import { translate as t } from '@/boot/i18n'
import { computed, onScopeDispose, ref } from 'vue'
import { useQuerySchemeApi } from '@/api/services/query-scheme'
import { useQueryScope } from '@/composables/query-scope'
import {
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
  type QuerySchemeIssue,
  type QuerySchemePayloadV1,
  type QuerySchemeResolveResult,
  type QuerySchemeSummary,
} from '@/modules/query-scheme/types'
import type { Query } from '@/types/global'
import { subscribeQuerySchemeDeleted } from '@/modules/query-scheme/events'

export type SchemeAwareQueryState<TQuery extends Query> = {
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
    if (!name) return t('ui.queryScheme')
    return queryState.dirty.value ? t('ui.modifiedSchemeName', { name: name }) : name
  })

  const personalDefault = computed(() =>
    schemes.value.find((scheme) => scheme.type === QuerySchemeType.PERSONAL && scheme.is_default),
  )
  const pageDefault = computed(() =>
    schemes.value.find(
      (scheme) => scheme.type === QuerySchemeType.PAGE_DEFAULT && scheme.is_default,
    ),
  )

  const loadAvailable = async () => {
    if (!scope.scopeCode.value) return null
    loading.value = true
    error.value = ''
    try {
      const response = await api.available(scope.scopeCode.value)
      schemes.value = response.data || []
      return schemes.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : t('ui.failedToLoadQueryScheme')
      schemes.value = []
      return null
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
      if (!result) throw new Error(t('ui.queryProjectResolutionResultsAreEmpty'))
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
      error.value = cause instanceof Error ? cause.message : t('ui.failedToQueryProgramApplication')
      return false
    } finally {
      loading.value = false
    }
  }

  const initialize = async (
    requestedSchemeId?: number,
    options: { preserveInitialQuery?: boolean } = {},
  ) => {
    if (initialized.value) return false
    const config = await scope.loadScope()
    if (!config) return false
    const availableSchemes = await loadAvailable()
    if (!availableSchemes) return false
    const requestedScheme = requestedSchemeId
      ? schemes.value.find((scheme) => scheme.id === requestedSchemeId)
      : undefined
    const defaultScheme =
      requestedScheme ||
      (options.preserveInitialQuery ? undefined : personalDefault.value || pageDefault.value)
    if (!defaultScheme) {
      initialized.value = true
      return false
    }
    const applied = await applyScheme(defaultScheme)
    initialized.value = applied || blockedScheme.value !== null
    return applied
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
    if (!scope.scopeCode.value) throw new Error(t('ui.theCurrentPageDoesNotConfigureTheQueryRange'))
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
    if (!detail) throw new Error(t('ui.querySchemeSaveResultIsEmpty'))
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
