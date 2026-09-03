<template>
  <base-content
    class="q-pa-sm dictionary-page"
    :class="{ 'dictionary-page--dark': $q.dark.isActive }"
  >
    <div class="dictionary-workspace">
      <master-detail-page
        :mode="SysMasterDetailMode.SUMMARY"
        :master-title="t('ui.dictionaryType')"
        master-width="372px"
        min-width="980px"
        min-height="calc(100vh - 142px)"
      >
        <template #master-actions>
          <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
            <template #right-actions>
              <q-btn
                v-for="btn in masterTopButtons"
                :key="btn.id"
                v-bind="menuButtonDisplayProps(btn, { label: masterButtonLabel(btn) })"
                dense
                color="primary"
                :disable="loading"
                @click="handleButtonClick(btn)"
              >
                <q-tooltip>{{ btn.name }}</q-tooltip>
              </q-btn>
            </template>
          </standard-table-toolbar>
        </template>

        <template #master-toolbar>
          <div class="dictionary-master-toolbar">
            <div class="row q-gutter-sm no-wrap">
              <q-input
                class="master-search"
                dense
                outlined
                debounce="300"
                v-model="keyword"
                :placeholder="t('ui.searchForDictionaryNameEncoding')"
              >
                <template v-slot:append>
                  <q-icon name="search" />
                </template>
              </q-input>
              <q-btn
                color="primary"
                :label="t('ui.search')"
                :disable="loading"
                @click="handleBasicSearch"
              />
            </div>
          </div>
        </template>

        <template #master-content>
          <q-list separator class="dictionary-list">
            <q-item
              v-for="dict in rows"
              :key="dict.id"
              class="dictionary-row"
              clickable
              :active="currentDict && currentDict.id === dict.id"
              active-class="dictionary-row--active"
              @click="selectDict(dict)"
            >
              <q-item-section>
                <q-item-label class="dictionary-name">{{ dict.dict_name }}</q-item-label>
                <q-item-label caption class="dictionary-code">
                  {{ dict.dict_code }}
                </q-item-label>
              </q-item-section>

              <q-item-section v-if="masterLineButtons.length" side>
                <div class="row q-gutter-xs dictionary-row-actions">
                  <q-btn
                    v-for="btn in masterLineButtons"
                    :key="btn.id"
                    v-bind="menuButtonDisplayProps(btn, { label: masterButtonLabel(btn) })"
                    flat
                    :color="btn.color || 'primary'"
                    size="sm"
                    :disable="loading"
                    @click.stop="handleButtonClick(btn, dict)"
                  >
                    <q-tooltip>{{ masterButtonLabel(btn) }}</q-tooltip>
                  </q-btn>
                </div>
              </q-item-section>
            </q-item>

            <q-item v-if="rows.length === 0 && !loading">
              <q-item-section class="text-center text-grey">
                {{ listEmptyMessage }}
              </q-item-section>
            </q-item>

            <q-item v-if="loading">
              <q-item-section class="text-center">
                <q-spinner color="primary" size="24px" />
                <div class="q-mt-sm text-grey">{{ t('ui.loading') }}</div>
              </q-item-section>
            </q-item>
          </q-list>
        </template>

        <template #master-footer>
          <div v-if="total" class="dictionary-master-footer">
            <table-pagination
              v-model:page="query.page"
              v-model:pageSize="query.num"
              :total="total"
            />
          </div>
        </template>

        <template #detail-context>
          <div class="dictionary-detail-context">
            <div class="dictionary-context-card">
              <div class="dictionary-context-icon">
                {{ currentDict ? currentDict.dict_name.slice(0, 1) : t('ui.word') }}
              </div>
              <div class="dictionary-context-copy">
                <div class="dictionary-context-title">
                  {{ currentDict ? currentDict.dict_name : t('ui.selectTheDictionaryType') }}
                </div>
                <div class="dictionary-context-code">
                  {{
                    currentDict
                      ? currentDict.dict_code
                      : t('ui.manageDictionariesAfterLeftSelection')
                  }}
                </div>
              </div>
            </div>
          </div>
        </template>

        <template #detail-toolbar>
          <standard-table-toolbar
            class="dictionary-detail-toolbar"
            :refreshing="itemsLoading"
            :disabled="!currentDict"
            @refresh="refreshDictItems"
          >
            <template #quick-search>
              <q-input
                class="detail-search"
                dense
                outlined
                debounce="300"
                v-model="itemSearchText"
                :placeholder="t('ui.searchDictionaryEntriesForEncodingValues')"
                :disable="!currentDict"
              >
                <template v-slot:append>
                  <q-icon name="search" />
                </template>
              </q-input>
            </template>
            <template #right-actions>
              <q-btn
                v-for="btn in detailTopButtons"
                :key="btn.id"
                v-bind="menuButtonDisplayProps(btn, { label: detailButtonLabel(btn) })"
                dense
                color="primary"
                :disable="!currentDict"
                @click="handleButtonClick(btn)"
              >
                <q-tooltip>{{ btn.name }}</q-tooltip>
              </q-btn>
            </template>
          </standard-table-toolbar>
        </template>

        <template #detail-content>
          <q-table
            class="fit sticky-header-table dictionary-item-table"
            :rows="filteredDictItems"
            :columns="itemColumns"
            row-key="id"
            flat
            :dark="$q.dark.isActive"
            :loading="itemsLoading"
            :no-data-label="
              !currentDict
                ? t('ui.pleaseSelectTheLeftDictionaryFirst')
                : itemsLoading
                  ? t('ui.loading')
                  : t('ui.noDictionaryEntryForNow')
            "
            :pagination="{ rowsPerPage: 0 }"
            hide-pagination
            virtual-scroll
            :virtual-scroll-item-size="48"
            :virtual-scroll-sticky-size-start="48"
          >
            <template v-slot:body-cell-actions="props">
              <q-td :props="props" class="dictionary-actions-cell">
                <q-btn
                  v-for="btn in detailLineButtons"
                  :key="btn.id"
                  v-bind="menuButtonDisplayProps(btn, { label: detailButtonLabel(btn) })"
                  flat
                  :color="btn.color || 'primary'"
                  size="sm"
                  :disable="!currentDict || itemsLoading"
                  @click="handleButtonClick(btn, props.row)"
                >
                  <q-tooltip>{{ detailButtonLabel(btn) }}</q-tooltip>
                </q-btn>
              </q-td>
            </template>
          </q-table>
        </template>
      </master-detail-page>
    </div>

    <!-- 字典表单对话框 -->
    <dynamic-form-dialog
      v-model="showDictFormDialog"
      :edit-data="currentEditDict"
      :title="currentEditDict ? t('ui.editDictionary') : t('ui.addDictionary')"
      :fields="dictFields"
      :submit-btn-text="currentEditDict ? t('ui.save') : t('ui.createRecord')"
      @submit="handleDictFormSubmit"
    />

    <!-- 字典项表单对话框 -->
    <dynamic-form-dialog
      v-model="showDictItemFormDialog"
      :edit-data="currentEditDictItem"
      :title="
        currentEditDictItem && currentEditDictItem.id
          ? t('ui.editDictionaryItems')
          : t('ui.addDictionaryEntry')
      "
      :fields="dictItemFields"
      :submit-btn-text="
        currentEditDictItem && currentEditDictItem.id ? t('ui.save') : t('ui.createRecord')
      "
      @submit="handleDictItemFormSubmit"
    />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'develop_dictionary' })
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import MasterDetailPage from '@/components/MasterDetail/MasterDetailPage.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import {
  type DictItemUpdateReq,
  useDictApi,
  type Dict,
  type DictItem,
} from '@/api/services/sys-dict'
import type { Query } from '@/types/global'
import DynamicFormDialog from '@/components/FormDialog/DynamicFormDialog.vue'
import { useRuntimeTableMetadata } from '@/composables/runtime-table-metadata'
import { useTableQueryState } from '@/composables/table-query-state'
import cloneDeep from 'lodash/cloneDeep'
import { useDictStore } from '@/stores/dict'
import { buildTableColumns } from '@/utils/column-format'
import { useMasterDetailPageButtons } from '@/composables/page-buttons'
import type { MenuButton } from '@/api/services/sys-menu'
import { menuButtonDisplayProps } from '@/utils/menu-button-display'
import { SysMasterDetailMode } from '@/types/enum'
import { useConfirmDialog } from '@/composables/confirm-dialog'
import { resolveTableEmptyMessage } from '@/utils/table-state'

