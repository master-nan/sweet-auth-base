import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  PRODUCTION_WRITE_CONFIRMATION,
  initExternalEnvFile,
  parseEnvContent,
  validateExternalEnv,
  validateExternalEnvFileSecurity,
  validateExternalEnvRepositorySafety,
  validateExternalEnvTemplate,
  validateExternalWriteTarget,
} from './preflight-external.mjs'

const validEnvContent = `
APP_ENV=production
APP_DBS_PRIMARY_HOST=postgres.primary.internal
APP_DBS_PRIMARY_PORT=5432
APP_DBS_PRIMARY_NAME=sweet_admin
APP_DBS_PRIMARY_USER=sweet_admin_app
APP_DBS_PRIMARY_PASSWORD=DbCred_2026_Strong!
APP_REDIS_HOST=redis.internal
APP_REDIS_PORT=6379
APP_REDIS_DB=5
APP_REDIS_PASSWORD=RedisCred_2026_Strong!
APP_SESSION_SECRET=zN8vYq3rT7pLm4sWx9Cb2Kj6Hf5Da0Rt
APP_CONF_SALT=qM7sWc2Jk8Pn4Vt6Ry9Lx5Ha3Db0Fu1G
APP_BOOTSTRAP_ADMIN_PASSWORD=StartAdmin_2026_Strong!
APP_RUN_MIGRATIONS=false
APP_REQUIRE_SECURE_CONFIG=true
APP_ENFORCE_CASBIN_POLICY_COVERAGE=true
APP_SECURITY_CORS_ALLOWED_ORIGINS=https://admin.company.test
APP_SECURITY_CORS_ALLOW_CREDENTIALS=false
APP_AUDIT_ACCESS_LOG_RETENTION_DAYS=180
SWEET_ADMIN_ADMIN_USER=admin
SWEET_ADMIN_ADMIN_PASSWORD=StartAdmin_2026_Strong!
SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE=staging
SWEET_ADMIN_BASE_URL=https://admin.company.test/sweet_admin
SWEET_ADMIN_HEALTH_BASE_URL=https://admin-health.company.test
APP_UPLOAD_DRIVER=local
APP_UPLOAD_DIR=/app/uploads
APP_UPLOAD_BASE_URL=/sweet_admin/files
APP_UPLOAD_MAX_SIZE=50
APP_UPLOAD_CHUNK_SIZE=5
APP_UPLOAD_ALLOWED_EXTENSIONS=.jpg,.png,.pdf,.txt,.csv,.docx,.xlsx
APP_UPLOAD_ALLOWED_MIME_TYPES=image/jpeg,image/png,application/pdf,text/plain,text/csv,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
APP_UPLOAD_PUBLIC_PREVIEW=false
`

const validTemplateContent = `${validEnvContent}
SWEET_ADMIN_SMOKE_TABLE=sys_user
SWEET_ADMIN_SMOKE_CAPTCHA_ID=
SWEET_ADMIN_SMOKE_CAPTCHA=
SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE=
APP_UPLOAD_OSS_ENDPOINT=
APP_UPLOAD_OSS_ACCESS_KEY_ID=
APP_UPLOAD_OSS_ACCESS_KEY_SECRET=
APP_UPLOAD_OSS_BUCKET_NAME=
APP_UPLOAD_OSS_BASE_URL=
APP_UPLOAD_OSS_BASE_PATH=uploads/
`

test('parseEnvContent supports comments, export, and quoted values', () => {
  const env = parseEnvContent(`
    # comment
    export APP_ENV=production
    APP_SESSION_SECRET="abc\\\\def"
    APP_CONF_SALT='xyz'
    APP_INLINE=value # trailing comment
  `)

  assert.equal(env.APP_ENV, 'production')
  assert.equal(env.APP_SESSION_SECRET, 'abc\\def')
  assert.equal(env.APP_CONF_SALT, 'xyz')
  assert.equal(env.APP_INLINE, 'value')
})

test('validateExternalEnv accepts hardened external deploy settings', () => {
  const result = validateExternalEnv(parseEnvContent(validEnvContent))

  assert.equal(result.ok, true, result.problems.join('\n'))
  assert.deepEqual(result.problems, [])
})

test('validateExternalEnvTemplate requires all template keys', () => {
  const result = validateExternalEnvTemplate(parseEnvContent(validTemplateContent))

  assert.equal(result.ok, true, result.problems.join('\n'))

  const missing = parseEnvContent(validTemplateContent)
  delete missing.SWEET_ADMIN_BASE_URL
  const missingResult = validateExternalEnvTemplate(missing)

  assert.equal(missingResult.ok, false)
  assert.match(missingResult.problems.join('\n'), /SWEET_ADMIN_BASE_URL/)
})

