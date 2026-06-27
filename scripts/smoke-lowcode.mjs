#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'

const baseUrl = normalizeBaseUrl(process.env.SWEET_ADMIN_BASE_URL || 'http://localhost:8008/sweet_admin')
const healthBaseUrl = normalizeBaseUrl(process.env.SWEET_ADMIN_HEALTH_BASE_URL || 'http://localhost:9009')
const username = process.env.SWEET_ADMIN_ADMIN_USER || 'admin'
const password = process.env.SWEET_ADMIN_ADMIN_PASSWORD || 'admin123'
const tableCode = process.env.SWEET_ADMIN_SMOKE_TABLE || `smk_publish_${Date.now().toString(36)}`
const crudTableCode =
  process.env.SWEET_ADMIN_SMOKE_CRUD_TABLE || `smk_${Date.now().toString(36)}`
const dropPhysicalSmokeTables = process.env.SWEET_ADMIN_SMOKE_DROP_PHYSICAL === '1'
const passwordSalt =
  process.env.SWEET_ADMIN_SMOKE_PASSWORD_SALT || 'local-docker-sweet-admin-salt-change-me'

let accessToken = ''

function normalizeBaseUrl(value) {
  return value.replace(/\/+$/, '')
}

function apiPath(path) {
  return `${baseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

function assertNoSecretField(record, field, message) {
  assert(record?.[field] === undefined || record[field] === '', message)
}

function parsePostgresBool(value) {
  return ['1', 't', 'true', 'yes', 'y'].includes(String(value || '').trim().toLowerCase())
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
  return { status: res.status, body }
}

async function requestMultipart(path, formData, options = {}) {
  const headers = {
    ...(options.headers || {}),
  }
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`
  }
  const res = await fetch(apiPath(path), {
    ...options,
    method: options.method || 'POST',
    headers,
    body: formData,
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
  return { status: res.status, body }
}

async function requestRaw(path, options = {}) {
  const url = /^https?:\/\//i.test(path) ? path : `${healthBaseUrl}${path.startsWith('/') ? path : `/${path}`}`
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

async function requestApiWithoutAuth(path, options = {}) {
  const res = await fetch(apiPath(path), {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
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

async function loginAs(userName, rawPassword) {
  const previousToken = accessToken
  accessToken = ''
  const login = await request('/admin/login', {
    method: 'POST',
    body: JSON.stringify({
      user_name: userName,
      password: rawPassword,
      captcha: '',
      captcha_id: '',
    }),
  })
  accessToken = previousToken
  return login
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message)
  }
}

function assertSmokeTableCode(code) {
  assert(
    /^(smk|smoke|smoke_long)_[A-Za-z0-9_]+$/.test(code),
    `refusing to hard cleanup non-smoke table: ${code}`,
  )
}

function isSmokeTableCode(code) {
  return /^(smk|smoke|smoke_long)_[A-Za-z0-9_]+$/.test(code)
}

function runPostgres(sql) {
  return execFileSync(
    'docker',
    [
      'compose',
      'exec',
      '-T',
      '-e',
      'PGPASSWORD=sweet_admin',
      'postgres',
      'psql',
      '-U',
      'sweet_admin',
      '-d',
      'sweet_admin',
      '-At',
      '-F',
      '\t',
      '-v',
      'ON_ERROR_STOP=1',
      '-c',
      sql,
    ],
    { encoding: 'utf8', stdio: 'pipe' },
  ).trim()
}

function hardCleanupSmokeTable(code) {
  if (!dropPhysicalSmokeTables) return
  assertSmokeTableCode(code)
  const sqlCode = code.replaceAll("'", "''")
  const menuWhere = `table_code = '${sqlCode}' OR "option" = '${sqlCode}'`
  const sql = `
DELETE FROM sys_user_data_scope_override WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_role_data_scope WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_data_scope_binding WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_role_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_menu WHERE id IN (SELECT id FROM sys_menu WHERE ${menuWhere});
DELETE FROM sys_table_index_field WHERE index_id IN (SELECT id FROM sys_table_index WHERE table_id IN (SELECT id FROM sys_table WHERE table_code = '${sqlCode}' LIMIT 1));
DELETE FROM sys_table_index WHERE table_id IN (SELECT id FROM sys_table WHERE table_code = '${sqlCode}' LIMIT 1);
DELETE FROM sys_table_relation WHERE table_id IN (SELECT id FROM sys_table WHERE table_code = '${sqlCode}' LIMIT 1);
DELETE FROM sys_table_field WHERE table_id IN (SELECT id FROM sys_table WHERE table_code = '${sqlCode}' LIMIT 1);
DELETE FROM sys_table WHERE id IN (SELECT id FROM sys_table WHERE table_code = '${sqlCode}' LIMIT 1);
DROP TABLE IF EXISTS "${code}";
`
  runPostgres(sql)
}

function cleanupStaleSmokeArtifacts() {
  if (!dropPhysicalSmokeTables) return
  const output = runPostgres(`
SELECT table_code
FROM sys_table
WHERE table_code ~ '^(smk|smoke|smoke_long)_[A-Za-z0-9_]+$'
UNION
SELECT table_code
FROM sys_menu
WHERE table_code ~ '^(smk|smoke|smoke_long)_[A-Za-z0-9_]+$'
UNION
SELECT "option"
FROM sys_menu
WHERE "option" ~ '^(smk|smoke|smoke_long)_[A-Za-z0-9_]+$';
`)
  const codes = output ? output.split('\n').map((item) => item.trim()).filter(Boolean) : []
  for (const code of codes) {
    hardCleanupSmokeTable(code)
  }
}

function cleanupSmokeRole(roleName) {
  if (!dropPhysicalSmokeTables) return
  assert(/^smoke_role_[A-Za-z0-9_]+$/.test(roleName), `refusing to cleanup non-smoke role: ${roleName}`)
  const sqlName = sqlString(roleName)
  runPostgres(`
DELETE FROM casbin_rule WHERE v0 = '${sqlName}';
DELETE FROM sys_role_menu_button WHERE role_id IN (SELECT id FROM sys_role WHERE name = '${sqlName}' LIMIT 1);
DELETE FROM sys_role_menu WHERE role_id IN (SELECT id FROM sys_role WHERE name = '${sqlName}' LIMIT 1);
DELETE FROM sys_user_role WHERE role_id IN (SELECT id FROM sys_role WHERE name = '${sqlName}' LIMIT 1);
DELETE FROM sys_role WHERE id IN (SELECT id FROM sys_role WHERE name = '${sqlName}' LIMIT 1);
	`)
}

function hashUserPassword(rawPassword, userId) {
  return createHash('md5').update(`${rawPassword}${userId}${passwordSalt}`).digest('hex')
}

function cleanupSmokeUser(userName) {
  if (!dropPhysicalSmokeTables) return
  assert(/^smoke_user_[A-Za-z0-9_]+$/.test(userName), `refusing to cleanup non-smoke user: ${userName}`)
  const sqlName = sqlString(userName)
  const existing = runPostgres(`
SELECT id, phone_number
FROM sys_user
WHERE user_name = '${sqlName}'
LIMIT 1;
`)
  let userId = ''
  let phone = ''
  if (existing) {
    const parts = existing.split('\t')
    userId = parts[0] || ''
    phone = parts[1] || ''
	  }
	  runPostgres(`
	DELETE FROM sys_user_data_scope_override WHERE user_id IN (SELECT id FROM sys_user WHERE user_name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_user_role WHERE user_id IN (SELECT id FROM sys_user WHERE user_name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_user WHERE id IN (SELECT id FROM sys_user WHERE user_name = '${sqlName}' LIMIT 1);
	`)
  clearUserCacheKeys(userId ? `USER_CACHE_KEY_${userId}` : '', `USER_CACHE_KEY_${userName}`, phone ? `USER_CACHE_KEY_${phone}` : '')
}

function createSmokeUserWithRole(userName, roleId) {
  if (!dropPhysicalSmokeTables) return null
  assert(/^smoke_user_[A-Za-z0-9_]+$/.test(userName), `refusing to create non-smoke user: ${userName}`)
  const suffix = Math.floor(Date.now() % 100000000)
  const userId = 800000000 + suffix
  const phone = `139${String(userId).slice(-8)}`
  const email = `${userName}@smoke.local`
  const rawPassword = 'admin123'
  const passwordHash = hashUserPassword(rawPassword, userId)
  const sqlName = sqlString(userName)
  cleanupSmokeUser(userName)
  runPostgres(`
DELETE FROM sys_user_role WHERE user_id IN (
  SELECT id FROM sys_user WHERE id = ${userId} OR phone_number = '${sqlString(phone)}'
);
DELETE FROM sys_user WHERE id = ${userId} OR phone_number = '${sqlString(phone)}';
INSERT INTO sys_user
  (id, gmt_create, gmt_modify, state, user_name, password, email, phone_number, password_changed_at, language, access_tokens, is_reset)
VALUES
  (${userId}, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, true, '${sqlName}', '${passwordHash}', '${sqlString(email)}', '${sqlString(phone)}', CURRENT_TIMESTAMP, 'zh-CN', '', false);
INSERT INTO sys_user_role (user_id, role_id) VALUES (${userId}, ${Number(roleId)});
`)
  clearUserCacheKeys(`USER_CACHE_KEY_${userId}`, `USER_CACHE_KEY_${userName}`, `USER_CACHE_KEY_${phone}`)
  return { userId, userName, phone, password: rawPassword }
}

function cleanupSmokeMenu(menuName) {
  if (!dropPhysicalSmokeTables) return
	  assert(/^smoke_menu_[A-Za-z0-9_]+$/.test(menuName), `refusing to cleanup non-smoke menu: ${menuName}`)
	  const sqlName = sqlString(menuName)
	  runPostgres(`
	DELETE FROM sys_user_data_scope_override WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_role_data_scope WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_data_scope_binding WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_role_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
	DELETE FROM sys_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
DELETE FROM sys_menu WHERE id IN (SELECT id FROM sys_menu WHERE name = '${sqlName}' LIMIT 1);
`)
}

function prepareLocalSmokeRuntime() {
  if (!dropPhysicalSmokeTables) return
  runPostgres("UPDATE sys_configure SET enable_captcha = false, sender_password = 'smoke-secret';")
  runPostgres("UPDATE sys_user SET is_reset = false, password_changed_at = CURRENT_TIMESTAMP WHERE id = 1;")
  execFileSync(
    'docker',
    [
      'compose',
      'exec',
      '-T',
      'redis',
      'redis-cli',
      '-n',
      '5',
      'DEL',
      'CONFIGURE_CACHE_KEY_',
      'USER_CACHE_KEY_1',
      `USER_CACHE_KEY_${username}`,
      'USER_CACHE_KEY_13800000000',
    ],
    { stdio: 'pipe' },
  )
}

function clearConfigureSmokeSecret() {
  if (!dropPhysicalSmokeTables) return
  runPostgres("UPDATE sys_configure SET sender_password = '';")
  execFileSync(
    'docker',
    ['compose', 'exec', '-T', 'redis', 'redis-cli', '-n', '5', 'DEL', 'CONFIGURE_CACHE_KEY_'],
    { stdio: 'pipe' },
  )
}

function clearUserCache() {
  if (!dropPhysicalSmokeTables) return
  clearUserCacheKeys('USER_CACHE_KEY_1', `USER_CACHE_KEY_${username}`, 'USER_CACHE_KEY_13800000000')
}

function clearUserCacheKeys(...keys) {
  if (!dropPhysicalSmokeTables) return
  const cacheKeys = keys.filter(Boolean)
  if (cacheKeys.length === 0) return
  execFileSync(
    'docker',
    [
      'compose',
      'exec',
      '-T',
      'redis',
      'redis-cli',
      '-n',
      '5',
      'DEL',
      ...cacheKeys,
    ],
    { stdio: 'pipe' },
  )
}

function sqlString(value) {
  return String(value).replaceAll("'", "''")
}

function latestAccessLogAfter(id, urlLike) {
  const output = runPostgres(`
SELECT id, COALESCE(url, ''), COALESCE(body, ''), COALESCE(query, ''), COALESCE(response, '')
FROM access_log
WHERE id > ${Number(id)}
  AND url LIKE '${sqlString(urlLike)}'
ORDER BY id DESC
LIMIT 1;
`)
  if (!output) return null
  const [rowId, url, body, query, ...responseParts] = output.split('\t')
  return {
    id: Number(rowId),
    url: url || '',
    body: body || '',
    query: query || '',
    response: responseParts.join('\t') || '',
  }
}

async function waitForAccessLogAfter(id, urlLike, label) {
  for (let index = 0; index < 20; index += 1) {
    const row = latestAccessLogAfter(id, urlLike)
    if (row) return row
    await new Promise((resolve) => setTimeout(resolve, 100))
  }
  throw new Error(`missing ${label} access log after id ${id}`)
}

function assertAuditLogs(code, actions) {
  if (!dropPhysicalSmokeTables) return
  const actionList = actions.map((action) => `'${sqlString(action)}'`).join(',')
  const count = Number(
    runPostgres(`
SELECT COUNT(*)
FROM access_log
WHERE resource_code = '${sqlString(code)}'
  AND user_name = '${sqlString(username)}'
  AND success = true
  AND action IN (${actionList});
`),
  )
  assert(
    count >= actions.length,
    `missing operation audit logs for ${code}: found ${count}, expected at least ${actions.length}`,
  )
}

function assertSensitiveTableFieldsHidden(table, fieldCodes) {
  const fields = Array.isArray(table?.table_fields) ? table.table_fields : []
  for (const fieldCode of fieldCodes) {
    const field = fields.find((item) => item.field_code === fieldCode)
    assert(field, `sensitive field metadata missing: ${fieldCode}`)
    assert(
      !field.is_list_show &&
        !field.is_insert_show &&
        !field.is_update_show &&
        !field.is_quick_search &&
        !field.is_advanced_search,
      `sensitive field ${fieldCode} is exposed in low-code metadata: ${JSON.stringify(field)}`,
    )
  }
}

function assertSensitiveRecordFieldsAbsent(records, fieldCodes) {
  assert(Array.isArray(records), 'low-code records should be an array')
  for (const record of records) {
    for (const fieldCode of fieldCodes) {
      assert(
        !Object.prototype.hasOwnProperty.call(record || {}, fieldCode),
        `sensitive field ${fieldCode} leaked in low-code response: ${JSON.stringify(record)}`,
      )
    }
  }
}

function assertLowCodeManagedFieldMetadata(table, requiredFieldCodes = []) {
  const fields = Array.isArray(table?.table_fields) ? table.table_fields : []
  for (const fieldCode of requiredFieldCodes) {
    const field = fields.find((item) => item.field_code === fieldCode)
    assert(field, `managed field metadata missing: ${fieldCode}`)
    assert(
      !field.is_list_show &&
        !field.is_insert_show &&
        !field.is_update_show &&
        !field.is_quick_search &&
        !field.is_advanced_search,
      `managed field ${fieldCode} is exposed in low-code list/write/search metadata: ${JSON.stringify(field)}`,
    )
  }
}

function assertAuditPermissionsSeeded() {
  if (!dropPhysicalSmokeTables) return
  const buttonCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM sys_menu_button b
JOIN sys_menu m ON m.id = b.menu_id
WHERE m.name = 'system_audit'
  AND (
    (b.code = 'system_audit_query' AND b.path = '/admin/log/access/query' AND b.method = 'POST')
    OR (b.code = 'system_audit_detail' AND b.path = '/admin/log/access/:id' AND b.method = 'GET')
  );
`),
  )
  assert(buttonCount === 2, `audit permission buttons missing: ${buttonCount}`)

  const policyCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p'
  AND v0 = 'super_admin'
  AND (
    (v1 = '/admin/log/access/query' AND v2 = 'POST')
    OR (v1 = '/admin/log/access/:id' AND v2 = 'GET')
  );
`),
  )
  assert(policyCount === 2, `audit casbin policies missing: ${policyCount}`)

  const userPermissionButtonCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM sys_menu_button b
JOIN sys_menu m ON m.id = b.menu_id
	WHERE m.name = 'system_user'
	  AND (
	    (b.code = 'system_user_data_permission' AND b.path = '/admin/user/:id/data-permissions' AND b.method = 'PUT')
	    OR (b.code = 'system_user_menu_query' AND b.path = '/admin/menu/user/:id' AND b.method = 'GET')
	    OR (b.code = 'system_user_data_permission_query' AND b.path = '/admin/user/:id/data-permissions' AND b.method = 'GET')
	  );
	`),
	  )
  assert(userPermissionButtonCount === 3, `user data permission buttons missing: ${userPermissionButtonCount}`)

  const userPermissionPolicyCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
	WHERE ptype = 'p'
	  AND v0 = 'super_admin'
	  AND (
	    (v1 = '/admin/user/:id/data-permissions' AND v2 = 'PUT')
	    OR (v1 = '/admin/menu/user/:id' AND v2 = 'GET')
	    OR (v1 = '/admin/user/:id/data-permissions' AND v2 = 'GET')
	  );
	`),
  )
  assert(userPermissionPolicyCount === 3, `user data permission casbin policies missing: ${userPermissionPolicyCount}`)

  const routePolicyCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p'
  AND v0 = 'super_admin'
  AND (
    (v1 = '/admin/table/publish/:code' AND v2 = 'POST')
    OR (v1 = '/admin/generalization/query/code/:code' AND v2 = 'POST')
    OR (v1 = '/admin/generalization/create' AND v2 = 'POST')
    OR (v1 = '/admin/role/assign-permissions' AND v2 = 'POST')
    OR (v1 = '/admin/menu/button/:id' AND v2 = 'PUT')
    OR (v1 = '/admin/file/upload/init' AND v2 = 'POST')
  );
