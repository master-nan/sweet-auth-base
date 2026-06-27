#!/usr/bin/env node

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { parseEnvContent } from './preflight-external.mjs'

let runtime = null
let accessToken = ''

function normalizeBaseUrl(value) {
  return value.replace(/\/+$/, '')
}

export function loadOptionalEnvFile(envPath, targetEnv = process.env) {
  if (!envPath) return
  const resolvedPath = path.resolve(envPath)
  if (!fs.existsSync(resolvedPath)) {
    throw new Error(`readonly smoke env file not found: ${resolvedPath}`)
  }
  const env = parseEnvContent(fs.readFileSync(resolvedPath, 'utf8'))
  for (const [key, value] of Object.entries(env)) {
    if (targetEnv[key] === undefined) {
      targetEnv[key] = value
    }
  }
}

export function resolveSmokeCredential(env = process.env) {
  return {
    username: env.SWEET_ADMIN_ADMIN_USER || 'admin',
    password: env.SWEET_ADMIN_ADMIN_PASSWORD || env.APP_BOOTSTRAP_ADMIN_PASSWORD || 'admin123',
  }
}

export function createSmokeRuntime(env = process.env) {
  const credential = resolveSmokeCredential(env)
  return {
    baseUrl: normalizeBaseUrl(env.SWEET_ADMIN_BASE_URL || 'http://localhost:8080/sweet_admin'),
    healthBaseUrl: normalizeBaseUrl(env.SWEET_ADMIN_HEALTH_BASE_URL || 'http://localhost:9005'),
    username: credential.username,
    password: credential.password,
    tableCode: env.SWEET_ADMIN_SMOKE_TABLE || 'sys_user',
    captcha: {
      enabled: false,
      id: env.SWEET_ADMIN_SMOKE_CAPTCHA_ID || '',
      value: env.SWEET_ADMIN_SMOKE_CAPTCHA || '',
      imageFile: env.SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE || '',
    },
  }
}

export function resolveConfiguredCaptchaLoginFields(smokeRuntime) {
  if (!smokeRuntime.captcha?.enabled) {
    return { captcha: '', captcha_id: '' }
  }
  if (smokeRuntime.captcha.value && smokeRuntime.captcha.id) {
    return {
      captcha: smokeRuntime.captcha.value,
      captcha_id: smokeRuntime.captcha.id,
    }
  }
  return null
}

function apiPath(path) {
  return `${runtime.baseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`
  }
  const res = await fetch(apiPath(path), {
    ...options,
    headers,
  })
  const text = await res.text()
  let body = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = text
    }
  }
  return { status: res.status, body, headers: res.headers }
}

async function requestRaw(path, options = {}) {
  const url = /^https?:\/\//i.test(path)
    ? path
    : `${runtime.healthBaseUrl}${path.startsWith('/') ? path : `/${path}`}`
  const res = await fetch(url, options)
  const text = await res.text()
  let body = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = text
    }
  }
  return { status: res.status, body, headers: res.headers }
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

function assertNoSecretFields(value, context) {
  const rendered = JSON.stringify(value || {})
  for (const key of ['app_secret', 'ding_secret', 'sender_password']) {
    assert(!new RegExp(`"${key}"\\s*:\\s*"[^"]+`).test(rendered), `${context} leaked ${key}: ${rendered}`)
  }
}

function findMenuByOption(menus, option) {
  for (const menu of menus || []) {
    if (menu?.option === option) return menu
    const child = findMenuByOption(menu?.children || [], option)
    if (child) return child
  }
  return null
}

async function assertHealth() {
  const health = await requestRaw('/healthz')
  assert(health.status === 200 && health.body?.status === 'ok', `healthz failed: ${JSON.stringify(health.body)}`)
  const ready = await requestRaw('/readyz')
  assert(ready.status === 200 && ready.body?.status === 'ready', `readyz failed: ${JSON.stringify(ready.body)}`)
  assert(
    ready.body?.components?.db_primary?.ok || ready.body?.components?.db?.ok,
    `readyz primary database missing or unhealthy: ${JSON.stringify(ready.body)}`,
  )
  assert(
    ready.body?.components?.redis?.ok,
    `readyz dependencies missing or unhealthy: ${JSON.stringify(ready.body)}`,
  )
  console.log('OK health readiness')
}

async function assertPublicConfigure() {
  const configure = await request('/admin/configure')
  assert(configure.status === 200 && configure.body?.success, `configure endpoint failed: ${JSON.stringify(configure.body)}`)
  assertNoSecretFields(configure.body?.data, 'public configure')
  runtime.captcha.enabled = Boolean(configure.body?.data?.enable_captcha)
  console.log(`OK configure redaction${runtime.captcha.enabled ? ' (captcha enabled)' : ''}`)
}

function decodeCaptchaImage(image) {
  if (!image) return Buffer.alloc(0)
  if (Array.isArray(image)) return Buffer.from(image)
  if (typeof image !== 'string') return Buffer.alloc(0)
  const payload = image.includes(',') ? image.split(',').pop() : image
  return Buffer.from(payload || '', 'base64')
}

