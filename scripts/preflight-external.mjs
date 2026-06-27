#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const TRUE_VALUES = new Set(['1', 'true', 'yes', 'y', 'on'])
const FALSE_VALUES = new Set(['0', 'false', 'no', 'n', 'off'])
export const PRODUCTION_WRITE_CONFIRMATION = 'I_UNDERSTAND_THIS_WRITES_PRODUCTION'

const DANGEROUS_EXTENSIONS = new Set([
  '.html',
  '.htm',
  '.svg',
  '.js',
  '.mjs',
  '.cjs',
  '.jsx',
  '.ts',
  '.tsx',
  '.vue',
  '.sh',
  '.bash',
  '.zsh',
  '.fish',
  '.bat',
  '.cmd',
  '.ps1',
  '.exe',
  '.dll',
  '.msi',
  '.app',
  '.dmg',
  '.pkg',
  '.jar',
  '.war',
  '.class',
  '.php',
  '.jsp',
  '.asp',
  '.aspx',
  '.py',
  '.rb',
  '.pl',
  '.go',
  '.rs',
])

const DANGEROUS_MIME_TYPES = new Set([
  'text/html',
  'image/svg+xml',
  'application/javascript',
  'text/javascript',
  'application/x-javascript',
  'application/ecmascript',
  'text/ecmascript',
  'text/x-shellscript',
  'application/x-sh',
  'application/x-msdownload',
  'application/x-msdos-program',
  'application/x-ms-installer',
  'application/x-dosexec',
  'application/java-archive',
  'application/x-httpd-php',
  'application/octet-stream',
])

export const REQUIRED_KEYS = [
  'APP_ENV',
  'APP_DBS_PRIMARY_HOST',
  'APP_DBS_PRIMARY_PORT',
  'APP_DBS_PRIMARY_NAME',
  'APP_DBS_PRIMARY_USER',
  'APP_DBS_PRIMARY_PASSWORD',
  'APP_REDIS_HOST',
  'APP_REDIS_PORT',
  'APP_REDIS_DB',
  'APP_REDIS_PASSWORD',
  'APP_SESSION_SECRET',
  'APP_CONF_SALT',
  'APP_BOOTSTRAP_ADMIN_PASSWORD',
  'APP_RUN_MIGRATIONS',
  'APP_REQUIRE_SECURE_CONFIG',
  'APP_ENFORCE_CASBIN_POLICY_COVERAGE',
  'APP_SECURITY_CORS_ALLOWED_ORIGINS',
  'APP_SECURITY_CORS_ALLOW_CREDENTIALS',
  'APP_AUDIT_ACCESS_LOG_RETENTION_DAYS',
  'APP_UPLOAD_DRIVER',
  'APP_UPLOAD_DIR',
  'APP_UPLOAD_BASE_URL',
  'APP_UPLOAD_MAX_SIZE',
  'APP_UPLOAD_CHUNK_SIZE',
  'APP_UPLOAD_ALLOWED_EXTENSIONS',
  'APP_UPLOAD_ALLOWED_MIME_TYPES',
  'APP_UPLOAD_PUBLIC_PREVIEW',
]

export const TEMPLATE_KEYS = [
  ...REQUIRED_KEYS,
  'SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE',
  'SWEET_ADMIN_BASE_URL',
  'SWEET_ADMIN_HEALTH_BASE_URL',
  'SWEET_ADMIN_ADMIN_USER',
  'SWEET_ADMIN_ADMIN_PASSWORD',
  'SWEET_ADMIN_SMOKE_TABLE',
  'SWEET_ADMIN_SMOKE_CAPTCHA_ID',
  'SWEET_ADMIN_SMOKE_CAPTCHA',
  'SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE',
  'APP_UPLOAD_OSS_ENDPOINT',
  'APP_UPLOAD_OSS_ACCESS_KEY_ID',
  'APP_UPLOAD_OSS_ACCESS_KEY_SECRET',
  'APP_UPLOAD_OSS_BUCKET_NAME',
  'APP_UPLOAD_OSS_BASE_URL',
  'APP_UPLOAD_OSS_BASE_PATH',
]