`),
  )
  assert(routePolicyCount === 6, `super_admin route policy coverage missing: ${routePolicyCount}`)
  assertSuperAdminRoutePolicyCoverage()
}

function assertSuperAdminRoutePolicyCoverage() {
  if (!dropPhysicalSmokeTables) return
  const source = readFileSync('backend/initialize/router.go', 'utf8')
  const allowMissing = new Set(['POST /admin/logout', 'GET /admin/user/me', 'GET /admin/menu/my'])
  const routeRegex = /adminGroup\.(GET|POST|PUT|DELETE)\("([^"]+)"/g
  const expected = []
  for (const match of source.matchAll(routeRegex)) {
    const method = match[1]
    const path = `/admin${match[2]}`
    const key = `${method} ${path}`
    if (!allowMissing.has(key)) {
      expected.push(key)
    }
  }
  const policyRows = runPostgres(`
SELECT CONCAT(v2, ' ', v1)
FROM casbin_rule
WHERE ptype = 'p' AND v0 = 'super_admin';
`)
  const policies = new Set(policyRows ? policyRows.split('\n').map((item) => item.trim()).filter(Boolean) : [])
  const missing = expected.filter((key) => !policies.has(key))
  assert(
    missing.length === 0,
    `super_admin Casbin route policies missing: ${missing.slice(0, 10).join(', ')}`,
  )
}

function assertBuiltinPermissionButtonsSeeded() {
  if (!dropPhysicalSmokeTables) return
  const expected = [
    ['develop_configure', 'develop_configure_detail', 'GET', '/admin/configure/detail', true],
    ['develop_configure', 'develop_configure_save', 'PUT', '/admin/configure/:id', false],
    ['system_application', 'system_application_query', 'POST', '/admin/application/query', true],
    ['system_application', 'system_application_metadata', 'GET', '/admin/table/code/:code', true],
    ['system_application', 'system_application_detail', 'GET', '/admin/application/:id', true],
    ['system_application', 'system_application_create', 'POST', '/admin/application', false],
    ['system_application', 'system_application_update', 'PUT', '/admin/application/:id', false],
    ['system_application', 'system_application_rotate_secret', 'POST', '/admin/application/:id/rotate-secret', false],
    ['system_application', 'system_application_delete', 'DELETE', '/admin/application/:id', false],
    ['system_sms', 'system_sms_query', 'POST', '/admin/sms/template/query', true],
    ['system_sms', 'system_sms_metadata', 'GET', '/admin/table/code/:code', true],
    ['system_sms', 'system_sms_detail', 'GET', '/admin/sms/template/:id', true],
    ['system_menu', 'system_menu_query', 'POST', '/admin/menu/query', true],
    ['system_menu', 'system_menu_button_query', 'GET', '/admin/menu/buttons/:menuId', true],
    ['system_menu', 'system_menu_order', 'PUT', '/admin/menu/order', true],
    ['system_menu', 'system_menu_refresh_cache', 'POST', '/admin/menu/refresh-cache', true],
    ['system_menu', 'system_menu_create', 'POST', '/admin/menu', false],
    ['system_menu', 'system_menu_update', 'PUT', '/admin/menu/:id', false],
    ['system_menu', 'system_menu_button_create', 'POST', '/admin/menu/button', false],
    ['system_menu', 'system_menu_button_update', 'PUT', '/admin/menu/button/:id', false],
    ['system_role', 'system_role_query', 'POST', '/admin/role/query', true],
    ['system_role', 'system_role_metadata', 'GET', '/admin/table/code/:code', true],
    ['system_role', 'system_role_detail', 'GET', '/admin/role/:id', true],
    ['system_role', 'system_role_permission_menu_query', 'POST', '/admin/menu/query', true],
    ['system_role', 'system_role_assign_permission', 'POST', '/admin/role/assign-permissions', false],
    ['system_user', 'system_user_query', 'POST', '/admin/user/query', true],
    ['system_user', 'system_user_metadata', 'GET', '/admin/table/code/:code', true],
    ['system_user', 'system_user_detail', 'GET', '/admin/user/:id', true],
    ['system_user', 'system_user_create', 'POST', '/admin/user', false],
    ['system_user', 'system_user_update', 'PUT', '/admin/user/:id', false],
    ['system_user', 'system_user_reset_password', 'POST', '/admin/user/reset_password/:id', false],
    ['develop_database', 'develop_database_query', 'POST', '/admin/table/query', true],
    ['develop_database', 'develop_database_metadata', 'GET', '/admin/table/code/:code', true],
    ['develop_database', 'develop_database_field_query', 'GET', '/admin/table/fields/:id', true],
    ['develop_database', 'develop_database_field_update', 'PUT', '/admin/table/field/:id', true],
    ['develop_database', 'develop_database_index_query', 'GET', '/admin/table/indexes/:id', true],
    ['develop_database', 'develop_database_index_sync', 'POST', '/admin/table/sync/index/:code', true],
    ['develop_database', 'develop_database_relation_query', 'GET', '/admin/table/relations/:id', true],
    ['develop_database', 'develop_database_relation_create', 'POST', '/admin/table/relation', true],
    ['develop_dictionary', 'develop_dictionary_query', 'POST', '/admin/dict/query', true],
    ['develop_dictionary', 'develop_dictionary_metadata', 'GET', '/admin/table/code/:code', true],
    ['develop_dictionary', 'develop_dictionary_item_query', 'GET', '/admin/dict/items/:id', true],
    ['develop_dictionary', 'develop_dictionary_create', 'POST', '/admin/dict', false],
    ['develop_dictionary', 'develop_dictionary_item_create', 'POST', '/admin/dict/item', false],
  ]
  const menuNames = [...new Set(expected.map(([menu]) => menu))]
    .map((menu) => `'${menu.replaceAll("'", "''")}'`)
    .join(',')
  const rows = runPostgres(`
SELECT CONCAT(m.name, '\t', b.code, '\t', b.method, '\t', b.path, '\t', NOT b.is_button)
FROM sys_menu_button b
JOIN sys_menu m ON m.id = b.menu_id
WHERE m.name IN (${menuNames});
`)
  const actual = new Map(
    rows
      ? rows.split('\n').map((line) => {
          const [menu, code, method, path, apiOnly] = line.split('\t')
          return [`${menu}|${code}`, { method, path, apiOnly: parsePostgresBool(apiOnly) }]
        })
      : [],
  )
  const missing = []
  const mismatched = []
  for (const [menu, code, method, path, apiOnly] of expected) {
    const key = `${menu}|${code}`
    const item = actual.get(key)
    if (!item) {
      missing.push(key)
      continue
    }
    if (item.method !== method || item.path !== path || item.apiOnly !== apiOnly) {
      mismatched.push(`${key}: ${item.method} ${item.path} api_only=${item.apiOnly}`)
    }
  }
  assert(missing.length === 0, `builtin permission buttons missing: ${missing.join(', ')}`)
  assert(mismatched.length === 0, `builtin permission buttons mismatched: ${mismatched.join('; ')}`)
}

async function assertApplicationSecretsRedacted() {
  let smokeApplicationId = 0
  try {
    const smokeApplicationName = `Smoke Application ${Date.now().toString(36)}`
    const created = await request('/admin/application', {
      method: 'POST',
      body: JSON.stringify({
        name: smokeApplicationName,
        expiration: 3600,
        ding_key: '',
        ding_secret: '',
        ding_app_id: '',
        remark: 'smoke application secret lifecycle',
      }),
    })
    assert(
      created.status === 200 && created.body?.success && created.body.data?.id,
      `application create failed: ${JSON.stringify(created.body)}`,
    )
    smokeApplicationId = created.body.data.id
    assert(
      created.body.data?.app_key && created.body.data?.app_secret,
      `application create did not return one-time credential: ${JSON.stringify(created.body)}`,
    )
    const originalSecret = created.body.data.app_secret

    const createdDetail = await request(`/admin/application/${smokeApplicationId}`)
    assert(
      createdDetail.status === 200 && createdDetail.body?.success,
      `created application detail failed: ${JSON.stringify(createdDetail.body)}`,
    )
    assertNoSecretField(
      createdDetail.body.data,
      'app_secret',
      `created application detail leaked app_secret: ${JSON.stringify(createdDetail.body.data)}`,
    )

    const rotated = await request(`/admin/application/${smokeApplicationId}/rotate-secret`, { method: 'POST' })
    assert(
      rotated.status === 200 && rotated.body?.success && rotated.body.data?.app_secret,
      `application secret rotate failed: ${JSON.stringify(rotated.body)}`,
    )
    assert(
      rotated.body.data.app_secret !== originalSecret,
      `application secret did not rotate: ${JSON.stringify(rotated.body.data)}`,
    )

    const rotatedDetail = await request(`/admin/application/${smokeApplicationId}`)
    assert(
      rotatedDetail.status === 200 && rotatedDetail.body?.success,
      `rotated application detail failed: ${JSON.stringify(rotatedDetail.body)}`,
    )
    assertNoSecretField(
      rotatedDetail.body.data,
      'app_secret',
      `rotated application detail leaked app_secret: ${JSON.stringify(rotatedDetail.body.data)}`,
    )
  } finally {
    if (smokeApplicationId > 0) {
      const deleted = await request(`/admin/application/${smokeApplicationId}`, { method: 'DELETE' })
      assert(deleted.status === 200 && deleted.body?.success, `delete smoke application failed: ${JSON.stringify(deleted.body)}`)
    }
  }

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
  const rows = query.body.data || []
  const application = rows.find((item) => item.app_key === 'sweet-admin') || rows[0]
  assert(application?.id, `application query returned no rows: ${JSON.stringify(query.body)}`)
  assertNoSecretField(application, 'app_secret', `application query leaked app_secret: ${JSON.stringify(application)}`)
  assert(!application.ding_secret, `application query leaked ding_secret: ${JSON.stringify(application)}`)

  const detail = await request(`/admin/application/${application.id}`)
  assert(detail.status === 200 && detail.body?.success, `application detail failed: ${JSON.stringify(detail.body)}`)
  assertNoSecretField(detail.body.data, 'app_secret', `application detail leaked app_secret: ${JSON.stringify(detail.body.data)}`)
  assert(!detail.body.data?.ding_secret, `application detail leaked ding_secret: ${JSON.stringify(detail.body.data)}`)
  console.log('OK application secret redaction')
}

