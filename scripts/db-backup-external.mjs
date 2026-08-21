#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import crypto from 'node:crypto'
import { spawn } from 'node:child_process'
import { finished } from 'node:stream/promises'
import { fileURLToPath } from 'node:url'

import {
  parseEnvContent,
  validateExternalEnv,
  validateExternalEnvFileSecurity,
  validateExternalWriteTarget,
} from './preflight-external.mjs'

const DEFAULT_POSTGRES_CLIENT_IMAGE = 'postgres:16-alpine'
const DEFAULT_RESTORE_EVIDENCE_DIR = 'reports'
const RESTORE_CONFIRMATION = 'I_UNDERSTAND_THIS_OVERWRITES_DATA'
const PROJECT_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const EXTERNAL_COMPOSE_FILE = path.join(PROJECT_ROOT, 'docker-compose.external.yml')

function parseBoolean(value) {
  const normalized = String(value || '').toLowerCase().trim()
  if (['1', 'true', 'yes', 'y', 'on'].includes(normalized)) return true
  if (['0', 'false', 'no', 'n', 'off'].includes(normalized)) return false
  return null
}

function readExternalEnv(envPath) {
  const resolvedPath = path.resolve(envPath || '.env.external')
  if (!fs.existsSync(resolvedPath)) {
    throw new Error(`External env file not found: ${resolvedPath}`)
  }
  const stat = fs.statSync(resolvedPath)
  const fileProblems = validateExternalEnvFileSecurity(resolvedPath, stat)
  if (fileProblems.length > 0) {
    throw new Error(`External env file is not safe:\n- ${fileProblems.join('\n- ')}`)
  }
  return {
    env: parseEnvContent(fs.readFileSync(resolvedPath, 'utf8')),
    envPath: resolvedPath,
  }
}

function requireValidExternalEnv(env) {
  const result = validateExternalEnv(env, {
    allowNonProduction: parseBoolean(process.env.SWEET_ADMIN_PREFLIGHT_ALLOW_NON_PRODUCTION || '') === true,
    requireStartupWritesDisabled: true,
  })
  if (!result.ok) {
    throw new Error(`External env is not safe for database backup:\n- ${result.problems.join('\n- ')}`)
  }
  return result
}

export function mergeRuntimeOptions(env, runtimeEnv = process.env) {
  const merged = { ...env }
  for (const key of [
    'APP_DB_BACKUP_TARGET',
    'APP_DB_RESTORE_TARGET',
    'SWEET_ADMIN_DB_BACKUP_DIR',
    'SWEET_ADMIN_POSTGRES_CLIENT_IMAGE',
  ]) {
    if (runtimeEnv[key]) {
      merged[key] = runtimeEnv[key]
    }
  }
  return merged
}

function dbConfig(env, label) {
  if (label !== 'primary') {
    throw new Error('Only the primary PostgreSQL database can be backed up or restored')
  }
  const prefix = 'APP_DBS_PRIMARY'
  return {
    label,
    host: required(env, `${prefix}_HOST`),
    port: required(env, `${prefix}_PORT`),
    name: required(env, `${prefix}_NAME`),
    user: required(env, `${prefix}_USER`),
    password: required(env, `${prefix}_PASSWORD`),
    tls: {
      mode: required(env, `${prefix}_TLS_MODE`).toLowerCase(),
      rootCAFile: optional(env, `${prefix}_TLS_ROOT_CA_FILE`),
      certFile: optional(env, `${prefix}_TLS_CERT_FILE`),
      keyFile: optional(env, `${prefix}_TLS_KEY_FILE`),
    },
  }
}

function required(env, key) {
  const value = (env[key] || '').trim()
  if (!value) {
    throw new Error(`${key} must be set`)
  }
  return value
}

function optional(env, key) {
  return (env[key] || '').trim()
}

