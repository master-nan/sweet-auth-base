import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/sync-error/Index.vue'),
  'utf8',
)

describe('organization sync error query scheme integration', () => {
  it('uses the shared page composable and unified query scheme bindings', () => {
    expect(source).toContain(
      "useQuerySchemePage('organization_sync_error', queryState, resetAndFetch)",
    )
    expect(source).toContain('<query-scheme-selector')
    expect(source).toContain('<query-quick-presets')
    expect(source).toContain('<query-scheme-save-dialog')
    expect(source).toContain(':schemes="schemePage.runtime.schemes.value"')
    expect(source).toContain(':config="schemePage.runtime.scope.config.value"')
    expect(source).toContain('v-model="schemePage.showSaveDialog.value"')
    expect(source).toContain('v-model:bindings="queryState.bindings.value"')
    expect(source).toContain(':source-name="queryState.schemeSource.value?.name || \'\'"')
    expect(source).toContain(':dirty="queryState.dirty.value"')
  })

  it('applies contextual diagnostics after scheme initialization and before fetching', () => {
    const mounted = source.slice(source.indexOf('onMounted(async () => {'))
    const initializeIndex = mounted.indexOf('await schemePage.initialize()')
    const routeFilterIndex = mounted.indexOf('applyRouteFilters()')
    const fetchIndex = mounted.indexOf('await fetchData()')
    expect(initializeIndex).toBeGreaterThanOrEqual(0)
    expect(initializeIndex).toBeLessThan(routeFilterIndex)
    expect(routeFilterIndex).toBeLessThan(fetchIndex)
    expect(source).toContain('@refresh="fetchData"')
    expect(source.match(/schemePage\.initialize\(\)/g)).toHaveLength(1)
  })

  it('keeps failed-record defaults, details, and error diagnosis', () => {
    expect(source).toContain("status: 'failed'")
    expect(source).toContain('getSyncRecordDetail')
    expect(source).toContain('getSyncRecordError')
    expect(source).toContain("button.event_action === 'view_error'")
  })
})