export function parseEnvContent(content) {
  const env = {}
  const lines = content.replace(/^\uFEFF/, '').split(/\r?\n/)
  for (let index = 0; index < lines.length; index += 1) {
    let line = lines[index].trim()
    if (!line || line.startsWith('#')) continue
    if (line.startsWith('export ')) {
      line = line.slice('export '.length).trim()
    }
    const separatorIndex = line.indexOf('=')
    if (separatorIndex <= 0) continue

    const key = line.slice(0, separatorIndex).trim()
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) continue

    let value = line.slice(separatorIndex + 1).trim()
    if (value.startsWith("'") && value.endsWith("'")) {
      value = value.slice(1, -1)
    } else if (value.startsWith('"') && value.endsWith('"')) {
      value = value
        .slice(1, -1)
        .replace(/\\n/g, '\n')
        .replace(/\\"/g, '"')
        .replace(/\\\\/g, '\\')
    } else {
      value = value.replace(/\s+#.*$/, '').trim()
    }
    env[key] = value
  }
  return env
}

export function validateExternalEnv(env, options = {}) {
  const problems = []
  const warnings = []
  const allowNonProduction = options.allowNonProduction === true
  const requireMigrationsDisabled = options.requireMigrationsDisabled === true
  const requireSmokeCredentials = options.requireSmokeCredentials === true

  for (const key of REQUIRED_KEYS) {
    if (!Object.prototype.hasOwnProperty.call(env, key) || env[key].trim() === '') {
      problems.push(`${key} must be set`)
    }
  }

  const environment = get(env, 'APP_ENV').toLowerCase()
  if (!['pro', 'prod', 'production'].includes(environment) && !allowNonProduction) {
    problems.push('APP_ENV should be pro, prod, or production for external deploy validation')
  }

  requireBoolean(env, 'APP_REQUIRE_SECURE_CONFIG', true, problems)
  requireBoolean(env, 'APP_ENFORCE_CASBIN_POLICY_COVERAGE', true, problems)
  requireBoolean(env, 'APP_UPLOAD_PUBLIC_PREVIEW', false, problems)
  const runMigrations = parseBoolean(get(env, 'APP_RUN_MIGRATIONS'))
  if (runMigrations === null) {
    problems.push('APP_RUN_MIGRATIONS must be a boolean')
  } else if (runMigrations && requireMigrationsDisabled) {
    problems.push('APP_RUN_MIGRATIONS must be false for readonly external checks')
  } else if (runMigrations) {
    warnings.push('APP_RUN_MIGRATIONS is true; backend startup will apply migrations to the target database')
  }

  if (parseBoolean(get(env, 'APP_SECURITY_CORS_ALLOW_CREDENTIALS')) === true) {
    warnings.push('APP_SECURITY_CORS_ALLOW_CREDENTIALS is true; keep it false unless cross-site cookies are required')
  }

  validateDB(env, 'APP_DBS_PRIMARY', problems)
  validateRedis(env, problems)
  validateSecrets(env, problems)
  validateCors(env, problems)
  validateAudit(env, problems)
  validateUpload(env, problems)
  if (requireSmokeCredentials) {
    validateSmokeTargets(env, problems)
    validateSmokeCredentials(env, problems, warnings)
  }

  return {
    ok: problems.length === 0,
    problems,
    warnings,
  }
}

export function validateExternalEnvTemplate(env) {
  const problems = []
  for (const key of TEMPLATE_KEYS) {
    if (!Object.prototype.hasOwnProperty.call(env, key)) {
      problems.push(`${key} must exist in .env.external.example`)
    }
  }
  const duplicateKeys = findDuplicateEnvKeys(env.__lines || [])
  for (const key of duplicateKeys) {
    problems.push(`${key} is duplicated in .env.external.example`)
  }
  return {
    ok: problems.length === 0,
    problems,
  }
}

export function initExternalEnvFile({
  targetPath = '.env.external',
  templatePath = '.env.external.example',
  force = false,
  repositorySafetyOptions = {},
} = {}) {
  const resolvedTargetPath = path.resolve(targetPath)
  const resolvedTemplatePath = path.resolve(templatePath)
  if (!fs.existsSync(resolvedTemplatePath)) {
    throw new Error(`External env template not found: ${resolvedTemplatePath}`)
  }
  if (fs.existsSync(resolvedTargetPath) && !force) {
    throw new Error(`Refusing to overwrite existing external env file: ${resolvedTargetPath}`)
  }

  const templateContent = fs.readFileSync(resolvedTemplatePath, 'utf8')
  const templateEnv = parseEnvContentWithLines(templateContent)
  const templateResult = validateExternalEnvTemplate(templateEnv)
  if (!templateResult.ok) {
    throw new Error(`External env template is incomplete:\n- ${templateResult.problems.join('\n- ')}`)
  }
  const repositoryProblems = validateExternalEnvRepositorySafety(resolvedTargetPath, repositorySafetyOptions)
  if (repositoryProblems.length > 0) {
    throw new Error(`External env target is not safe for secrets:\n- ${repositoryProblems.join('\n- ')}`)
  }

  fs.mkdirSync(path.dirname(resolvedTargetPath), { recursive: true })
  fs.writeFileSync(resolvedTargetPath, templateContent, { mode: 0o600, flag: force ? 'w' : 'wx' })
  fs.chmodSync(resolvedTargetPath, 0o600)
  return {
    path: resolvedTargetPath,
    template_path: resolvedTemplatePath,
  }
}

export function validateExternalEnvFileSecurity(filePath, stat) {
  const problems = []
  const label = filePath || 'external env file'
  if (!stat || typeof stat.isFile !== 'function' || !stat.isFile()) {
    problems.push(`External env file must be a regular file: ${label}`)
    return problems
  }
  if (Number.isInteger(stat.mode) && ((stat.mode & 0o777) & 0o077) !== 0) {
    problems.push(`External env file permissions must be owner-only (chmod 600): ${label}`)
  }
  return problems
}

export function validateExternalEnvRepositorySafety(filePath, options = {}) {
  const resolvedPath = path.resolve(filePath || '')
  const gitRoot = options.gitRoot === undefined ? findGitRoot(options.cwd || process.cwd()) : options.gitRoot
  if (!gitRoot) return []

  const resolvedGitRoot = path.resolve(gitRoot)
  if (!pathInside(resolvedPath, resolvedGitRoot)) return []

  const relativePath = path.relative(resolvedGitRoot, resolvedPath)
  const isTracked =
    options.isTracked === undefined ? gitPathTracked(resolvedGitRoot, relativePath) : options.isTracked
  const isIgnored =
    options.isIgnored === undefined ? gitPathIgnored(resolvedGitRoot, relativePath) : options.isIgnored
  const problems = []
  if (isTracked) {
    problems.push(`External env file must not be tracked by git: ${resolvedPath}`)
  }
  if (!isIgnored) {
    problems.push(`External env file inside repository must be ignored by git: ${resolvedPath}`)
  }
  return problems
}

export function validateExternalWriteTarget(env, options = {}) {
  const problems = []
  const operation = options.operation || 'external write'
  const purpose = get(env, 'SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE').toLowerCase()
  const allowedPurposes = new Set(['staging', 'stage', 'test', 'testing', 'development', 'dev', 'production', 'prod'])
  if (!allowedPurposes.has(purpose)) {
    problems.push(
      `SWEET_ADMIN_EXTERNAL_TARGET_PURPOSE must be staging, test, development, or production before ${operation}`,
    )
    return problems
  }
  if (purpose === 'production' || purpose === 'prod') {
    const confirmation = String(options.productionConfirmation || '').trim()
    if (confirmation !== PRODUCTION_WRITE_CONFIRMATION) {
      problems.push(
        `Refusing ${operation} against production without CONFIRM_EXTERNAL_PRODUCTION_WRITE=${PRODUCTION_WRITE_CONFIRMATION}`,
      )
    }
  }
  return problems
}

function validateSmokeTargets(env, problems) {
  validateExternalSmokeURL(env, 'SWEET_ADMIN_BASE_URL', {
    requirePath: true,
    pathHint: '/sweet_admin',
    problems,
  })
  validateExternalSmokeURL(env, 'SWEET_ADMIN_HEALTH_BASE_URL', {
    requirePath: false,
    problems,
  })
}

function validateExternalSmokeURL(env, key, { requirePath = false, pathHint = '', problems }) {
  const value = get(env, key)
  if (missingConfigValue(value)) {
    problems.push(`${key} must be set to the external readonly target URL`)
    return
  }

  let parsed
  try {
    parsed = new URL(value)
  } catch {
    problems.push(`${key} must be an explicit http(s) URL`)
    return
  }
  if (!['http:', 'https:'].includes(parsed.protocol) || !parsed.hostname) {
    problems.push(`${key} must be an explicit http(s) URL`)
    return
  }
  if (isLocalSmokeHost(parsed.hostname)) {
    problems.push(`${key} must not point to localhost for external readonly validation`)
  }
  if (parsed.username || parsed.password || parsed.search || parsed.hash) {
    problems.push(`${key} must not include credentials, query, or fragment`)
  }
  if (requirePath && pathHint && parsed.pathname.replace(/\/+$/, '') !== pathHint) {
    problems.push(`${key} should include the ${pathHint} base path`)
  }
}

function isLocalSmokeHost(hostname) {
  const normalized = String(hostname || '').toLowerCase().replace(/^\[|\]$/g, '')
  return normalized === 'localhost' ||
    normalized === '127.0.0.1' ||
    normalized === '0.0.0.0' ||
    normalized === '::1'
}

function validateSmokeCredentials(env, problems, warnings) {
  const userName = get(env, 'SWEET_ADMIN_ADMIN_USER') || 'admin'
  if (placeholderConfigValue(userName.toLowerCase())) {
    problems.push('SWEET_ADMIN_ADMIN_USER must be a real admin user for readonly smoke')
  }
  const password = get(env, 'SWEET_ADMIN_ADMIN_PASSWORD') || get(env, 'APP_BOOTSTRAP_ADMIN_PASSWORD')
  if (insecureCredentialValue(password)) {
    problems.push(
      'SWEET_ADMIN_ADMIN_PASSWORD or APP_BOOTSTRAP_ADMIN_PASSWORD must be set to the current admin password for readonly smoke',
    )
  }

  const captchaId = get(env, 'SWEET_ADMIN_SMOKE_CAPTCHA_ID')
  const captcha = get(env, 'SWEET_ADMIN_SMOKE_CAPTCHA')
  if ((captchaId && !captcha) || (!captchaId && captcha)) {
    problems.push('SWEET_ADMIN_SMOKE_CAPTCHA_ID and SWEET_ADMIN_SMOKE_CAPTCHA must be provided together')
  } else if (!captchaId && !captcha) {
    warnings.push(
      'readonly smoke can run without captcha variables only when login captcha is disabled; if captcha is enabled, first run with SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE and then rerun with SWEET_ADMIN_SMOKE_CAPTCHA_ID/SWEET_ADMIN_SMOKE_CAPTCHA',
    )
  }
}

function validateDB(env, prefix, problems) {
  const host = get(env, `${prefix}_HOST`)
  const port = get(env, `${prefix}_PORT`)
  const name = get(env, `${prefix}_NAME`)
  const user = get(env, `${prefix}_USER`)
  const password = get(env, `${prefix}_PASSWORD`)

  if (missingConfigValue(host)) problems.push(`${prefix}_HOST must be a real host, not a placeholder`)
  validatePort(port, `${prefix}_PORT`, problems)
  if (missingConfigValue(name)) problems.push(`${prefix}_NAME must be a real database name`)
  if (insecureDBUserValue(user)) problems.push(`${prefix}_USER must be a non-root, non-admin user`)
  if (insecureCredentialValue(password)) {
    problems.push(`${prefix}_PASSWORD must be a non-default credential with at least 8 characters`)
  }
}

function validateRedis(env, problems) {
  if (missingConfigValue(get(env, 'APP_REDIS_HOST'))) {
    problems.push('APP_REDIS_HOST must be a real host, not a placeholder')
  }
  validatePort(get(env, 'APP_REDIS_PORT'), 'APP_REDIS_PORT', problems)
  validateIntegerRange(get(env, 'APP_REDIS_DB'), 'APP_REDIS_DB', 0, 15, problems)
  if (insecureCredentialValue(get(env, 'APP_REDIS_PASSWORD'))) {
    problems.push('APP_REDIS_PASSWORD must be a non-default credential with at least 8 characters')
  }
}

function validateSecrets(env, problems) {
  if (insecureSecureConfigValue(get(env, 'APP_SESSION_SECRET'))) {
    problems.push('APP_SESSION_SECRET must be non-default and at least 32 characters')
  }
  if (insecureSecureConfigValue(get(env, 'APP_CONF_SALT'))) {
    problems.push('APP_CONF_SALT must be non-default and at least 32 characters')
  }
  const bootstrapPassword = get(env, 'APP_BOOTSTRAP_ADMIN_PASSWORD').toLowerCase()
  if (bootstrapPassword.length < 12 || insecureCredentialValue(bootstrapPassword)) {
    problems.push('APP_BOOTSTRAP_ADMIN_PASSWORD must be strong and must not be admin123 or a placeholder')
  }
}

function validateCors(env, problems) {
  const origins = splitCSV(get(env, 'APP_SECURITY_CORS_ALLOWED_ORIGINS'))
  if (origins.length === 0) {
    problems.push('APP_SECURITY_CORS_ALLOWED_ORIGINS must contain at least one trusted origin')
    return
  }
  for (const origin of origins) {
    const normalized = origin.toLowerCase()
    if (
      normalized === '*' ||
      placeholderConfigValue(normalized) ||
      !/^https?:\/\/[^/]+/.test(normalized)
    ) {
      problems.push('APP_SECURITY_CORS_ALLOWED_ORIGINS must contain only explicit http(s) origins')
      return
    }
  }
}

function validateAudit(env, problems) {
  validateIntegerRange(get(env, 'APP_AUDIT_ACCESS_LOG_RETENTION_DAYS'), 'APP_AUDIT_ACCESS_LOG_RETENTION_DAYS', 1, 3650, problems)
}

function validateUpload(env, problems) {
  const driver = get(env, 'APP_UPLOAD_DRIVER').toLowerCase()
  if (!['local', 'oss'].includes(driver)) {
    problems.push('APP_UPLOAD_DRIVER must be local or oss')
  }
  if (missingConfigValue(get(env, 'APP_UPLOAD_DIR'))) {
    problems.push('APP_UPLOAD_DIR must be set')
  }
  if (missingConfigValue(get(env, 'APP_UPLOAD_BASE_URL'))) {
    problems.push('APP_UPLOAD_BASE_URL must be set')
  }
  validateUploadBaseURL(get(env, 'APP_UPLOAD_BASE_URL'), 'APP_UPLOAD_BASE_URL', true, problems)
  validateIntegerRange(get(env, 'APP_UPLOAD_MAX_SIZE'), 'APP_UPLOAD_MAX_SIZE', 1, 512, problems)
  validateIntegerRange(get(env, 'APP_UPLOAD_CHUNK_SIZE'), 'APP_UPLOAD_CHUNK_SIZE', 1, 128, problems)

  const extensions = splitCSV(get(env, 'APP_UPLOAD_ALLOWED_EXTENSIONS')).map(normalizeExtension)
  const mimeTypes = splitCSV(get(env, 'APP_UPLOAD_ALLOWED_MIME_TYPES')).map(normalizeMimeType)
  if (extensions.length === 0) problems.push('APP_UPLOAD_ALLOWED_EXTENSIONS must not be empty')
  if (mimeTypes.length === 0) problems.push('APP_UPLOAD_ALLOWED_MIME_TYPES must not be empty')

  for (const ext of extensions) {
    if (DANGEROUS_EXTENSIONS.has(ext)) {
      problems.push(`APP_UPLOAD_ALLOWED_EXTENSIONS must not include active or executable type ${ext}`)
    }
  }
  for (const mimeType of mimeTypes) {
    if (DANGEROUS_MIME_TYPES.has(mimeType)) {
      problems.push(`APP_UPLOAD_ALLOWED_MIME_TYPES must not include active or executable type ${mimeType}`)
    }
  }

  if (driver === 'oss') {
    requireConfiguredValue(env, 'APP_UPLOAD_OSS_ENDPOINT', problems)
    requireCredentialValue(env, 'APP_UPLOAD_OSS_ACCESS_KEY_ID', problems)
    requireCredentialValue(env, 'APP_UPLOAD_OSS_ACCESS_KEY_SECRET', problems)
    requireConfiguredValue(env, 'APP_UPLOAD_OSS_BUCKET_NAME', problems)
    if (get(env, 'APP_UPLOAD_OSS_BASE_URL')) {
      validateUploadBaseURL(get(env, 'APP_UPLOAD_OSS_BASE_URL'), 'APP_UPLOAD_OSS_BASE_URL', false, problems)
    }
    validateOSSBasePath(get(env, 'APP_UPLOAD_OSS_BASE_PATH'), problems)
  }
}

function validateUploadBaseURL(value, key, allowRelativePath, problems) {
  const normalized = value.trim()
  if (missingConfigValue(normalized)) return
  if (/\s/.test(normalized) || normalized.includes('?') || normalized.includes('#')) {
    problems.push(`${key} must be a clean URL prefix without whitespace, query, or fragment`)
    return
  }
  if (normalized.startsWith('/')) {
    if (!allowRelativePath || normalized.startsWith('//') || normalized === '/' || pathHasTraversal(normalized)) {
      problems.push(`${key} must be ${allowRelativePath ? 'a rooted path or HTTPS URL' : 'an HTTPS URL'}`)
    }
    return
  }
  let parsed
  try {
    parsed = new URL(normalized)
  } catch {
    problems.push(`${key} must be ${allowRelativePath ? 'a rooted path or HTTPS URL' : 'an HTTPS URL'}`)
    return
  }
  if (parsed.protocol !== 'https:' || !parsed.hostname) {
    problems.push(`${key} must use HTTPS when configured as an absolute URL`)
  }
}

function validateOSSBasePath(value, problems) {
  const normalized = value.trim()
  if (!normalized) return
  if (
    normalized.startsWith('/') ||
    normalized.startsWith('\\') ||
    /^https?:\/\//i.test(normalized) ||
    pathHasTraversal(normalized)
  ) {
    problems.push('APP_UPLOAD_OSS_BASE_PATH must be a relative object-key prefix without path traversal')
  }
}

function pathHasTraversal(value) {
  const normalized = value.replaceAll('\\', '/')
  return normalized === '..' || normalized.startsWith('../') || normalized.endsWith('/..') || normalized.includes('/../')
}

function requireConfiguredValue(env, key, problems) {
  if (missingConfigValue(get(env, key))) {
    problems.push(`${key} must be set to a real value`)
  }
}

function requireCredentialValue(env, key, problems) {
  if (insecureCredentialValue(get(env, key))) {
    problems.push(`${key} must be set to a real non-default value`)
  }
}

function requireBoolean(env, key, expected, problems) {
  const parsed = parseBoolean(get(env, key))
  if (parsed === null) {
    problems.push(`${key} must be a boolean`)
  } else if (parsed !== expected) {
    problems.push(`${key} must be ${expected}`)
  }
}

function validatePort(value, key, problems) {
  validateIntegerRange(value, key, 1, 65535, problems)
}

function validateIntegerRange(value, key, min, max, problems) {
  if (!/^-?\d+$/.test(value.trim())) {
    problems.push(`${key} must be an integer`)
    return
  }
  const number = Number.parseInt(value, 10)
  if (number < min || number > max) {
    problems.push(`${key} must be between ${min} and ${max}`)
  }
}

function parseBoolean(value) {
  const normalized = value.toLowerCase().trim()
  if (TRUE_VALUES.has(normalized)) return true
  if (FALSE_VALUES.has(normalized)) return false
  return null
}

function splitCSV(value) {
  return value
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
}

function normalizeExtension(value) {
  const ext = value.toLowerCase().trim()
  if (!ext) return ''
  return ext.startsWith('.') ? ext : `.${ext}`
}

function normalizeMimeType(value) {
  const index = value.indexOf(';')
  const trimmed = index >= 0 ? value.slice(0, index) : value
  return trimmed.toLowerCase().trim()
}

function insecureSecureConfigValue(value) {
  const normalized = value.toLowerCase().trim()
  if (normalized.length < 32) return true
  return ['replace-with', 'change-me', 'changeme', 'local-docker', 'local-external', 'sweet-admin', 'placeholder', 'example', 'secret'].some((marker) =>
    normalized.includes(marker),
  )
}

function insecureDBUserValue(value) {
  const normalized = value.toLowerCase().trim()
  return missingConfigValue(normalized) || normalized === 'root' || normalized === 'admin'
}

function insecureCredentialValue(value) {
  const normalized = value.toLowerCase().trim()
  if (normalized === '' || normalized.length < 8 || placeholderConfigValue(normalized)) return true
  return ['sweet_admin', 'admin123', 'password', '123456', 'local-docker', 'local-external'].some((marker) =>
    normalized === marker || normalized.includes(marker),
  )
}

function missingConfigValue(value) {
  const normalized = value.toLowerCase().trim()
  return normalized === '' || placeholderConfigValue(normalized)
}

function placeholderConfigValue(normalized) {
  return ['replace-with', 'change-me', 'changeme', 'placeholder', 'example', '<', '>'].some((marker) =>
    normalized.includes(marker),
  )
}

function get(env, key) {
  return (env[key] ?? '').trim()
}

function runtimeEnvValue(keys) {
  for (const key of keys) {
    const value = (process.env[key] || '').trim()
    if (value) return value
  }
  return ''
}

function runtimeBoolean(keys) {
  const value = runtimeEnvValue(keys)
  return value ? parseBoolean(value) : null
}

function main() {
  const command = process.argv[2] || ''
  if (command === 'init') {
    try {
      const result = initExternalEnvFile({
        targetPath:
          process.argv[3] ||
          runtimeEnvValue(['SWEET_ADMIN_EXTERNAL_ENV_FILE']) ||
          '.env.external',
        templatePath:
          process.argv[4] ||
          runtimeEnvValue(['SWEET_ADMIN_EXTERNAL_ENV_TEMPLATE']) ||
          '.env.external.example',
        force:
          runtimeBoolean(['SWEET_ADMIN_EXTERNAL_ENV_INIT_FORCE']) === true,
      })
      console.log(`External env file initialized: ${result.path}`)
      console.log('Fill real staging/production values, then run make preflight-external-readonly.')
    } catch (error) {
      console.error(error.message)
      process.exitCode = 1
    }
    return
  }

  if (command === 'template-check') {
    const templatePath = path.resolve(process.argv[3] || '.env.external.example')
    if (!fs.existsSync(templatePath)) {
      console.error(`External env template not found: ${templatePath}`)
      process.exitCode = 1
      return
    }
    const result = validateExternalEnvTemplate(parseEnvContentWithLines(fs.readFileSync(templatePath, 'utf8')))
    if (!result.ok) {
      console.error(`External env template check failed for ${templatePath}:`)
      for (const problem of result.problems) {
        console.error(`- ${problem}`)
      }
      process.exitCode = 1
      return
    }
    console.log(`External env template check passed for ${templatePath}`)
    return
  }

  const envPath =
    process.argv[2] ||
    runtimeEnvValue(['SWEET_ADMIN_EXTERNAL_ENV_FILE']) ||
    '.env.external'
  const writeOperation = process.argv[3] || ''
  const resolvedPath = path.resolve(envPath)
  if (!fs.existsSync(resolvedPath)) {
    console.error(`External env file not found: ${resolvedPath}`)
    console.error('Copy .env.external.example to .env.external and fill real staging/production values first.')
    process.exitCode = 1
    return
  }

  const fileProblems = [
    ...validateExternalEnvFileSecurity(resolvedPath, fs.statSync(resolvedPath)),
    ...validateExternalEnvRepositorySafety(resolvedPath),
  ]
  const env = parseEnvContent(fs.readFileSync(resolvedPath, 'utf8'))
  const result = validateExternalEnv(env, {
    allowNonProduction:
      runtimeBoolean(['SWEET_ADMIN_PREFLIGHT_ALLOW_NON_PRODUCTION']) === true,
    requireMigrationsDisabled:
      runtimeBoolean(['SWEET_ADMIN_PREFLIGHT_REQUIRE_MIGRATIONS_DISABLED']) === true,
    requireSmokeCredentials:
      runtimeBoolean(['SWEET_ADMIN_PREFLIGHT_REQUIRE_SMOKE_CREDENTIALS']) === true,
  })
  const writeProblems = writeOperation
    ? validateExternalWriteTarget(env, {
        operation: writeOperation,
        productionConfirmation: process.env.CONFIRM_EXTERNAL_PRODUCTION_WRITE,
      })
    : []

  if (!result.ok || fileProblems.length > 0 || writeProblems.length > 0) {
    console.error(`External deploy preflight failed for ${resolvedPath}:`)
    for (const problem of fileProblems) {
      console.error(`- ${problem}`)
    }
    for (const problem of writeProblems) {
      console.error(`- ${problem}`)
    }
    for (const problem of result.problems) {
      console.error(`- ${problem}`)
    }
    process.exitCode = 1
    return
  }

  console.log(`External deploy preflight passed for ${resolvedPath}`)
  for (const warning of result.warnings) {
    console.warn(`Warning: ${warning}`)
  }
}

const currentFile = fileURLToPath(import.meta.url)
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  main()
}

