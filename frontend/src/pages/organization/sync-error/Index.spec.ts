import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/pages/organization/sync-error/Index.vue'),
  'utf8',
)

describe('organization sync error query scheme integration', () => {
  it('preserves routed diagnostics as standard expressions during initialization', () => {
    const mounted = source.slice(source.indexOf('onMounted(async () => {'))
    const initializeIndex = mounted.indexOf(
      'await schemePage.initialize({ preserveInitialQuery: hasRouteContext })',
    )
    const fetchIndex = mounted.indexOf('await fetchData()')
    expect(initializeIndex).toBeGreaterThanOrEqual(0)
    expect(initializeIndex).toBeLessThan(fetchIndex)
    expect(source).toContain('@refresh="fetchData"')
    expect(source).toContain("field: 'object_type'")
    expect(source).toContain("field: 'local_id'")
    expect(source.match(/expression_type: ExpressionType\.EQ/g)).toHaveLength(2)
    expect(source).not.toContain('applyRouteFilters')
  })

  it('keeps failed-record defaults, details, and error diagnosis', () => {
    expect(source).toContain("status: 'failed'")
    expect(source).toContain('getSyncRecordDetail')
    expect(source).toContain('getSyncRecordError')
    expect(source).toContain("button.event_action === 'view_error'")
  })
})