function postgresDockerInvocation(image, database, { interactive = false } = {}) {
  const args = ['run', '--rm']
  if (interactive) args.push('-i')

  const env = {
    ...process.env,
    PGPASSWORD: database.password,
    PGSSLMODE: database.tls.mode,
  }
  args.push('-e', 'PGPASSWORD', '-e', 'PGSSLMODE')

  const tlsFiles = [
    ['PGSSLROOTCERT', database.tls.rootCAFile, 'root-ca.pem'],
    ['PGSSLCERT', database.tls.certFile, 'client-cert.pem'],
    ['PGSSLKEY', database.tls.keyFile, 'client-key.pem'],
  ]
  for (const [key, configuredPath, filename] of tlsFiles) {
    if (!configuredPath) continue
    const source = path.resolve(configuredPath)
    const destination = `/run/sweet-admin-db-tls/${filename}`
    args.push('--mount', `type=bind,src=${source},dst=${destination},readonly`, '-e', key)
    env[key] = destination
  }
  args.push(image)
  return { args, env }
}

export function resolveBackupTargetNames(value = 'primary') {
  const normalized = String(value || 'primary').toLowerCase().trim()
  const targets = normalized
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
  if (targets.length === 0) return ['primary']
  for (const target of targets) {
    if (target !== 'primary') {
      throw new Error('APP_DB_BACKUP_TARGET must be primary')
    }
  }
  return [...new Set(targets)]
}

function uniqueDatabases(databases) {
  const seen = new Set()
  const result = []
  for (const database of databases) {
    const key = [database.host, database.port, database.name, database.user].join('\0')
    if (seen.has(key)) continue
    seen.add(key)
    result.push(database)
  }
  return result
}

function timestampForFile(date = new Date()) {
  const pad = (value) => String(value).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
    '-',
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join('')
}

function safeFilenamePart(value) {
  return String(value || '')
    .replace(/[^A-Za-z0-9_.-]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 80) || 'database'
}

export function createBackupPlans(env, options = {}) {
  const image = options.image || env.SWEET_ADMIN_POSTGRES_CLIENT_IMAGE || DEFAULT_POSTGRES_CLIENT_IMAGE
  const outputDir = path.resolve(options.outputDir || env.SWEET_ADMIN_DB_BACKUP_DIR || 'backups')
  const timestamp = options.timestamp || timestampForFile(options.now || new Date())
  const targetNames = resolveBackupTargetNames(options.target || env.APP_DB_BACKUP_TARGET || 'primary')
  const databases = uniqueDatabases(targetNames.map((target) => dbConfig(env, target)))

  return databases.map((database) => {
    const outputPath = path.join(
      outputDir,
      `${timestamp}-${safeFilenamePart(database.label)}-${safeFilenamePart(database.name)}.sql`,
    )
    const invocation = postgresDockerInvocation(image, database)
    return {
      action: 'backup',
      image,
      outputPath,
      manifestPath: backupManifestPath(outputPath),
      database,
      executable: 'docker',
      clientEnv: invocation.env,
      args: [
        ...invocation.args,
        'pg_dump',
        '--format=plain',
        '--no-owner',
        '--no-privileges',
        '--clean',
        '--if-exists',
        '--no-password',
        '-h',
        database.host,
        '-p',
        database.port,
        '-U',
        database.user,
        '-d',
        database.name,
      ],
    }
  })
}

export function createRestorePlan(env, backupFile, options = {}) {
  const resolvedBackupFile = path.resolve(backupFile || '')
  if (!backupFile) {
    throw new Error('BACKUP_FILE must be provided for restore')
  }
  const image = options.image || env.SWEET_ADMIN_POSTGRES_CLIENT_IMAGE || DEFAULT_POSTGRES_CLIENT_IMAGE
  const target = options.target || env.APP_DB_RESTORE_TARGET || 'primary'
  if (String(target).toLowerCase().trim() !== 'primary') {
    throw new Error('APP_DB_RESTORE_TARGET must be primary')
  }
  const database = dbConfig(env, String(target).toLowerCase().trim())
  const invocation = postgresDockerInvocation(image, database, { interactive: true })
  return {
    action: 'restore',
    image,
    backupFile: resolvedBackupFile,
    database,
    executable: 'docker',
    clientEnv: invocation.env,
    args: [
      ...invocation.args,
      'psql',
      '--set',
      'ON_ERROR_STOP=on',
      '--single-transaction',
      '--no-password',
      '-h',
      database.host,
      '-p',
      database.port,
      '-U',
      database.user,
      '-d',
      database.name,
    ],
  }
}

