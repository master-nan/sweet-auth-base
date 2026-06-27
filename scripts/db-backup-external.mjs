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
  }
}

function required(env, key) {
  const value = (env[key] || '').trim()
  if (!value) {
    throw new Error(`${key} must be set`)
  }
  return value
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
    return {
      action: 'backup',
      image,
      outputPath,
      manifestPath: backupManifestPath(outputPath),
      database,
      executable: 'docker',
      args: [
        'run',
        '--rm',
        '-e',
        'PGPASSWORD',
        image,
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
  return {
    action: 'restore',
    image,
    backupFile: resolvedBackupFile,
    database,
    executable: 'docker',
    args: [
      'run',
      '--rm',
      '-i',
      '-e',
      'PGPASSWORD',
      image,
      'psql',
      '--set',
      'ON_ERROR_STOP=on',
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
    },
    executable: plan.executable,
    args: plan.args,
    passwordEnv: 'PGPASSWORD',
  }
}

export function backupManifestPath(outputPath) {
  return `${outputPath}.manifest.json`
}

export function createBackupManifest(plan, { sizeBytes, sha256, createdAt = new Date() } = {}) {
  return {
    schema_version: 1,
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
    command: {
      executable: plan.executable,
      args: plan.args,
      passwordEnv: 'PGPASSWORD',
    },
  }
}

export function createRestoreEvidence(plan, backupEvidence, { envPath = '', targetPurpose = '', restoredAt = new Date() } = {}) {
  return {
    schema_version: 1,
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
  if (manifest.schema_version !== 1) {
    return `Backup manifest schema_version must be 1 for ${label}`
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

async function runBackup(plan) {
  await fs.promises.mkdir(path.dirname(plan.outputPath), { recursive: true })
  const output = fs.createWriteStream(plan.outputPath, { flags: 'wx', mode: 0o600 })
  try {
    await runProcess(plan, {
      env: { ...process.env, PGPASSWORD: plan.database.password },
      stdout: output,
      stdin: 'ignore',
    })
    await finished(output)
    const stat = await fs.promises.stat(plan.outputPath)
    const sha256 = await fileSha256(plan.outputPath)
    const manifest = createBackupManifest(plan, {
      sizeBytes: stat.size,
      sha256,
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
  const input = fs.createReadStream(plan.backupFile)
  await runProcess(plan, {
    env: { ...process.env, PGPASSWORD: plan.database.password },
    stdin: input,
    stdout: 'inherit',
  })
  return writeRestoreEvidence(plan, evidence, {
    envPath: options.envPath,
    targetPurpose: options.targetPurpose,
    outputDir: options.evidenceDir,
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
