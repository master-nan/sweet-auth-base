import cloneDeep from 'lodash/cloneDeep'
import { computed, ref, type Ref } from 'vue'
import type { ExpressionGroup, Query } from 'src/types/global'

export interface TableQueryStateOptions<TQuery extends Query> {
  createInitialQuery: () => TQuery
  createEmptyExpressions?: () => ExpressionGroup[]
}

const defaultEmptyExpressions = (): ExpressionGroup[] => [
  { rules: [{ field: '', value: null }], nested: [] },
]

export function useTableQueryState<TQuery extends Query>(
  options: TableQueryStateOptions<TQuery>,
) {
  const query = ref(options.createInitialQuery()) as Ref<TQuery>
  const draftAdvanced = ref(cloneDeep(query.value)) as Ref<TQuery>
  const appliedAdvanced = ref(cloneDeep(query.value)) as Ref<TQuery>

  const keyword = computed({
    get: () => query.value.quick_query?.keyword || '',
    set: (value: string) => {
      query.value.quick_query = { keyword: value }
    },
  })

  const resetPage = () => {
    query.value.page = 1
  }

  const submitQuickSearch = () => {
    query.value.expressions = (options.createEmptyExpressions || defaultEmptyExpressions)()
    appliedAdvanced.value = cloneDeep(query.value)
    resetPage()
  }

  const beginAdvancedEdit = () => {
    draftAdvanced.value = cloneDeep(query.value)
  }

  const applyAdvancedQuery = (value: TQuery = draftAdvanced.value) => {
    query.value.expressions = cloneDeep(value.expressions)
    appliedAdvanced.value = cloneDeep(query.value)
    resetPage()
  }

  const applySorting = (
    field: string,
    descending: boolean,
    allowedFields?: ReadonlySet<string>,
  ) => {
    if (field && allowedFields && !allowedFields.has(field)) return false
    query.value.order = { field, is_asc: field ? !descending : false }
    resetPage()
    return true
  }

  const setPage = (page: number) => {
    query.value.page = Math.max(1, page)
  }

  const setPageSize = (pageSize: number) => {
    query.value.num = pageSize
    resetPage()
  }

  const refreshSnapshot = () => cloneDeep(query.value)

  const clearQuery = () => {
    query.value = options.createInitialQuery()
    draftAdvanced.value = cloneDeep(query.value)
    appliedAdvanced.value = cloneDeep(query.value)
  }

  return {
    query,
    keyword,
    draftAdvanced,
    appliedAdvanced,
    resetPage,
    submitQuickSearch,
    beginAdvancedEdit,
    applyAdvancedQuery,
    applySorting,
    setPage,
    setPageSize,
    refreshSnapshot,
    clearQuery,
  }
}
