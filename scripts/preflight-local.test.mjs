import assert from 'node:assert/strict'
import test from 'node:test'

import {
  evaluateDiskCheck,
  evaluateDockerCache,
  parseComposePs,
  parseDfPk,
  parseDockerSystemDf,
  parseSizeToMB,
  validateServiceHealth,
} from './preflight-local.mjs'

test('parseSizeToMB handles docker size units', () => {
  assert.equal(parseSizeToMB('0B'), 0)
  assert.equal(parseSizeToMB('512MB'), 512)
  assert.equal(parseSizeToMB('12.82GB'), 13128)
  assert.equal(parseSizeToMB('1TB'), 1048576)
  assert.equal(parseSizeToMB('not-a-size'), null)
})

test('parseDockerSystemDf parses json lines', () => {
  const rows = parseDockerSystemDf(`
{"Type":"Images","Size":"3.4GB","Reclaimable":"0B (0%)"}
{"Type":"Build Cache","Size":"13.01GB","Reclaimable":"12.82GB"}
`)

  assert.equal(rows.length, 2)
  assert.equal(rows[1].Type, 'Build Cache')
})

test('parseDfPk reads available space from POSIX df output', () => {
  const disk = parseDfPk(`
Filesystem     1024-blocks     Used Available Capacity Mounted on
/dev/vda1         61202244 28784596  29276324      50% /var/lib/postgresql/data
`)

  assert.equal(disk.availableMB, 28590)
  assert.equal(disk.capacity, '50%')
  assert.equal(disk.mount, '/var/lib/postgresql/data')
})

test('evaluateDiskCheck rejects low free space', () => {
  assert.equal(
    evaluateDiskCheck('postgres', { availableMB: 1024 }, 2048),
    'postgres has 1024MB free; require at least 2048MB',
  )
  assert.equal(evaluateDiskCheck('postgres', { availableMB: 4096 }, 2048), '')
})

test('evaluateDockerCache warns for large reclaimable build cache', () => {
  const warning = evaluateDockerCache([{ Type: 'Build Cache', Reclaimable: '12.82GB' }], 10240)

  assert.match(warning, /Docker build cache/)
  assert.equal(evaluateDockerCache([{ Type: 'Build Cache', Reclaimable: '512MB' }], 10240), '')
})

test('parseComposePs and validateServiceHealth check required services', () => {
  const services = parseComposePs(`
{"Service":"postgres","State":"running","Health":"healthy"}
{"Service":"redis","State":"running","Health":"healthy"}
{"Service":"backend","State":"running","Health":"starting"}
`)

  const problems = validateServiceHealth(services, ['postgres', 'redis', 'backend', 'frontend'])

  assert.match(problems.join('\n'), /backend health is starting/)
  assert.match(problems.join('\n'), /frontend is not present/)
})
