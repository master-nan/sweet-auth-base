import { ref } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import { useQuerySchemes, type SchemeAwareQueryState } from 'src/composables/query-schemes'
import type { QuerySchemePayloadV1, QuerySchemeSummary } from 'src/modules/query-scheme/types'
import type { Query } from 'src/types/global'

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

  const initialize = async () => {
    const requestedID = Number(route.query.query_scheme_id)
    return runtime.initialize(
      Number.isSafeInteger(requestedID) && requestedID > 0 ? requestedID : undefined,
    )
  }

  const selectScheme = async (scheme: QuerySchemeSummary) => {
    const previousPage = queryState.query.value.page
    if (await runtime.applyScheme(scheme)) {
      applyWithSingleRefresh(previousPage)
      return
    }
    $q.notify({
      type: 'warning',
      message: runtime.issues.value[0]?.message || runtime.error.value || '该方案当前不可用',
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
      // The shared HTTP interceptor owns the safe user-facing error message.
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
