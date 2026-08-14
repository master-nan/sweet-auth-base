import { describe, expect, it } from 'vitest'
import { useTableQueryState } from 'src/composables/table-query-state'
import type { Query } from 'src/types/global'

const createQuery = (): Query => ({
  page: 3,
  num: 20,
  order: { field: '', is_asc: false },
  quick_query: { keyword: 'Sweet' },
  expressions: [{ rules: [{ field: 'status', value: 'enabled' }], nested: [] }],
})

describe('useTableQueryState', () => {
  it('refreshes without changing query, pagination or filters', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    const snapshot = state.refreshSnapshot()

    expect(snapshot).toEqual(createQuery())
    expect(state.query.value.page).toBe(3)
    expect(state.query.value.quick_query?.keyword).toBe('Sweet')
  })

  it('resets page for advanced apply, sorting and page size changes', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    state.beginAdvancedEdit()
    state.draftAdvanced.value.expressions = [
      { rules: [{ field: 'name', value: 'Platform' }], nested: [] },
    ]
    state.applyAdvancedQuery()
    expect(state.query.value.page).toBe(1)
    expect(state.query.value.expressions[0]?.rules[0]?.field).toBe('name')

    state.query.value.page = 4
    expect(state.applySorting('name', true, new Set(['name']))).toBe(true)
    expect(state.query.value.order).toEqual({ field: 'name', is_asc: false })
    expect(state.query.value.page).toBe(1)

    state.query.value.page = 2
    state.setPageSize(50)
    expect(state.query.value).toMatchObject({ page: 1, num: 50 })
  })

  it('rejects sorting fields outside the server allowlist', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    expect(state.applySorting('secret', false, new Set(['name']))).toBe(false)
    expect(state.query.value.order).toEqual({ field: '', is_asc: false })
  })
})
