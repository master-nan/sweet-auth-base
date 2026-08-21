import cloneDeep from 'lodash/cloneDeep'
import { computed, ref, type Ref } from 'vue'
import type { Query } from 'src/types/global'
import type {
  QuerySchemeBinding,
  QuerySchemePayloadV1,
  QuerySchemeResolvedQuery,
  QuerySchemeSource,
} from 'src/modules/query-scheme/types'
import { normalizeQuerySchemePayload, serializeQuerySchemePayload } from 'src/utils/query-state'

export interface TableQueryStateOptions<TQuery extends Query> {
  createInitialQuery: () => TQuery
}

export function useTableQueryState<TQuery extends Query>(options: TableQueryStateOptions<TQuery>) {
  const query = ref(options.createInitialQuery()) as Ref<TQuery>
  const draftAdvanced = ref(cloneDeep(query.value)) as Ref<TQuery>
  const appliedAdvanced = ref(cloneDeep(query.value)) as Ref<TQuery>
  const schemeSource = ref<QuerySchemeSource | null>(null)
  const schemeBaseline = ref<QuerySchemePayloadV1 | null>(null)
  const bindings = ref<QuerySchemeBinding[]>([])

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

  const currentSchemePayload = computed(() =>
    normalizeQuerySchemePayload(query.value, bindings.value),
  )
  const dirty = computed(() => {
    if (!schemeSource.value || !schemeBaseline.value) return false
    return (
      serializeQuerySchemePayload(currentSchemePayload.value) !==
      serializeQuerySchemePayload(schemeBaseline.value)
    )
  })

  const applyResolvedScheme = (
    source: QuerySchemeSource,
    resolved: QuerySchemeResolvedQuery,
    sourceBindings: QuerySchemeBinding[] = [],
  ) => {
    query.value.expressions = cloneDeep(resolved.expressions)
    query.value.quick_query = cloneDeep(resolved.quick_query)
    query.value.order = cloneDeep(resolved.order)
    bindings.value = cloneDeep(sourceBindings)
    resetPage()
    draftAdvanced.value = cloneDeep(query.value)
    appliedAdvanced.value = cloneDeep(query.value)
    schemeSource.value = { ...source }
    schemeBaseline.value = cloneDeep(currentSchemePayload.value)
  }

  const applySchemePayload = (payload: QuerySchemePayloadV1) => {
    query.value.expressions = cloneDeep(payload.expressions)
    query.value.quick_query = cloneDeep(payload.quick_query)
    query.value.order = cloneDeep(payload.order)
    bindings.value = cloneDeep(payload.bindings)
    resetPage()
    draftAdvanced.value = cloneDeep(query.value)
    appliedAdvanced.value = cloneDeep(query.value)
  }

  const markSchemeSaved = (source: QuerySchemeSource) => {
    schemeSource.value = { ...source }
    schemeBaseline.value = cloneDeep(currentSchemePayload.value)
  }

  const detachSchemeSource = () => {
    schemeSource.value = null
    schemeBaseline.value = null
  }

  const discardSchemeChanges = () => {
    if (!schemeBaseline.value) return false
    applySchemePayload(schemeBaseline.value)
    return true
  }

  const clearQuery = () => {
    query.value = options.createInitialQuery()
    draftAdvanced.value = cloneDeep(query.value)
    appliedAdvanced.value = cloneDeep(query.value)
    bindings.value = []
    detachSchemeSource()
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
    currentSchemePayload,
    schemeSource,
    schemeBaseline,
    bindings,
    dirty,
    applyResolvedScheme,
    applySchemePayload,
    markSchemeSaved,
    detachSchemeSource,
    discardSchemeChanges,
    clearQuery,
  }
}

export type TableQueryState<TQuery extends Query> = ReturnType<typeof useTableQueryState<TQuery>>
