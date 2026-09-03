import { describe, expect, it } from 'vitest'
import { useTableQueryState } from '@/composables/table-query-state'
import type { Query } from '@/types/global'
import { QuerySchemeBindingKind, QuerySchemeType } from '@/modules/query-scheme/types'

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

  it('submits quick search without clearing advanced conditions, bindings or sorting', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    state.markSchemeSaved({
      id: 9,
      name: '启用记录',
      type: QuerySchemeType.PERSONAL,
      revision: 1,
      is_default: false,
    })
    state.bindings.value = [
      {
        pointer: '/expressions/0/rules/0/value',
        kind: QuerySchemeBindingKind.CURRENT_USER,
      },
    ]
    state.query.value.order = { field: 'gmt_create', is_asc: false }
    state.keyword.value = 'Platform'

    state.submitQuickSearch()

    expect(state.query.value).toMatchObject({
      page: 1,
      quick_query: { keyword: 'Platform' },
      expressions: [{ rules: [{ field: 'status', value: 'enabled' }], nested: [] }],
      order: { field: 'gmt_create', is_asc: false },
    })
    expect(state.bindings.value).toEqual([
      {
        pointer: '/expressions/0/rules/0/value',
        kind: QuerySchemeBindingKind.CURRENT_USER,
      },
    ])
    expect(state.schemeSource.value?.id).toBe(9)
  })

  it('rejects sorting fields outside the server allowlist', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    expect(state.applySorting('secret', false, new Set(['name']))).toBe(false)
    expect(state.query.value.order).toEqual({ field: '', is_asc: false })
  })

  it('tracks scheme dirty state without pagination changes', () => {
    const state = useTableQueryState({ createInitialQuery: createQuery })
    state.applyResolvedScheme(
      { id: 8, name: '常用查询', type: QuerySchemeType.PERSONAL, revision: 2, is_default: false },
      {
        expressions: createQuery().expressions,
        quick_query: { keyword: 'Sweet' },
        order: { field: '', is_asc: false },
      },
    )
    expect(state.dirty.value).toBe(false)

    state.setPage(9)
    state.setPageSize(50)
    expect(state.dirty.value).toBe(false)

    state.keyword.value = 'Platform'
    expect(state.dirty.value).toBe(true)
    expect(state.discardSchemeChanges()).toBe(true)
    expect(state.keyword.value).toBe('Sweet')
    expect(state.dirty.value).toBe(false)
  })
})
