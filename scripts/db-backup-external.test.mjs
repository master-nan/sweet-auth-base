import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import crypto from 'node:crypto'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import {
  backupManifestPath,
  createBackupManifest,
  createBackupPlans,
  createMigrationLedgerSummary,
  createRestoreEvidence,
  createRestorePlan,
  mergeRuntimeOptions,
  parseDatabaseIdentityOutput,
  parseMigrationLedgerRows,
  redactPlan,
  resolveReadinessURL,
  resolveBackupTargetNames,
  validateReadinessPayload,
  validateRestoredMigrationLedger,
  validateRestoreBackupManifest,
  validateRestoreTarget,
  verifyRestoreBackupEvidence,
  writeRestoreEvidence,
} from './db-backup-external.mjs'

const env = {
  APP_DBS_PRIMARY_HOST: 'postgres.primary.internal',
  APP_DBS_PRIMARY_PORT: '5432',
  APP_DBS_PRIMARY_NAME: 'sweet_admin',
  APP_DBS_PRIMARY_USER: 'sweet_admin_app',
  APP_DBS_PRIMARY_PASSWORD: 'PrimaryPassword_2026!',
  APP_DBS_PRIMARY_TLS_MODE: 'verify-full',
  APP_DBS_PRIMARY_TLS_ROOT_CA_FILE: '',
  APP_DBS_PRIMARY_TLS_CERT_FILE: '',
  APP_DBS_PRIMARY_TLS_KEY_FILE: '',
}

const databaseIdentity = {
  database_name: 'sweet_admin',
  database_user: 'sweet_admin_app',
  database_schema: 'public',
  server_version_num: '160010',
  server_address: '10.0.0.10',
  server_port: 5432,
  database_oid: '16384',
}

const migrationLedger = createMigrationLedgerSummary([
  {
    version: '202606070001',
    key: 'baseline',
    checksum: 'b'.repeat(64),
    applied_at: '2026-06-07 12:00:00+00',
  },
])

test('resolveBackupTargetNames supports the primary PostgreSQL database only', () => {
  assert.deepEqual(resolveBackupTargetNames(), ['primary'])
  assert.deepEqual(resolveBackupTargetNames('primary,primary'), ['primary'])
  assert.throws(() => resolveBackupTargetNames('both'), /APP_DB_BACKUP_TARGET/)
  assert.throws(() => resolveBackupTargetNames('prod'), /APP_DB_BACKUP_TARGET/)
})

test('createBackupPlans builds safe pg_dump plans without password arguments', () => {
  const plans = createBackupPlans(
    {
      ...env,
      APP_DB_BACKUP_TARGET: 'primary',
      SWEET_ADMIN_POSTGRES_CLIENT_IMAGE: 'postgres:16-alpine',
    },
    {
      outputDir: '/tmp/sweet-admin-backups',
      timestamp: '20260607-120000',
    },
  )

  assert.equal(plans.length, 1)
  assert.equal(plans[0].outputPath, path.join('/tmp/sweet-admin-backups', '20260607-120000-primary-sweet_admin.sql'))
  assert.equal(
    plans[0].manifestPath,
    path.join('/tmp/sweet-admin-backups', '20260607-120000-primary-sweet_admin.sql.manifest.json'),
  )
  assert.equal(plans[0].database.password, 'PrimaryPassword_2026!')
  assert.equal(plans[0].args.includes('PrimaryPassword_2026!'), false)
  assert.deepEqual(plans[0].args.slice(0, 7), ['run', '--rm', '-e', 'PGPASSWORD', '-e', 'PGSSLMODE', 'postgres:16-alpine'])
  assert.equal(plans[0].args.includes('pg_dump'), true)
  assert.equal(plans[0].args.includes('--no-owner'), true)
  assert.equal(plans[0].clientEnv.PGSSLMODE, 'verify-full')
  assert.equal(plans[0].args.at(-1), 'sweet_admin')

  const redacted = redactPlan(plans[0])
  assert.equal('password' in redacted.database, false)
  assert.equal(redacted.manifestPath.endsWith('.manifest.json'), true)
  assert.equal(JSON.stringify(redacted).includes('PrimaryPassword_2026!'), false)
})

test('createRestorePlan targets one database and keeps password out of arguments', () => {
  const plan = createRestorePlan(
    {
      ...env,
      APP_DB_RESTORE_TARGET: 'primary',
    },
    '/tmp/sweet-admin-backups/primary.sql',
  )

  assert.equal(plan.database.label, 'primary')
  assert.equal(plan.backupFile, '/tmp/sweet-admin-backups/primary.sql')
  assert.equal(plan.args.includes('PrimaryPassword_2026!'), false)
  assert.equal(plan.args.includes('-i'), true)
  assert.equal(plan.args.includes('psql'), true)
  assert.equal(plan.args.includes('--single-transaction'), true)
  assert.equal(plan.args.at(-1), 'sweet_admin')
})