async function fetchCaptchaChallenge() {
  const captcha = await request('/admin/captcha')
  assert(captcha.status === 200 && captcha.body?.success, `captcha endpoint failed: ${JSON.stringify(captcha.body)}`)
  assert(captcha.body?.data?.captcha_id, `captcha response missing captcha_id: ${JSON.stringify(captcha.body)}`)
  return captcha.body.data
}

function saveCaptchaImageIfRequested(challenge) {
  if (!runtime.captcha.imageFile) return ''
  const image = decodeCaptchaImage(challenge.image)
  assert(image.length > 0, `captcha response missing image data: ${JSON.stringify(challenge)}`)
  const imagePath = path.resolve(runtime.captcha.imageFile)
  fs.mkdirSync(path.dirname(imagePath), { recursive: true })
  fs.writeFileSync(imagePath, image)
  return imagePath
}

async function resolveLoginCaptcha() {
  const configured = resolveConfiguredCaptchaLoginFields(runtime)
  if (configured) return configured

  const challenge = await fetchCaptchaChallenge()
  const imagePath = saveCaptchaImageIfRequested(challenge)
  const imageHint = imagePath ? ` Captcha image saved to ${imagePath}.` : ''
  throw new Error(
    `captcha is enabled; provide SWEET_ADMIN_SMOKE_CAPTCHA_ID=${challenge.captcha_id} and SWEET_ADMIN_SMOKE_CAPTCHA=<code> before running readonly smoke again.${imageHint}`,
  )
}

async function assertLogin() {
  const captchaFields = await resolveLoginCaptcha()
  const login = await request('/admin/login', {
    method: 'POST',
    body: JSON.stringify({
      user_name: runtime.username,
      password: runtime.password,
      ...captchaFields,
    }),
  })
  assert(login.status === 200 && login.body?.success, `login failed: ${JSON.stringify(login.body)}`)
  accessToken = login.body.data?.access_token || ''
  assert(accessToken, 'login response did not include access_token')
  assert(login.body.data?.must_change_password === false, `admin requires password change: ${JSON.stringify(login.body.data)}`)
  console.log('OK login')
}

async function assertUserAndMenu() {
  const me = await request('/admin/user/me')
  assert(me.status === 200 && me.body?.success && me.body?.data?.id, `current user failed: ${JSON.stringify(me.body)}`)

  const menus = await request('/admin/menu/my')
  assert(menus.status === 200 && menus.body?.success && Array.isArray(menus.body?.data), `menu query failed: ${JSON.stringify(menus.body)}`)
  assert(menus.body.data.length > 0, 'current user has no visible menus')
  console.log('OK user menus')
}

async function assertTableMetadata() {
  const tableCode = runtime.tableCode
  const table = await request(`/admin/table/code/${tableCode}`)
  assert(table.status === 200 && table.body?.success && table.body?.data?.id, `table metadata missing: ${JSON.stringify(table.body)}`)
  assert(Array.isArray(table.body.data?.table_fields), `table fields missing: ${JSON.stringify(table.body.data)}`)

  const menus = await request('/admin/menu/my')
  const menu = findMenuByOption(menus.body?.data, tableCode)
  assert(menu?.id || tableCode === 'sys_user', `published menu not visible for ${tableCode}`)
  console.log(`OK metadata ${tableCode}`)
}

async function assertApplicationRedaction() {
  const query = await request('/admin/application/query', {
    method: 'POST',
    body: JSON.stringify({
      table_code: 'application',
      page: 1,
      num: 10,
      quick_query: { keyword: '' },
      expressions: [],
    }),
  })
  assert(query.status === 200 && query.body?.success, `application query failed: ${JSON.stringify(query.body)}`)
  assertNoSecretFields(query.body?.data, 'application query')
  const rows = query.body?.data || []
  const application = rows.find((item) => item.app_key === 'sweet-admin') || rows[0]
  if (application?.id) {
    const detail = await request(`/admin/application/${application.id}`)
    assert(detail.status === 200 && detail.body?.success, `application detail failed: ${JSON.stringify(detail.body)}`)
    assertNoSecretFields(detail.body?.data, 'application detail')
  }
  console.log('OK application redaction')
}

async function assertAuditQuery() {
  const audit = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({
      table_code: 'access_log',
      page: 1,
      num: 1,
      quick_query: { keyword: '' },
      expressions: [],
    }),
  })
  assert(audit.status === 200 && audit.body?.success, `audit query failed: ${JSON.stringify(audit.body)}`)
  assert(Array.isArray(audit.body?.data), `audit query returned invalid data: ${JSON.stringify(audit.body)}`)
  assertNoSecretFields(audit.body?.data, 'audit query')
  console.log('OK audit query')
}

async function main() {
  loadOptionalEnvFile(process.env.SWEET_ADMIN_EXTERNAL_ENV_FILE)
  runtime = createSmokeRuntime(process.env)
  accessToken = ''

  console.log(`Readonly smoke target: ${runtime.baseUrl}`)
  console.log(`Health target: ${runtime.healthBaseUrl}`)

  await assertHealth()
  await assertPublicConfigure()
  await assertLogin()
  await assertUserAndMenu()
  await assertTableMetadata()
  await assertApplicationRedaction()
  await assertAuditQuery()

  console.log('Readonly smoke passed')
}

const currentFile = fileURLToPath(import.meta.url)
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