export function redactPlan(plan) {
  return {
    action: plan.action,
    image: plan.image,
    outputPath: plan.outputPath,
    manifestPath: plan.manifestPath,
    backupFile: plan.backupFile,
    database: {
      label: plan.database.label,
      host: plan.database.host,
      port: plan.database.port,
      name: plan.database.name,
      user: plan.database.user,
      tlsMode: plan.database.tls.mode,
    },
    executable: plan.executable,
    args: plan.args,
    passwordEnv: 'PGPASSWORD',
  }
}

export function backupManifestPath(outputPath) {
  return `${outputPath}.manifest.json`
}

function createPostgresQueryPlan(plan, sql) {
  const invocation = postgresDockerInvocation(plan.image, plan.database)
  return {
    action: 'database inspection',
    executable: 'docker',
    clientEnv: invocation.env,
    args: [
      ...invocation.args,
      'psql',
      '--no-password',
      '--tuples-only',
      '--no-align',
      '--field-separator',
      '\t',
      '-h',
      plan.database.host,
      '-p',
      plan.database.port,
      '-U',
      plan.database.user,
      '-d',
      plan.database.name,
      '--command',
      sql,
    ],
  }
}

export function parseDatabaseIdentityOutput(output) {
  const value = String(output || '').trim()
  if (!value) throw new Error('Database identity query returned no data')
  let identity
  try {
    identity = JSON.parse(value)
  } catch (error) {
    throw new Error(`Database identity query returned invalid JSON: ${error.message}`)
  }
  for (const key of ['database_name', 'database_user', 'database_schema', 'server_version_num', 'database_oid']) {
    if (String(identity[key] ?? '').trim() === '') {
      throw new Error(`Database identity is missing ${key}`)
    }
  }
  return identity
}

export function parseMigrationLedgerRows(output) {
  const value = String(output || '').trim()
  if (!value) return []
  return value.split(/\r?\n/).map((line) => {
    const [version, key, checksum, appliedAt] = line.split('\t')
    if (!/^-?\d+$/.test(version || '') || !key || !/^[a-f0-9]{64}$/i.test(checksum || '') || !appliedAt) {
      throw new Error('Migration ledger query returned an invalid row')
    }
    return {
      version,
      key,
      checksum: checksum.toLowerCase(),
      applied_at: appliedAt,
    }
  })
}

export function createMigrationLedgerSummary(entries, { exists = true } = {}) {
  const normalized = [...entries]
    .map((entry) => ({
      version: String(entry.version),
      key: String(entry.key),
      checksum: String(entry.checksum).toLowerCase(),
      applied_at: String(entry.applied_at),
    }))
    .sort((left, right) => BigInt(left.version) < BigInt(right.version) ? -1 : BigInt(left.version) > BigInt(right.version) ? 1 : 0)
  const first = normalized[0] || null
  const latest = normalized.at(-1) || null
  return {
    table: 'schema_migration',
    exists,
    entry_count: normalized.length,
    first_version: first?.version || null,
    latest_version: latest?.version || null,
    latest_key: latest?.key || null,
    latest_applied_at: latest?.applied_at || null,
    entries_sha256: exists
      ? crypto.createHash('sha256').update(JSON.stringify(normalized)).digest('hex')
      : null,
  }
}

