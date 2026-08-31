import { translate as t } from 'src/boot/i18n'
import { ref } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import { useQuerySchemes, type SchemeAwareQueryState } from 'src/composables/query-schemes'
import type { QuerySchemePayloadV1, QuerySchemeSummary } from 'src/modules/query-scheme/types'
import type { Query } from 'src/types/global'

export const QUERY_SCHEME_NAVIGATION_STATE_KEY = 'query_scheme_id'

export function useQuerySchemePage<TQuery extends Query>(
  routeName: string,
  queryState: SchemeAwareQueryState<TQuery>,
  onQueryChanged: () => void | Promise<void>,
) {
  const $q = useQuasar()
  const route = useRoute()
  const router = useRouter()
  const runtime = useQuerySchemes(routeName, queryState)
  const showSaveDialog = ref(false)
  const saving = ref(false)

  const applyWithSingleRefresh = (previousPage: number) => {
    if (queryState.query.value.page === previousPage) void onQueryChanged()
  }

  const runQueryChange = (change: () => void) => {
    const previousPage = queryState.query.value.page
    change()
    applyWithSingleRefresh(previousPage)
  }

  const initialize = async (options: { preserveInitialQuery?: boolean } = {}) => {
    const routeRequestedID = Number(route.query[QUERY_SCHEME_NAVIGATION_STATE_KEY])
    const stateRequestedID = Number(
      typeof window === 'undefined'
        ? undefined
        : window.history.state?.[QUERY_SCHEME_NAVIGATION_STATE_KEY],
    )
    const requestedID =
      Number.isSafeInteger(routeRequestedID) && routeRequestedID > 0
        ? routeRequestedID
        : Number.isSafeInteger(stateRequestedID) && stateRequestedID > 0
          ? stateRequestedID
          : undefined
    try {
      return await runtime.initialize(requestedID, options)
    } finally {
      if (
        typeof window !== 'undefined' &&
        Object.prototype.hasOwnProperty.call(
          window.history.state || {},
          QUERY_SCHEME_NAVIGATION_STATE_KEY,
        )
      ) {
        const nextState = { ...(window.history.state || {}) }
        delete nextState[QUERY_SCHEME_NAVIGATION_STATE_KEY]
        window.history.replaceState(nextState, '')
      }
    }
  }

  const selectScheme = async (scheme: QuerySchemeSummary) => {
    const previousPage = queryState.query.value.page
    if (await runtime.applyScheme(scheme)) {
      applyWithSingleRefresh(previousPage)
      return
    }
    $q.notify({
      type: 'warning',
      message:
        runtime.issues.value[0]?.message ||
        runtime.error.value ||
        t('ui.theProgramIsCurrentlyUnavailable'),
    })
  }

  const applyPreset = (payload: QuerySchemePayloadV1) => {
    const previousPage = queryState.query.value.page
    runtime.applyPreset(payload)
    applyWithSingleRefresh(previousPage)
  }

  const restoreCurrent = () => {
    const previousPage = queryState.query.value.page
    if (runtime.restoreCurrentScheme()) applyWithSingleRefresh(previousPage)
  }

  const resetDefault = async () => {
    const previousPage = queryState.query.value.page
    if (await runtime.resetToDefault()) applyWithSingleRefresh(previousPage)
  }

  const openManager = () => {
    void router.push({ name: 'query_scheme_manager' })
  }

  const savePersonal = async (value: { name: string; isDefault: boolean; saveAs: boolean }) => {
    saving.value = true
    try {
      await runtime.savePersonal(value.name, value.isDefault, value.saveAs)
      showSaveDialog.value = false
    } catch {
      // 安全的用户提示由共享HTTP拦截器统一处理，页面不展示后端技术正文。
    } finally {
      saving.value = false
    }
  }

  return {
    runtime,
    showSaveDialog,
    saving,
    initialize,
    runQueryChange,
    selectScheme,
    applyPreset,
    restoreCurrent,
    resetDefault,
    openManager,
    savePersonal,
  }
}

export type QuerySchemePageController<TQuery extends Query> = ReturnType<
  typeof useQuerySchemePage<TQuery>
>