test('mergeRuntimeOptions lets command environment override backup and restore options', () => {
  const merged = mergeRuntimeOptions(
    {
      APP_DB_BACKUP_TARGET: 'primary',
      APP_DB_RESTORE_TARGET: 'primary',
      SWEET_ADMIN_POSTGRES_CLIENT_IMAGE: 'postgres:16-alpine',
    },
    {
      APP_DB_BACKUP_TARGET: 'primary',
      APP_DB_RESTORE_TARGET: 'primary',
      SWEET_ADMIN_POSTGRES_CLIENT_IMAGE: 'postgres:17-alpine',
    },
  )

  assert.equal(merged.APP_DB_BACKUP_TARGET, 'primary')
  assert.equal(merged.APP_DB_RESTORE_TARGET, 'primary')
  assert.equal(merged.SWEET_ADMIN_POSTGRES_CLIENT_IMAGE, 'postgres:17-alpine')
})

test('createBackupManifest records backup evidence without secrets', () => {
  const [plan] = createBackupPlans(env, {
    outputDir: '/tmp/sweet-admin-backups',
    timestamp: '20260607-120000',
  })

  const manifest = createBackupManifest(plan, {
    sizeBytes: 2048,
    sha256: 'a'.repeat(64),
    databaseIdentity,
    migrationLedger,
    createdAt: new Date('2026-06-07T12:30:00.000Z'),
  })

  assert.equal(backupManifestPath(plan.outputPath), plan.manifestPath)
  assert.equal(manifest.schema_version, 2)
  assert.equal(manifest.action, 'backup')
  assert.equal(manifest.created_at, '2026-06-07T12:30:00.000Z')
  assert.equal(manifest.size_bytes, 2048)
  assert.equal(manifest.sha256, 'a'.repeat(64))
  assert.equal(manifest.database.name, 'sweet_admin')
  assert.deepEqual(manifest.database_identity, databaseIdentity)
  assert.equal(manifest.migration_ledger.entries_sha256, migrationLedger.entries_sha256)
  assert.equal(manifest.command.passwordEnv, 'PGPASSWORD')
  assert.equal(JSON.stringify(manifest).includes('PrimaryPassword_2026!'), false)
})

test('database identity and migration ledger parsers produce a stable summary', () => {
  assert.deepEqual(parseDatabaseIdentityOutput(JSON.stringify(databaseIdentity)), databaseIdentity)

  const rows = parseMigrationLedgerRows([
    `202606070002\tsecond\t${'c'.repeat(64)}\t2026-06-07 12:05:00+00`,
    `202606070001\tbaseline\t${'b'.repeat(64)}\t2026-06-07 12:00:00+00`,
  ].join('\n'))
  const summary = createMigrationLedgerSummary(rows)
  const reversedSummary = createMigrationLedgerSummary([...rows].reverse())

  assert.equal(summary.exists, true)
  assert.equal(summary.entry_count, 2)
  assert.equal(summary.first_version, '202606070001')
  assert.equal(summary.latest_version, '202606070002')
  assert.equal(summary.entries_sha256, reversedSummary.entries_sha256)
  assert.match(summary.entries_sha256, /^[a-f0-9]{64}$/)
})

test('restore target, migration ledger, and readiness checks fail closed', () => {
  const plan = createRestorePlan(env, '/tmp/sweet-admin-backups/primary.sql')
  const backupEvidence = {
    manifest: {
      schema_version: 2,
      database: { name: 'sweet_admin' },
      database_identity: databaseIdentity,
      migration_ledger: migrationLedger,
    },
  }

  assert.equal(validateRestoreTarget(plan, backupEvidence), '')
  assert.match(
    validateRestoreTarget(
      plan,
      { manifest: { database_identity: { database_name: 'another_database' } } },
    ),
    /does not match restore target/,
  )
  assert.equal(validateRestoredMigrationLedger(backupEvidence.manifest, migrationLedger), '')
  assert.match(
    validateRestoredMigrationLedger(
      backupEvidence.manifest,
      { ...migrationLedger, entries_sha256: 'f'.repeat(64) },
    ),
    /digest does not match/,
  )
  assert.equal(validateReadinessPayload({
    status: 'ready',
    components: { db_primary: { ok: true }, redis: { ok: true } },
  }), '')
  assert.match(
    validateReadinessPayload({ status: 'ready', components: { redis: { ok: false } } }),
    /Every readiness dependency component/,
  )
  assert.equal(resolveReadinessURL('https://health.company.test').toString(), 'https://health.company.test/readyz')
  assert.throws(
    () => resolveReadinessURL('https://operator:credential@health.company.test?token=value'),
    /without credentials, query, or fragment/,
  )
})