export async function inspectDatabaseState(plan) {
  const identityQuery = `SELECT json_build_object(
    'database_name', current_database(),
    'database_user', current_user,
    'database_schema', current_schema(),
    'server_version_num', current_setting('server_version_num'),
    'server_address', inet_server_addr()::text,
    'server_port', inet_server_port(),
    'database_oid', (SELECT oid::text FROM pg_database WHERE datname = current_database())
  )::text;`
  const identityOutput = await runCapturedProcess(createPostgresQueryPlan(plan, identityQuery))
  const identity = parseDatabaseIdentityOutput(identityOutput)

  const ledgerExistsOutput = await runCapturedProcess(createPostgresQueryPlan(
    plan,
    "SELECT COALESCE(to_regclass('schema_migration')::text, '');",
  ))
  const ledgerExists = ledgerExistsOutput.trim() !== ''
  let entries = []
  if (ledgerExists) {
    const ledgerOutput = await runCapturedProcess(createPostgresQueryPlan(
      plan,
      `SELECT version::text, key, checksum,
        to_char(applied_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
      FROM schema_migration ORDER BY version;`,
    ))
    entries = parseMigrationLedgerRows(ledgerOutput)
  }
  return {
    identity,
    migrationLedger: createMigrationLedgerSummary(entries, { exists: ledgerExists }),
  }
}

export function createBackupManifest(plan, {
  sizeBytes,
  sha256,
  databaseIdentity,
  migrationLedger,
  createdAt = new Date(),
} = {}) {
  return {
    schema_version: 2,
    action: 'backup',
    created_at: createdAt instanceof Date ? createdAt.toISOString() : String(createdAt || ''),
    image: plan.image,
    output_path: plan.outputPath,
    size_bytes: Number(sizeBytes || 0),
    sha256,
    database: {
      label: plan.database.label,
      host: plan.database.host,
      port: plan.database.port,
      name: plan.database.name,
      user: plan.database.user,
    },
    database_identity: databaseIdentity,
    migration_ledger: migrationLedger,
    command: {
      executable: plan.executable,
      args: plan.args,
      passwordEnv: 'PGPASSWORD',
    },
  }
}

export function createRestoreEvidence(plan, backupEvidence, {
  envPath = '',
  targetPurpose = '',
  verification = null,
  restoredAt = new Date(),
} = {}) {
  return {
    schema_version: 2,
    action: 'restore',
    restored_at: restoredAt instanceof Date ? restoredAt.toISOString() : String(restoredAt || ''),
    env_path: envPath ? path.resolve(envPath) : null,
    target_purpose: targetPurpose || null,
    image: plan.image,
    backup: {
      path: backupEvidence.backupFile,
      manifest_path: backupEvidence.manifestPath,
      size_bytes: backupEvidence.sizeBytes,
      sha256: backupEvidence.sha256,
      database: backupEvidence.manifest?.database
        ? {
            label: backupEvidence.manifest.database.label,
            host: backupEvidence.manifest.database.host,
            port: backupEvidence.manifest.database.port,
            name: backupEvidence.manifest.database.name,
            user: backupEvidence.manifest.database.user,
          }
        : null,
    },
    target_database: {
      label: plan.database.label,
      host: plan.database.host,
      port: plan.database.port,
      name: plan.database.name,
      user: plan.database.user,
    },
    command: {
      executable: plan.executable,
      args: plan.args,
      passwordEnv: 'PGPASSWORD',
    },
    post_restore: verification,
  }
}

export function restoreEvidenceName(plan, backupEvidence, restoredAt = new Date()) {
  const timestamp = timestampForFile(restoredAt instanceof Date ? restoredAt : new Date())
  const hashPrefix = String(backupEvidence.sha256 || '').slice(0, 12) || 'nohash'
  return `${timestamp}-restore-${safeFilenamePart(plan.database.label)}-${safeFilenamePart(plan.database.name)}-${hashPrefix}.json`
}

export async function writeRestoreEvidence(plan, backupEvidence, options = {}) {
  const outputDir = path.resolve(options.outputDir || DEFAULT_RESTORE_EVIDENCE_DIR)
  const restoredAt = options.restoredAt || new Date()
  const evidence = createRestoreEvidence(plan, backupEvidence, {
    envPath: options.envPath,
    targetPurpose: options.targetPurpose,
    verification: options.verification,
    restoredAt,
  })
  const outputPath = path.join(outputDir, restoreEvidenceName(plan, backupEvidence, restoredAt))
  await fs.promises.mkdir(outputDir, { recursive: true })
  await fs.promises.writeFile(outputPath, `${JSON.stringify(evidence, null, 2)}\n`, {
    flag: 'wx',
    mode: 0o600,
  })
  return { outputPath, evidence }
}

