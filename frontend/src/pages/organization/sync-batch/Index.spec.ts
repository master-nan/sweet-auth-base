import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/sync-batch/Index.vue'),
  'utf8',
)

describe('organization sync batch query scheme integration', () => {
  it('initializes before fetching and refreshes business data directly', () => {
    const mounted = source.slice(source.indexOf('onMounted(async () => {'))
    const initializeIndex = mounted.indexOf('await schemePage.initialize()')
    expect(initializeIndex).toBeGreaterThanOrEqual(0)
    expect(initializeIndex).toBeLessThan(mounted.indexOf('await fetchData()'))
    expect(source).toContain('@refresh="fetchData"')
    expect(source.match(/schemePage\.initialize\(\)/g)).toHaveLength(1)
  })

  it('keeps batch detail and error diagnosis behavior', () => {
    expect(source).toContain('getSyncBatchError')
    expect(source).toContain("buildOrganizationDetailRoute('org_sync_batch'")
    expect(source).toContain("button.event_action === 'view_error'")
  })
})
