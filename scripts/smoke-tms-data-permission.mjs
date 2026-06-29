#!/usr/bin/env node

import { execFileSync } from 'node:child_process'
import { createHash } from 'node:crypto'

const baseUrl = normalizeBaseUrl(process.env.SWEET_ADMIN_BASE_URL || 'http://localhost:8008/sweet_admin')
const username = process.env.SWEET_ADMIN_ADMIN_USER || 'admin'
const password = process.env.SWEET_ADMIN_ADMIN_PASSWORD || 'admin123'
const passwordSalt =
  process.env.SWEET_ADMIN_SMOKE_PASSWORD_SALT || 'local-docker-sweet-admin-salt-change-me'

const tables = ['tms_waybill', 'tms_vehicle', 'tms_company']
const dimensionCode = 'tms_company'
const roleName = 'tms_operator'
const eastOperatorName = 'tms_east_operator_user'
const southOperatorName = 'tms_south_operator_user'
const operatorPassword = 'admin123'
const eastCompanyName = '华东运输公司'
const southCompanyName = '华南运输公司'
const tmsMenuGroupId = 810000100
const companyRelationLinkage = JSON.stringify({
  linkage: {
    enabled: true,
    mode: 'relation',
    tableCode: 'tms_company',
    labelKey: 'company_name',
    valueKey: 'id',
    pageSize: 200,
  },
})

let accessToken = ''

function normalizeBaseUrl(value) {
  return value.replace(/\/+$/, '')
}

function apiPath(path) {
  return `${baseUrl}${path.startsWith('/') ? path : `/${path}`}`
}

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