export async function fileSha256(filePath) {
  const hash = crypto.createHash('sha256')
  const input = fs.createReadStream(filePath)
  input.on('data', (chunk) => hash.update(chunk))
  await finished(input)
  return hash.digest('hex')
}

export function validateRestoreBackupManifest({ backupFile, manifest, stat, manifestStat, sha256 }) {
  const label = backupFile || 'backup file'
  if (!manifest || typeof manifest !== 'object') {
    return `Backup manifest is missing or invalid for ${label}`
  }
  if (![1, 2].includes(manifest.schema_version)) {
    return `Backup manifest schema_version must be 1 or 2 for ${label}`
  }
  if (manifest.action !== 'backup') {
    return `Backup manifest action must be backup for ${label}`
  }
  if (!/^[a-f0-9]{64}$/i.test(String(manifest.sha256 || ''))) {
    return `Backup manifest sha256 is missing or invalid for ${label}`
  }
  if (Number(manifest.size_bytes || 0) !== Number(stat?.size || 0)) {
    return `Backup manifest size does not match ${label}`
  }
  if (String(manifest.sha256).toLowerCase() !== String(sha256 || '').toLowerCase()) {
    return `Backup manifest sha256 does not match ${label}`
  }
  if (manifest.output_path && path.basename(manifest.output_path) !== path.basename(label)) {
    return `Backup manifest output_path does not match ${label}`
  }
  if (!manifest.database?.label || !manifest.database?.name || !manifest.database?.host) {
    return `Backup manifest database summary is incomplete for ${label}`
  }
  if (manifest.schema_version === 2) {
    const identity = manifest.database_identity
    if (
      !identity?.database_name
      || !identity?.database_user
      || !identity?.database_schema
      || !identity?.server_version_num
      || !identity?.database_oid
    ) {
      return `Backup manifest database identity is incomplete for ${label}`
    }
    if (identity.database_name !== manifest.database.name) {
      return `Backup manifest database identity does not match its configured database for ${label}`
    }
    const ledger = manifest.migration_ledger
    if (
      !ledger
      || ledger.table !== 'schema_migration'
      || typeof ledger.exists !== 'boolean'
      || !Number.isInteger(ledger.entry_count)
      || ledger.entry_count < 0
    ) {
      return `Backup manifest migration ledger summary is incomplete for ${label}`
    }
    if (ledger.exists && !/^[a-f0-9]{64}$/i.test(String(ledger.entries_sha256 || ''))) {
      return `Backup manifest migration ledger digest is invalid for ${label}`
    }
    if (!ledger.exists && ledger.entry_count !== 0) {
      return `Backup manifest missing migration ledger must have zero entries for ${label}`
    }
  }
  if (!Number.isFinite(Date.parse(manifest.created_at || ''))) {
    return `Backup manifest created_at is invalid for ${label}`
  }
  if (Number.isInteger(stat?.mode) && ((stat.mode & 0o777) & 0o077) !== 0) {
    return `Backup file permissions must be owner-only for ${label}`
  }
  if (Number.isInteger(manifestStat?.mode) && ((manifestStat.mode & 0o777) & 0o077) !== 0) {
    return `Backup manifest permissions must be owner-only for ${label}`
  }
  return ''
}