const { t } = useI18n({ useScope: 'global' })

const loading = ref(false)
const loadError = ref('')
const dictStore = useDictStore()

const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const dictApi = useDictApi()
const rows = ref<Dict[]>([])
const total = ref(0)

const {
  master_top_buttons: rawMasterTopButtons,
  master_line_buttons: rawMasterLineButtons,
  detail_top_buttons: rawDetailTopButtons,
  detail_line_buttons: rawDetailLineButtons,
} = useMasterDetailPageButtons('develop_dictionary', (btn) =>
  ['create_item', 'update_item', 'delete_item'].includes(btn.event_action),
)

const action_handlers: Record<string, (row?: any) => void> = {
  create: () => openAddDictDialog(),
  update: (row) => row && openEditDictDialog(row),
  delete: (row) => row && confirmDeleteDict(row),
  create_item: () => openAddDictItemDialog(),
  update_item: (row) => row && openEditDictItemDialog(row),
  delete_item: (row) => row && confirmDeleteDictItem(row),
}

const handleButtonClick = (btn: MenuButton, row?: any) => {
  const handler = action_handlers[btn.event_action]
  if (handler) handler(row)
}

const executableButtons = (buttons: MenuButton[]) =>
  buttons.filter((btn) => !!action_handlers[btn.event_action])

