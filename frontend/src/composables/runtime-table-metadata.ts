import { translate as t } from 'src/i18n/runtime/instance'
import { computed, ref, toValue, type MaybeRefOrGetter } from 'vue'
import { useTableApi, type RuntimeTableMetadata, type TableField } from 'src/api/services/sys-table'

type MetadataLoader = (tableCode: string) => Promise<{
  success: boolean
  message?: string
  data: RuntimeTableMetadata
}>

export interface RuntimeTableMetadataOptions {
  loader?: MetadataLoader
}

const sortFields = (fields: TableField[]) =>
  fields.slice().sort((left, right) => left.sequence - right.sequence)

export function useRuntimeTableMetadata(
  tableCode: MaybeRefOrGetter<string>,
  options: RuntimeTableMetadataOptions = {},
) {
  const tableApi = useTableApi()
  const metadata = ref<RuntimeTableMetadata | null>(null)
  const metadataLoading = ref(false)
  const metadataError = ref('')

  const fields = computed(() => sortFields(metadata.value?.table_fields || []))
  const listFields = computed(() => fields.value.filter((field) => field.is_list_show))
  const quickSearchFields = computed(() => fields.value.filter((field) => field.is_quick_search))
  const quickSearchPlaceholder = computed(() => {
    const labels = Array.from(
      new Set(quickSearchFields.value.map((field) => field.field_name.trim()).filter(Boolean)),
    )
    if (!labels.length) return t('ui.searchKeywords')
    return t('ui.searchFieldSummary', {
      value1: labels.slice(0, 3).join(t('ui.listSeparator')),
      value2: labels.length > 3 ? t('ui.andMore') : '',
    })
  })
  const advancedSearchFields = computed(() =>
    fields.value.filter((field) => field.is_advanced_search),
  )
  const formFields = computed(() =>
    fields.value.filter((field) => field.is_insert_show || field.is_update_show),
  )
  const detailFields = computed(() => fields.value.filter((field) => (field.detail_span || 0) > 0))

  const loadMetadata = async () => {
    const code = toValue(tableCode).trim()
    if (!code) {
      metadata.value = null
      metadataError.value = t('ui.synchronisingFolder')
      return null
    }
    metadataLoading.value = true
    metadataError.value = ''
    try {
      const response = options.loader
        ? await options.loader(code)
        : await tableApi.queryRuntimeTableByCode(code)
      if (!response.success || !response.data) {
        throw new Error(response.message || t('ui.loadingMetadataFailed'))
      }
      metadata.value = response.data
      return response.data
    } catch (error) {
      metadata.value = null
      metadataError.value = error instanceof Error ? error.message : t('ui.loadingMetadataFailed')
      return null
    } finally {
      metadataLoading.value = false
    }
  }

  return {
    metadata,
    metadataLoading,
    metadataError,
    fields,
    listFields,
    quickSearchFields,
    quickSearchPlaceholder,
    advancedSearchFields,
    formFields,
    detailFields,
    loadMetadata,
  }
}