function assertMetadataDictionariesSeeded() {
  if (!dropPhysicalSmokeTables) return
  const dictCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM sys_dict
WHERE dict_code IN (
  'whether',
  'sys_table_type',
  'sys_table_field_type',
  'sys_table_field_input_type',
  'sys_menu_button_position',
  'sys_table_relation_type',
  'sys_table_field_category'
);
`),
  )
  assert(dictCount === 7, `metadata dictionaries missing: ${dictCount}`)

  const fieldCount = Number(
    runPostgres(`
SELECT COUNT(*)
FROM sys_table_field f
JOIN sys_table t ON t.id = f.table_id
WHERE (t.table_code = 'sys_table' AND f.field_code = 'table_type' AND f.dict_code = 'sys_table_type')
   OR (t.table_code = 'sys_table_field' AND f.field_code = 'field_type' AND f.dict_code = 'sys_table_field_type')
   OR (t.table_code = 'sys_table_field' AND f.field_code = 'input_type' AND f.dict_code = 'sys_table_field_input_type')
   OR (t.table_code = 'sys_table_field' AND f.field_code = 'field_category' AND f.dict_code = 'sys_table_field_category')
   OR (t.table_code = 'sys_menu_button' AND f.field_code = 'position' AND f.dict_code = 'sys_menu_button_position')
   OR (t.table_code = 'sys_table_relation' AND f.field_code = 'relation_type' AND f.dict_code = 'sys_table_relation_type');
`),
  )
  assert(fieldCount === 6, `metadata field dictionary bindings missing: ${fieldCount}`)
  console.log('OK metadata dictionaries')
}

async function assertProtectedGeneralizationWriteGuard(menus) {
  const databaseMenu = findMenuByName(menus, 'develop_database')
  assert(databaseMenu?.id, 'develop_database menu not found for protected write guard')

  const denied = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: 'sys_table',
      menu_id: databaseMenu.id,
      data: {
        table_name: 'Smoke Protected Table',
        table_code: `smk_blocked_${Date.now().toString(36)}`,
        table_type: 1,
        parent_id: 0,
      },
    }),
  })
  assert(
    [400, 403].includes(denied.status) && denied.body?.success === false,
    `protected system table write was not blocked: ${JSON.stringify(denied.body)}`,
  )
  console.log('OK protected table write guard')
}

function pickDataPermissionField(table) {
  const fields = table?.table_fields || []
  const integerTypes = new Set([1, 9, 11])
  const priority = ['scope_id', 'tenant_id', 'project_id', 'owner_id', 'id']
  for (const fieldCode of priority) {
    const field = fields.find((item) => item.field_code === fieldCode && integerTypes.has(Number(item.field_type)))
    if (field) return field.field_code
  }
  const firstInteger = fields.find((item) => integerTypes.has(Number(item.field_type)))
  assert(firstInteger?.field_code, `table has no integer field for data permission smoke: ${JSON.stringify(table)}`)
  return firstInteger.field_code
}

async function assertUserDataPermissionApi(menuId, fixedMenuId, table) {
  const userMenus = await request('/admin/menu/user/1')
  assert(userMenus.status === 200 && userMenus.body?.success, `user menu query failed: ${JSON.stringify(userMenus.body)}`)
  const fieldCode = pickDataPermissionField(table)
  const dimensionCode = `smoke_scope_${Date.now().toString(36)}`

  const createdDimension = await request('/admin/data-permission/dimension', {
    method: 'POST',
    body: JSON.stringify({
      code: dimensionCode,
      name: 'Smoke Scope',
      value_type: 'number',
      source_type: 'none',
      memo: 'smoke data permission dimension',
      state: true,
    }),
  })
  assert(
    createdDimension.status === 200 && createdDimension.body?.success,
    `create data permission dimension failed: ${JSON.stringify(createdDimension.body)}`,
  )

  try {
    const invalidBinding = await request(`/admin/data-permission/bindings/menu/${menuId}`, {
      method: 'PUT',
      body: JSON.stringify({
        menu_id: menuId,
        bindings: [
          {
            dimension_code: dimensionCode,
            field_code: 'missing_data_field',
            match_type: 'in',
            actions: ['query'],
            required: true,
          },
        ],
      }),
    })
    assert(
      invalidBinding.status === 400 && invalidBinding.body?.error_code === 10000,
      `invalid data permission binding field was not rejected: ${JSON.stringify(invalidBinding.body)}`,
    )

    if (fixedMenuId) {
      const fixedMenuBinding = await request(`/admin/data-permission/bindings/menu/${fixedMenuId}`, {
        method: 'PUT',
        body: JSON.stringify({
          menu_id: fixedMenuId,
          bindings: [{ dimension_code: dimensionCode, field_code: 'id', match_type: 'in', actions: ['query'] }],
        }),
      })
      assert(
        fixedMenuBinding.status === 200 && fixedMenuBinding.body?.success,
        `fixed data permission menu binding failed: ${JSON.stringify(fixedMenuBinding.body)}`,
      )
      await request(`/admin/data-permission/bindings/menu/${fixedMenuId}`, {
        method: 'PUT',
        body: JSON.stringify({ menu_id: fixedMenuId, bindings: [] }),
      })
    }

    const savedBinding = await request(`/admin/data-permission/bindings/menu/${menuId}`, {
      method: 'PUT',
      body: JSON.stringify({
        menu_id: menuId,
        bindings: [
          {
            dimension_code: dimensionCode,
            field_code: fieldCode,
            match_type: 'in',
            actions: ['query', 'detail', 'create', 'update', 'delete'],
            required: true,
          },
        ],
      }),
    })
    assert(
      savedBinding.status === 200 && savedBinding.body?.success,
      `save menu data permission binding failed: ${JSON.stringify(savedBinding.body)}`,
    )

    const savedOverride = await request('/admin/user/1/data-permissions', {
      method: 'PUT',
      body: JSON.stringify({
        user_id: 1,
        overrides: [
          {
            menu_id: menuId,
            table_code: table.table_code,
            dimension_code: dimensionCode,
            strategy: 'specified',
            scope_values: ['3', '1', '3'],
            override_mode: 'replace',
          },
        ],
      }),
    })
    assert(
      savedOverride.status === 200 && savedOverride.body?.success,
      `save user data permission override failed: ${JSON.stringify(savedOverride.body)}`,
    )

    const overrides = await request('/admin/user/1/data-permissions')
    assert(
      overrides.status === 200 && overrides.body?.success,
      `user data permission override query failed: ${JSON.stringify(overrides.body)}`,
    )
    assert(
      overrides.body.data?.some(
        (item) =>
          item.menu_id === menuId &&
          item.dimension_code === dimensionCode &&
          Array.isArray(item.scope_values) &&
          item.scope_values.join(',') === '1,3',
      ),
      `saved user data permission override missing or not normalized: ${JSON.stringify(overrides.body.data)}`,
    )

    const cleared = await request('/admin/user/1/data-permissions', {
      method: 'PUT',
      body: JSON.stringify({ user_id: 1, overrides: [] }),
    })
    assert(cleared.status === 200 && cleared.body?.success, `clear user data permission failed: ${JSON.stringify(cleared.body)}`)
  } finally {
    await request(`/admin/data-permission/bindings/menu/${menuId}`, {
      method: 'PUT',
      body: JSON.stringify({ menu_id: menuId, bindings: [] }),
    })
    const dimensions = await request('/admin/data-permission/dimension/query', {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 1000,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    const created = dimensions.body?.data?.find((item) => item.code === dimensionCode)
    if (created?.id) {
      await request(`/admin/data-permission/dimension/${created.id}`, { method: 'DELETE' })
    }
  }
}

function normalizeDataScopeValues(scopeValues) {
  return String(scopeValues || '')
    .split(/[,\s;]+/)
    .map((item) => item.trim())
    .filter(Boolean)
    .sort()
}

function smokeDimensionCode(fieldCode) {
  assert(/^[A-Za-z_][A-Za-z0-9_]{0,48}$/.test(fieldCode), `invalid smoke data permission field: ${fieldCode}`)
  return `smoke_${fieldCode}`
}

function setMenuDataScope(menuId, scopeValues, fieldCode = 'id') {
  if (!dropPhysicalSmokeTables) return
  const numericMenuId = Number(menuId)
  const values = normalizeDataScopeValues(scopeValues)
  assert(values.length > 0, 'data scope values are required')
  const tableCode = runPostgres(`SELECT table_code FROM sys_menu WHERE id = ${numericMenuId} LIMIT 1;`)
  assert(tableCode, `menu ${numericMenuId} has no table_code for data scope smoke`)
  const dimensionCode = smokeDimensionCode(fieldCode)
  const idBase = Date.now() * 1000 + Math.floor(Math.random() * 1000)
  const actionsJson = sqlString(JSON.stringify(['batch_delete', 'create', 'delete', 'detail', 'export', 'query', 'update']))
  const valuesJson = sqlString(JSON.stringify(values))
  runPostgres(`
INSERT INTO sys_data_dimension
  (id, gmt_create, gmt_modify, state, code, name, value_type, source_type, source_code, label_field, value_field, parent_field, memo)
VALUES
  (${idBase}, NOW(), NOW(), true, '${sqlString(dimensionCode)}', 'Smoke ${sqlString(fieldCode)}', 'number', 'none', '', '', '', '', 'smoke data permission')
ON CONFLICT (code) DO UPDATE SET
  gmt_modify = NOW(),
  state = true,
  value_type = 'number',
  source_type = 'none',
  memo = 'smoke data permission';
DELETE FROM sys_user_data_scope_override WHERE menu_id = ${numericMenuId} AND dimension_code = '${sqlString(dimensionCode)}';
DELETE FROM sys_role_data_scope WHERE menu_id = ${numericMenuId} AND dimension_code = '${sqlString(dimensionCode)}';
DELETE FROM sys_data_scope_binding WHERE menu_id = ${numericMenuId} AND dimension_code = '${sqlString(dimensionCode)}';
INSERT INTO sys_data_scope_binding
  (id, gmt_create, gmt_modify, state, menu_id, table_code, dimension_code, field_code, match_type, required, actions)
VALUES
  (${idBase + 1}, NOW(), NOW(), true, ${numericMenuId}, '${sqlString(tableCode)}', '${sqlString(dimensionCode)}', '${sqlString(fieldCode)}', 'in', true, '${actionsJson}');
WITH numbered_roles AS (
  SELECT role_id, row_number() OVER (ORDER BY role_id) AS rn
  FROM sys_user_role
  WHERE user_id = 1
)
INSERT INTO sys_role_data_scope
  (id, gmt_create, gmt_modify, state, role_id, menu_id, table_code, dimension_code, strategy, scope_values)
SELECT
  ${idBase + 10} + rn,
  NOW(),
  NOW(),
  true,
  role_id,
  ${numericMenuId},
  '${sqlString(tableCode)}',
  '${sqlString(dimensionCode)}',
  'specified',
  '${valuesJson}'
FROM numbered_roles;
`)
}

function clearMenuDataScope(menuId) {
  if (!dropPhysicalSmokeTables) return
  runPostgres(`
DELETE FROM sys_user_data_scope_override WHERE menu_id = ${Number(menuId)};
DELETE FROM sys_role_data_scope WHERE menu_id = ${Number(menuId)};
DELETE FROM sys_data_scope_binding WHERE menu_id = ${Number(menuId)};
`)
}

function setMenuButtonDisabled(buttonId, disabled) {
  if (!dropPhysicalSmokeTables) return
  runPostgres(`UPDATE sys_menu_button SET is_disabled = ${disabled ? 'true' : 'false'}, gmt_modify = NOW() WHERE id = ${Number(buttonId)};`)
}

async function createSmokeMetadataTable(code, name) {
  await cleanupTable(code)
  const created = await request('/admin/table', {
    method: 'POST',
    body: JSON.stringify({
      table_name: name,
      table_code: code,
      table_type: 1,
      parent_id: 0,
      sql: '',
    }),
  })
  assert(created.status === 200 && created.body?.success, `create ${code} table failed: ${JSON.stringify(created.body)}`)
  const table = await fetchTableByCode(code)
  assert(table?.id, `${code} metadata missing after create`)
  return table
}

async function preparePublishSmokeTable(code) {
  if (!dropPhysicalSmokeTables || !isSmokeTableCode(code)) return
  const table = await createSmokeMetadataTable(code, 'Smoke Publish Item')
  await createSmokeTableField(table.id, {
    field_name: '名称',
    field_code: 'name',
    is_quick_search: true,
    is_advanced_search: true,
    is_null: false,
    binding: 'min=1|max=64',
    sequence: 9,
  })
  await createSmokeTableField(table.id, {
    field_name: '范围ID',
    field_code: 'scope_id',
    type: 11,
    input_type: 2,
    is_index: true,
    is_advanced_search: true,
    is_null: false,
    binding: 'min=1',
    sequence: 10,
  })
}

async function createSmokeTableField(tableId, overrides) {
  const field = {
    table_id: tableId,
    field_name: '名称',
    field_code: 'name',
    type: 3,
    field_length: 64,
    field_decimal_length: 0,
    input_type: 1,
    default_value: '',
    dict_code: '',
    is_primary_key: false,
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: true,
    is_list_show: true,
    is_insert_show: true,
    is_update_show: true,
    sequence: 9,
    binding: '',
    original_field_id: 0,
    field_category: 'normal_field',
    expression: '',
    linkage_config: '',
    ...overrides,
  }
  const created = await request('/admin/table/field', {
    method: 'POST',
    body: JSON.stringify(field),
  })
  assert(
    created.status === 200 && created.body?.success,
    `create ${field.field_code} field failed: ${JSON.stringify(created.body)}`,
  )
}

async function assertMetadataIdentifierGuard() {
  if (!dropPhysicalSmokeTables) return
  const invalidTable = await request('/admin/table', {
    method: 'POST',
    body: JSON.stringify({
      table_name: 'Smoke Invalid Identifier',
      table_code: `smk_${'a'.repeat(70)}`,
      table_type: 1,
      parent_id: 0,
      sql: '',
    }),
  })
  assert(invalidTable.status === 400, `overlong table_code was not rejected: ${JSON.stringify(invalidTable.body)}`)

  const code = `smk_ident_${Date.now().toString(36)}`
  await cleanupTable(code)
  try {
    const table = await createSmokeMetadataTable(code, 'Smoke Identifier Guard')
    const invalidField = await request('/admin/table/field', {
      method: 'POST',
      body: JSON.stringify({
        table_id: table.id,
        field_name: '非法字段',
        field_code: 'bad-name',
        type: 3,
        field_length: 64,
        field_decimal_length: 0,
        input_type: 1,
        default_value: '',
        dict_code: '',
        is_primary_key: false,
        is_index: false,
        is_quick_search: false,
        is_advanced_search: false,
        is_sort: false,
        is_null: true,
        is_list_show: true,
        is_insert_show: true,
        is_update_show: true,
        sequence: 9,
        binding: '',
        original_field_id: 0,
        field_category: 'normal_field',
        expression: '',
        linkage_config: '',
      }),
    })
    assert(invalidField.status === 400, `unsafe field_code was not rejected: ${JSON.stringify(invalidField.body)}`)

    await createSmokeTableField(table.id, {
      field_name: '名称',
      field_code: 'name',
      sequence: 9,
    })
    const { field } = await waitForTableField(code, 'name')
    const unsafeIndex = await request('/admin/table/index', {
      method: 'POST',
      body: JSON.stringify({
        table_id: table.id,
        index_name: 'idx-bad',
        is_unique: false,
        index_fields: [{ table_id: table.id, field_id: field.id, field_code: field.field_code }],
      }),
    })
    assert(unsafeIndex.status === 400, `unsafe index_name was not rejected: ${JSON.stringify(unsafeIndex.body)}`)

    const mismatchedIndex = await request('/admin/table/index', {
      method: 'POST',
      body: JSON.stringify({
        table_id: table.id,
        index_name: `idx_${Date.now().toString(36)}_mismatch`,
        is_unique: false,
        index_fields: [{ table_id: table.id, field_id: field.id, field_code: 'missing_name' }],
      }),
    })
    assert(
      mismatchedIndex.status === 400,
      `mismatched index field id/code was not rejected: ${JSON.stringify(mismatchedIndex.body)}`,
    )
    console.log('OK metadata identifier guard')
  } finally {
    await cleanupTable(code)
  }
}

async function assertForcedPasswordChangeFlow() {
  if (!dropPhysicalSmokeTables) return

  runPostgres("UPDATE sys_user SET is_reset = true WHERE id = 1;")
  clearUserCache()

  const previousToken = accessToken
  accessToken = ''
  const forcedLogin = await request('/admin/login', {
    method: 'POST',
    body: JSON.stringify({
      user_name: username,
      password,
      captcha: '',
      captcha_id: '',
    }),
  })
  assert(
    forcedLogin.status === 200 && forcedLogin.body?.success,
    `forced password-change login failed: ${JSON.stringify(forcedLogin.body)}`,
  )
  assert(
    forcedLogin.body.data?.must_change_password === true &&
      forcedLogin.body.data?.password_change_reason === 'initial_reset',
    `login did not require password change: ${JSON.stringify(forcedLogin.body.data)}`,
  )
  accessToken = forcedLogin.body.data.access_token
  assert(accessToken, 'forced password-change login did not include access_token')

  const changed = await request('/admin/user/password', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
  assert(changed.status === 200 && changed.body?.success, `password change failed: ${JSON.stringify(changed.body)}`)

  clearUserCache()
  const me = await request('/admin/user/me')
  assert(me.status === 200 && me.body?.success, `me after password change failed: ${JSON.stringify(me.body)}`)
  assert(me.body.data?.is_reset === false, `password change did not clear is_reset: ${JSON.stringify(me.body.data)}`)

  accessToken = previousToken || accessToken
  console.log('OK forced password change')
}

async function assertAuditApi(code, action) {
  const audit = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      resource_code: code,
      action,
      success: true,
    }),
  })
  assert(audit.status === 200 && audit.body?.success, `audit query failed: ${JSON.stringify(audit.body)}`)
  assert(audit.body.data?.length > 0, `audit query missing ${action} for ${code}`)
  const detailId = audit.body.data[0].id
  const detail = await request(`/admin/log/access/${detailId}`)
  assert(detail.status === 200 && detail.body?.success && detail.body.data?.id, 'audit detail failed')
}

async function assertAuditQueryGuard() {
  const invalidTime = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({ page: 1, num: 5, start_time: 'not-a-date' }),
  })
  assert(invalidTime.status === 400, `invalid audit time was not rejected: ${JSON.stringify(invalidTime.body)}`)

  const reversedRange = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      start_time: '2026-06-05 00:00:00',
      end_time: '2026-06-04 00:00:00',
    }),
  })
  assert(reversedRange.status === 400, `reversed audit time range was not rejected: ${JSON.stringify(reversedRange.body)}`)
  console.log('OK audit query guard')
}

async function assertAuditGenericQuery() {
  const quickQuery = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      quick_query: { keyword: '/admin/configure' },
      expressions: [],
    }),
  })
  assert(
    quickQuery.status === 200 && quickQuery.body?.success && quickQuery.body.data?.length > 0,
    `audit generic quick query failed: ${JSON.stringify(quickQuery.body)}`,
  )

  const expressionQuery = await request('/admin/log/access/query', {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      expressions: [
        {
          rules: [{ field: 'method', expression_type: 5, value: 'GET', type: 3 }],
          nested: [],
        },
      ],
    }),
  })
  assert(
    expressionQuery.status === 200 && expressionQuery.body?.success && expressionQuery.body.data?.length > 0,
    `audit generic expression query failed: ${JSON.stringify(expressionQuery.body)}`,
  )
  console.log('OK audit generic query')
}

async function assertAuditSensitiveRedaction() {
  if (!dropPhysicalSmokeTables) return

  const captchaBeforeId = Number(runPostgres('SELECT COALESCE(MAX(id), 0) FROM access_log;'))
  const captcha = await requestApiWithoutAuth('/admin/captcha')
  assert(captcha.status === 200 && captcha.body?.success, `captcha request failed: ${JSON.stringify(captcha.body)}`)
  const captchaId = captcha.body.data?.captcha_id || ''
  const captchaImage = captcha.body.data?.image || ''
  const captchaLog = await waitForAccessLogAfter(captchaBeforeId, '%/admin/captcha', 'captcha')
  const captchaPayload = `${captchaLog.body}\n${captchaLog.query}\n${captchaLog.response}`
  assert(!captchaId || !captchaPayload.includes(captchaId), 'audit log leaked captcha_id')
  assert(
    !captchaImage || !captchaPayload.includes(String(captchaImage).slice(0, 48)),
    'audit log leaked captcha image',
  )
  assert(
    captchaLog.response.includes('"captcha_id":"***"') || captchaLog.response.includes('"captchaId":"***"'),
    `captcha audit log did not mask captcha_id: ${captchaLog.response}`,
  )
  assert(captchaLog.response.includes('"image":"***"'), `captcha audit log did not mask image: ${captchaLog.response}`)

  const mobile = '13800138000'
  const sendSmsBeforeId = Number(runPostgres('SELECT COALESCE(MAX(id), 0) FROM access_log;'))
  await requestApiWithoutAuth(`/api/send_sms/${mobile}/LOGIN_CODE`, { method: 'POST', body: '{}' })
  const sendSmsLog = await waitForAccessLogAfter(sendSmsBeforeId, '%/api/send_sms/%', 'send_sms')
  assert(!sendSmsLog.url.includes(mobile), `send_sms audit url leaked mobile: ${sendSmsLog.url}`)
  assert(
    sendSmsLog.url.includes('/api/send_sms/***/LOGIN_CODE'),
    `send_sms audit url did not mask mobile: ${sendSmsLog.url}`,
  )

  const oneTimeCode = '654321'
  const smsLoginBeforeId = Number(runPostgres('SELECT COALESCE(MAX(id), 0) FROM access_log;'))
  await requestApiWithoutAuth('/api/sms_code_login', {
    method: 'POST',
    body: JSON.stringify({ mobile, code: oneTimeCode }),
  })
  const smsLoginLog = await waitForAccessLogAfter(smsLoginBeforeId, '%/api/sms_code_login', 'sms_code_login')
  const smsLoginPayload = `${smsLoginLog.body}\n${smsLoginLog.query}\n${smsLoginLog.response}`
  assert(!smsLoginPayload.includes(oneTimeCode), 'audit log leaked sms login code')
  assert(smsLoginPayload.includes('"code":"***"'), `sms login audit log did not mask code: ${smsLoginPayload}`)
  console.log('OK audit sensitive redaction')
}

async function assertRoleValidationGuard() {
  const invalidCreate = await request('/admin/role', {
    method: 'POST',
    body: JSON.stringify({ memo: 'missing role name' }),
  })
  assert(invalidCreate.status === 400, `invalid role create was not rejected: ${JSON.stringify(invalidCreate.body)}`)

  const invalidUpdate = await request('/admin/role/1', {
    method: 'PUT',
    body: JSON.stringify({ memo: 'missing role name' }),
  })
  assert(invalidUpdate.status === 400, `invalid role update was not rejected: ${JSON.stringify(invalidUpdate.body)}`)

  const invalidAssign = await request('/admin/role/assign-permissions', {
    method: 'POST',
    body: JSON.stringify({ menu_ids: [], button_ids: [] }),
  })
  assert(invalidAssign.status === 400, `invalid role permission assignment was not rejected: ${JSON.stringify(invalidAssign.body)}`)
  console.log('OK role validation guard')
}

async function assertRoleButtonPolicyScope(selectedMenu, foreignMenu) {
  if (!dropPhysicalSmokeTables) return
  const selectedButton = (selectedMenu.menu_buttons || []).find((item) => (item.event_action || item.code) === 'create')
  const foreignButton = (foreignMenu.menu_buttons || []).find((item) => (item.event_action || item.code) === 'query')
  assert(selectedButton?.id && selectedButton?.api_path && selectedButton?.http_method, 'selected role scope button missing')
  assert(foreignButton?.id && foreignButton?.api_path && foreignButton?.http_method, 'foreign role scope button missing')

  const roleName = `smoke_role_${Date.now().toString(36)}`
  cleanupSmokeRole(roleName)
  try {
    const created = await request('/admin/role', {
      method: 'POST',
      body: JSON.stringify({ name: roleName, memo: 'Smoke role policy scope' }),
    })
    assert(created.status === 200 && created.body?.success, `create smoke role failed: ${JSON.stringify(created.body)}`)
    const roleId = Number(
      runPostgres(`SELECT id FROM sys_role WHERE name = '${sqlString(roleName)}' AND gmt_delete IS NULL ORDER BY id DESC LIMIT 1;`),
    )
    assert(roleId > 0, 'created smoke role id missing')

    const assigned = await request('/admin/role/assign-permissions', {
      method: 'POST',
      body: JSON.stringify({
        role_id: roleId,
        menu_ids: [selectedMenu.id],
        button_ids: [selectedButton.id, foreignButton.id],
      }),
    })
    assert(assigned.status === 200 && assigned.body?.success, `assign smoke role permissions failed: ${JSON.stringify(assigned.body)}`)

    const foreignJoinCount = Number(
      runPostgres(`
SELECT COUNT(*)
FROM sys_role_menu_button
WHERE role_id = ${roleId}
  AND button_id = ${Number(foreignButton.id)};
`),
    )
    assert(foreignJoinCount === 0, `foreign button relation was persisted: ${foreignJoinCount}`)

    const foreignPolicyCount = Number(
      runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p'
  AND v0 = '${sqlString(roleName)}'
  AND v1 = '${sqlString(foreignButton.api_path)}'
  AND v2 = '${sqlString(String(foreignButton.http_method).toUpperCase())}';
`),
    )
    assert(foreignPolicyCount === 0, `foreign button policy was persisted: ${foreignPolicyCount}`)

    const selectedPolicyCount = Number(
      runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p'
  AND v0 = '${sqlString(roleName)}'
  AND v1 = '${sqlString(selectedButton.api_path)}'
  AND v2 = '${sqlString(String(selectedButton.http_method).toUpperCase())}';
`),
    )
    assert(selectedPolicyCount === 1, `selected button policy missing: ${selectedPolicyCount}`)
  } finally {
    cleanupSmokeRole(roleName)
  }
  console.log('OK role button policy scope')
}

async function assertLowCodeQueryRequiresQueryButton(lowcodeMenu) {
  if (!dropPhysicalSmokeTables) return
  const suffix = Date.now().toString(36)
  const buttonCode = `smoke_button_query_guard_${suffix}`
  const roleName = `smoke_role_${suffix}`
  const userName = `smoke_user_${suffix}`
  const queryPath = '/admin/generalization/query/code/:code'
  const queryMethod = 'POST'
  const code = lowcodeMenu.table_code || lowcodeMenu.option || tableCode
  let buttonId = 0
  const adminToken = accessToken
  cleanupSmokeUser(userName)
  cleanupSmokeRole(roleName)
  try {
    buttonId = await createSmokeMenuButton(lowcodeMenu.id, buttonCode, queryPath, queryMethod, 'custom')
    const roleId = await createSmokeRoleWithButton(roleName, lowcodeMenu.id, buttonId)
    assertRolePolicyCount(roleName, queryPath, queryMethod, 1, 'custom query-route policy missing before query guard')
    const user = createSmokeUserWithRole(userName, roleId)
    const login = await loginAs(user.userName, user.password)
    assert(login.status === 200 && login.body?.success, `smoke user login failed: ${JSON.stringify(login.body)}`)

    accessToken = login.body.data.access_token
    const denied = await request(`/admin/generalization/query/code/${code}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 5,
        table_code: code,
        menu_id: lowcodeMenu.id,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      denied.status === 403 && denied.body?.error_code === 30006,
      `query was allowed without query button action: ${JSON.stringify(denied.body)}`,
    )
  } finally {
    accessToken = adminToken
    if (buttonId > 0) {
      await request(`/admin/menu/button/${buttonId}`, { method: 'DELETE' })
    }
    cleanupSmokeUser(userName)
    cleanupSmokeRole(roleName)
    runPostgres(`DELETE FROM sys_menu_button WHERE code = '${sqlString(buttonCode)}';`)
  }
  console.log('OK low-code query button guard')
}

