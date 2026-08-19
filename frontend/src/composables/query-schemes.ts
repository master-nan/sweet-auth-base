import { computed, onScopeDispose, ref } from 'vue'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import { useQueryScope } from 'src/composables/query-scope'
import {
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
  type QuerySchemeIssue,
  type QuerySchemePayloadV1,
  type QuerySchemeResolveResult,
  type QuerySchemeSummary,
} from 'src/modules/query-scheme/types'
import type { Query } from 'src/types/global'
import { subscribeQuerySchemeDeleted } from 'src/modules/query-scheme/events'

type SchemeAwareQueryState<TQuery extends Query> = {
  query: { value: TQuery }
  schemeSource: { value: QuerySchemeResolveResult['scheme'] | null }
  bindings: { value: QuerySchemePayloadV1['bindings'] }
  dirty: { value: boolean }
  currentSchemePayload: { value: QuerySchemePayloadV1 }
  applyResolvedScheme: (
    source: QuerySchemeResolveResult['scheme'],
    query: NonNullable<QuerySchemeResolveResult['resolved_query']>,
    bindings: QuerySchemePayloadV1['bindings'],
  ) => void
  applySchemePayload: (payload: QuerySchemePayloadV1) => void
  markSchemeSaved: (source: QuerySchemeResolveResult['scheme']) => void
  detachSchemeSource: () => void
  discardSchemeChanges: () => boolean
  clearQuery: () => void
}

export function useQuerySchemes<TQuery extends Query>(
  routeName: string,
  queryState: SchemeAwareQueryState<TQuery>,
) {
  const api = useQuerySchemeApi()
  const scope = useQueryScope(routeName)
  const schemes = ref<QuerySchemeSummary[]>([])
  const loading = ref(false)
  const error = ref('')
  const issues = ref<QuerySchemeIssue[]>([])
  const blockedScheme = ref<QuerySchemeSummary | null>(null)
  const initialized = ref(false)

  const unsubscribeDeleted = subscribeQuerySchemeDeleted((id) => {
    if (queryState.schemeSource.value?.id === id) queryState.detachSchemeSource()
  })
  onScopeDispose(unsubscribeDeleted)

  const currentLabel = computed(() => {
    const name = queryState.schemeSource.value?.name
    if (!name) return '查询方案'
    return queryState.dirty.value ? `${name}（已修改）` : name
  })

  const personalDefault = computed(() =>
    schemes.value.find(
      (scheme) => scheme.type === QuerySchemeType.PERSONAL && scheme.is_default,
    ),
  )
  const pageDefault = computed(() =>
    schemes.value.find(
      (scheme) => scheme.type === QuerySchemeType.PAGE_DEFAULT && scheme.is_default,
    ),
  )

  const loadAvailable = async () => {
    if (!scope.scopeCode.value) return []
    loading.value = true
    error.value = ''
    try {
      const response = await api.available(scope.scopeCode.value)
      schemes.value = response.data || []
      return schemes.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '查询方案加载失败'
      schemes.value = []
      return []
    } finally {
      loading.value = false
    }
  }

  const applyScheme = async (scheme: QuerySchemeSummary) => {
    if (!scope.scopeCode.value) return false
    loading.value = true
    error.value = ''
    issues.value = []
    blockedScheme.value = null
    try {
      const response = await api.resolve(scheme.id, scope.scopeCode.value)
      const result = response.data
      if (!result) throw new Error('查询方案解析结果为空')
      if (
        result.validation_status !== QuerySchemeValidationStatus.VALID ||
        !result.resolved_query
      ) {
        issues.value = result.issues || []
        blockedScheme.value = scheme
        return false
      }
      queryState.applyResolvedScheme(result.scheme, result.resolved_query, result.bindings || [])
      return true
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '查询方案应用失败'
      return false
    } finally {
      loading.value = false
    }
  }

  const initialize = async (requestedSchemeId?: number) => {
    if (initialized.value) return false
    const config = await scope.loadScope()
    if (!config) return false
    await loadAvailable()
    initialized.value = true
    const requestedScheme = requestedSchemeId
      ? schemes.value.find((scheme) => scheme.id === requestedSchemeId)
      : undefined
    const defaultScheme = requestedScheme || personalDefault.value || pageDefault.value
    return defaultScheme ? applyScheme(defaultScheme) : false
  }

  const applyPreset = (payload: QuerySchemePayloadV1) => {
    queryState.applySchemePayload(payload)
  }

  const restoreCurrentScheme = () => queryState.discardSchemeChanges()
  const resetToDefault = async () => {
    const defaultScheme = personalDefault.value || pageDefault.value
    if (defaultScheme) return applyScheme(defaultScheme)
    queryState.clearQuery()
    return true
  }

  const savePersonal = async (name: string, isDefault: boolean, saveAs: boolean) => {
    if (!scope.scopeCode.value) throw new Error('当前页面未配置查询方案范围')
    const source = queryState.schemeSource.value
    let detail: QuerySchemeDetail | undefined
    if (!saveAs && source?.type === QuerySchemeType.PERSONAL) {
      const response = await api.updatePersonal(source.id, {
        name,
        query_payload: queryState.currentSchemePayload.value,
        is_default: isDefault,
        revision: source.revision,
      })
      detail = response.data
    } else {
      const response = await api.createPersonal({
        name,
        scope_code: scope.scopeCode.value,
        query_payload: queryState.currentSchemePayload.value,
        is_default: isDefault,
      })
      detail = response.data
    }
    if (!detail) throw new Error('查询方案保存结果为空')
    queryState.markSchemeSaved({
      id: detail.id,
      name: detail.name,
      type: detail.type,
      revision: detail.revision,
      is_default: detail.is_default,
    })
    await loadAvailable()
    return detail
  }

  return {
    scope,
    schemes,
    loading,
    error,
    issues,
    blockedScheme,
    initialized,
    currentLabel,
    initialize,
    loadAvailable,
    applyScheme,
    applyPreset,
    restoreCurrentScheme,
    resetToDefault,
    savePersonal,
  }
}
