import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/develop/dictionary/Index.vue'),
  'utf8',
)

const functionSource = (name: string, nextName: string) => {
  const start = source.indexOf(`const ${name} =`)
  const end = source.indexOf(`const ${nextName} =`, start)
  return source.slice(start, end)
}

describe('Dictionary master-detail workspace', () => {
  it('keeps dictionary Item loading outside the Master query scheme state', () => {
    const masterFetch = functionSource('fetchData', 'syncCurrentDictAfterFetch')
    const itemFetch = functionSource('fetchDictItems', 'refreshDictItems')

    expect(masterFetch).toContain('dictApi.queryDict(query.value)')
    expect(itemFetch).toContain('dictApi.queryDictItemsByDictId(dictId)')
    expect(itemFetch).not.toContain('queryState')
    expect(itemFetch).not.toContain('query.value')
    expect(source).toContain('v-model="itemSearchText"')
  })

  it('keeps Query Center controls out of the narrow master pane', () => {
    expect(source).not.toContain('QuerySchemeControls')
    expect(source).not.toContain('useQuerySchemePage')
    expect(source).not.toContain('<query-scheme-controls')
    expect(source).toContain('master-width="372px"')
    expect(source).toContain('v-model="keyword"')
  })
})