export async function verifyRestoreBackupEvidence(backupFile) {
  if (!backupFile) {
    throw new Error('BACKUP_FILE must be provided for backup verification')
  }
  const resolvedBackupFile = path.resolve(backupFile)
  if (!fs.existsSync(resolvedBackupFile)) {
    throw new Error(`Backup file not found: ${resolvedBackupFile}`)
  }
  const manifestPath = backupManifestPath(resolvedBackupFile)
  if (!fs.existsSync(manifestPath)) {
    throw new Error(`Backup manifest not found: ${manifestPath}`)
  }

  const stat = await fs.promises.stat(resolvedBackupFile)
  if (!stat.isFile()) {
    throw new Error(`Backup file is not a regular file: ${resolvedBackupFile}`)
  }
  const manifestStat = await fs.promises.stat(manifestPath)
  if (!manifestStat.isFile()) {
    throw new Error(`Backup manifest is not a regular file: ${manifestPath}`)
  }

  let manifest
  try {
    manifest = JSON.parse(await fs.promises.readFile(manifestPath, 'utf8'))
  } catch (error) {
    throw new Error(`Backup manifest is not valid JSON: ${manifestPath}: ${error.message}`)
  }
  const sha256 = await fileSha256(resolvedBackupFile)
  const problem = validateRestoreBackupManifest({
    backupFile: resolvedBackupFile,
    manifest,
    stat,
    manifestStat,
    sha256,
  })
  if (problem) {
    throw new Error(problem)
  }

  return {
    backupFile: resolvedBackupFile,
    manifestPath,
    sizeBytes: stat.size,
    sha256,
    manifest,
  }
}

export function validateRestoreTarget(plan, backupEvidence) {
  const sourceName = backupEvidence.manifest?.database_identity?.database_name
    || backupEvidence.manifest?.database?.name
  if (!sourceName) return 'Backup manifest does not identify its source database'
  if (sourceName !== plan.database.name) {
    return `Backup source database ${sourceName} does not match restore target ${plan.database.name}`
  }
  return ''
}

export function validateRestoredMigrationLedger(manifest, targetLedger) {
  if (manifest?.schema_version !== 2) return ''
  const sourceLedger = manifest.migration_ledger
  if (!sourceLedger || !targetLedger) return 'Migration ledger summary is unavailable after restore'
  if (sourceLedger.exists !== targetLedger.exists) {
    return 'Restored migration ledger existence does not match the backup manifest'
  }
  if (sourceLedger.entry_count !== targetLedger.entry_count) {
    return 'Restored migration ledger entry count does not match the backup manifest'
  }
  if (sourceLedger.entries_sha256 !== targetLedger.entries_sha256) {
    return 'Restored migration ledger digest does not match the backup manifest'
  }
  return ''
}

export function validateReadinessPayload(payload) {
  if (!payload || payload.status !== 'ready') {
    return 'Readiness status must be ready after restore'
  }
  if (!payload.components || typeof payload.components !== 'object') {
    return 'Readiness response must include dependency components'
  }
  const components = Object.entries(payload.components)
  if (components.length === 0 || components.some(([, status]) => status?.ok !== true)) {
    return 'Every readiness dependency component must be healthy after restore'
  }
  return ''
}

async function runDatabasePreflight(envPath) {
  const plan = {
    action: 'post-restore database preflight',
    executable: 'docker',
    args: [
      'compose',
      '--env-file',
      envPath,
      '-f',
      EXTERNAL_COMPOSE_FILE,
      'run',
      '--rm',
      '--no-deps',
      '-e',
      'APP_DB_PREFLIGHT_REQUIRE_MIGRATED=true',
      'backend',
      '/app/db-preflight',
    ],
  }
  await runProcess(plan, {
    env: process.env,
    stdin: 'ignore',
    stdout: 'inherit',
  })
}

export function resolveReadinessURL(baseURL) {
  let configuredURL
  try {
    configuredURL = new URL(required({ SWEET_ADMIN_HEALTH_BASE_URL: baseURL }, 'SWEET_ADMIN_HEALTH_BASE_URL'))
  } catch (error) {
    throw new Error(`SWEET_ADMIN_HEALTH_BASE_URL must be a valid http(s) URL: ${error.message}`)
  }
  if (
    !['http:', 'https:'].includes(configuredURL.protocol)
    || configuredURL.username
    || configuredURL.password
    || configuredURL.search
    || configuredURL.hash
  ) {
    throw new Error('SWEET_ADMIN_HEALTH_BASE_URL must be an http(s) URL without credentials, query, or fragment')
  }
  return new URL('/readyz', configuredURL)
}

