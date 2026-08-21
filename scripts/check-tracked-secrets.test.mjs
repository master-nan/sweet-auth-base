import assert from 'node:assert/strict'
import test from 'node:test'

import { scanTrackedEntries } from './check-tracked-secrets.mjs'

test('scanTrackedEntries detects high-confidence credential formats without returning values', () => {
  const privateKey = ['-----BEGIN', 'PRIVATE KEY-----'].join(' ')
  const githubToken = ['ghp', 'abcdefghijklmnopqrstuvwxyz1234567890'].join('_')
  const credentialURI = ['postgresql://admin', 'cleartext-password@db.internal/app'].join(':')
  const findings = scanTrackedEntries([
    { path: 'config/private.pem', content: privateKey },
    { path: 'scripts/deploy.sh', content: `TOKEN=${githubToken}` },
    { path: 'docs/example.md', content: credentialURI },
  ])

  assert.deepEqual(findings.map(({ path, rule }) => ({ path, rule })), [
    { path: 'config/private.pem', rule: 'private-key' },
    { path: 'scripts/deploy.sh', rule: 'github-token' },
    { path: 'docs/example.md', rule: 'credential-uri' },
  ])
  assert.equal(JSON.stringify(findings).includes('cleartext-password'), false)
})

test('scanTrackedEntries detects fixed production defaults but permits runtime interpolation', () => {
  const findings = scanTrackedEntries([
    {
      path: 'docker-compose.external.yml',
      content: [
        'APP_SESSION_SECRET: ${APP_SESSION_SECRET:-known-production-secret}',
        'APP_CONF_SALT: ${APP_CONF_SALT}',
        'APP_BOOTSTRAP_APPLICATION_SECRET: replace-with-real-value',
      ].join('\n'),
    },
    {
      path: 'docker-compose.yml',
      content: 'APP_SESSION_SECRET: ${APP_SESSION_SECRET:-local-development-secret}',
    },
  ])

  assert.deepEqual(findings.map(({ path, line, rule }) => ({ path, line, rule })), [
    { path: 'docker-compose.external.yml', line: 1, rule: 'production-secret-default' },
  ])
})

test('scanTrackedEntries supports a narrow inline rule allowance', () => {
  const trainingURI = ['redis://training', 'training-password@example.invalid:6379/0'].join(':')
  const findings = scanTrackedEntries([{
    path: 'docs/security.md',
    content: [
      '<!-- secret-scan: allow credential-uri; intentionally invalid training value -->',
      trainingURI,
    ].join('\n'),
  }])

  assert.deepEqual(findings, [])
})

test('scanTrackedEntries ignores binary entries and placeholder DSNs without passwords', () => {
  const binaryCredentialURI = ['postgresql://admin', 'password@db/app'].join(':')
  const findings = scanTrackedEntries([
    { path: 'asset.bin', content: `\0${binaryCredentialURI}` },
    { path: 'docs/operations.md', content: 'postgresql://<user>:<password>@<host>/<database>' },
    { path: '.github/workflows/release.yml', content: 'POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}' },
  ])

  assert.deepEqual(findings, [])
})
