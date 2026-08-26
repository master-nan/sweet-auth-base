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
    throw new Error(`只读基础可用性测试的环境文件不存在：${resolvedPath}`)
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
    baseUrl: normalizeBaseUrl(env.SWEET_ADMIN_BASE_URL || 'http://localhost:8008/sweet_admin'),
    healthBaseUrl: normalizeBaseUrl(env.SWEET_ADMIN_HEALTH_BASE_URL || 'http://localhost:9009'),
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
    assert(!new RegExp(`"${key}"\\s*:\\s*"[^"]+`).test(rendered), `${context} 返回了敏感字段 ${key}：${rendered}`)
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
  assert(health.status === 200 && health.body?.status === 'ok', `healthz 检查失败：${JSON.stringify(health.body)}`)
  const ready = await requestRaw('/readyz')
  assert(ready.status === 200 && ready.body?.status === 'ready', `readyz 检查失败：${JSON.stringify(ready.body)}`)
  assert(
    ready.body?.components?.db_primary?.ok || ready.body?.components?.db?.ok,
    `readyz 未返回可用的主数据库状态：${JSON.stringify(ready.body)}`,
  )
  assert(
    ready.body?.components?.redis?.ok,
    `readyz 未返回可用的 Redis 状态：${JSON.stringify(ready.body)}`,
  )
  console.log('通过：healthz 与 readyz')
}

async function assertPublicConfigure() {
  const configure = await request('/admin/configure')
  assert(configure.status === 200 && configure.body?.success, `公开配置接口检查失败：${JSON.stringify(configure.body)}`)
  assertNoSecretFields(configure.body?.data, 'public configure')
  runtime.captcha.enabled = Boolean(configure.body?.data?.enable_captcha)
  console.log(`通过：公开配置未泄露秘密${runtime.captcha.enabled ? '（已启用验证码）' : ''}`)
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
  assert(captcha.status === 200 && captcha.body?.success, `验证码接口检查失败：${JSON.stringify(captcha.body)}`)
  assert(captcha.body?.data?.captcha_id, `验证码响应缺少 captcha_id：${JSON.stringify(captcha.body)}`)
  return captcha.body.data
}

function saveCaptchaImageIfRequested(challenge) {
  if (!runtime.captcha.imageFile) return ''
  const image = decodeCaptchaImage(challenge.image)
  assert(image.length > 0, `验证码响应缺少图片数据：${JSON.stringify(challenge)}`)
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
  const imageHint = imagePath ? ` 验证码图片已保存到 ${imagePath}。` : ''
  throw new Error(
    `登录已启用验证码；请设置 SWEET_ADMIN_SMOKE_CAPTCHA_ID=${challenge.captcha_id} 和 SWEET_ADMIN_SMOKE_CAPTCHA=<code>，再重新运行只读基础可用性测试。${imageHint}`,
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
  assert(login.status === 200 && login.body?.success, `登录检查失败：${JSON.stringify(login.body)}`)
  accessToken = login.body.data?.access_token || ''
  assert(accessToken, '登录响应缺少 access_token')
  assert(login.body.data?.must_change_password === false, `测试管理员需要先修改密码：${JSON.stringify(login.body.data)}`)
  console.log('通过：登录')
}

async function assertUserAndMenu() {
  const me = await request('/admin/user/me')
  assert(me.status === 200 && me.body?.success && me.body?.data?.id, `当前用户接口检查失败：${JSON.stringify(me.body)}`)

  const menus = await request('/admin/menu/my')
  assert(menus.status === 200 && menus.body?.success && Array.isArray(menus.body?.data), `菜单查询失败：${JSON.stringify(menus.body)}`)
  assert(menus.body.data.length > 0, '当前用户没有可见菜单')
  console.log('通过：当前用户与菜单')
}

async function assertTableMetadata() {
  const tableCode = runtime.tableCode
  const table = await request(`/admin/table/code/${tableCode}`)
  assert(table.status === 200 && table.body?.success && table.body?.data?.id, `找不到表 Metadata：${JSON.stringify(table.body)}`)
  assert(Array.isArray(table.body.data?.table_fields), `表 Metadata 缺少字段：${JSON.stringify(table.body.data)}`)

  const menus = await request('/admin/menu/my')
  const menu = findMenuByOption(menus.body?.data, tableCode)
  assert(menu?.id || tableCode === 'sys_user', `已发布表 ${tableCode} 没有可见菜单`)
  console.log(`通过：Metadata ${tableCode}`)
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
  assert(query.status === 200 && query.body?.success, `应用查询失败：${JSON.stringify(query.body)}`)
  assertNoSecretFields(query.body?.data, 'application query')
  const rows = query.body?.data || []
  const application = rows.find((item) => item.app_key === 'sweet-admin') || rows[0]
  if (application?.id) {
    const detail = await request(`/admin/application/${application.id}`)
    assert(detail.status === 200 && detail.body?.success, `应用详情检查失败：${JSON.stringify(detail.body)}`)
    assertNoSecretFields(detail.body?.data, 'application detail')
  }
  console.log('通过：应用列表和详情未泄露秘密')
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
  assert(audit.status === 200 && audit.body?.success, `审计日志查询失败：${JSON.stringify(audit.body)}`)
  assert(Array.isArray(audit.body?.data), `审计日志返回格式错误：${JSON.stringify(audit.body)}`)
  assertNoSecretFields(audit.body?.data, 'audit query')
  console.log('通过：审计日志查询')
}

async function main() {
  loadOptionalEnvFile(process.env.SWEET_ADMIN_EXTERNAL_ENV_FILE)
  runtime = createSmokeRuntime(process.env)
  accessToken = ''

  console.log(`只读基础可用性测试地址：${runtime.baseUrl}`)
  console.log(`健康检查地址：${runtime.healthBaseUrl}`)

  await assertHealth()
  await assertPublicConfigure()
  await assertLogin()
  await assertUserAndMenu()
  await assertTableMetadata()
  await assertApplicationRedaction()
  await assertAuditQuery()

  console.log('只读基础可用性测试通过。')
}

const currentFile = fileURLToPath(import.meta.url)
if (process.argv[1] && path.resolve(process.argv[1]) === currentFile) {
  main().catch((error) => {
    console.error(error.message)
    process.exit(1)
  })
}
