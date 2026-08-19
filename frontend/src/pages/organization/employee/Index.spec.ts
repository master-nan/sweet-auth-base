import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/employee/Index.vue'),
  'utf8',
)

describe('organization employee query scheme integration', () => {
  it('uses the shared page composable and unified query scheme bindings', () => {
    expect(source).toContain(
      "useQuerySchemePage('organization_employee', queryState, resetAndFetch)",
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

  it('initializes the scheme before fetching without changing refresh behavior', () => {
    const mounted = source.slice(source.indexOf('onMounted(async () => {'))
    const initializeIndex = mounted.indexOf('await schemePage.initialize()')
    expect(initializeIndex).toBeGreaterThanOrEqual(0)
    expect(initializeIndex).toBeLessThan(mounted.indexOf('await fetchData()'))
    expect(source).toContain('@refresh="fetchData"')
    expect(source.match(/schemePage\.initialize\(\)/g)).toHaveLength(1)
  })

  it('keeps employee account binding, assignments, and sync diagnostics', () => {
    expect(source).toContain('bindEmployeeUser')
    expect(source).toContain('queryAssignments')
    expect(source).toContain("name: 'organization_sync_error'")
  })
})