const masterTopButtons = computed(() => executableButtons(rawMasterTopButtons.value))
const masterLineButtons = computed(() => executableButtons(rawMasterLineButtons.value))
const detailTopButtons = computed(() => executableButtons(rawDetailTopButtons.value))
const detailLineButtons = computed(() => executableButtons(rawDetailLineButtons.value))

const masterButtonLabel = (btn: MenuButton) => {
  if (btn.event_action === 'create') return t('ui.addDictionary')
  if (btn.event_action === 'update') return t('ui.editDictionary')
  if (btn.event_action === 'delete') return t('ui.removeDictionary')
  return btn.name
}

const detailButtonLabel = (btn: MenuButton) => {
  if (btn.event_action === 'create_item') return t('ui.addDictionaryEntry')
  if (btn.event_action === 'update_item') return t('ui.editDictionaryItems')
  if (btn.event_action === 'delete_item') return t('ui.removeDictionaryEntry')
  return btn.name
}

const refreshDictItemColumns = () => {
  const visibleItemCols = (rawDictItemColumns.value || [])
    .filter((c) => c.name !== 'dict_id')
    .map((c) => {
      if (c.name === 'actions') {
        return {
          ...c,
          classes: 'dictionary-actions-col',
          headerClasses: 'dictionary-actions-col',
          style: 'width: 96px; min-width: 96px; max-width: 96px;',
          headerStyle: 'width: 96px; min-width: 96px; max-width: 96px;',
        }
      }
      if (c.name === 'item_value') {
        return {
          ...c,
          align: 'center' as const,
          style: 'width: 92px; min-width: 92px; max-width: 120px;',
          headerStyle: 'width: 92px; min-width: 92px; max-width: 120px;',
        }
      }
      return c
    })
  itemColumns.value = detailLineButtons.value.length
    ? visibleItemCols
    : visibleItemCols.filter((c) => c.name !== 'actions')
}

// 当前选中的字典
const currentDict = ref<Dict | null>(null)

// 字典项相关
const dictItems = ref<DictItem[]>([])
const itemsLoading = ref(false)
const itemSearchText = ref('')

// 过滤后的字典项列表
const filteredDictItems = computed(() => {
  if (!itemSearchText.value) return dictItems.value

  const searchText = itemSearchText.value.toLowerCase()
  return dictItems.value.filter((item) => {
    return (
      (item.item_name && item.item_name.toLowerCase().includes(searchText)) ||
      (item.item_value && item.item_value.toString().toLowerCase().includes(searchText))
    )
  })
})

