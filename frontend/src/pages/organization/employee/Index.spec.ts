import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/employee/Index.vue'),
  'utf8',
)

describe('organization employee query scheme integration', () => {
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