async function queryLowCodeRows(code, menuId, payload) {
  const response = await request(`/admin/generalization/query/code/${code}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 20,
      table_code: code,
      menu_id: menuId,
      quick_query: { keyword: '' },
      include_deleted: false,
      ...payload,
    }),
  })
  assert(response.status === 200 && response.body?.success, `low-code query failed: ${JSON.stringify(response.body)}`)
  return response.body
}

function assertQueryNames(body, expectedNames, context) {
  const rows = body.data || []
  const names = rows.map((row) => row.name).sort()
  const expected = [...expectedNames].sort()
  assert(
    JSON.stringify(names) === JSON.stringify(expected),
    `${context} names mismatch: expected=${JSON.stringify(expected)} actual=${JSON.stringify(names)} body=${JSON.stringify(body)}`,
  )
}

async function assertLowCodeAdvancedQueryMatrix(code, menuId) {
  for (const row of [
    {
      name: 'Query Matrix Alternate',
      scope_id: 3,
      status: 2,
      enabled: false,
      biz_date: '2026-06-15',
      started_at: '2026-06-15 15:45:00',
      start_time: '15:45:00',
      nullable_score: 7,
    },
    {
      name: 'Query Matrix Future',
      scope_id: 4,
      status: 1,
      enabled: true,
      biz_date: '2026-07-01',
      started_at: '2026-07-01 08:00:00',
      start_time: '08:00:00',
      nullable_score: 9,
    },
  ]) {
    const created = await request('/admin/generalization/create', {
      method: 'POST',
      body: JSON.stringify({
        table_code: code,
        menu_id: menuId,
        data: row,
      }),
    })
    assert(created.status === 200 && created.body?.success, `create advanced query row failed: ${JSON.stringify(created.body)}`)
  }

  const dictInQuery = await queryLowCodeRows(code, menuId, {
    expressions: [
      {
        logic: 1,
        rules: [{ field: 'status', expression_type: 9, value: '1, 2', type: 9 }],
        nested: [],
      },
    ],
  })
  assertQueryNames(dictInQuery, ['Smoke Item', 'Query Matrix Alternate', 'Query Matrix Future'], 'dict IN query')

  const boolDateQuery = await queryLowCodeRows(code, menuId, {
    expressions: [
      {
        logic: 1,
        rules: [
          { field: 'enabled', expression_type: 5, value: true, type: 5 },
          { field: 'biz_date', expression_type: 13, value: ['2026-06-01', '2026-06-30'], type: 6 },
        ],
        nested: [],
      },
    ],
  })
  assertQueryNames(boolDateQuery, ['Smoke Item'], 'boolean + date BETWEEN query')

  const nullableTimeQuery = await queryLowCodeRows(code, menuId, {
    expressions: [
      {
        logic: 1,
        rules: [
          { field: 'start_time', expression_type: 3, value: '09:00', type: 8 },
          { field: 'nullable_score', expression_type: 11, value: null, type: 11 },
        ],
        nested: [],
      },
    ],
  })
  assertQueryNames(nullableTimeQuery, ['Smoke Item'], 'time + IS_NULL query')

  const nestedOrQuery = await queryLowCodeRows(code, menuId, {
    expressions: [
      {
        logic: 2,
        rules: [{ field: 'status', expression_type: 5, value: 2, type: 9 }],
        nested: [
          {
            logic: 1,
            rules: [
              { field: 'enabled', expression_type: 5, value: true, type: 5 },
              { field: 'biz_date', expression_type: 3, value: '2026-07-01', type: 6 },
            ],
            nested: [],
          },
        ],
      },
    ],
  })
  assertQueryNames(nestedOrQuery, ['Query Matrix Alternate', 'Query Matrix Future'], 'nested OR query')

  const invalidDateQuery = await queryLowCodeRows(code, menuId, {
    expressions: [
      {
        logic: 1,
        rules: [{ field: 'biz_date', expression_type: 13, value: ['bad-date', '2026-06-30'], type: 6 }],
        nested: [],
      },
    ],
  })
  assert(
    Number(invalidDateQuery.total || 0) === 0,
    `invalid date query did not fail closed: ${JSON.stringify(invalidDateQuery)}`,
  )

  console.log('OK low-code advanced query matrix')
}

function menuButtonUpdatePayload(button, overrides = {}) {
  return {
    id: button.id,
    menu_id: button.menu_id,
    name: button.name || 'Smoke Button',
    code: button.code || `smoke_button_${Date.now().toString(36)}`,
    icon: button.icon || '',
    color: button.color || 'primary',
    sequence: Number(button.sequence || 0),
    memo: button.memo || '',
    position: Number(button.position || 2),
    event_type: button.event_type || '',
    event_action: button.event_action || '',
    api_path: button.api_path || '',
    http_method: button.http_method || '',
    params_schema: button.params_schema || '',
    confirm_text: button.confirm_text || '',
    disable_when: button.disable_when || '',
    is_hidden: Boolean(button.is_hidden),
    is_disabled: Boolean(button.is_disabled),
    before_hooks: button.before_hooks || '',
    after_hooks: button.after_hooks || '',
    ...overrides,
  }
}

async function assertMenuButtonConfigGuards(lowcodeMenu, normalMenu) {
  const createButton = (lowcodeMenu.menu_buttons || []).find((item) => (item.event_action || item.code) === 'create')
  assert(createButton?.id, 'low-code create button missing for config guard')

  const invalidLowCodeButton = await request(`/admin/menu/button/${createButton.id}`, {
    method: 'PUT',
    body: JSON.stringify(
      menuButtonUpdatePayload(createButton, {
        api_path: '/admin/user/reset_password/:id',
        http_method: 'POST',
      }),
    ),
  })
  assert(
    invalidLowCodeButton.status === 400 && invalidLowCodeButton.body?.error_code === 10000,
    `low-code button accepted unrelated api: ${JSON.stringify(invalidLowCodeButton.body)}`,
  )

  const incompleteApiButton = await request('/admin/menu/button', {
    method: 'POST',
    body: JSON.stringify({
      menu_id: normalMenu.id,
      name: 'Smoke Invalid API Pair',
      code: `smoke_invalid_api_${Date.now().toString(36)}`,
      icon: 'warning',
      color: 'warning',
      sequence: 99,
      memo: '',
      position: 2,
      event_type: '',
      event_action: 'query',
      api_path: '/admin/menu/query',
      http_method: '',
      params_schema: '',
      confirm_text: '',
      disable_when: '',
      is_hidden: true,
      is_disabled: false,
      before_hooks: '',
      after_hooks: '',
    }),
  })
  assert(
    incompleteApiButton.status === 400 && incompleteApiButton.body?.error_code === 10000,
    `button accepted incomplete api config: ${JSON.stringify(incompleteApiButton.body)}`,
  )

  const invalidParamsSchema = await request('/admin/menu/button', {
    method: 'POST',
    body: JSON.stringify({
      menu_id: normalMenu.id,
      name: 'Smoke Bad Params Schema',
      code: `smoke_bad_params_${Date.now().toString(36)}`,
      icon: 'bug_report',
      color: 'negative',
      sequence: 99,
      memo: '',
      position: 2,
      event_type: '',
      event_action: 'custom',
      api_path: '',
      http_method: '',
      params_schema: '[{"field_code":"bad-field"}]',
      confirm_text: '',
      disable_when: '',
      is_hidden: true,
      is_disabled: false,
      before_hooks: '',
      after_hooks: '',
    }),
  })
  assert(
    invalidParamsSchema.status === 400 && invalidParamsSchema.body?.error_code === 10000,
    `button accepted invalid params_schema: ${JSON.stringify(invalidParamsSchema.body)}`,
  )

  const invalidDisableWhen = await request('/admin/menu/button', {
    method: 'POST',
    body: JSON.stringify({
      menu_id: normalMenu.id,
      name: 'Smoke Bad Disable Rule',
      code: `smoke_bad_disable_${Date.now().toString(36)}`,
      icon: 'bug_report',
      color: 'negative',
      sequence: 99,
      memo: '',
      position: 2,
      event_type: '',
      event_action: 'custom',
      api_path: '',
      http_method: '',
      params_schema: '',
      confirm_text: '',
      disable_when: '{"field":"row.status","op":"exec","value":"x"}',
      is_hidden: true,
      is_disabled: false,
      before_hooks: '',
      after_hooks: '',
    }),
  })
  assert(
    invalidDisableWhen.status === 400 && invalidDisableWhen.body?.error_code === 10000,
    `button accepted invalid disable_when: ${JSON.stringify(invalidDisableWhen.body)}`,
  )
  console.log('OK menu button config guard')
}

async function createSmokeRoleWithButton(roleName, menuId, buttonId) {
  return createSmokeRoleWithButtons(roleName, menuId, [buttonId])
}

async function createSmokeRoleWithButtons(roleName, menuId, buttonIds) {
  cleanupSmokeRole(roleName)
  const created = await request('/admin/role', {
    method: 'POST',
    body: JSON.stringify({ name: roleName, memo: 'Smoke cleanup role' }),
  })
  assert(created.status === 200 && created.body?.success, `create cleanup smoke role failed: ${JSON.stringify(created.body)}`)
  const roleId = Number(
    runPostgres(`SELECT id FROM sys_role WHERE name = '${sqlString(roleName)}' AND gmt_delete IS NULL ORDER BY id DESC LIMIT 1;`),
  )
  assert(roleId > 0, 'cleanup smoke role id missing')
  const assigned = await request('/admin/role/assign-permissions', {
    method: 'POST',
    body: JSON.stringify({ role_id: roleId, menu_ids: [menuId], button_ids: buttonIds }),
  })
  assert(assigned.status === 200 && assigned.body?.success, `assign cleanup smoke role failed: ${JSON.stringify(assigned.body)}`)
  return roleId
}

async function createSmokeMenuButton(
  menuId,
  code,
  path = '/admin/log/access/:id',
  method = 'GET',
  eventAction = 'detail',
) {
  const created = await request('/admin/menu/button', {
    method: 'POST',
    body: JSON.stringify({
      menu_id: menuId,
      name: 'Smoke Cleanup Button',
      code,
      icon: 'visibility',
      color: 'primary',
      sequence: 99,
      memo: '',
      position: 2,
      event_type: '',
      event_action: eventAction,
      api_path: path,
      http_method: method,
      params_schema: '',
      confirm_text: '',
      disable_when: '',
      is_hidden: true,
      is_disabled: false,
      before_hooks: '',
      after_hooks: '',
    }),
  })
  assert(created.status === 200 && created.body?.success, `create cleanup smoke button failed: ${JSON.stringify(created.body)}`)
  const buttonId = Number(
    runPostgres(`SELECT id FROM sys_menu_button WHERE code = '${sqlString(code)}' AND gmt_delete IS NULL ORDER BY id DESC LIMIT 1;`),
  )
  assert(buttonId > 0, `cleanup smoke button id missing for ${code}`)
  return buttonId
}

function assertRolePolicyCount(roleName, path, method, expected, message) {
  const count = Number(
    runPostgres(`
SELECT COUNT(*)
FROM casbin_rule
WHERE ptype = 'p'
  AND v0 = '${sqlString(roleName)}'
  AND v1 = '${sqlString(path)}'
  AND v2 = '${sqlString(method)}';
`),
  )
  assert(count === expected, `${message}: ${count}`)
}

async function assertMenuButtonDeleteCleanup(normalMenu) {
  if (!dropPhysicalSmokeTables) return
  const code = `smoke_button_delete_${Date.now().toString(36)}`
  const roleName = `smoke_role_${Date.now().toString(36)}`
  const path = '/admin/log/access/:id'
  const method = 'GET'
  cleanupSmokeRole(roleName)
  try {
    const buttonId = await createSmokeMenuButton(normalMenu.id, code, path, method)
    const roleId = await createSmokeRoleWithButton(roleName, normalMenu.id, buttonId)
    assertRolePolicyCount(roleName, path, method, 1, 'delete-button smoke policy missing before delete')

    const deleted = await request(`/admin/menu/button/${buttonId}`, { method: 'DELETE' })
    assert(deleted.status === 200 && deleted.body?.success, `delete cleanup smoke button failed: ${JSON.stringify(deleted.body)}`)

    const relationCount = Number(
      runPostgres(`SELECT COUNT(*) FROM sys_role_menu_button WHERE role_id = ${roleId} AND button_id = ${buttonId};`),
    )
    assert(relationCount === 0, `deleted button role relation remained: ${relationCount}`)
    assertRolePolicyCount(roleName, path, method, 0, 'deleted button policy remained')
  } finally {
    cleanupSmokeRole(roleName)
    runPostgres(`DELETE FROM sys_menu_button WHERE code = '${sqlString(code)}';`)
  }
  console.log('OK menu button delete cleanup')
}

async function assertMenuDeleteCleanup() {
  if (!dropPhysicalSmokeTables) return
  const suffix = Date.now().toString(36)
  const menuName = `smoke_menu_${suffix}`
  const buttonCode = `smoke_menu_button_${suffix}`
  const roleName = `smoke_role_${suffix}`
  const path = '/admin/log/access/:id'
  const method = 'GET'
  cleanupSmokeRole(roleName)
  cleanupSmokeMenu(menuName)
  try {
    const createdMenu = await request('/admin/menu', {
      method: 'POST',
      body: JSON.stringify({
        pid: 0,
        name: menuName,
        path: menuName,
        component: 'pages/system/audit/Index.vue',
        title: 'Smoke Cleanup Menu',
        is_hidden: false,
        sequence: 99,
        option: '',
        icon: 'manage_search',
        redirect: '',
      }),
    })
    assert(createdMenu.status === 200 && createdMenu.body?.success, `create cleanup smoke menu failed: ${JSON.stringify(createdMenu.body)}`)
    const menuId = Number(
      runPostgres(`SELECT id FROM sys_menu WHERE name = '${sqlString(menuName)}' AND gmt_delete IS NULL ORDER BY id DESC LIMIT 1;`),
    )
    assert(menuId > 0, 'cleanup smoke menu id missing')
    const buttonId = await createSmokeMenuButton(menuId, buttonCode, path, method)
    const roleId = await createSmokeRoleWithButton(roleName, menuId, buttonId)
    assertRolePolicyCount(roleName, path, method, 1, 'delete-menu smoke policy missing before delete')

    const deletedMenu = await request(`/admin/menu/${menuId}`, { method: 'DELETE' })
    assert(deletedMenu.status === 200 && deletedMenu.body?.success, `delete cleanup smoke menu failed: ${JSON.stringify(deletedMenu.body)}`)

    const relationCounts = runPostgres(`
SELECT
  (SELECT COUNT(*) FROM sys_role_menu WHERE role_id = ${roleId} AND menu_id = ${menuId}),
  (SELECT COUNT(*) FROM sys_role_menu_button WHERE role_id = ${roleId} AND menu_id = ${menuId});
`)
      .split(/\s+/)
      .map(Number)
    assert(relationCounts[0] === 0 && relationCounts[1] === 0, `deleted menu role relations remained: ${relationCounts.join(',')}`)
    assertRolePolicyCount(roleName, path, method, 0, 'deleted menu policy remained')
  } finally {
    cleanupSmokeRole(roleName)
    cleanupSmokeMenu(menuName)
  }
  console.log('OK menu delete cleanup')
}

async function assertFileUploadGuard() {
  const allowedForm = new FormData()
  allowedForm.append('file', new Blob(['hello smoke upload\n'], { type: 'text/plain' }), 'smoke-upload.txt')
  const allowed = await requestMultipart('/admin/file/upload', allowedForm)
  assert(
    allowed.status === 200 && allowed.body?.success && allowed.body.data?.id,
    `allowed text upload failed: ${JSON.stringify(allowed.body)}`,
  )

  const fileId = allowed.body.data.id
  assert(allowed.body.data?.file_uuid, 'text upload did not return file_uuid')
  const privatePreviewNoAuth = await requestApiWithoutAuth(`/admin/file/preview/${allowed.body.data.file_uuid}`)
  assert(
    privatePreviewNoAuth.status === 401 && privatePreviewNoAuth.body?.error_code === 20002,
    `private file preview without auth was not denied: ${JSON.stringify(privatePreviewNoAuth.body)}`,
  )
  const privateDownloadNoAuth = await requestApiWithoutAuth(`/admin/file/download/${allowed.body.data.file_uuid}`)
  assert(
    privateDownloadNoAuth.status === 401 && privateDownloadNoAuth.body?.error_code === 20002,
    `private file download without auth was not denied: ${JSON.stringify(privateDownloadNoAuth.body)}`,
  )

  const publicPreview = await requestRaw(allowed.body.data.file_url)
  assert(
    publicPreview.status === 404,
    `uploaded file public preview should be disabled: status=${publicPreview.status} body=${JSON.stringify(publicPreview.body)}`,
  )

  const previewAccess = await request(`/admin/file/preview-url/${allowed.body.data.file_uuid}?ttl=120`)
  assert(
    previewAccess.status === 200 && previewAccess.body?.success && previewAccess.body.data?.url,
    `signed preview url failed: ${JSON.stringify(previewAccess.body)}`,
  )
  const previewAccessUrl = new URL(previewAccess.body.data.url, healthBaseUrl)
  assert(
    previewAccessUrl.pathname.endsWith(`/files/access/preview/${allowed.body.data.file_uuid}`),
    `signed preview path should not expose token: ${previewAccess.body.data.url}`,
  )
  assert(previewAccessUrl.searchParams.get('token'), 'signed preview token missing from query')

  const signedPreview = await requestRaw(previewAccess.body.data.url)
  assert(
    signedPreview.status === 200 &&
      signedPreview.body === 'hello smoke upload\n' &&
      signedPreview.headers.get('content-disposition')?.startsWith('inline') &&
      signedPreview.headers.get('x-content-type-options') === 'nosniff',
    `signed preview failed: status=${signedPreview.status} body=${JSON.stringify(signedPreview.body)}`,
  )

  previewAccessUrl.searchParams.set('token', `${previewAccessUrl.searchParams.get('token')}x`)
  const tamperedPreview = await requestRaw(previewAccessUrl.toString())
  assert(tamperedPreview.status === 404, `tampered signed preview was not rejected: ${tamperedPreview.status}`)

  const downloadAccess = await request(`/admin/file/download-url/${allowed.body.data.file_uuid}?ttl=120`)
  assert(
    downloadAccess.status === 200 && downloadAccess.body?.success && downloadAccess.body.data?.url,
    `signed download url failed: ${JSON.stringify(downloadAccess.body)}`,
  )
  const signedDownload = await requestRaw(downloadAccess.body.data.url)
  assert(
    signedDownload.status === 200 &&
      signedDownload.body === 'hello smoke upload\n' &&
      signedDownload.headers.get('content-disposition')?.startsWith('attachment') &&
      signedDownload.headers.get('cache-control') === 'private, no-store',
    `signed download failed: status=${signedDownload.status} body=${JSON.stringify(signedDownload.body)}`,
  )

  const deleted = await request(`/admin/file/${fileId}`, { method: 'DELETE' })
  assert(deleted.status === 200 && deleted.body?.success, `uploaded file cleanup failed: ${JSON.stringify(deleted.body)}`)

  const rejectedForm = new FormData()
  rejectedForm.append('file', new Blob(['#!/bin/sh\necho nope\n'], { type: 'text/x-shellscript' }), 'smoke-upload.sh')
  const rejected = await requestMultipart('/admin/file/upload', rejectedForm)
  assert(rejected.status === 400, `script upload was not rejected: ${JSON.stringify(rejected.body)}`)
  console.log('OK file upload guard')
}

function findMenuByName(menus, name) {
  for (const menu of menus || []) {
    if (menu.name === name) {
      return menu
    }
    const child = findMenuByName(menu.children, name)
    if (child) {
      return child
    }
  }
  return null
}

function findMenuByOption(menus, option) {
  for (const menu of menus || []) {
    if (menu.table_code === option || menu.option === option) {
      return menu
    }
    const child = findMenuByOption(menu.children, option)
    if (child) {
      return child
    }
  }
  return null
}

async function fetchTableByCode(code) {
  const table = await request(`/admin/table/code/${code}`)
  if (table.status === 200 && table.body?.success && table.body.data?.id) {
    return table.body.data
  }
  return null
}

async function waitForTableRelations(code, predicate, message) {
  let lastRelations = []
  for (let i = 0; i < 24; i++) {
    const table = await fetchTableByCode(code)
    lastRelations = table?.table_relations || []
    if (predicate(lastRelations, table)) {
      return table
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`${message}: ${JSON.stringify(lastRelations)}`)
}

async function waitForTableField(code, fieldCode) {
  let lastFields = []
  for (let i = 0; i < 24; i++) {
    const table = await fetchTableByCode(code)
    lastFields = table?.table_fields || []
    const field = lastFields.find((item) => item.field_code === fieldCode)
    if (field?.id) {
      return { table, field }
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`table field ${fieldCode} not visible in ${code}: ${JSON.stringify(lastFields)}`)
}

async function cleanupTable(code) {
  const table = await fetchTableByCode(code)
  if (!table?.id) {
    hardCleanupSmokeTable(code)
    return
  }
  await request(`/admin/table/unpublish/${code}`, { method: 'POST' })
  const deleted = await request(`/admin/table/${table.id}`, { method: 'DELETE' })
  assert(deleted.status === 200 && deleted.body?.success, `cleanup table failed: ${JSON.stringify(deleted.body)}`)
  hardCleanupSmokeTable(code)
}

async function waitForTableMissing(code) {
  for (let i = 0; i < 20; i++) {
    const table = await fetchTableByCode(code)
    if (!table?.id) return
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`table ${code} still visible after cleanup`)
}

async function assertRelationCandidateMenuScope() {
  if (!dropPhysicalSmokeTables) return
  const suffix = Date.now().toString(36)
  const targetCode = `smk_rel_target_${suffix}`
  const sourceCode = `smk_rel_source_${suffix}`
  await cleanupTable(sourceCode)
  await cleanupTable(targetCode)
  try {
    const targetTable = await createSmokeMetadataTable(targetCode, 'Smoke Relation Target')
    await createSmokeTableField(targetTable.id, {
      field_name: '名称',
      field_code: 'name',
      is_quick_search: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=64',
      sequence: 9,
    })
    await createSmokeTableField(targetTable.id, {
      field_name: '范围ID',
      field_code: 'scope_id',
      type: 11,
      input_type: 2,
      is_index: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1',
      sequence: 10,
    })
    const publishTarget = await request(`/admin/table/publish/${targetCode}`, { method: 'POST' })
    assert(
      publishTarget.status === 200 && publishTarget.body?.success,
      `publish relation target failed: ${JSON.stringify(publishTarget.body)}`,
    )
    const targetMenus = await request('/admin/menu/my')
    const targetMenu = findMenuByName(targetMenus.body.data, `lowcode_${targetCode}`)
    assert(targetMenu?.id, 'relation target menu not found')
    await assertTableIndexLifecycle(targetCode, suffix)

    for (const row of [
      { name: 'Target In Scope', scope_id: 1 },
      { name: 'Target Out Scope', scope_id: 2 },
    ]) {
      const created = await request('/admin/generalization/create', {
        method: 'POST',
        body: JSON.stringify({
          table_code: targetCode,
          menu_id: targetMenu.id,
          data: row,
        }),
      })
      assert(created.status === 200 && created.body?.success, `create relation target row failed: ${JSON.stringify(created.body)}`)
    }
    const targetRowsQuery = await request(`/admin/generalization/query/code/${targetCode}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 20,
        table_code: targetCode,
        menu_id: targetMenu.id,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      targetRowsQuery.status === 200 && targetRowsQuery.body?.success,
      `query relation target rows failed: ${JSON.stringify(targetRowsQuery.body)}`,
    )
    const targetInScope = (targetRowsQuery.body.data || []).find((row) => row.name === 'Target In Scope')
    const targetOutScope = (targetRowsQuery.body.data || []).find((row) => row.name === 'Target Out Scope')
    assert(targetInScope?.id && targetOutScope?.id, `relation target row ids missing: ${JSON.stringify(targetRowsQuery.body.data)}`)

    const sourceTable = await createSmokeMetadataTable(sourceCode, 'Smoke Relation Source')
    await createSmokeTableField(sourceTable.id, {
      field_name: '目标',
      field_code: 'target_id',
      type: 1,
      input_type: 4,
      is_advanced_search: true,
      sequence: 9,
      linkage_config: JSON.stringify({
        linkage: {
          enabled: true,
          mode: 'relation',
          tableId: targetTable.id,
          labelKey: 'name',
          valueKey: 'id',
          searchPageSize: 20,
        },
      }),
    })
    await assertRelationDeleteCache(sourceTable, targetTable, sourceCode)
    const sourceMeta = await fetchTableByCode(sourceCode)
    const targetField = sourceMeta?.table_fields?.find((field) => field.field_code === 'target_id')
    assert(targetField?.linkage_config, 'source relation field linkage config missing')
    const linkageConfig = JSON.parse(targetField.linkage_config)
    assert(
      linkageConfig.linkage?.tableCode === targetCode,
      `source relation linkage tableCode was not normalized: ${targetField.linkage_config}`,
    )

    const publishSource = await request(`/admin/table/publish/${sourceCode}`, { method: 'POST' })
    assert(
      publishSource.status === 200 && publishSource.body?.success,
      `publish relation source failed: ${JSON.stringify(publishSource.body)}`,
    )
    const sourceMenus = await request('/admin/menu/my')
    const sourceMenu = findMenuByName(sourceMenus.body.data, `lowcode_${sourceCode}`)
    assert(sourceMenu?.id, 'relation source menu not found')

    const createdSourceRow = await request('/admin/generalization/create', {
      method: 'POST',
      body: JSON.stringify({
        table_code: sourceCode,
        menu_id: sourceMenu.id,
        data: { target_id: targetInScope.id },
      }),
    })
    assert(
      createdSourceRow.status === 200 && createdSourceRow.body?.success,
      `create relation source row failed: ${JSON.stringify(createdSourceRow.body)}`,
    )

    const sourceRelationQuery = await request(`/admin/generalization/query/code/${sourceCode}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 20,
        table_code: sourceCode,
        menu_id: sourceMenu.id,
        expressions: [
          {
            logic: 1,
            rules: [{ field: 'target_id', expression_type: 5, value: targetInScope.id, type: 1 }],
            nested: [],
          },
        ],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      sourceRelationQuery.status === 200 &&
        sourceRelationQuery.body?.success &&
        sourceRelationQuery.body.data?.length === 1 &&
        Number(sourceRelationQuery.body.data[0]?.target_id) === Number(targetInScope.id),
      `relation source field query failed: ${JSON.stringify(sourceRelationQuery.body)}`,
    )

    setMenuDataScope(targetMenu.id, '1', 'scope_id')
    const wrongMenuQuery = await request(`/admin/generalization/query/code/${targetCode}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 20,
        table_code: targetCode,
        menu_id: sourceMenu.id,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      wrongMenuQuery.status === 403 && wrongMenuQuery.body?.error_code === 30006,
      `relation candidate query accepted source menu_id: ${JSON.stringify(wrongMenuQuery.body)}`,
    )

    const scopedCandidateFilterQuery = await request(`/admin/generalization/query/code/${targetCode}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 20,
        table_code: targetCode,
        menu_id: targetMenu.id,
        filters: { id: [targetInScope.id, targetOutScope.id] },
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      scopedCandidateFilterQuery.status === 200 &&
        scopedCandidateFilterQuery.body?.success &&
        scopedCandidateFilterQuery.body.data?.length === 1 &&
        scopedCandidateFilterQuery.body.data[0]?.name === 'Target In Scope',
      `relation candidate filtered lookup failed: ${JSON.stringify(scopedCandidateFilterQuery.body)}`,
    )

    const scopedCandidateQuery = await request(`/admin/generalization/query/code/${targetCode}`, {
      method: 'POST',
      body: JSON.stringify({
        page: 1,
        num: 20,
        table_code: targetCode,
        menu_id: targetMenu.id,
        expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
        quick_query: { keyword: '' },
        include_deleted: false,
      }),
    })
    assert(
      scopedCandidateQuery.status === 200 &&
        scopedCandidateQuery.body?.success &&
        scopedCandidateQuery.body.data?.length === 1 &&
        scopedCandidateQuery.body.data[0]?.name === 'Target In Scope',
      `relation candidate data scope failed: ${JSON.stringify(scopedCandidateQuery.body)}`,
    )
    console.log('OK relation field query and candidate menu scope')
  } finally {
    const targetMenuId = runPostgres(`SELECT COALESCE(MAX(id), 0) FROM sys_menu WHERE table_code = '${sqlString(targetCode)}' OR "option" = '${sqlString(targetCode)}';`)
    const sourceMenuId = runPostgres(`SELECT COALESCE(MAX(id), 0) FROM sys_menu WHERE table_code = '${sqlString(sourceCode)}' OR "option" = '${sqlString(sourceCode)}';`)
    if (Number(targetMenuId) > 0) clearMenuDataScope(Number(targetMenuId))
    if (Number(sourceMenuId) > 0) clearMenuDataScope(Number(sourceMenuId))
    await cleanupTable(sourceCode)
    await cleanupTable(targetCode)
  }
}

async function assertRelationDeleteCache(sourceTable, targetTable, sourceCode) {
  const created = await request('/admin/table/relation', {
    method: 'POST',
    body: JSON.stringify({
      table_id: sourceTable.id,
      related_table_id: targetTable.id,
      reference_key: 'target_id',
      foreign_key: 'id',
      relation_type: 1,
      manyTableCode: '',
    }),
  })
  assert(created.status === 200 && created.body?.success, `create smoke relation failed: ${JSON.stringify(created.body)}`)

  const relations = await request(`/admin/table/relations/${sourceTable.id}`)
  assert(
    relations.status === 200 && relations.body?.success,
    `query smoke relations failed: ${JSON.stringify(relations.body)}`,
  )
  const relation = (relations.body.data || []).find(
    (item) =>
      Number(item.table_id) === Number(sourceTable.id) &&
      Number(item.related_table_id) === Number(targetTable.id) &&
      item.reference_key === 'target_id' &&
      item.foreign_key === 'id',
  )
  assert(relation?.id, `created smoke relation not found: ${JSON.stringify(relations.body.data)}`)

  await waitForTableRelations(
    sourceCode,
    (items) => items.some((item) => Number(item.id) === Number(relation.id)),
    'created relation did not enter table metadata cache',
  )

  const deleted = await request(`/admin/table/relation/${relation.id}`, { method: 'DELETE' })
  assert(deleted.status === 200 && deleted.body?.success, `delete smoke relation failed: ${JSON.stringify(deleted.body)}`)

  await waitForTableRelations(
    sourceCode,
    (items) => !items.some((item) => Number(item.id) === Number(relation.id)),
    'deleted relation remained in table metadata cache',
  )
  console.log('OK relation delete cache')
}

async function assertTableIndexLifecycle(tableCode, suffix) {
  const { table, field } = await waitForTableField(tableCode, 'name')
  const indexName = `idx_${suffix}_name`
  const created = await request('/admin/table/index', {
    method: 'POST',
    body: JSON.stringify({
      table_id: table.id,
      index_name: indexName,
      is_unique: false,
      index_fields: [
        {
          table_id: table.id,
          field_id: field.id,
          field_code: field.field_code,
        },
      ],
    }),
  })
  assert(created.status === 200 && created.body?.success, `create smoke index failed: ${JSON.stringify(created.body)}`)

  const indexes = await request(`/admin/table/indexes/${table.id}`)
  assert(indexes.status === 200 && indexes.body?.success, `query smoke indexes failed: ${JSON.stringify(indexes.body)}`)
  const index = (indexes.body.data || []).find((item) => item.index_name === indexName)
  assert(index?.id, `created smoke index not found: ${JSON.stringify(indexes.body.data)}`)
  assert(
    (index.index_fields || []).some((item) => item.field_code === 'name'),
    `created smoke index fields missing name: ${JSON.stringify(index)}`,
  )

  const deleted = await request(`/admin/table/index/${index.id}`, { method: 'DELETE' })
  assert(deleted.status === 200 && deleted.body?.success, `delete smoke index failed: ${JSON.stringify(deleted.body)}`)

  const afterDelete = await request(`/admin/table/indexes/${table.id}`)
  assert(
    afterDelete.status === 200 &&
      afterDelete.body?.success &&
      !(afterDelete.body.data || []).some((item) => item.index_name === indexName),
    `deleted smoke index still visible: ${JSON.stringify(afterDelete.body)}`,
  )
  console.log('OK table index lifecycle')
}

async function main() {
  console.log(`Smoke target: ${baseUrl}`)
  console.log(`Health target: ${healthBaseUrl}`)
  prepareLocalSmokeRuntime()
  cleanupStaleSmokeArtifacts()

  const health = await requestRaw('/healthz')
  assert(health.status === 200 && health.body?.status === 'ok', `healthz failed: ${JSON.stringify(health.body)}`)
  const ready = await requestRaw('/readyz')
  assert(ready.status === 200 && ready.body?.status === 'ready', `readyz failed: ${JSON.stringify(ready.body)}`)
  assert(
    (ready.body?.components?.db_primary?.ok || ready.body?.components?.db?.ok) &&
      ready.body?.components?.redis?.ok,
    `readyz dependencies missing or unhealthy: ${JSON.stringify(ready.body)}`,
  )
  console.log('OK health readiness')

  const configure = await request('/admin/configure')
  assert(configure.status === 200 && configure.body?.success, 'configure endpoint failed')
  assert(!configure.body?.data?.enable_captcha, 'captcha is enabled; smoke login requires captcha disabled')
  assert(
    configure.body?.data?.password_length > 0 && configure.body?.data?.password_policy,
    `public configure defaults missing: ${JSON.stringify(configure.body?.data)}`,
  )
  assert(
    !Object.prototype.hasOwnProperty.call(configure.body?.data || {}, 'sender_password'),
    'public configure endpoint leaked sender_password',
  )
  assert(
    !Object.prototype.hasOwnProperty.call(configure.body?.data || {}, 'sender_email') &&
      !Object.prototype.hasOwnProperty.call(configure.body?.data || {}, 'smtp_server'),
    'public configure endpoint leaked email settings',
  )
  console.log('OK configure')

  const login = await request('/admin/login', {
    method: 'POST',
    body: JSON.stringify({
      user_name: username,
      password,
      captcha: '',
      captcha_id: '',
    }),
  })
  assert(login.status === 200 && login.body?.success, `login failed: ${JSON.stringify(login.body)}`)
  accessToken = login.body.data.access_token
  assert(accessToken, 'login response did not include access_token')
  assert(login.body.data.must_change_password === false, `admin should not require password change: ${JSON.stringify(login.body.data)}`)
  console.log('OK login')
  await assertForcedPasswordChangeFlow()

  const configureDetail = await request('/admin/configure/detail')
  assert(
    configureDetail.status === 200 && configureDetail.body?.success,
    `configure detail failed: ${JSON.stringify(configureDetail.body)}`,
  )
  assert(
    !Object.prototype.hasOwnProperty.call(configureDetail.body?.data || {}, 'sender_password'),
    'configure detail leaked sender_password',
  )
  clearConfigureSmokeSecret()
  console.log('OK configure redaction')
  await assertApplicationSecretsRedacted()
  await assertAuditQueryGuard()
  await assertAuditGenericQuery()
  await assertAuditSensitiveRedaction()
  await assertRoleValidationGuard()
  await assertFileUploadGuard()

  await preparePublishSmokeTable(tableCode)
  const publish = await request(`/admin/table/publish/${tableCode}`, { method: 'POST' })
  assert(publish.status === 200 && publish.body?.success, `publish failed: ${JSON.stringify(publish.body)}`)
  console.log(`OK publish ${tableCode}`)

  const table = await request(`/admin/table/code/${tableCode}`)
  assert(table.status === 200 && table.body?.success && table.body.data?.id, 'table metadata missing')
  if (tableCode === 'sys_user') {
    assertSensitiveTableFieldsHidden(table.body.data, ['password', 'access_tokens'])
  }
  console.log(`OK table metadata id=${table.body.data.id}`)

  const menus = await request('/admin/menu/my')
  assert(menus.status === 200 && menus.body?.success, 'my menu query failed')
  const auditMenu = findMenuByName(menus.body.data, 'system_audit')
  assert(auditMenu, 'system audit menu not found')
  assertAuditPermissionsSeeded()
  assertBuiltinPermissionButtonsSeeded()
  assertMetadataDictionariesSeeded()
  await assertMetadataIdentifierGuard()
  await assertProtectedGeneralizationWriteGuard(menus.body.data)
  const lowcodeMenu = findMenuByName(menus.body.data, `lowcode_${tableCode}`)
  assert(lowcodeMenu, `published menu lowcode_${tableCode} not found`)
  assert(lowcodeMenu.table_code === tableCode, `published menu table_code mismatch: ${lowcodeMenu.table_code}`)
  const buttonActions = new Set((lowcodeMenu.menu_buttons || []).map((button) => button.event_action || button.code))
  for (const action of ['create', 'detail', 'update', 'delete']) {
    assert(buttonActions.has(action), `published menu missing ${action} button`)
  }
  const lowcodeButtons = lowcodeMenu.menu_buttons || []
  const expectedButtonApis = {
    query: ['POST', '/admin/generalization/query/code/:code'],
    create: ['POST', '/admin/generalization/create'],
    update: ['PUT', '/admin/generalization/update'],
    delete: ['DELETE', '/admin/generalization/delete'],
  }
  for (const [action, [method, path]] of Object.entries(expectedButtonApis)) {
    const button = lowcodeButtons.find((item) => (item.event_action || item.code) === action)
    assert(button, `published menu missing ${action} API button`)
    assert(
      button.http_method === method && button.api_path === path,
      `published ${action} button API mismatch: ${button.http_method} ${button.api_path}`,
    )
  }
  console.log(`OK menu id=${lowcodeMenu.id}`)
  await assertMenuButtonConfigGuards(lowcodeMenu, auditMenu)
  await assertMenuButtonDeleteCleanup(auditMenu)
  await assertMenuDeleteCleanup()
  await assertRoleButtonPolicyScope(lowcodeMenu, auditMenu)
  await assertLowCodeQueryRequiresQueryButton(lowcodeMenu)

  const autoMenuQuery = await request(`/admin/generalization/query/code/${tableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: tableCode,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    autoMenuQuery.status === 200 && autoMenuQuery.body?.success && Array.isArray(autoMenuQuery.body.data),
    `published table query without menu_id did not resolve menu permission: ${JSON.stringify(autoMenuQuery.body)}`,
  )
  console.log('OK query menu auto resolve')

  setMenuDataScope(lowcodeMenu.id, '1')
  const unpublish = await request(`/admin/table/unpublish/${tableCode}`, { method: 'POST' })
  assert(
    unpublish.status === 200 && unpublish.body?.success,
    `unpublish failed: ${JSON.stringify(unpublish.body)}`,
  )
  const unpublishedMenus = await request('/admin/menu/my')
  assert(unpublishedMenus.status === 200 && unpublishedMenus.body?.success, 'my menu query failed after unpublish')
  assert(!findMenuByName(unpublishedMenus.body.data, `lowcode_${tableCode}`), 'unpublished menu is still visible')
  const staleMenuQuery = await request(`/admin/generalization/query/code/${tableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: tableCode,
      menu_id: lowcodeMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    staleMenuQuery.status === 403 && staleMenuQuery.body?.error_code === 30006,
    'stale unpublished menu_id query was not denied',
  )
  if (dropPhysicalSmokeTables) {
	    const activeDataPermissionCount = Number(
	      runPostgres(`
	SELECT
	  (SELECT COUNT(*) FROM sys_data_scope_binding WHERE menu_id = ${Number(lowcodeMenu.id)} AND gmt_delete IS NULL) +
	  (SELECT COUNT(*) FROM sys_role_data_scope WHERE menu_id = ${Number(lowcodeMenu.id)} AND gmt_delete IS NULL) +
	  (SELECT COUNT(*) FROM sys_user_data_scope_override WHERE menu_id = ${Number(lowcodeMenu.id)} AND gmt_delete IS NULL);
	`),
	    )
    assert(activeDataPermissionCount === 0, `unpublish did not clear active data permissions: ${activeDataPermissionCount}`)
  }
  console.log('OK unpublish')

  const republish = await request(`/admin/table/publish/${tableCode}`, { method: 'POST' })
  assert(
    republish.status === 200 && republish.body?.success,
    `republish failed: ${JSON.stringify(republish.body)}`,
  )
  const republishedMenus = await request('/admin/menu/my')
  assert(republishedMenus.status === 200 && republishedMenus.body?.success, 'my menu query failed after republish')
  const republishedMenu = findMenuByName(republishedMenus.body.data, `lowcode_${tableCode}`)
  assert(republishedMenu, `republished menu lowcode_${tableCode} not found`)
  console.log(`OK republish menu id=${republishedMenu.id}`)

  const query = await request(`/admin/generalization/query/code/${tableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: tableCode,
      menu_id: republishedMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(query.status === 200 && query.body?.success && Array.isArray(query.body.data), 'low-code query failed')
  if (tableCode === 'sys_user') {
    assertSensitiveRecordFieldsAbsent(query.body.data, ['password', 'access_tokens'])
  }
  console.log(`OK low-code query rows=${query.body.data.length}`)

  const unknownFilterQuery = await request(`/admin/generalization/query/code/${tableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: tableCode,
      menu_id: republishedMenu.id,
      filters: { unknown_field: 'should-not-expand-results' },
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    unknownFilterQuery.status === 200 &&
      unknownFilterQuery.body?.success &&
      Number(unknownFilterQuery.body?.total || 0) === 0,
    `unknown low-code filter did not fail closed: ${JSON.stringify(unknownFilterQuery.body)}`,
  )
  console.log('OK low-code query field guard')
  await assertRelationCandidateMenuScope()

  await assertUserDataPermissionApi(republishedMenu.id, auditMenu.id, table.body.data)
  console.log('OK user data permission API')

  const autoMenuCreate = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: tableCode,
      data: { name: 'Auto Menu Resolve', scope_id: 1 },
    }),
  })
  assert(
    autoMenuCreate.status === 200 && autoMenuCreate.body?.success,
    `missing menu_id write did not resolve menu permission: ${JSON.stringify(autoMenuCreate.body)}`,
  )
  console.log('OK write menu auto resolve')

  if (dropPhysicalSmokeTables && isSmokeTableCode(tableCode)) {
    await cleanupTable(tableCode)
  }

  await cleanupTable(crudTableCode)
  const createTable = await request('/admin/table', {
    method: 'POST',
    body: JSON.stringify({
      table_name: 'Smoke Lowcode Item',
      table_code: crudTableCode,
      table_type: 1,
      parent_id: 0,
      sql: '',
    }),
  })
  assert(createTable.status === 200 && createTable.body?.success, `create smoke table failed: ${JSON.stringify(createTable.body)}`)
  const crudTable = await fetchTableByCode(crudTableCode)
  assert(crudTable?.id, 'created smoke table metadata missing')

  const createField = await request('/admin/table/field', {
    method: 'POST',
    body: JSON.stringify({
      table_id: crudTable.id,
      field_name: '名称',
      field_code: 'name',
      type: 3,
      field_length: 64,
      field_decimal_length: 0,
      input_type: 1,
      default_value: '',
      dict_code: '',
      is_primary_key: false,
      is_index: false,
      is_quick_search: true,
      is_advanced_search: true,
      is_sort: false,
      is_null: false,
      is_list_show: true,
      is_insert_show: true,
      is_update_show: true,
      sequence: 9,
      binding: 'min=1|max=64',
      original_field_id: 0,
      field_category: 'normal_field',
      expression: '',
      linkage_config: '',
    }),
  })
  assert(createField.status === 200 && createField.body?.success, `create smoke field failed: ${JSON.stringify(createField.body)}`)

  const createScopeField = await request('/admin/table/field', {
    method: 'POST',
    body: JSON.stringify({
      table_id: crudTable.id,
      field_name: '范围ID',
      field_code: 'scope_id',
      type: 11,
      field_length: 0,
      field_decimal_length: 0,
      input_type: 2,
      default_value: '',
      dict_code: '',
      is_primary_key: false,
      is_index: true,
      is_quick_search: false,
      is_advanced_search: true,
      is_sort: false,
      is_null: false,
      is_list_show: true,
      is_insert_show: true,
      is_update_show: true,
      sequence: 10,
      binding: 'min=1',
      original_field_id: 0,
      field_category: 'normal_field',
      expression: '',
      linkage_config: '',
    }),
  })
  assert(createScopeField.status === 200 && createScopeField.body?.success, `create scope field failed: ${JSON.stringify(createScopeField.body)}`)

  await createSmokeTableField(crudTable.id, {
    field_name: '状态',
    field_code: 'status',
    type: 9,
    field_length: 0,
    input_type: 4,
    dict_code: 'sys_table_type',
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 11,
  })

  await createSmokeTableField(crudTable.id, {
    field_name: '启用',
    field_code: 'enabled',
    type: 5,
    field_length: 0,
    input_type: 11,
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 12,
  })

  await createSmokeTableField(crudTable.id, {
    field_name: '业务日期',
    field_code: 'biz_date',
    type: 6,
    field_length: 0,
    input_type: 5,
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 13,
  })

  await createSmokeTableField(crudTable.id, {
    field_name: '开始时间',
    field_code: 'started_at',
    type: 7,
    field_length: 0,
    input_type: 6,
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 14,
  })

  await createSmokeTableField(crudTable.id, {
    field_name: '开始时刻',
    field_code: 'start_time',
    type: 8,
    field_length: 0,
    input_type: 7,
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 15,
  })

  await createSmokeTableField(crudTable.id, {
    field_name: '可空分数',
    field_code: 'nullable_score',
    type: 11,
    field_length: 0,
    input_type: 2,
    is_quick_search: false,
    is_advanced_search: true,
    is_null: true,
    sequence: 16,
  })

  const createAttachmentField = await request('/admin/table/field', {
    method: 'POST',
    body: JSON.stringify({
      table_id: crudTable.id,
      field_name: '附件',
      field_code: 'attachment_ids',
      type: 3,
      field_length: 512,
      field_decimal_length: 0,
      input_type: 10,
      default_value: '',
      dict_code: '',
      is_primary_key: false,
      is_index: false,
      is_quick_search: false,
      is_advanced_search: false,
      is_sort: false,
      is_null: true,
      is_list_show: false,
      is_insert_show: true,
      is_update_show: true,
      sequence: 17,
      binding: '',
      original_field_id: 0,
      field_category: 'normal_field',
      expression: '',
      linkage_config: '',
    }),
  })
  assert(
    createAttachmentField.status === 200 && createAttachmentField.body?.success,
    `create attachment field failed: ${JSON.stringify(createAttachmentField.body)}`,
  )

  const createRichField = await request('/admin/table/field', {
    method: 'POST',
    body: JSON.stringify({
      table_id: crudTable.id,
      field_name: '富文本',
      field_code: 'rich_content',
      type: 4,
      field_length: 0,
      field_decimal_length: 0,
      input_type: 16,
      default_value: '',
      dict_code: '',
      is_primary_key: false,
      is_index: false,
      is_quick_search: false,
      is_advanced_search: false,
      is_sort: false,
      is_null: true,
      is_list_show: false,
      is_insert_show: true,
      is_update_show: true,
      sequence: 18,
      binding: '',
      original_field_id: 0,
      field_category: 'normal_field',
      expression: '',
      linkage_config: '',
    }),
  })
  assert(createRichField.status === 200 && createRichField.body?.success, `create rich field failed: ${JSON.stringify(createRichField.body)}`)

  const crudTableAfterFields = await fetchTableByCode(crudTableCode)
  assertLowCodeManagedFieldMetadata(crudTableAfterFields, [
    'id',
    'gmt_create',
    'gmt_create_user',
    'gmt_modify',
    'gmt_modify_user',
    'gmt_delete',
    'gmt_delete_user',
  ])
  const nameFieldMeta = crudTableAfterFields?.table_fields?.find((field) => field.field_code === 'name')
  assert(nameFieldMeta && !nameFieldMeta.is_null, `name field is_null metadata was not persisted false: ${JSON.stringify(nameFieldMeta)}`)
  const scopeFieldMeta = crudTableAfterFields?.table_fields?.find((field) => field.field_code === 'scope_id')
  assert(scopeFieldMeta && !scopeFieldMeta.is_null, `scope_id field is_null metadata was not persisted false: ${JSON.stringify(scopeFieldMeta)}`)
  console.log('OK low-code metadata bool guards')

  const publishCrud = await request(`/admin/table/publish/${crudTableCode}`, { method: 'POST' })
  assert(publishCrud.status === 200 && publishCrud.body?.success, `publish smoke table failed: ${JSON.stringify(publishCrud.body)}`)
  const crudMenus = await request('/admin/menu/my')
  const crudMenu = findMenuByName(crudMenus.body.data, `lowcode_${crudTableCode}`)
  assert(crudMenu?.id, 'smoke low-code menu not found')

  if (dropPhysicalSmokeTables) {
    const crudCreateButton = (crudMenu.menu_buttons || []).find((item) => (item.event_action || item.code) === 'create')
    assert(crudCreateButton?.id, 'smoke low-code create button not found')
    setMenuButtonDisabled(crudCreateButton.id, true)
    const disabledCreate = await request('/admin/generalization/create', {
      method: 'POST',
      body: JSON.stringify({
        table_code: crudTableCode,
        menu_id: crudMenu.id,
        data: { name: 'Smoke Disabled Button Item', scope_id: 2 },
      }),
    })
    assert(
      disabledCreate.status === 403 && disabledCreate.body?.error_code === 30006,
      `disabled create button was not denied: ${JSON.stringify(disabledCreate.body)}`,
    )
    setMenuButtonDisabled(crudCreateButton.id, false)
    console.log('OK disabled low-code button guard')
  }

  const recordFileForm = new FormData()
  recordFileForm.append('file', new Blob(['hello record file access\n'], { type: 'text/plain' }), 'record-file-access.txt')
  const recordFile = await requestMultipart('/admin/file/upload', recordFileForm)
  assert(
    recordFile.status === 200 && recordFile.body?.success && recordFile.body.data?.id && recordFile.body.data?.file_uuid,
    `record file upload failed: ${JSON.stringify(recordFile.body)}`,
  )

  const createdRow = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: {
        name: 'Smoke Item',
        scope_id: 2,
        status: 1,
        enabled: true,
        biz_date: '2026-06-01',
        started_at: '2026-06-01 09:30:00',
        start_time: '09:30:00',
        nullable_score: null,
        attachment_ids: JSON.stringify([recordFile.body.data.id]),
        rich_content: `<p><img src="/sweet_admin/files/${recordFile.body.data.file_uuid}" data-file-uuid="${recordFile.body.data.file_uuid}"></p>`,
      },
    }),
  })
  assert(createdRow.status === 200 && createdRow.body?.success, `low-code create failed: ${JSON.stringify(createdRow.body)}`)

  const crudQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: 'Smoke Item' },
      include_deleted: false,
    }),
  })
  assert(crudQuery.status === 200 && crudQuery.body?.success && crudQuery.body.data?.length === 1, 'low-code create query failed')
  const rowId = Number(crudQuery.body.data[0].id)
  assert(rowId > 0, 'created smoke row id missing')
  await assertLowCodeAdvancedQueryMatrix(crudTableCode, crudMenu.id)

  const missingRowId = rowId + 999999
  const missingUpdate = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: missingRowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Missing Smoke Item', scope_id: 2 },
    }),
  })
  assert(
    missingUpdate.status === 400 && missingUpdate.body?.error_code === 20013,
    `missing row update was not rejected as data-not-found: ${JSON.stringify(missingUpdate.body)}`,
  )

  const missingDelete = await request('/admin/generalization/delete', {
    method: 'DELETE',
    body: JSON.stringify({
      id: missingRowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
    }),
  })
  assert(
    missingDelete.status === 400 && missingDelete.body?.error_code === 20013,
    `missing row delete was not rejected as data-not-found: ${JSON.stringify(missingDelete.body)}`,
  )
  console.log('OK low-code missing row write guard')

  const invalidTypedExpressionQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [
        {
          logic: 1,
          rules: [{ field: 'scope_id', expression_type: 5, value: 'not-a-number', type: 3 }],
          nested: [],
        },
      ],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    invalidTypedExpressionQuery.status === 200 &&
      invalidTypedExpressionQuery.body?.success &&
      Number(invalidTypedExpressionQuery.body?.total || 0) === 0,
    `invalid typed expression did not fail closed: ${JSON.stringify(invalidTypedExpressionQuery.body)}`,
  )

  const invalidTypedFilterQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      filters: { scope_id: 'not-a-number' },
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    invalidTypedFilterQuery.status === 200 &&
      invalidTypedFilterQuery.body?.success &&
      Number(invalidTypedFilterQuery.body?.total || 0) === 0,
    `invalid typed filter did not fail closed: ${JSON.stringify(invalidTypedFilterQuery.body)}`,
  )

  const typedBetweenQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [
        {
          logic: 1,
          rules: [{ field: 'scope_id', expression_type: 13, value: [1, 3], type: 11 }],
          nested: [],
        },
      ],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    typedBetweenQuery.status === 200 &&
      typedBetweenQuery.body?.success &&
      Number(typedBetweenQuery.body?.total || 0) >= 1,
    `typed BETWEEN expression did not match expected rows: ${JSON.stringify(typedBetweenQuery.body)}`,
  )

  const invalidTypedBetweenQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [
        {
          logic: 1,
          rules: [{ field: 'scope_id', expression_type: 13, value: [1], type: 11 }],
          nested: [],
        },
      ],
      quick_query: { keyword: '' },
      include_deleted: false,
    }),
  })
  assert(
    invalidTypedBetweenQuery.status === 200 &&
      invalidTypedBetweenQuery.body?.success &&
      Number(invalidTypedBetweenQuery.body?.total || 0) === 0,
    `invalid typed BETWEEN expression did not fail closed: ${JSON.stringify(invalidTypedBetweenQuery.body)}`,
  )

  const invalidTypedCreate = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Invalid Smoke Item', scope_id: 'not-a-number' },
    }),
  })
  assert(invalidTypedCreate.status === 400, `invalid typed create was not rejected: ${JSON.stringify(invalidTypedCreate.body)}`)

  const emptyRequiredUpdate = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: '', scope_id: 2 },
    }),
  })
  assert(
    emptyRequiredUpdate.status === 400,
    `empty required update was not rejected: ${JSON.stringify(emptyRequiredUpdate.body)}`,
  )
  console.log('OK low-code typed query and write guards')

  const updatedRow = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Updated', scope_id: 2, id: 1, gmt_create: '2000-01-01 00:00:00' },
    }),
  })
  assert(updatedRow.status === 200 && updatedRow.body?.success, `low-code update failed: ${JSON.stringify(updatedRow.body)}`)

  setMenuDataScope(crudMenu.id, '1')

  const scopedQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: 'Smoke Item Updated' },
      include_deleted: false,
    }),
  })
  assert(
    scopedQuery.status === 200 && scopedQuery.body?.success && scopedQuery.body.data?.length === 0,
    `scoped query exposed data outside the configured id scope: ${JSON.stringify(scopedQuery.body)}`,
  )

  const scopedCreateDenied = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Out Of Scope' },
    }),
  })
  assert(
    scopedCreateDenied.status === 403 && scopedCreateDenied.body?.error_code === 30006,
    `scoped create without configured id value was not denied: ${JSON.stringify(scopedCreateDenied.body)}`,
  )

  const scopedUpdateDenied = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Out Of Scope Update' },
    }),
  })
  assert(
    scopedUpdateDenied.status === 403 && scopedUpdateDenied.body?.error_code === 30006,
    `scoped update without configured id value was not denied: ${JSON.stringify(scopedUpdateDenied.body)}`,
  )

  const scopedDeleteDenied = await request('/admin/generalization/delete', {
    method: 'DELETE',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
    }),
  })
  assert(
    scopedDeleteDenied.status === 403 && scopedDeleteDenied.body?.error_code === 30006,
    `scoped delete without configured id value was not denied: ${JSON.stringify(scopedDeleteDenied.body)}`,
  )

  clearMenuDataScope(crudMenu.id)
  setMenuDataScope(crudMenu.id, '1', 'scope_id')
  let scopedRowId = 0

  const customScopedQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: 'Smoke Item Updated' },
      include_deleted: false,
    }),
  })
  assert(
    customScopedQuery.status === 200 && customScopedQuery.body?.success && customScopedQuery.body.data?.length === 0,
    `custom scoped query exposed out-of-scope data: ${JSON.stringify(customScopedQuery.body)}`,
  )

  const customScopedCreateDenied = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Out Of Scope', scope_id: 2 },
    }),
  })
  assert(
    customScopedCreateDenied.status === 403 && customScopedCreateDenied.body?.error_code === 30006,
    `custom scoped create was not denied: ${JSON.stringify(customScopedCreateDenied.body)}`,
  )

  const customScopedCreateAllowed = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item In Scope', scope_id: 1 },
    }),
  })
  assert(
    customScopedCreateAllowed.status === 200 && customScopedCreateAllowed.body?.success,
    `custom scoped create in scope failed: ${JSON.stringify(customScopedCreateAllowed.body)}`,
  )
  const customScopedAllowedQuery = await request(`/admin/generalization/query/code/${crudTableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 5,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: 'Smoke Item In Scope' },
      include_deleted: false,
    }),
  })
  assert(
    customScopedAllowedQuery.status === 200 &&
      customScopedAllowedQuery.body?.success &&
      customScopedAllowedQuery.body.data?.length === 1,
    `custom scoped in-scope query failed: ${JSON.stringify(customScopedAllowedQuery.body)}`,
  )
  scopedRowId = Number(customScopedAllowedQuery.body.data[0].id)
  assert(scopedRowId > 0, 'custom scoped in-scope row id missing')

  const customScopedMoveDenied = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: scopedRowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Scope Move', scope_id: 2 },
    }),
  })
  assert(
    customScopedMoveDenied.status === 403 && customScopedMoveDenied.body?.error_code === 30006,
    `custom scoped update moved row out of scope: ${JSON.stringify(customScopedMoveDenied.body)}`,
  )

  const customScopedUpdateDenied = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
      data: { name: 'Smoke Item Custom Scope Update' },
    }),
  })
  assert(
    customScopedUpdateDenied.status === 403 && customScopedUpdateDenied.body?.error_code === 30006,
    `custom scoped update was not denied: ${JSON.stringify(customScopedUpdateDenied.body)}`,
  )

  const customScopedDeleteDenied = await request('/admin/generalization/delete', {
    method: 'DELETE',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
    }),
  })
  assert(
    customScopedDeleteDenied.status === 403 && customScopedDeleteDenied.body?.error_code === 30006,
    `custom scoped delete was not denied: ${JSON.stringify(customScopedDeleteDenied.body)}`,
  )

  clearMenuDataScope(crudMenu.id)
  console.log('OK data-scope guard')

  if (scopedRowId > 0) {
    const deletedScopedRow = await request('/admin/generalization/delete', {
      method: 'DELETE',
      body: JSON.stringify({
        id: scopedRowId,
        table_code: crudTableCode,
        menu_id: crudMenu.id,
      }),
    })
    assert(
      deletedScopedRow.status === 200 && deletedScopedRow.body?.success,
      `low-code scoped row delete failed: ${JSON.stringify(deletedScopedRow.body)}`,
    )
  }

  const deletedRow = await request('/admin/generalization/delete', {
    method: 'DELETE',
    body: JSON.stringify({
      id: rowId,
      table_code: crudTableCode,
      menu_id: crudMenu.id,
    }),
  })
  assert(deletedRow.status === 200 && deletedRow.body?.success, `low-code delete failed: ${JSON.stringify(deletedRow.body)}`)
  const deletedRecordFile = await request(`/admin/file/${recordFile.body.data.id}`, { method: 'DELETE' })
  assert(
    deletedRecordFile.status === 200 && deletedRecordFile.body?.success,
    `record file cleanup failed: ${JSON.stringify(deletedRecordFile.body)}`,
  )
  assertAuditLogs(crudTableCode, ['table_create', 'table_publish', 'lowcode_create', 'lowcode_update', 'lowcode_delete'])
  await assertAuditApi(crudTableCode, 'lowcode_create')
  await cleanupTable(crudTableCode)
  console.log('OK low-code custom CRUD')

  const longTableCode = `smoke_long_${Date.now().toString(36)}_table_code_check`
  await cleanupTable(longTableCode)
  const createLongTable = await request('/admin/table', {
    method: 'POST',
    body: JSON.stringify({
      table_name: 'Smoke Long Table Code',
      table_code: longTableCode,
      table_type: 1,
      parent_id: 0,
      sql: '',
    }),
  })
  assert(
    createLongTable.status === 200 && createLongTable.body?.success,
    `create long table failed: ${JSON.stringify(createLongTable.body)}`,
  )
  const publishLongTable = await request(`/admin/table/publish/${longTableCode}`, { method: 'POST' })
  assert(
    publishLongTable.status === 200 && publishLongTable.body?.success,
    `publish long table failed: ${JSON.stringify(publishLongTable.body)}`,
  )
  const longMenus = await request('/admin/menu/my')
  const longMenu = findMenuByOption(longMenus.body.data, longTableCode)
  assert(longMenu?.id, 'long table published menu not found by option')
  assert(String(longMenu.name || '').length <= 32, 'long table menu name exceeds 32 chars')
  await cleanupTable(longTableCode)
  console.log('OK long table publish')

  console.log('Low-code smoke passed')
}

main().catch((error) => {
  console.error(error.message)
  process.exit(1)
})