// 表单对话框相关
const showDictFormDialog = ref(false)
const currentEditDict = ref<Dict | null>(null)
const showDictItemFormDialog = ref(false)
const currentEditDictItem = ref<DictItemUpdateReq | null>(null)

// 表格列定义
const itemColumns = ref<QTableProps['columns']>([])
const rawDictItemColumns = ref<QTableProps['columns']>([])

const {
  fields: dictMetadataFields,
  formFields: dictFields,
  loadMetadata: loadDictMetadata,
} = useRuntimeTableMetadata('sys_dict')
const {
  fields: dictItemMetadataFields,
  formFields: runtimeDictItemFields,
  loadMetadata: loadDictItemMetadata,
} = useRuntimeTableMetadata('sys_dict_item')
const dictItemFields = computed(() =>
  runtimeDictItemFields.value.filter((field) => field.field_code !== 'dict_id'),
)

// 默认空查询
const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 20,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})

// 查询参数
const queryState = useTableQueryState<Query>({
  createInitialQuery: () => ({
    page: 1,
    num: 20,
    order: { field: '', is_asc: false },
    table_code: 'sys_dict',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }),
})
const { query, keyword } = queryState
const listEmptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: true,
    error: loadError.value,
    hasQuery: !!keyword.value,
  }),
)

// 获取字典列表数据
const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const res = await dictApi.queryDict(query.value)
    rows.value = res.data
    total.value = res.total || 0
    await syncCurrentDictAfterFetch()
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = t('ui.loadingOfDictionaryListFailed')
  } finally {
    loading.value = false
  }
}

const syncCurrentDictAfterFetch = async () => {
  if (!rows.value.length) {
    clearCurrentSelection()
    return
  }
  if (currentDict.value) {
    const matched = rows.value.find((dict) => dict.id === currentDict.value?.id)
    if (matched) {
      currentDict.value = matched
      return
    }
  }
  const firstDict = rows.value[0]
  if (firstDict) {
    await selectDict(firstDict)
  }
}

const resetToFirstPageOrFetch = () => {
  clearCurrentSelection()
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  void fetchData()
}

const handleBasicSearch = () => {
  const previousPage = query.value.page
  queryState.submitQuickSearch()
  if (previousPage === 1) resetToFirstPageOrFetch()
}

const clearCurrentSelection = () => {
  currentDict.value = null
  dictItems.value = []
  itemSearchText.value = ''
}

// 选择字典
const selectDict = async (dict: Dict) => {
  currentDict.value = dict
  await fetchDictItems(dict.id)
}

// 获取字典项列表
const fetchDictItems = async (dictId: number) => {
  if (!dictId) return

  itemsLoading.value = true
  try {
    const res = await dictApi.queryDictItemsByDictId(dictId)
    dictItems.value = res.data
  } catch (error) {
    console.error('获取字典项列表失败', error)
    dictItems.value = []
  } finally {
    itemsLoading.value = false
  }
}

// 刷新字典项列表
const refreshDictItems = async () => {
  if (currentDict.value) {
    await fetchDictItems(currentDict.value.id)
  }
}

// 打开新增字典对话框
const openAddDictDialog = () => {
  currentEditDict.value = null
  showDictFormDialog.value = true
}

// 打开编辑字典对话框
const openEditDictDialog = (dict: Dict) => {
  currentEditDict.value = cloneDeep(dict)
  showDictFormDialog.value = true
}

// 打开新增字典项对话框
const openAddDictItemDialog = () => {
  if (!currentDict.value) return

  // 重置编辑状态，确保是新增，不要误传字典id作为字典项id
  currentEditDictItem.value = {
    dict_id: currentDict.value.id, // 正确地设置关联的字典ID
    item_name: '',
    item_code: '',
    item_value: '',
    id: 0,
  }
  showDictItemFormDialog.value = true
}

// 打开编辑字典项对话框
const openEditDictItemDialog = (dictItem: DictItem) => {
  // 克隆字典项数据，避免直接修改原对象
  currentEditDictItem.value = cloneDeep({
    ...dictItem,
    dict_id: dictItem.dict_id || currentDict.value!.id, // 确保dict_id有值
  })
  showDictItemFormDialog.value = true
}