test('initExternalEnvFile creates owner-only env file and refuses overwrite', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-external-env-'))
  try {
    const templatePath = path.join(dir, '.env.external.example')
    const targetPath = path.join(dir, '.env.external')
    fs.writeFileSync(templatePath, validTemplateContent)

    const result = initExternalEnvFile({ templatePath, targetPath })

    assert.equal(result.path, targetPath)
    assert.equal(fs.statSync(targetPath).mode & 0o077, 0)
    assert.equal(fs.readFileSync(targetPath, 'utf8'), validTemplateContent)
    assert.throws(
      () => initExternalEnvFile({ templatePath, targetPath }),
      /Refusing to overwrite/,
    )
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('validateExternalEnvFileSecurity rejects broad file permissions', () => {
  assert.deepEqual(
    validateExternalEnvFileSecurity('/secure/.env.external', {
      mode: 0o600,
      isFile: () => true,
    }),
    [],
  )

  assert.match(
    validateExternalEnvFileSecurity('/secure/.env.external', {
      mode: 0o644,
      isFile: () => true,
    }).join('\n'),
    /chmod 600/,
  )

  assert.match(
    validateExternalEnvFileSecurity('/secure/.env.external', {
      mode: 0o600,
      isFile: () => false,
    }).join('\n'),
    /regular file/,
  )
})

test('validateExternalEnvRepositorySafety requires ignored untracked env files inside repository', () => {
  assert.deepEqual(
    validateExternalEnvRepositorySafety('/repo/.env.external', {
      gitRoot: '/repo',
      isTracked: false,
      isIgnored: true,
    }),
    [],
  )

  assert.match(
    validateExternalEnvRepositorySafety('/repo/.env.external', {
      gitRoot: '/repo',
      isTracked: true,
      isIgnored: true,
    }).join('\n'),
    /must not be tracked/,
  )

  assert.match(
    validateExternalEnvRepositorySafety('/repo/secrets/.env.external', {
      gitRoot: '/repo',
      isTracked: false,
      isIgnored: false,
    }).join('\n'),
    /must be ignored/,
  )

  assert.deepEqual(
    validateExternalEnvRepositorySafety('/secure/.env.external', {
      gitRoot: '/repo',
      isTracked: false,
      isIgnored: false,
    }),
    [],
  )
})

test('initExternalEnvFile refuses repository targets that are not ignored', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-external-env-repo-'))
  try {
    const templatePath = path.join(dir, '.env.external.example')
    const targetPath = path.join(dir, '.env.external')
    fs.writeFileSync(templatePath, validTemplateContent)

    assert.throws(
      () => initExternalEnvFile({ templatePath, targetPath, repositorySafetyOptions: {
        gitRoot: dir,
        isTracked: false,
        isIgnored: false,
      } }),
      /not safe for secrets/,
    )
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

test('validateExternalWriteTarget protects destructive external operations', () => {
  const staging = parseEnvContent(validEnvContent)
  assert.deepEqual(validateExternalWriteTarget(staging, { operation: 'migration' }), [])

  const missingPurpose = { ...staging, SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE: '' }
  assert.match(
    validateExternalWriteTarget(missingPurpose, { operation: 'restore' }).join('\n'),
    /SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE/,
  )

  const production = { ...staging, SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE: 'production' }
  assert.match(
    validateExternalWriteTarget(production, { operation: 'migration' }).join('\n'),
    /CONFIRM_EXTERNAL_PRODUCTION_WRITE/,
  )
  assert.deepEqual(
    validateExternalWriteTarget(production, {
      operation: 'migration',
      productionConfirmation: PRODUCTION_WRITE_CONFIRMATION,
    }),
    [],
  )
})

test('validateExternalEnv rejects placeholders and unsafe deploy settings', () => {
  const env = parseEnvContent(validEnvContent)
  env.APP_ENV = 'docker'
  env.APP_DBS_PRIMARY_HOST = 'replace-with-primary-postgres-host'
  env.APP_REDIS_PASSWORD = ''
  env.APP_SESSION_SECRET = 'local-external-session-secret-change-me'
  env.APP_REQUIRE_SECURE_CONFIG = 'false'
  env.APP_SECURITY_CORS_ALLOWED_ORIGINS = '*'
  env.APP_UPLOAD_PUBLIC_PREVIEW = 'true'
  env.APP_UPLOAD_ALLOWED_EXTENSIONS = '.jpg,.sh'
  env.APP_UPLOAD_ALLOWED_MIME_TYPES = 'image/jpeg,text/html'

  const result = validateExternalEnv(env)

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /APP_ENV should be pro, prod, or production/)
  assert.match(result.problems.join('\n'), /APP_DBS_PRIMARY_HOST/)
  assert.match(result.problems.join('\n'), /APP_REDIS_PASSWORD/)
  assert.match(result.problems.join('\n'), /APP_SESSION_SECRET/)
  assert.match(result.problems.join('\n'), /APP_REQUIRE_SECURE_CONFIG must be true/)
  assert.match(result.problems.join('\n'), /APP_SECURITY_CORS_ALLOWED_ORIGINS/)
  assert.match(result.problems.join('\n'), /APP_UPLOAD_PUBLIC_PREVIEW must be false/)
  assert.match(result.problems.join('\n'), /\.sh/)
  assert.match(result.problems.join('\n'), /text\/html/)
})

test('validateExternalEnv accepts oss settings with short real bucket names', () => {
  const env = parseEnvContent(validEnvContent)
  env.APP_UPLOAD_DRIVER = 'oss'
  env.APP_UPLOAD_OSS_ENDPOINT = 'oss-cn-hangzhou.aliyuncs.com'
  env.APP_UPLOAD_OSS_ACCESS_KEY_ID = 'LTAI5StrongKey'
  env.APP_UPLOAD_OSS_ACCESS_KEY_SECRET = 'OssCred_2026_Strong!'
  env.APP_UPLOAD_OSS_BUCKET_NAME = 'ac'
  env.APP_UPLOAD_OSS_BASE_URL = 'https://cdn.company.test/sweet-admin'
  env.APP_UPLOAD_OSS_BASE_PATH = 'tenant-a/uploads/'

  const result = validateExternalEnv(env)

  assert.equal(result.ok, true, result.problems.join('\n'))
})

test('validateExternalEnv rejects unsafe upload access URLs', () => {
  const env = parseEnvContent(validEnvContent)
  env.APP_UPLOAD_BASE_URL = 'http://files.company.test/sweet-admin/files'
  env.APP_UPLOAD_DRIVER = 'oss'
  env.APP_UPLOAD_OSS_ENDPOINT = 'oss-cn-hangzhou.aliyuncs.com'
  env.APP_UPLOAD_OSS_ACCESS_KEY_ID = 'LTAI5StrongKey'
  env.APP_UPLOAD_OSS_ACCESS_KEY_SECRET = 'OssCred_2026_Strong!'
  env.APP_UPLOAD_OSS_BUCKET_NAME = 'sweet-admin'
  env.APP_UPLOAD_OSS_BASE_URL = 'http://cdn.company.test/sweet-admin'
  env.APP_UPLOAD_OSS_BASE_PATH = '../uploads'

  const result = validateExternalEnv(env)

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /APP_UPLOAD_BASE_URL must use HTTPS/)
  assert.match(result.problems.join('\n'), /APP_UPLOAD_OSS_BASE_URL must use HTTPS/)
  assert.match(result.problems.join('\n'), /APP_UPLOAD_OSS_BASE_PATH/)
})

test('validateExternalEnv warns when external startup migrations are enabled', () => {
  const env = parseEnvContent(validEnvContent)
  env.APP_RUN_MIGRATIONS = 'true'

  const result = validateExternalEnv(env)

  assert.equal(result.ok, true, result.problems.join('\n'))
  assert.match(result.warnings.join('\n'), /APP_RUN_MIGRATIONS is true/)
})

test('validateExternalEnv rejects migrations when readonly mode requires them disabled', () => {
  const env = parseEnvContent(validEnvContent)
  env.APP_RUN_MIGRATIONS = 'true'

  const result = validateExternalEnv(env, { requireMigrationsDisabled: true })

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /APP_RUN_MIGRATIONS must be false/)
})

test('validateExternalEnv rejects placeholder readonly smoke credentials when required', () => {
  const env = parseEnvContent(validEnvContent)
  env.SWEET_ADMIN_ADMIN_PASSWORD = 'replace-with-current-admin-password'

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /current admin password/)
})