function sqlString(value) {
  return String(value ?? '').replaceAll("'", "''")
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

async function request(path, options = {}) {
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  }
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`
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

function clearUserCacheKeys(...cacheKeys) {
  const keys = cacheKeys.filter(Boolean)
  if (!keys.length) return
  execFileSync(
    'docker',
    ['compose', 'exec', '-T', 'redis', 'redis-cli', '-n', '5', 'DEL', ...keys],
    { stdio: 'pipe' },
  )
}

function clearRedisKeysByPattern(pattern) {
  const keys = execFileSync(
    'docker',
    ['compose', 'exec', '-T', 'redis', 'redis-cli', '-n', '5', '--scan', '--pattern', pattern],
    { encoding: 'utf8', stdio: 'pipe' },
  )
    .split(/\r?\n/)
    .map((key) => key.trim())
    .filter(Boolean)
  if (!keys.length) return
  execFileSync(
    'docker',
    ['compose', 'exec', '-T', 'redis', 'redis-cli', '-n', '5', 'DEL', ...keys],
    { stdio: 'pipe' },
  )
}

function clearTmsCache() {
  clearRedisKeysByPattern('TABLE_CACHE_KEY_tms_*')
  clearRedisKeysByPattern('GENERALIZATION_CACHE_KEY_tms_*')
  clearRedisKeysByPattern('GENERALIZATION_CACHE_KEY_*tms_*')
}

function clearTmsMetadataCacheByIds() {
  let tableIds = []
  try {
    tableIds = runPostgres(
      "SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company');",
    )
      .split(/\r?\n/)
      .map((id) => id.trim())
      .filter(Boolean)
  } catch {
    tableIds = []
  }
  for (const tableId of tableIds) {
    clearRedisKeysByPattern(`TABLE_CACHE_KEY_${tableId}`)
    clearRedisKeysByPattern(`TABLE_FIELD_CACHE_KEY_${tableId}*`)
  }
}

function hashUserPassword(rawPassword, userId) {
  return createHash('md5').update(`${rawPassword}${userId}${passwordSalt}`).digest('hex')
}

function cleanupTmsDemo() {
  clearTmsCache()
  clearTmsMetadataCacheByIds()
  runPostgres(`
DELETE FROM sys_user_data_scope_override WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_role_data_scope WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_data_scope_binding WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_role_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_role_menu WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%' OR name = 'tms_demo');
DELETE FROM sys_menu_button WHERE menu_id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_menu WHERE id IN (SELECT id FROM sys_menu WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company') OR name LIKE 'lowcode_tms_%');
DELETE FROM sys_menu WHERE name = 'tms_demo' OR id = ${tmsMenuGroupId};
DELETE FROM sys_user_role WHERE role_id IN (SELECT id FROM sys_role WHERE name = '${sqlString(roleName)}');
DELETE FROM sys_user_data_scope_override WHERE user_id IN (SELECT id FROM sys_user WHERE user_name IN ('${sqlString(eastOperatorName)}','${sqlString(southOperatorName)}'));
DELETE FROM sys_user_dimension_value WHERE user_id IN (SELECT id FROM sys_user WHERE user_name IN ('${sqlString(eastOperatorName)}','${sqlString(southOperatorName)}'));
DELETE FROM sys_user WHERE user_name IN ('${sqlString(eastOperatorName)}','${sqlString(southOperatorName)}');
DELETE FROM casbin_rule WHERE v0 = '${sqlString(roleName)}';
DELETE FROM sys_role WHERE name = '${sqlString(roleName)}';
DELETE FROM sys_data_dimension WHERE code = '${sqlString(dimensionCode)}';
DELETE FROM sys_table_index_field WHERE index_id IN (SELECT id FROM sys_table_index WHERE table_id IN (SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company')));
DELETE FROM sys_table_index WHERE table_id IN (SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company'));
DELETE FROM sys_table_relation WHERE table_id IN (SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company')) OR related_table_id IN (SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company'));
DELETE FROM sys_table_field WHERE table_id IN (SELECT id FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company'));
DELETE FROM sys_table WHERE table_code IN ('tms_waybill','tms_vehicle','tms_company');
DROP TABLE IF EXISTS tms_waybill;
DROP TABLE IF EXISTS tms_vehicle;
DROP TABLE IF EXISTS tms_company;
`)
  clearTmsMetadataCacheByIds()
  clearTmsCache()
}

async function loginAdmin() {
  const configure = await request('/admin/configure')
  assert(configure.status === 200 && configure.body?.success, `configure failed: ${JSON.stringify(configure.body)}`)
  assert(!configure.body.data?.enable_captcha, 'captcha is enabled; local smoke login requires captcha disabled')
  const login = await loginAs(username, password)
  assert(login.status === 200 && login.body?.success, `admin login failed: ${JSON.stringify(login.body)}`)
  accessToken = login.body.data.access_token
  assert(accessToken, 'admin login did not return access token')
  console.log('OK admin login')
}

async function createTable(tableCode, tableName, fields) {
  const created = await request('/admin/table', {
    method: 'POST',
    body: JSON.stringify({
      table_name: tableName,
      table_code: tableCode,
      table_type: 1,
      parent_id: 0,
      sql: '',
    }),
  })
  assert(created.status === 200 && created.body?.success, `create table ${tableCode} failed: ${JSON.stringify(created.body)}`)
  const table = await fetchTable(tableCode)
  assert(table?.id, `created table ${tableCode} metadata missing`)
  for (const [index, field] of fields.entries()) {
    await createField(table.id, { sequence: 9 + index, ...field })
  }
  const publish = await request(`/admin/table/publish/${tableCode}`, { method: 'POST' })
  assert(publish.status === 200 && publish.body?.success, `publish ${tableCode} failed: ${JSON.stringify(publish.body)}`)
  console.log(`OK table published ${tableCode}`)
}

async function createField(tableId, overrides) {
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
  assert(created.status === 200 && created.body?.success, `create field ${field.field_code} failed: ${JSON.stringify(created.body)}`)
}

async function fetchTable(tableCode) {
  const res = await request(`/admin/table/code/${tableCode}`)
  if (res.status === 200 && res.body?.success) return res.body.data
  return null
}

async function createTmsTables() {
  await createTable('tms_company', 'TMS公司', [
    {
      field_name: '公司名称',
      field_code: 'company_name',
      type: 3,
      input_type: 1,
      field_length: 64,
      is_quick_search: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=64',
    },
    {
      field_name: '上级公司',
      field_code: 'parent_id',
      type: 1,
      input_type: 2,
      is_index: true,
      is_null: true,
    },
  ])
  await createTable('tms_waybill', 'TMS运单', [
    {
      field_name: '运单号',
      field_code: 'waybill_no',
      type: 3,
      input_type: 1,
      field_length: 64,
      is_quick_search: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=64',
    },
    {
      field_name: '所属公司',
      field_code: 'company_id',
      type: 1,
      input_type: 4,
      is_index: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1',
      linkage_config: companyRelationLinkage,
    },
    {
      field_name: '客户名称',
      field_code: 'customer_name',
      type: 3,
      input_type: 1,
      field_length: 64,
      is_quick_search: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=64',
    },
    {
      field_name: '运单状态',
      field_code: 'status',
      type: 3,
      input_type: 1,
      field_length: 32,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=32',
    },
  ])
  await createTable('tms_vehicle', 'TMS车辆', [
    {
      field_name: '车牌号',
      field_code: 'plate_no',
      type: 3,
      input_type: 1,
      field_length: 32,
      is_quick_search: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1|max=32',
    },
    {
      field_name: '所属公司',
      field_code: 'company_id',
      type: 1,
      input_type: 4,
      is_index: true,
      is_advanced_search: true,
      is_null: false,
      binding: 'min=1',
      linkage_config: companyRelationLinkage,
    },
    {
      field_name: '司机姓名',
      field_code: 'driver_name',
      type: 3,
      input_type: 1,
      field_length: 64,
      is_quick_search: true,
      is_null: false,
      binding: 'min=1|max=64',
    },
  ])
}

async function verifyTmsCompanyFieldLinkage() {
  for (const tableCode of ['tms_waybill', 'tms_vehicle']) {
    const table = await fetchTable(tableCode)
    const companyField = table?.table_fields?.find((field) => field.field_code === 'company_id')
    assert(companyField, `${tableCode}.company_id metadata missing`)
    assert(Number(companyField.input_type) === 4, `${tableCode}.company_id should use select input`)
    const parsed = JSON.parse(companyField.linkage_config || '{}')
    assert(
      parsed.linkage?.enabled === true &&
        parsed.linkage?.mode === 'relation' &&
        parsed.linkage?.tableCode === 'tms_company' &&
        parsed.linkage?.labelKey === 'company_name' &&
        parsed.linkage?.valueKey === 'id',
      `${tableCode}.company_id linkage mismatch: ${companyField.linkage_config}`,
    )
  }
  console.log('OK TMS company relation field metadata')
}

function ensureTmsMenuGroup() {
  runPostgres(`
DELETE FROM sys_menu WHERE name = 'tms_demo' OR id = ${tmsMenuGroupId};
INSERT INTO sys_menu
  (id, gmt_create, gmt_modify, state, pid, name, path, component, title, is_hidden, sequence, page_type, table_code, option, icon, redirect, is_unfold)
VALUES
  (${tmsMenuGroupId}, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, true, 0, 'tms_demo', 'tms-demo', 'src/components/Layout/Layout.vue', 'TMS测试', false, 4, 'directory', '', '', 'local_shipping', '', true);
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT id, ${tmsMenuGroupId} FROM sys_role WHERE name = 'super_admin'
ON CONFLICT DO NOTHING;
UPDATE sys_menu
SET pid = ${tmsMenuGroupId},
    path = CASE table_code
      WHEN 'tms_company' THEN 'company'
      WHEN 'tms_waybill' THEN 'waybill'
      WHEN 'tms_vehicle' THEN 'vehicle'
      ELSE path
    END,
    sequence = CASE table_code
      WHEN 'tms_company' THEN 1
      WHEN 'tms_waybill' THEN 2
      WHEN 'tms_vehicle' THEN 3
      ELSE sequence
    END,
    icon = CASE table_code
      WHEN 'tms_company' THEN 'business'
      WHEN 'tms_waybill' THEN 'receipt_long'
      WHEN 'tms_vehicle' THEN 'local_shipping'
      ELSE icon
    END,
    gmt_modify = CURRENT_TIMESTAMP
WHERE table_code IN ('tms_company','tms_waybill','tms_vehicle');
`)
  console.log('OK TMS menu group')
}

async function menuByTable(tableCode) {
  const menus = await request('/admin/menu/my')
  assert(menus.status === 200 && menus.body?.success, `menu query failed: ${JSON.stringify(menus.body)}`)
  const menu = findMenuByTable(menus.body.data, tableCode)
  assert(menu?.id, `menu for ${tableCode} not found`)
  return menu
}

async function menuByName(name) {
  const menus = await request('/admin/menu/my')
  assert(menus.status === 200 && menus.body?.success, `menu query failed: ${JSON.stringify(menus.body)}`)
  const menu = findMenuByName(menus.body.data, name)
  assert(menu?.id, `menu ${name} not found`)
  return menu
}

function findMenuByTable(menus, tableCode) {
  for (const menu of menus || []) {
    if (menu.table_code === tableCode || menu.option === tableCode) return menu
    const child = findMenuByTable(menu.children, tableCode)
    if (child) return child
  }
  return null
}

function findMenuByName(menus, name) {
  for (const menu of menus || []) {
    if (menu.name === name) return menu
    const child = findMenuByName(menu.children, name)
    if (child) return child
  }
  return null
}

async function createLowCodeRow(tableCode, menuId, data) {
  const created = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({ table_code: tableCode, menu_id: menuId, data }),
  })
  assert(created.status === 200 && created.body?.success, `create ${tableCode} row failed: ${JSON.stringify(created.body)}`)
}

async function queryLowCodeRows(tableCode, menuId, payload = {}) {
  const queried = await request(`/admin/generalization/query/code/${tableCode}`, {
    method: 'POST',
    body: JSON.stringify({
      page: 1,
      num: 50,
      table_code: tableCode,
      menu_id: menuId,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: '' },
      include_deleted: false,
      ...payload,
    }),
  })
  assert(queried.status === 200 && queried.body?.success, `query ${tableCode} failed: ${JSON.stringify(queried.body)}`)
  return queried.body
}

async function seedTmsRows(companyMenu, waybillMenu, vehicleMenu) {
  await createLowCodeRow('tms_company', companyMenu.id, { company_name: eastCompanyName, parent_id: null })
  await createLowCodeRow('tms_company', companyMenu.id, { company_name: southCompanyName, parent_id: null })
  const eastId = Number(runPostgres(`SELECT id FROM tms_company WHERE company_name = '${sqlString(eastCompanyName)}' LIMIT 1;`))
  const southId = Number(runPostgres(`SELECT id FROM tms_company WHERE company_name = '${sqlString(southCompanyName)}' LIMIT 1;`))
  assert(eastId > 0 && southId > 0, 'seeded company ids missing')

  await createLowCodeRow('tms_waybill', waybillMenu.id, {
    waybill_no: 'WB-EAST-001',
    company_id: eastId,
    customer_name: '华东客户',
    status: '待发车',
  })
  await createLowCodeRow('tms_waybill', waybillMenu.id, {
    waybill_no: 'WB-SOUTH-001',
    company_id: southId,
    customer_name: '华南客户',
    status: '待发车',
  })
  await createLowCodeRow('tms_vehicle', vehicleMenu.id, {
    plate_no: '沪A10001',
    company_id: eastId,
    driver_name: '张华东',
  })
  await createLowCodeRow('tms_vehicle', vehicleMenu.id, {
    plate_no: '粤B20002',
    company_id: southId,
    driver_name: '李华南',
  })

  const eastWaybillId = Number(runPostgres("SELECT id FROM tms_waybill WHERE waybill_no = 'WB-EAST-001' LIMIT 1;"))
  const southWaybillId = Number(runPostgres("SELECT id FROM tms_waybill WHERE waybill_no = 'WB-SOUTH-001' LIMIT 1;"))
  assert(eastWaybillId > 0 && southWaybillId > 0, 'seeded waybill ids missing')
  console.log(`OK TMS rows eastCompany=${eastId} southCompany=${southId}`)
  return { eastId, southId, eastWaybillId, southWaybillId }
}

async function createDataPermissionDimension() {
  const created = await request('/admin/data-permission/dimension', {
    method: 'POST',
    body: JSON.stringify({
      code: dimensionCode,
      name: '所属公司',
      value_type: 'number',
      source_type: 'table',
      source_code: 'tms_company',
      label_field: 'company_name',
      value_field: 'id',
      parent_field: 'parent_id',
      memo: 'TMS公司维度真实测试',
      state: true,
    }),
  })
  assert(created.status === 200 && created.body?.success, `create dimension failed: ${JSON.stringify(created.body)}`)
  console.log('OK TMS company dimension')
}

async function bindMenuDataScope(menu, tableCode) {
  const saved = await request(`/admin/data-permission/bindings/menu/${menu.id}`, {
    method: 'PUT',
    body: JSON.stringify({
      menu_id: menu.id,
      bindings: [
        {
          dimension_code: dimensionCode,
          field_code: 'company_id',
          match_type: 'in',
          required: true,
          actions: ['query', 'detail', 'create', 'update', 'delete', 'export', 'batch_delete'],
          state: true,
        },
      ],
    }),
  })
  assert(saved.status === 200 && saved.body?.success, `bind ${tableCode} data scope failed: ${JSON.stringify(saved.body)}`)
}

async function createOperatorRoleAndUsers(tmsMenuGroup, waybillMenu, vehicleMenu, dataPermissionMenu, ids) {
  const roleCreated = await request('/admin/role', {
    method: 'POST',
    body: JSON.stringify({ name: roleName, memo: 'TMS运营真实测试角色，按当前用户归属过滤公司' }),
  })
  assert(roleCreated.status === 200 && roleCreated.body?.success, `create role failed: ${JSON.stringify(roleCreated.body)}`)
  const roleId = Number(runPostgres(`SELECT id FROM sys_role WHERE name = '${sqlString(roleName)}' AND gmt_delete IS NULL LIMIT 1;`))
  assert(roleId > 0, 'created role id missing')
  const debugButton = (dataPermissionMenu.menu_buttons || []).find((button) => button.code === 'system_data_permission_debug')
  assert(debugButton?.id, 'data permission debug API button missing')
  const menuIds = [tmsMenuGroup.id, waybillMenu.id, vehicleMenu.id, dataPermissionMenu.id]
  const buttonIds = [...(waybillMenu.menu_buttons || []), ...(vehicleMenu.menu_buttons || []), debugButton]
    .filter((button) => !button.is_disabled)
    .map((button) => button.id)
  assert(buttonIds.length >= 10, `expected low-code buttons for TMS menus, got ${buttonIds.length}`)
  const assigned = await request('/admin/role/assign-permissions', {
    method: 'POST',
    body: JSON.stringify({
      role_id: roleId,
      menu_ids: menuIds,
      button_ids: buttonIds,
      data_permissions: [
        {
          menu_id: waybillMenu.id,
          table_code: 'tms_waybill',
          dimension_code: dimensionCode,
          strategy: 'user_dimension',
          scope_values: [],
          state: true,
        },
        {
          menu_id: vehicleMenu.id,
          table_code: 'tms_vehicle',
          dimension_code: dimensionCode,
          strategy: 'user_dimension',
          scope_values: [],
          state: true,
        },
      ],
    }),
  })
  assert(assigned.status === 200 && assigned.body?.success, `assign role failed: ${JSON.stringify(assigned.body)}`)

  const users = [
    { id: 810000001, name: eastOperatorName, phone: '13910000001', companyId: ids.eastId },
    { id: 810000002, name: southOperatorName, phone: '13910000002', companyId: ids.southId },
  ]
  for (const user of users) {
    const email = `${user.name}@tms.local`
    const passwordHash = hashUserPassword(operatorPassword, user.id)
    runPostgres(`
DELETE FROM sys_user_role WHERE user_id = ${user.id};
DELETE FROM sys_user_dimension_value WHERE user_id = ${user.id};
DELETE FROM sys_user WHERE id = ${user.id} OR user_name = '${sqlString(user.name)}' OR phone_number = '${sqlString(user.phone)}';
INSERT INTO sys_user
  (id, gmt_create, gmt_modify, state, user_name, password, email, phone_number, password_changed_at, language, access_tokens, is_reset)
VALUES
  (${user.id}, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, true, '${sqlString(user.name)}', '${passwordHash}', '${sqlString(email)}', '${sqlString(user.phone)}', CURRENT_TIMESTAMP, 'zh-CN', '', false);
INSERT INTO sys_user_role (user_id, role_id) VALUES (${user.id}, ${roleId});
`)
    const savedOwnership = await request(`/admin/user/${user.id}/dimension-values`, {
      method: 'PUT',
      body: JSON.stringify({
        user_id: user.id,
        items: [
          {
            dimension_code: dimensionCode,
            scope_values: [String(user.companyId)],
            state: true,
          },
        ],
      }),
    })
    assert(
      savedOwnership.status === 200 && savedOwnership.body?.success,
      `save user dimension values failed: ${JSON.stringify(savedOwnership.body)}`,
    )
    clearUserCacheKeys(`USER_CACHE_KEY_${user.id}`, `USER_CACHE_KEY_${user.name}`, `USER_CACHE_KEY_${user.phone}`)
  }
  console.log(`OK operator users ${eastOperatorName}/${southOperatorName}/${operatorPassword}`)
}

function rowCodes(rows, field) {
  return (rows || []).map((row) => row[field]).sort()
}

async function assertDenied(response, context) {
  assert(
    [400, 403].includes(response.status) && response.body?.success === false,
    `${context} should be denied: status=${response.status} body=${JSON.stringify(response.body)}`,
  )
}

async function verifyAsOperator(operatorName, waybillMenu, vehicleMenu, ids, expected) {
  const login = await loginAs(operatorName, operatorPassword)
  assert(login.status === 200 && login.body?.success, `${operatorName} login failed: ${JSON.stringify(login.body)}`)
  accessToken = login.body.data.access_token
  assert(accessToken, 'operator token missing')

  const waybills = await queryLowCodeRows('tms_waybill', waybillMenu.id)
  assert(Number(waybills.total) === 1, `${operatorName} should see one waybill: ${JSON.stringify(waybills)}`)
  assert(
    JSON.stringify(rowCodes(waybills.data, 'waybill_no')) === JSON.stringify([expected.waybillNo]),
    `${operatorName} waybill scope mismatch: ${JSON.stringify(waybills.data)}`,
  )

  const vehicles = await queryLowCodeRows('tms_vehicle', vehicleMenu.id)
  assert(Number(vehicles.total) === 1, `${operatorName} should see one vehicle: ${JSON.stringify(vehicles)}`)
  assert(
    JSON.stringify(rowCodes(vehicles.data, 'plate_no')) === JSON.stringify([expected.plateNo]),
    `${operatorName} vehicle scope mismatch: ${JSON.stringify(vehicles.data)}`,
  )

  const allowedDetail = await request(`/admin/generalization/detail/code/tms_waybill/${expected.allowedWaybillId}?menu_id=${waybillMenu.id}`)
  assert(allowedDetail.status === 200 && allowedDetail.body?.success, `${operatorName} allowed detail failed: ${JSON.stringify(allowedDetail.body)}`)

  const deniedDetail = await request(`/admin/generalization/detail/code/tms_waybill/${expected.deniedWaybillId}?menu_id=${waybillMenu.id}`)
  await assertDenied(deniedDetail, `${operatorName} denied detail`)

  const allowedCreate = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: 'tms_waybill',
      menu_id: waybillMenu.id,
      data: {
        waybill_no: `${expected.waybillNo}-NEW`,
        company_id: expected.allowedCompanyId,
        customer_name: `${expected.companyLabel}新增客户`,
        status: '已创建',
      },
    }),
  })
  assert(allowedCreate.status === 200 && allowedCreate.body?.success, `${operatorName} create should be allowed: ${JSON.stringify(allowedCreate.body)}`)

  const deniedCreate = await request('/admin/generalization/create', {
    method: 'POST',
    body: JSON.stringify({
      table_code: 'tms_waybill',
      menu_id: waybillMenu.id,
      data: {
        waybill_no: `${expected.waybillNo}-DENIED`,
        company_id: expected.deniedCompanyId,
        customer_name: '越权新增客户',
        status: '已创建',
      },
    }),
  })
  await assertDenied(deniedCreate, `${operatorName} denied create`)

  const allowedUpdate = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: expected.allowedWaybillId,
      table_code: 'tms_waybill',
      menu_id: waybillMenu.id,
      data: { status: '已发车', company_id: expected.allowedCompanyId },
    }),
  })
  assert(allowedUpdate.status === 200 && allowedUpdate.body?.success, `${operatorName} update should be allowed: ${JSON.stringify(allowedUpdate.body)}`)

  const deniedUpdateToOtherCompany = await request('/admin/generalization/update', {
    method: 'PUT',
    body: JSON.stringify({
      id: expected.allowedWaybillId,
      table_code: 'tms_waybill',
      menu_id: waybillMenu.id,
      data: { company_id: expected.deniedCompanyId },
    }),
  })
  await assertDenied(deniedUpdateToOtherCompany, `${operatorName} update waybill to denied company`)

  const deniedDelete = await request('/admin/generalization/delete', {
    method: 'DELETE',
    body: JSON.stringify({
      id: expected.deniedWaybillId,
      table_code: 'tms_waybill',
      menu_id: waybillMenu.id,
    }),
  })
  await assertDenied(deniedDelete, `${operatorName} denied delete`)

  const debug = await request(`/admin/data-permission/debug?menu_id=${waybillMenu.id}&table_code=tms_waybill&action=query`)
  assert(debug.status === 200 && debug.body?.success, `data permission debug failed: ${JSON.stringify(debug.body)}`)
  const conditions = debug.body.data?.scope?.Conditions || debug.body.data?.scope?.conditions || []
  assert(
    conditions.some((item) => item.Field === 'company_id' || item.field === 'company_id'),
    `debug scope missing company_id condition: ${JSON.stringify(debug.body.data)}`,
  )
  assert(
    JSON.stringify(debug.body.data).includes(String(expected.allowedCompanyId)) && !JSON.stringify(debug.body.data).includes(String(expected.deniedCompanyId)),
    `debug scope values mismatch: ${JSON.stringify(debug.body.data)}`,
  )
  assert(
    (debug.body.data?.user_dimensions || []).length === 1,
    `debug should include user dimension values: ${JSON.stringify(debug.body.data)}`,
  )

  console.log(`OK ${operatorName} data permission query/detail/create/update/delete/debug`)
}

async function main() {
  cleanupTmsDemo()
  await loginAdmin()
  await createTmsTables()
  await verifyTmsCompanyFieldLinkage()
  ensureTmsMenuGroup()
  const tmsMenuGroup = await menuByName('tms_demo')
  const companyMenu = await menuByTable('tms_company')
  const waybillMenu = await menuByTable('tms_waybill')
  const vehicleMenu = await menuByTable('tms_vehicle')
  const dataPermissionMenu = await menuByName('system_data_permission')
  const ids = await seedTmsRows(companyMenu, waybillMenu, vehicleMenu)
  await createDataPermissionDimension()
  await bindMenuDataScope(waybillMenu, 'tms_waybill')
  await bindMenuDataScope(vehicleMenu, 'tms_vehicle')
  await createOperatorRoleAndUsers(tmsMenuGroup, waybillMenu, vehicleMenu, dataPermissionMenu, ids)
  await verifyAsOperator(eastOperatorName, waybillMenu, vehicleMenu, ids, {
    waybillNo: 'WB-EAST-001',
    plateNo: '沪A10001',
    allowedCompanyId: ids.eastId,
    deniedCompanyId: ids.southId,
    allowedWaybillId: ids.eastWaybillId,
    deniedWaybillId: ids.southWaybillId,
    companyLabel: '华东',
  })
  await verifyAsOperator(southOperatorName, waybillMenu, vehicleMenu, ids, {
    waybillNo: 'WB-SOUTH-001',
    plateNo: '粤B20002',
    allowedCompanyId: ids.southId,
    deniedCompanyId: ids.eastId,
    allowedWaybillId: ids.southWaybillId,
    deniedWaybillId: ids.eastWaybillId,
    companyLabel: '华南',
  })
  console.log('OK TMS data permission smoke completed')
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