test('createRestoreEvidence records restore rehearsal details without secrets', () => {
  const restoredAt = new Date(2026, 5, 7, 13, 0, 0)
  const plan = createRestorePlan(
    {
      ...env,
      APP_DB_RESTORE_TARGET: 'primary',
    },
    '/tmp/sweet-admin-backups/primary.sql',
  )
  const evidence = createRestoreEvidence(
    plan,
    {
      backupFile: '/tmp/sweet-admin-backups/primary.sql',
      manifestPath: '/tmp/sweet-admin-backups/primary.sql.manifest.json',
      sizeBytes: 4096,
      sha256: 'c'.repeat(64),
      manifest: {
        database: {
          label: 'primary',
          host: 'postgres.source.internal',
          port: '5432',
          name: 'sweet_admin',
          user: 'dump_user',
        },
      },
    },
    {
      envPath: '/secure/.env.external',
      targetPurpose: 'staging',
      restoredAt,
      verification: {
        db_preflight: { status: 'passed', require_migrated: true },
        readiness: { status: 'ready' },
      },
    },
  )

  assert.equal(evidence.schema_version, 2)
  assert.equal(evidence.action, 'restore')
  assert.equal(evidence.restored_at, restoredAt.toISOString())
  assert.equal(evidence.target_purpose, 'staging')
  assert.equal(evidence.backup.sha256, 'c'.repeat(64))
  assert.equal(evidence.backup.database.name, 'sweet_admin')
  assert.equal(evidence.target_database.name, 'sweet_admin')
  assert.equal(evidence.command.passwordEnv, 'PGPASSWORD')
  assert.equal(evidence.post_restore.db_preflight.status, 'passed')
  assert.equal(evidence.post_restore.readiness.status, 'ready')
  assert.equal(JSON.stringify(evidence).includes('PrimaryPassword_2026!'), false)
})

