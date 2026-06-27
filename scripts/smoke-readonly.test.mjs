import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  createSmokeRuntime,
  loadOptionalEnvFile,
  resolveConfiguredCaptchaLoginFields,
  resolveSmokeCredential,
} from './smoke-readonly.mjs'

test('loadOptionalEnvFile loads env values without overriding explicit variables', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'sweet-admin-smoke-env-'))
  const envPath = path.join(dir, '.env.external')
  fs.writeFileSync(
    envPath,
    `
SWEET_ADMIN_ADMIN_USER=file_admin
SWEET_ADMIN_ADMIN_PASSWORD=file_password
SWEET_ADMIN_BASE_URL=http://localhost:8008/sweet_admin
`,
  )

  const targetEnv = {
    SWEET_ADMIN_ADMIN_USER: 'explicit_admin',
  }
  loadOptionalEnvFile(envPath, targetEnv)

  assert.equal(targetEnv.SWEET_ADMIN_ADMIN_USER, 'explicit_admin')
  assert.equal(targetEnv.SWEET_ADMIN_ADMIN_PASSWORD, 'file_password')
  assert.equal(targetEnv.SWEET_ADMIN_BASE_URL, 'http://localhost:8008/sweet_admin')
})

test('resolveSmokeCredential falls back to bootstrap password', () => {
  const credential = resolveSmokeCredential({
    APP_BOOTSTRAP_ADMIN_PASSWORD: 'bootstrap_password',
  })

  assert.deepEqual(credential, {
    username: 'admin',
    password: 'bootstrap_password',
  })
})

test('resolveSmokeCredential prefers smoke-specific password', () => {
  const credential = resolveSmokeCredential({
    SWEET_ADMIN_ADMIN_USER: 'audit_admin',
    SWEET_ADMIN_ADMIN_PASSWORD: 'current_password',
    APP_BOOTSTRAP_ADMIN_PASSWORD: 'bootstrap_password',
  })

  assert.deepEqual(credential, {
    username: 'audit_admin',
    password: 'current_password',
  })
})

test('createSmokeRuntime normalizes URLs and resolves table code', () => {
  const runtime = createSmokeRuntime({
    SWEET_ADMIN_BASE_URL: 'http://localhost:8008/sweet_admin/',
    SWEET_ADMIN_HEALTH_BASE_URL: 'http://localhost:9009/',
    SWEET_ADMIN_ADMIN_USER: 'audit_admin',
    SWEET_ADMIN_ADMIN_PASSWORD: 'current_password',
    SWEET_ADMIN_SMOKE_TABLE: 'application',
    SWEET_ADMIN_SMOKE_CAPTCHA_ID: 'captcha-id',
    SWEET_ADMIN_SMOKE_CAPTCHA: '123456',
    SWEET_ADMIN_SMOKE_CAPTCHA_IMAGE_FILE: '/tmp/captcha.png',
  })

  assert.deepEqual(runtime, {
    baseUrl: 'http://localhost:8008/sweet_admin',
    healthBaseUrl: 'http://localhost:9009',
    username: 'audit_admin',
    password: 'current_password',
    tableCode: 'application',
    captcha: {
      enabled: false,
      id: 'captcha-id',
      value: '123456',
      imageFile: '/tmp/captcha.png',
    },
  })
})

test('resolveConfiguredCaptchaLoginFields returns empty fields when captcha is disabled', () => {
  const runtime = createSmokeRuntime({})

  assert.deepEqual(resolveConfiguredCaptchaLoginFields(runtime), {
    captcha: '',
    captcha_id: '',
  })
})

test('resolveConfiguredCaptchaLoginFields requires both captcha id and value when enabled', () => {
  const runtime = createSmokeRuntime({
    SWEET_ADMIN_SMOKE_CAPTCHA_ID: 'captcha-id',
  })
  runtime.captcha.enabled = true

  assert.equal(resolveConfiguredCaptchaLoginFields(runtime), null)
})

test('resolveConfiguredCaptchaLoginFields uses configured captcha when enabled', () => {
  const runtime = createSmokeRuntime({
    SWEET_ADMIN_SMOKE_CAPTCHA_ID: 'captcha-id',
    SWEET_ADMIN_SMOKE_CAPTCHA: '123456',
  })
  runtime.captcha.enabled = true

  assert.deepEqual(resolveConfiguredCaptchaLoginFields(runtime), {
    captcha: '123456',
    captcha_id: 'captcha-id',
  })
})
