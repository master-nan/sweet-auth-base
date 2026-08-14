import { computed, ref, toValue, type MaybeRefOrGetter } from 'vue'
import {
  useTableApi,
  type RuntimeTableMetadata,
  type TableField,
} from 'src/api/services/sys-table'

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
      metadataError.value = '缺少表编码'
      return null
    }
    metadataLoading.value = true
    metadataError.value = ''
    try {
      const response = options.loader
        ? await options.loader(code)
        : await tableApi.queryTableByCode(code)
      if (!response.success || !response.data) {
        throw new Error(response.message || '元数据加载失败')
      }
      metadata.value = response.data
      return response.data
    } catch (error) {
      metadata.value = null
      metadataError.value = error instanceof Error ? error.message : '元数据加载失败'
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
    advancedSearchFields,
    formFields,
    detailFields,
    loadMetadata,
  }
}