function parseEnvContentWithLines(content) {
  const env = parseEnvContent(content)
  Object.defineProperty(env, '__lines', {
    value: content.replace(/^\uFEFF/, '').split(/\r?\n/),
    enumerable: false,
  })
  return env
}

function findDuplicateEnvKeys(lines) {
  const seen = new Set()
  const duplicates = new Set()
  for (const rawLine of lines) {
    let line = rawLine.trim()
    if (!line || line.startsWith('#')) continue
    if (line.startsWith('export ')) line = line.slice('export '.length).trim()
    const separatorIndex = line.indexOf('=')
    if (separatorIndex <= 0) continue
    const key = line.slice(0, separatorIndex).trim()
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(key)) continue
    if (seen.has(key)) duplicates.add(key)
    seen.add(key)
  }
  return [...duplicates].sort()
}

function findGitRoot(cwd) {
  const result = spawnSync('git', ['rev-parse', '--show-toplevel'], { cwd, encoding: 'utf8' })
  if (result.status !== 0) return ''
  return result.stdout.trim()
}

function gitPathTracked(gitRoot, relativePath) {
  const result = spawnSync('git', ['ls-files', '--error-unmatch', '--', relativePath], {
    cwd: gitRoot,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'ignore'],
  })
  return result.status === 0
}

function gitPathIgnored(gitRoot, relativePath) {
  const result = spawnSync('git', ['check-ignore', '--quiet', '--', relativePath], {
    cwd: gitRoot,
    encoding: 'utf8',
  })
  return result.status === 0
}

function pathInside(candidatePath, rootPath) {
  const relative = path.relative(rootPath, candidatePath)
  return relative === '' || (!relative.startsWith('..') && !path.isAbsolute(relative))
}