// 确认删除字典
const confirmDeleteDict = (dict: Dict) => {
  confirmDanger({
    get message() {
      return t('ui.areYouSureYouWantToDeleteTheDictionaryTheDeletionWill', {
        value1: dict.dict_name,
      })
    },
  }).onOk(() => {
    void (async () => {
      const result = await dictApi.deleteDict(dict.id)
      if (result.success) {
        if (currentDict.value && currentDict.value.id === dict.id) {
          currentDict.value = null
          dictItems.value = []
        }
        fetchData()
        dictStore.clearDict(dict.dict_code)
      }
    })()
  })
}

// 确认删除字典项
const confirmDeleteDictItem = (dictItem: DictItem) => {
  confirmDanger({
    get message() {
      return t('ui.areYouSureYouWantToDeleteTheDictionaryEntry', { value1: dictItem.item_name })
    },
  }).onOk(() => {
    void (async () => {
      const result = await dictApi.deleteDictItem(dictItem.id)
      if (result.success) {
        if (currentDict.value) {
          await fetchDictItems(currentDict.value.id)
          // 清除缓存
          dictStore.clearDict(currentDict.value.dict_code)
        }
      }
    })()
  })
}

// 处理字典表单提交
const handleDictFormSubmit = async (formData: { data: Dict; isEdit: boolean; id?: number }) => {
  if (currentEditDict.value) {
    // 编辑字典
    await dictApi.updateDict({
      ...formData.data,
      id: currentEditDict.value.id,
    })
    // 如果更新的是当前选中的字典，更新当前选中
    if (currentDict.value && currentDict.value.id === currentEditDict.value.id) {
      currentDict.value = {
        ...currentDict.value,
        ...formData.data,
      }
    }
    // 清除缓存
    if (formData.data.dict_code) {
      dictStore.clearDict(formData.data.dict_code)
    }
  } else {
    // 新增字典
    const res = await dictApi.createDict(formData.data)
    // 选中新创建的字典
    if (res.data) {
      await fetchDictItems(res.data! as number)
    }
  }
  // 刷新字典列表
  fetchData()
  showDictFormDialog.value = false
}

// 处理字典项表单提交
const handleDictItemFormSubmit = async (formData: { data: any; isEdit: boolean; id?: number }) => {
  if (!currentDict.value) return

  try {
    // 清晰区分编辑和新增模式
    if (formData.isEdit && formData.id) {
      // 编辑字典项 - 使用传入的id
      await dictApi.updateDictItem({
        ...formData.data,
        id: formData.id,
      })
    } else {
      // 新增字典项 - 确保绑定正确的字典id
      await dictApi.createDictItem({
        ...formData.data,
        dict_id: currentDict.value.id, // 显式设置字典ID，确保关联到当前字典
      })
    }

    // 刷新字典项列表
    await fetchDictItems(currentDict.value.id)
    showDictItemFormDialog.value = false

    // 清除缓存
    dictStore.clearDict(currentDict.value.dict_code)
  } catch (error) {
    console.error('保存字典项失败', error)
  }
}

// 获取表结构信息
const fetchTableFields = async () => {
  try {
    if (await loadDictMetadata()) {
      const dictCodes1 = dictMetadataFields.value
        .map((f) => f.dict_code)
        .filter((c): c is string => !!c)
      await dictStore.loadDicts(dictCodes1)
    }

    if (await loadDictItemMetadata()) {
      const dictCodes2 = dictItemMetadataFields.value
        .map((f) => f.dict_code)
        .filter((c): c is string => !!c)
      await dictStore.loadDicts(dictCodes2)

      const { columns: itemCols } = buildTableColumns(dictItemMetadataFields.value, {
        getDictLabel: dictStore.getDictLabel,
      })
      rawDictItemColumns.value = itemCols
      refreshDictItemColumns()
    }
  } catch (error) {
    console.error('获取表结构信息失败', error)
  }
}

const initialized = ref(false)

onMounted(async () => {
  await fetchTableFields()
  await fetchData()
  initialized.value = true
})

