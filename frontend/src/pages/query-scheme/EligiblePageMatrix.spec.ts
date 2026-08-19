import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const eligiblePages = [
  ['system.application.list', 'system/application/Index.vue', 'system_application'],
  ['system.user.list', 'system/user/Index.vue', 'system_user'],
  ['system.role.list', 'system/role/Index.vue', 'system_role'],
  ['system.sms.list', 'system/sms/Index.vue', 'system_sms'],
  ['system.audit.list', 'system/audit/Index.vue', 'system_audit'],
  ['organization.employee.list', 'organization/employee/Index.vue', 'organization_employee'],
  ['organization.position.list', 'organization/position/Index.vue', 'organization_position'],
  ['organization.sync_batch.list', 'organization/sync-batch/Index.vue', 'organization_sync_batch'],
  ['organization.sync_error.list', 'organization/sync-error/Index.vue', 'organization_sync_error'],
  [
    'integration.external_system.list',
    'integration/external-system/Index.vue',
    'integration_external_system',
  ],
  [
    'integration.interface_definition.list',
    'integration/interface-definition/Index.vue',
    'integration_interface_definition',
  ],
  ['integration.credential.list', 'integration/credential/Index.vue', 'integration_credential'],
  [
    'integration.retry_policy.list',
    'integration/retry-policy/Index.vue',
    'integration_retry_policy',
  ],
  ['integration.sync_task.list', 'integration/sync-task/Index.vue', 'integration_sync_task'],
  ['integration.sync_batch.list', 'integration/sync-batch/Index.vue', 'integration_sync_batch'],
  ['integration.execution.list', 'integration/execution/Index.vue', 'integration_execution'],
  ['integration.log.list', 'integration/log/Index.vue', 'integration_log'],
  ['develop.dictionary.master', 'develop/dictionary/Index.vue', 'develop_dictionary'],
] as const

const pageSource = (path: string) => readFileSync(resolve(process.cwd(), 'src/pages', path), 'utf8')

describe('Query Center eligible page matrix', () => {
  it('freezes the 18 approved scopes without duplicate identities', () => {
    expect(eligiblePages).toHaveLength(18)
    expect(new Set(eligiblePages.map(([scope]) => scope)).size).toBe(18)
    expect(new Set(eligiblePages.map(([, , routeName]) => routeName)).size).toBe(18)
  })

  it.each(eligiblePages)('%s uses the shared page integration', (_, path, routeName) => {
    const source = pageSource(path)

    expect(source).toContain(`useQuerySchemePage('${routeName}', queryState, reset`)
    expect(source).toContain('<query-scheme-selector')
    expect(source).toContain('<query-quick-presets')
    expect(source).toContain('<query-scheme-save-dialog')
    expect(source).toContain('<advanced-query')
    expect(source).toContain('runQueryChange')
    expect(source).not.toContain("from 'src/composables/query-schemes'")
  })
})
