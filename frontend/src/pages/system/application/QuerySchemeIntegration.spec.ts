import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'
import { describe, expect, it } from 'vitest'

const pageSource = (relativePath: string) =>
  readFileSync(fileURLToPath(new URL(relativePath, import.meta.url)), 'utf8')

const pages = [
  { path: '../user/Index.vue', routeName: 'system_user' },
  { path: '../role/Index.vue', routeName: 'system_role' },
  { path: '../sms/Index.vue', routeName: 'system_sms' },
  { path: '../audit/Index.vue', routeName: 'system_audit' },
  { path: './Index.vue', routeName: 'system_application' },
]

describe('QC-002C system query scheme page integration', () => {
  it.each(pages)('$routeName uses the shared page integration contract', ({ path, routeName }) => {
    const source = pageSource(path)

    expect(source).toContain('<query-scheme-selector')
    expect(source).toContain('<query-quick-presets')
    expect(source).toContain('<query-scheme-save-dialog')
    expect(source).toContain(`useQuerySchemePage('${routeName}', queryState, resetAndFetch)`)
    expect(source).toContain(':schemes="schemePage.runtime.schemes.value"')
    expect(source).toContain(':config="schemePage.runtime.scope.config.value"')
    expect(source).toContain('v-model="schemePage.showSaveDialog.value"')
    expect(source).toContain('v-model:bindings="queryState.bindings.value"')
    expect(source).toContain(':source-name="queryState.schemeSource.value?.name || \'\'"')
    expect(source).toContain(':dirty="queryState.dirty.value"')
    expect(source).toContain('<standard-table-toolbar :refreshing="loading" @refresh="fetchData">')
    expect(source).not.toContain("from 'src/composables/query-schemes'")

    const mounted = source.match(/onMounted\(async \(\) => \{([\s\S]*?)\n\}\)/)?.[1] || ''
    expect(mounted).toContain('await schemePage.initialize()')
    expect(mounted).toContain('await fetchData()')
    expect(mounted.indexOf('await schemePage.initialize()')).toBeLessThan(
      mounted.indexOf('await fetchData()'),
    )
    expect(source.match(/schemePage\.initialize\(\)/g)).toHaveLength(1)
  })

  it('keeps application on the shared composable without legacy page glue', () => {
    const source = pageSource('./Index.vue')

    expect(source).not.toContain("from 'src/composables/query-schemes'")
    expect(source).not.toContain('const applySelectedScheme =')
    expect(source).not.toContain('const applyQuickPreset =')
    expect(source).not.toContain('const saveScheme =')
    expect(source).not.toContain('currentRoute?.value?.query?.query_scheme_id')
  })
})