async function checkReadiness(baseURL) {
  const readinessURL = resolveReadinessURL(baseURL)

  const response = await fetch(readinessURL, {
    headers: { accept: 'application/json' },
    redirect: 'error',
    signal: AbortSignal.timeout(15_000),
  })
  let payload
  try {
    payload = await response.json()
  } catch (error) {
    throw new Error(`Readiness response is not valid JSON: ${error.message}`)
  }
  const problem = validateReadinessPayload(payload)
  if (!response.ok || problem) {
    throw new Error(problem || `Readiness endpoint returned HTTP ${response.status}`)
  }
  return {
    status: payload.status,
    url: readinessURL.toString(),
    checked_at: new Date().toISOString(),
    components: payload.components,
  }
}

async function verifyRestoredDatabase(plan, backupEvidence, options) {
  const targetState = await inspectDatabaseState(plan)
  const ledgerProblem = validateRestoredMigrationLedger(
    backupEvidence.manifest,
    targetState.migrationLedger,
  )
  if (ledgerProblem) throw new Error(ledgerProblem)

  await runDatabasePreflight(options.envPath)
  const readiness = await checkReadiness(options.healthBaseURL)
  return {
    database_identity: targetState.identity,
    migration_ledger: targetState.migrationLedger,
    db_preflight: {
      status: 'passed',
      require_migrated: true,
    },
    readiness,
  }
}

async function runBackup(plan) {
  await fs.promises.mkdir(path.dirname(plan.outputPath), { recursive: true })
  const output = fs.createWriteStream(plan.outputPath, { flags: 'wx', mode: 0o600 })
  try {
    const databaseState = await inspectDatabaseState(plan)
    await runProcess(plan, {
      env: plan.clientEnv,
      stdout: output,
      stdin: 'ignore',
    })
    await finished(output)
    const stat = await fs.promises.stat(plan.outputPath)
    const sha256 = await fileSha256(plan.outputPath)
    const manifest = createBackupManifest(plan, {
      sizeBytes: stat.size,
      sha256,
      databaseIdentity: databaseState.identity,
      migrationLedger: databaseState.migrationLedger,
    })
    await fs.promises.writeFile(plan.manifestPath, `${JSON.stringify(manifest, null, 2)}\n`, {
      flag: 'wx',
      mode: 0o600,
    })
    return {
      backupPath: plan.outputPath,
      manifestPath: plan.manifestPath,
    }
  } catch (error) {
    output.destroy()
    await fs.promises.unlink(plan.outputPath).catch(() => {})
    await fs.promises.unlink(plan.manifestPath).catch(() => {})
    throw error
  }
}

async function runRestore(plan, options = {}) {
  const evidence = await verifyRestoreBackupEvidence(plan.backupFile)
  console.log(`Restore backup evidence verified: ${evidence.manifestPath}`)
  const targetProblem = validateRestoreTarget(plan, evidence)
  if (targetProblem) throw new Error(targetProblem)
  const input = fs.createReadStream(plan.backupFile)
  await runProcess(plan, {
    env: plan.clientEnv,
    stdin: input,
    stdout: 'inherit',
  })
  const verification = await verifyRestoredDatabase(plan, evidence, {
    envPath: options.envPath,
    healthBaseURL: options.healthBaseURL,
  })
  return writeRestoreEvidence(plan, evidence, {
    envPath: options.envPath,
    targetPurpose: options.targetPurpose,
    outputDir: options.evidenceDir,
    verification,
  })
}

function runCapturedProcess(plan) {
  return new Promise((resolve, reject) => {
    const chunks = []
    const child = spawn(plan.executable, plan.args, {
      env: plan.clientEnv,
      stdio: ['ignore', 'pipe', 'inherit'],
    })
    child.stdout.on('data', (chunk) => chunks.push(chunk))
    child.on('error', reject)
    child.on('close', (code) => {
      if (code === 0) {
        resolve(Buffer.concat(chunks).toString('utf8'))
        return
      }
      reject(new Error(`${plan.action} failed with exit code ${code}`))
    })
  })
}