test('writeRestoreEvidence writes owner-only restore evidence report', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-restore-evidence-'))
  try {
    const restoredAt = new Date(2026, 5, 7, 13, 0, 0)
    const plan = createRestorePlan(env, '/tmp/sweet-admin-backups/primary.sql')
    const backupEvidence = {
      backupFile: '/tmp/sweet-admin-backups/primary.sql',
      manifestPath: '/tmp/sweet-admin-backups/primary.sql.manifest.json',
      sizeBytes: 4096,
      sha256: 'd'.repeat(64),
      manifest: {
        database: {
          label: 'primary',
          host: 'postgres.source.internal',
          port: '5432',
          name: 'sweet_admin',
          user: 'dump_user',
        },
      },
    }

    const result = await writeRestoreEvidence(plan, backupEvidence, {
      outputDir: dir,
      envPath: '/secure/.env.external',
      targetPurpose: 'staging',
      restoredAt,
    })

    assert.equal(path.dirname(result.outputPath), dir)
    assert.match(path.basename(result.outputPath), /20260607-130000-restore-primary-sweet_admin-dddddddddddd\.json/)
    const stat = fs.statSync(result.outputPath)
    assert.equal(stat.mode & 0o077, 0)
    const parsed = JSON.parse(fs.readFileSync(result.outputPath, 'utf8'))
    assert.equal(parsed.backup.sha256, 'd'.repeat(64))
    assert.equal(JSON.stringify(parsed).includes('PrimaryPassword_2026!'), false)
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('validateRestoreBackupManifest accepts matching manifest evidence', () => {
  const backupFile = '/tmp/sweet-admin-backups/20260607-120000-primary-sweet_admin.sql'
  const manifest = {
    schema_version: 1,
    action: 'backup',
    created_at: '2026-06-07T12:30:00.000Z',
    output_path: backupFile,
    size_bytes: 2048,
    sha256: 'a'.repeat(64),
    database: {
      label: 'primary',
      host: 'postgres.primary.internal',
      name: 'sweet_admin',
    },
  }

  const problem = validateRestoreBackupManifest({
    backupFile,
    manifest,
    stat: { size: 2048, mode: 0o600 },
    manifestStat: { mode: 0o600 },
    sha256: 'a'.repeat(64),
  })

  assert.equal(problem, '')
})

test('validateRestoreBackupManifest requires database identity and ledger evidence in schema 2', () => {
  const backupFile = '/tmp/sweet-admin-backups/20260607-120000-primary-sweet_admin.sql'
  const manifest = {
    schema_version: 2,
    action: 'backup',
    created_at: '2026-06-07T12:30:00.000Z',
    output_path: backupFile,
    size_bytes: 2048,
    sha256: 'a'.repeat(64),
    database: {
      label: 'primary',
      host: 'postgres.primary.internal',
      name: 'sweet_admin',
    },
    database_identity: databaseIdentity,
    migration_ledger: migrationLedger,
  }

  assert.equal(validateRestoreBackupManifest({
    backupFile,
    manifest,
    stat: { size: 2048, mode: 0o600 },
    manifestStat: { mode: 0o600 },
    sha256: 'a'.repeat(64),
  }), '')

  assert.match(validateRestoreBackupManifest({
    backupFile,
    manifest: { ...manifest, migration_ledger: null },
    stat: { size: 2048, mode: 0o600 },
    manifestStat: { mode: 0o600 },
    sha256: 'a'.repeat(64),
  }), /migration ledger summary is incomplete/)
})

test('validateRestoreBackupManifest rejects mismatched hash and broad permissions', () => {
  const backupFile = '/tmp/sweet-admin-backups/20260607-120000-primary-sweet_admin.sql'
  const manifest = {
    schema_version: 1,
    action: 'backup',
    created_at: '2026-06-07T12:30:00.000Z',
    output_path: backupFile,
    size_bytes: 2048,
    sha256: 'a'.repeat(64),
    database: {
      label: 'primary',
      host: 'postgres.primary.internal',
      name: 'sweet_admin',
    },
  }

  assert.match(
    validateRestoreBackupManifest({
      backupFile,
      manifest,
      stat: { size: 2048, mode: 0o600 },
      manifestStat: { mode: 0o600 },
      sha256: 'b'.repeat(64),
    }),
    /sha256 does not match/,
  )
  assert.match(
    validateRestoreBackupManifest({
      backupFile,
      manifest,
      stat: { size: 2048, mode: 0o644 },
      manifestStat: { mode: 0o600 },
      sha256: 'a'.repeat(64),
    }),
    /permissions must be owner-only/,
  )
})

test('verifyRestoreBackupEvidence reads real backup and sidecar manifest', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-restore-'))
  try {
    const backupFile = path.join(dir, '20260607-120000-primary-sweet_admin.sql')
    const content = 'create table restore_check(id bigint);\n'
    fs.writeFileSync(backupFile, content, { mode: 0o600 })
    const sha256 = crypto.createHash('sha256').update(content).digest('hex')
    const manifest = {
      schema_version: 1,
      action: 'backup',
      created_at: '2026-06-07T12:30:00.000Z',
      output_path: backupFile,
      size_bytes: Buffer.byteLength(content),
      sha256,
      database: {
        label: 'primary',
        host: 'postgres.primary.internal',
        name: 'sweet_admin',
      },
    }
    fs.writeFileSync(backupManifestPath(backupFile), `${JSON.stringify(manifest)}\n`, { mode: 0o600 })

    const evidence = await verifyRestoreBackupEvidence(backupFile)

    assert.equal(evidence.backupFile, backupFile)
    assert.equal(evidence.manifestPath, backupManifestPath(backupFile))
    assert.equal(evidence.sha256, sha256)
    assert.equal(evidence.sizeBytes, Buffer.byteLength(content))
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('verify command validates backup evidence without requiring external env', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-restore-cli-'))
  try {
    const backupFile = path.join(dir, '20260607-120000-primary-sweet_admin.sql')
    const content = 'create table restore_cli_check(id bigint);\n'
    fs.writeFileSync(backupFile, content, { mode: 0o600 })
    const sha256 = crypto.createHash('sha256').update(content).digest('hex')
    const manifest = {
      schema_version: 1,
      action: 'backup',
      created_at: '2026-06-07T12:30:00.000Z',
      output_path: backupFile,
      size_bytes: Buffer.byteLength(content),
      sha256,
      database: {
        label: 'primary',
        host: 'postgres.primary.internal',
        name: 'sweet_admin',
      },
    }
    fs.writeFileSync(backupManifestPath(backupFile), `${JSON.stringify(manifest)}\n`, { mode: 0o600 })

    const scriptPath = path.join(path.dirname(fileURLToPath(import.meta.url)), 'db-backup-external.mjs')
    const result = spawnSync(process.execPath, [scriptPath, 'verify', backupFile], {
      encoding: 'utf8',
      env: {
        PATH: process.env.PATH,
      },
    })

    assert.equal(result.status, 0, `${result.stdout}\n${result.stderr}`)
    assert.match(result.stdout, /Backup verified:/)
    assert.match(result.stdout, /Backup manifest verified:/)
    assert.match(result.stdout, new RegExp(sha256))
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('verifyRestoreBackupEvidence rejects missing backup file or sidecar manifest', async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-restore-'))
  try {
    await assert.rejects(
      () => verifyRestoreBackupEvidence(''),
      /BACKUP_FILE must be provided for backup verification/,
    )

    const backupFile = path.join(dir, '20260607-120000-primary-sweet_admin.sql')
    fs.writeFileSync(backupFile, 'select 1;\n', { mode: 0o600 })

    await assert.rejects(
      () => verifyRestoreBackupEvidence(backupFile),
      /Backup manifest not found/,
    )
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})
