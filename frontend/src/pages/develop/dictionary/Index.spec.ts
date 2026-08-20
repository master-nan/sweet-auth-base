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

describe('Dictionary query scheme integration', () => {
  it('keeps dictionary Item loading outside the Master query scheme state', () => {
    const masterFetch = functionSource('fetchData', 'syncCurrentDictAfterFetch')
    const itemFetch = functionSource('fetchDictItems', 'refreshDictItems')

    expect(masterFetch).toContain('dictApi.queryDict(query.value)')
    expect(itemFetch).toContain('dictApi.queryDictItemsByDictId(dictId)')
    expect(itemFetch).not.toContain('queryState')
    expect(itemFetch).not.toContain('query.value')
    expect(source).toContain('v-model="itemSearchText"')
  })
})
