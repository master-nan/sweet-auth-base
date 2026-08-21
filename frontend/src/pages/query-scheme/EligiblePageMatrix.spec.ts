import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const eligiblePages = [
  ['system/application/Index.vue', 'system_application'],
  ['system/user/Index.vue', 'system_user'],
  ['system/role/Index.vue', 'system_role'],
  ['system/sms/Index.vue', 'system_sms'],
  ['system/audit/Index.vue', 'system_audit'],
  ['organization/employee/Index.vue', 'organization_employee'],
  ['organization/position/Index.vue', 'organization_position'],
  ['organization/sync-batch/Index.vue', 'organization_sync_batch'],
  ['organization/sync-error/Index.vue', 'organization_sync_error'],
  ['integration/external-system/Index.vue', 'integration_external_system'],
  ['integration/interface-definition/Index.vue', 'integration_interface_definition'],
  ['integration/credential/Index.vue', 'integration_credential'],
  ['integration/retry-policy/Index.vue', 'integration_retry_policy'],
  ['integration/sync-task/Index.vue', 'integration_sync_task'],
  ['integration/sync-batch/Index.vue', 'integration_sync_batch'],
  ['integration/execution/Index.vue', 'integration_execution'],
  ['integration/log/Index.vue', 'integration_log'],
] as const

const pageSource = (path: string) => readFileSync(resolve(process.cwd(), 'src/pages', path), 'utf8')

describe('Query Center eligible page matrix', () => {
  it('keeps each eligible page on the single page integration boundary', () => {
    expect(eligiblePages).toHaveLength(17)
    for (const [path, routeName] of eligiblePages) {
      const source = pageSource(path)
      expect(source, path).toContain(`useQuerySchemePage('${routeName}'`)
      expect(source.match(/useQuerySchemePage\(/g), path).toHaveLength(1)
    }
  })
})
