import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/position/Index.vue'),
  'utf8',
)

describe('organization position query scheme integration', () => {
  it('moves the legacy scheme handlers to the shared page composable', () => {
    expect(source).toContain(
      "useQuerySchemePage('organization_position', queryState, resetAndFetch)",
    )
    expect(source).not.toContain("useQuerySchemes('organization_position'")
    expect(source).not.toContain('const applySelectedScheme =')
    expect(source).not.toContain('const saveScheme =')
  })

  it('uses unified selector, preset, save, and advanced-query bindings', () => {
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

  it('initializes before fetching and keeps position detail diagnostics', () => {
    const mounted = source.slice(source.indexOf('onMounted(async () => {'))
    const initializeIndex = mounted.indexOf('await schemePage.initialize()')
    expect(initializeIndex).toBeGreaterThanOrEqual(0)
    expect(initializeIndex).toBeLessThan(mounted.indexOf('await fetchData()'))
    expect(source).toContain('@refresh="fetchData"')
    expect(source.match(/schemePage\.initialize\(\)/g)).toHaveLength(1)
    expect(source).toContain('getPositionDetail')
    expect(source).toContain("object_type: 'position'")
  })
})