// 监听查询参数变化
watch(
  () => [query.value.page, query.value.num],
  () => {
    if (!initialized.value) return
    clearCurrentSelection()
    fetchData()
  },
)

watch(
  () => detailLineButtons.value.length,
  () => refreshDictItemColumns(),
)
</script>

<style scoped lang="scss">
.dictionary-page {
  --dictionary-page: #f5f7fb;
  --dictionary-surface: #fff;
  --dictionary-border: #e3e8f2;
  --dictionary-text: #172033;
  --dictionary-muted: #657189;
  --dictionary-selected: linear-gradient(90deg, #f3f1ff, #fff 72%);
  background: var(--dictionary-page);
}

.dictionary-page--dark {
  --dictionary-page: var(--app-dark-page);
  --dictionary-surface: var(--app-dark-surface);
  --dictionary-border: var(--app-dark-border);
  --dictionary-text: var(--app-dark-heading);
  --dictionary-muted: var(--app-dark-muted);
  --dictionary-selected: linear-gradient(
    90deg,
    var(--app-primary-soft-strong),
    var(--app-dark-surface) 72%
  );
}

.dictionary-workspace {
  height: calc(100vh - 142px);
  min-height: 0;
  overflow-x: auto;
  overflow-y: hidden;
}

.dictionary-master-toolbar {
  padding: 12px 14px;
  border-bottom: 1px solid var(--dictionary-border);
  background: var(--dictionary-surface);
}

.master-search {
  min-width: 0;
  flex: 1;
}

.dictionary-master-footer {
  min-height: 50px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 8px 10px;
  border-top: 1px solid var(--dictionary-border);
  background: var(--dictionary-surface);
}

.dictionary-list {
  height: 100%;
  overflow: auto;
  background: var(--dictionary-surface);
}

.dictionary-row {
  min-height: 64px;
  padding: 10px 12px 10px 14px;
  border-left: 4px solid transparent;
}

.dictionary-row--active {
  color: var(--dictionary-text);
  border-left-color: var(--q-primary);
  background: var(--dictionary-selected);
}

.dictionary-name {
  color: var(--dictionary-text);
  font-weight: 800;
}

.dictionary-code {
  margin-top: 4px;
  color: var(--dictionary-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.dictionary-row-actions {
  opacity: 0.72;
  transition: opacity 0.16s ease;
}

.dictionary-row:hover .dictionary-row-actions,
.dictionary-row--active .dictionary-row-actions {
  opacity: 1;
}

.dictionary-detail-context {
  display: block;
}

.dictionary-context-card {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--dictionary-border);
  border-radius: 8px;
  background: var(--dictionary-surface);
}

.dictionary-context-icon {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  border-radius: 8px;
  color: #ffffff;
  background: linear-gradient(135deg, var(--q-primary), #2563eb);
  font-weight: 800;
}

.dictionary-context-copy {
  min-width: 0;
}

.dictionary-context-title {
  overflow: hidden;
  color: var(--dictionary-text);
  font-size: 18px;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dictionary-context-code {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  height: 24px;
  margin-top: 6px;
  padding: 0 8px;
  overflow: hidden;
  border: 1px solid rgba(111, 98, 242, 0.24);
  border-radius: 6px;
  color: var(--q-primary);
  background: #f3f1ff;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dictionary-detail-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--dictionary-border);
  background: var(--dictionary-surface);
}

.detail-search {
  min-width: 220px;
  flex: 1;
}

.dictionary-item-table :deep(th),
.dictionary-item-table :deep(td) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dictionary-item-table :deep(.dictionary-actions-col),
.dictionary-actions-cell {
  width: 96px;
  min-width: 96px;
  max-width: 96px;
}

.dictionary-actions-cell {
  padding-right: 8px;
  padding-left: 8px;
  text-align: center;
  white-space: nowrap;
}

.dictionary-actions-cell .q-btn {
  margin: 0 2px;
}

@media (max-width: 560px) {
  .dictionary-detail-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .detail-refresh-btn {
    align-self: flex-end;
  }

  .detail-search {
    width: 100%;
  }
}
</style>