test('validateExternalEnv rejects unsafe readonly smoke target URLs when required', () => {
  const env = parseEnvContent(validEnvContent)
  env.SWEET_ADMIN_BASE_URL = 'http://localhost:8008/sweet_admin'
  env.SWEET_ADMIN_HEALTH_BASE_URL = 'https://admin-health.company.test/readyz?debug=1'

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /SWEET_ADMIN_BASE_URL must not point to localhost/)
  assert.match(result.problems.join('\n'), /SWEET_ADMIN_HEALTH_BASE_URL must not include credentials, query, or fragment/)
})

test('validateExternalEnv rejects readonly smoke base URL without sweet_admin path', () => {
  const env = parseEnvContent(validEnvContent)
  env.SWEET_ADMIN_BASE_URL = 'https://admin.company.test/admin'

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /SWEET_ADMIN_BASE_URL should include the \/sweet_admin base path/)
})

test('validateExternalEnv warns about captcha variables for readonly smoke', () => {
  const env = parseEnvContent(validEnvContent)

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, true, result.problems.join('\n'))
  assert.match(result.warnings.join('\n'), /login captcha is disabled/)
})

test('validateExternalEnv rejects partial captcha variables for readonly smoke', () => {
  const env = parseEnvContent(validEnvContent)
  env.SWEET_ADMIN_SMOKE_CAPTCHA_ID = 'captcha-id'

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, false)
  assert.match(result.problems.join('\n'), /SWEET_ADMIN_SMOKE_CAPTCHA_ID and SWEET_ADMIN_SMOKE_CAPTCHA/)
})

test('validateExternalEnv accepts paired captcha variables for readonly smoke', () => {
  const env = parseEnvContent(validEnvContent)
  env.SWEET_ADMIN_SMOKE_CAPTCHA_ID = 'captcha-id'
  env.SWEET_ADMIN_SMOKE_CAPTCHA = '123456'

  const result = validateExternalEnv(env, { requireSmokeCredentials: true })

  assert.equal(result.ok, true, result.problems.join('\n'))
  assert.doesNotMatch(result.warnings.join('\n'), /login captcha is disabled/)
})
