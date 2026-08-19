import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
import { describe, expect, it } from 'vitest'

const pageSource = (relativePath: string) =>
  readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), 'utf8')

const standardListPages = [
  './integration/external-system/Index.vue',
  './integration/interface-definition/Index.vue',
  './integration/credential/Index.vue',
  './integration/retry-policy/Index.vue',
  './integration/sync-task/Index.vue',
  './integration/sync-batch/Index.vue',
  './integration/execution/Index.vue',
  './integration/log/Index.vue',
  './organization/position/Index.vue',
  './organization/employee/Index.vue',
  './organization/sync-batch/Index.vue',
  './organization/sync-error/Index.vue',
  './system/application/Index.vue',
  './system/user/Index.vue',
  './system/role/Index.vue',
  './system/sms/Index.vue',
  './system/audit/Index.vue',
  './develop/dictionary/Index.vue',
]

const metadataEntityPages = [
  './integration/external-system/Index.vue',
  './integration/interface-definition/Index.vue',
  './integration/credential/Index.vue',
  './integration/retry-policy/Index.vue',
  './integration/sync-task/Index.vue',
  './organization/position/Index.vue',
  './organization/employee/Index.vue',
  './system/application/Index.vue',
  './system/user/Index.vue',
  './system/role/Index.vue',
  './system/sms/Index.vue',
  './develop/dictionary/Index.vue',
]

describe('formal page consistency freeze', () => {
  it.each(standardListPages)('%s uses the shared toolbar and query state', (page) => {
    const source = pageSource(page)

    expect(source).toContain('StandardTableToolbar')
    expect(source).toContain('useTableQueryState')
  })

  it.each(metadataEntityPages)('%s reads safe runtime metadata', (page) => {
    expect(pageSource(page)).toContain('useRuntimeTableMetadata')
  })

  it('keeps production pages behind API services', () => {
    for (const page of standardListPages) {
      const source = pageSource(page)
      expect(source).not.toContain("from 'boot/axios'")
      expect(source).not.toMatch(/instance\.(get|post|put|delete|patch)\(/)
    }
  })

  it('does not provide an ungranted audit detail fallback', () => {
    const source = pageSource('./system/audit/Index.vue')

    expect(source).not.toContain("lineButtons.length === 0")
    expect(source).toContain('v-for="btn in lineButtons"')
  })
})
