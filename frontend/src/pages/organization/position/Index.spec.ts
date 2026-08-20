import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/position/Index.vue'),
  'utf8',
)

describe('organization position query scheme integration', () => {
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