function runProcess(plan, options) {
  return new Promise((resolve, reject) => {
    const child = spawn(plan.executable, plan.args, {
      env: options.env,
      stdio: [
        options.stdin === 'ignore' ? 'ignore' : 'pipe',
        options.stdout === 'inherit' ? 'inherit' : 'pipe',
        'inherit',
      ],
    })

    if (options.stdin && options.stdin !== 'ignore') {
      options.stdin.pipe(child.stdin)
      options.stdin.on('error', reject)
    }
    if (options.stdout && options.stdout !== 'inherit') {
      child.stdout.pipe(options.stdout)
      options.stdout.on('error', reject)
    }
    child.on('error', reject)
    child.on('close', (code) => {
      if (code === 0) {
        resolve()
        return
      }
      reject(new Error(`${plan.action} failed with exit code ${code}`))
    })
  })
}

async function main() {
  const command = process.argv[2] || 'backup'
  if (command === 'verify') {
    const evidence = await verifyRestoreBackupEvidence(process.argv[4] || process.argv[3] || process.env.BACKUP_FILE)
    console.log(`Backup verified: ${evidence.backupFile}`)
    console.log(`Backup manifest verified: ${evidence.manifestPath}`)
    console.log(`Backup size: ${evidence.sizeBytes} bytes`)
    console.log(`Backup sha256: ${evidence.sha256}`)
    return
  }

  const envFile =
    process.argv[3] ||
    process.env.SWEET_ADMIN_EXTERNAL_ENV_FILE ||
    '.env.external'
  const { env, envPath } = readExternalEnv(envFile)
  const validation = requireValidExternalEnv(env)
  const runtimeEnv = mergeRuntimeOptions(env)
  for (const warning of validation.warnings) {
    console.warn(`Warning: ${warning}`)
  }

  if (command === 'plan') {
    const plans = createBackupPlans(runtimeEnv, {
      outputDir: process.argv[4] || process.env.BACKUP_DIR,
    })
    console.log(JSON.stringify({ envPath, plans: plans.map(redactPlan) }, null, 2))
    return
  }

  if (command === 'backup') {
    const plans = createBackupPlans(runtimeEnv, {
      outputDir: process.argv[4] || process.env.BACKUP_DIR,
    })
    for (const plan of plans) {
      console.log(`Backing up ${plan.database.label}/${plan.database.name} from ${plan.database.host}:${plan.database.port}`)
      const result = await runBackup(plan)
      console.log(`Backup written: ${plan.outputPath}`)
      console.log(`Backup manifest written: ${result.manifestPath}`)
    }
    return
  }

  if (command === 'restore') {
    if (process.env.CONFIRM_EXTERNAL_RESTORE !== RESTORE_CONFIRMATION) {
      throw new Error(`Refusing restore without CONFIRM_EXTERNAL_RESTORE=${RESTORE_CONFIRMATION}`)
    }
    const writeProblems = validateExternalWriteTarget(runtimeEnv, {
      operation: 'restore',
      productionConfirmation: process.env.CONFIRM_EXTERNAL_PRODUCTION_WRITE,
    })
    if (writeProblems.length > 0) {
      throw new Error(`External restore target is not safe:\n- ${writeProblems.join('\n- ')}`)
    }
    const plan = createRestorePlan(runtimeEnv, process.argv[4] || process.env.BACKUP_FILE)
    console.log(`Restoring ${plan.backupFile} into ${plan.database.label}/${plan.database.name} at ${plan.database.host}:${plan.database.port}`)
    const result = await runRestore(plan, {
      envPath,
      targetPurpose: runtimeEnv.SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE,
      healthBaseURL: runtimeEnv.SWEET_ADMIN_HEALTH_BASE_URL,
      evidenceDir:
        process.env.RESTORE_EVIDENCE_DIR ||
        process.env.SWEET_ADMIN_RESTORE_EVIDENCE_DIR ||
        DEFAULT_RESTORE_EVIDENCE_DIR,
    })
    console.log('Restore completed')
    console.log(`Restore evidence written: ${result.outputPath}`)
    return
  }

  throw new Error('Usage: node scripts/db-backup-external.mjs backup|plan|restore|verify [env-file] [backup-dir-or-file]')
}

const currentFile = fileURLToPath(import.meta.url)
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
