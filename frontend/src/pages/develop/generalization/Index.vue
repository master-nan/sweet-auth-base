<template>
  <base-content class="q-pa-sm">
    <master-detail-page
      v-if="isMasterDetailEnabled"
      :mode="masterDetailMode"
      :master-title="currentTable?.table_name || '主表数据'"
      :master-subtitle="currentTable?.table_code || ''"
      :detail-title="detailTable?.table_name || '子表数据'"
      :detail-subtitle="detailRelationLabel"
      :master-width="masterDetailMasterWidth"
      min-height="calc(100vh - 132px)"
    >
      <template #master-actions>
        <q-btn
          v-for="btn in topButtons"
          :key="btn.id || btn.code"
          v-bind="menuButtonDisplayProps(btn)"
          :color="btn.color || 'primary'"
          :disable="loading || isButtonDisabled(btn)"
          @click="handleMenuButtonClick(btn)"
        />
        <q-separator
          v-if="topButtons.length && masterDetailMode !== MasterDetailDisplayMode.SUMMARY"
          vertical
          inset
        />
        <table-column-selector
          v-if="masterDetailMode !== MasterDetailDisplayMode.SUMMARY"
          v-model="visibleColumns"
          :columns="columns"
        />
        <q-btn round flat icon="refresh" color="primary" :disable="loading" @click="fetchData">
          <q-tooltip>刷新主表</q-tooltip>
        </q-btn>
      </template>

      <template #master-toolbar>
        <div class="generalization-md-toolbar">
          <q-input
            dense
            outlined
            debounce="300"
            v-model="query.quick_query!.keyword"
            placeholder="搜索主表关键词"
            class="generalization-md-search"
          >
            <template v-slot:append>
              <q-icon name="search" />
            </template>
          </q-input>
          <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
          <q-btn
            outline
            icon="tune"
            color="primary"
            :aria-label="
              hasAppliedAdvancedFilters
                ? `高级查询，已启用 ${activeFilterCount} 个条件`
                : '高级查询'
            "
            @click="showAdvancedQuery = true"
          >
            <q-badge v-if="activeFilterCount > 0" floating color="red">{{
              activeFilterCount
            }}</q-badge>
            <q-tooltip>{{
              hasAppliedAdvancedFilters
                ? `高级查询，已启用 ${activeFilterCount} 个条件`
                : '高级查询'
            }}</q-tooltip>
          </q-btn>
        </div>
      </template>

      <template #master-content>
        <q-table
          v-if="masterDetailMode !== MasterDetailDisplayMode.SUMMARY"
          class="fit sticky-header-table generalization-md-table"
          color="primary"
          flat
          bordered
          separator="cell"
          :rows="rows"
          :columns="columns"
          :visible-columns="visibleColumns"
          row-key="id"
          :loading="loading"
          :pagination="{ rowsPerPage: 0 }"
          hide-bottom
          @row-click="(_, row) => selectMasterRow(row)"
        >
          <template
            v-for="field in fileListFields"
            :key="`master-file-${field.field_code}`"
            #[`body-cell-${field.field_code}`]="props"
          >
            <q-td :props="props">
              <file-display
                :model-value="props.row[field.field_code]"
                :table-code="currentTable?.table_code || query.table_code || ''"
                :record-id="props.row.id"
                :menu-id="resolveMenuId() || 0"
                dense
              />
            </q-td>
          </template>

          <template v-slot:body-cell-actions="props">
            <q-td :props="props" class="q-gutter-xs">
              <q-btn
                v-for="btn in lineButtons"
                :key="btn.id || btn.code"
                flat
                v-bind="menuButtonDisplayProps(btn)"
                :color="btn.color || 'primary'"
                size="sm"
                :data-testid="
                  btn.event_action === SysMenuButtonEventAction.DETAIL
                    ? 'generalization-open-detail'
                    : undefined
                "
                :disable="loading || isButtonDisabled(btn, props.row)"
                @click.stop="handleMenuButtonClick(btn, props.row)"
              >
                <q-tooltip>{{ btn.name }}</q-tooltip>
              </q-btn>
            </q-td>
          </template>

          <template v-slot:no-data>
            <div class="full-width row flex-center q-pa-lg">
              <div class="column items-center">
                <q-icon :name="loadError ? 'cloud_off' : 'inbox'" color="grey-5" size="48px" />
                <div v-if="loadError" class="text-negative q-mb-sm">
                  {{ loadErrorMessage || t('generalization.loadFailed') }}
                </div>
                <div v-else class="text-grey-7 q-mb-sm">{{ t('generalization.noData') }}</div>
                <q-btn outline color="primary" size="sm" :disable="loading" @click="fetchData">
                  {{ t('generalization.retry') }}
                </q-btn>
              </div>
            </div>
          </template>
        </q-table>

        <q-scroll-area v-else class="fit generalization-master-scroll">
          <div v-if="loadError" class="generalization-empty-state">
            <q-icon name="cloud_off" color="grey-5" size="48px" />
            <div class="text-negative q-mb-sm">
              {{ loadErrorMessage || t('generalization.loadFailed') }}
            </div>
            <q-btn outline color="primary" size="sm" :disable="loading" @click="fetchData">
              {{ t('generalization.retry') }}
            </q-btn>
          </div>
          <div v-else-if="rows.length === 0" class="generalization-empty-state">
            <q-icon name="inbox" color="grey-5" size="48px" />
            <div class="text-grey-7 q-mb-sm">{{ t('generalization.noData') }}</div>
            <q-btn outline color="primary" size="sm" :disable="loading" @click="fetchData">
              {{ t('generalization.retry') }}
            </q-btn>
          </div>
          <div v-else class="generalization-master-list">
            <div
              v-for="row in rows"
              :key="row.id"
              class="generalization-master-item"
              :class="{ 'generalization-master-item--active': selectedMasterRow?.id === row.id }"
              @click="selectMasterRow(row)"
            >
              <div class="generalization-master-main">
                <div class="generalization-master-title">{{ getMasterRowTitle(row) }}</div>
                <div class="generalization-master-code">{{ getMasterRowSubtitle(row) }}</div>
              </div>
              <div v-if="lineButtons.length" class="generalization-master-actions">
                <q-btn
                  v-for="btn in lineButtons"
                  :key="btn.id || btn.code"
                  flat
                  dense
                  v-bind="menuButtonDisplayProps(btn)"
                  :color="btn.color || 'primary'"
                  size="sm"
                  :data-testid="
                    btn.event_action === SysMenuButtonEventAction.DETAIL
                      ? 'generalization-open-detail'
                      : undefined
                  "
                  :disable="loading || isButtonDisabled(btn, row)"
                  @click.stop="handleMenuButtonClick(btn, row)"
                >
                  <q-tooltip>{{ btn.name }}</q-tooltip>
                </q-btn>
              </div>
              <div v-if="masterSummaryMetaFields.length" class="generalization-master-meta">
                <span
                  v-for="field in masterSummaryMetaFields"
                  :key="field.field_code"
                  class="generalization-master-meta-item"
                >
                  <span class="generalization-master-meta-label">{{ field.field_name }}</span>
                  <span>{{ formatFieldValue(row, field, relationLookups) }}</span>
                </span>
              </div>
            </div>
          </div>
        </q-scroll-area>
      </template>

      <template #master-footer>
        <div class="generalization-md-footer">
          <div class="row q-gutter-xs">
            <q-btn
              v-for="btn in bottomButtons"
              :key="btn.id || btn.code"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              :disable="loading || isButtonDisabled(btn)"
              @click="handleMenuButtonClick(btn)"
            />
          </div>
          <q-space />
          <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
        </div>
      </template>

      <template #detail-context>
        <div class="generalization-detail-context">
          <div class="generalization-detail-title-wrap">
            <div class="generalization-detail-title">
              {{ detailTable?.table_name || '子表数据' }}
            </div>
            <div class="generalization-detail-subtitle">
              {{ selectedMasterRow ? getMasterRowTitle(selectedMasterRow) : '请选择左侧主表数据' }}
            </div>
          </div>
          <q-space />
          <div class="generalization-detail-relation">{{ detailRelationLabel }}</div>
        </div>
      </template>

      <template #detail-toolbar>
        <div class="generalization-md-toolbar generalization-detail-toolbar">
          <q-btn
            v-if="detailCanWrite"
            color="primary"
            icon="add"
            label="新增子数据"
            :disable="loading || detailLoading || !selectedMasterRow"
            @click="openDetailAddDialog"
          />
          <q-input
            dense
            outlined
            debounce="300"
            v-model="detailQuery.quick_query!.keyword"
            placeholder="搜索子表关键词"
            class="generalization-md-search"
            @keyup.enter="handleDetailSearch"
          >
            <template v-slot:append>
              <q-icon name="search" />
            </template>
          </q-input>
          <q-btn
            outline
            color="primary"
            icon="search"
            label="搜索"
            :disable="detailLoading || !selectedMasterRow"
            @click="handleDetailSearch"
          />
          <q-btn
            round
            flat
            icon="refresh"
            color="primary"
            :disable="detailLoading || !selectedMasterRow"
            @click="fetchDetailData"
          >
            <q-tooltip>刷新子表</q-tooltip>
          </q-btn>
        </div>
      </template>

      <template #detail-content>
        <q-table
          class="fit sticky-header-table generalization-md-table"
          color="primary"
          flat
          bordered
          separator="cell"
          :rows="detailRows"
          :columns="detailColumns"
          :visible-columns="detailVisibleColumns"
          row-key="id"
          :loading="detailLoading"
          :pagination="{ rowsPerPage: 0 }"
          hide-bottom
        >
          <template
            v-for="field in detailFileListFields"
            :key="`detail-file-${field.field_code}`"
            #[`body-cell-${field.field_code}`]="props"
          >
            <q-td :props="props">
              <file-display
                :model-value="props.row[field.field_code]"
                :table-code="detailTable?.table_code || ''"
                :record-id="props.row.id"
                dense
              />
            </q-td>
          </template>

          <template v-slot:body-cell-actions="props">
            <q-td :props="props" class="q-gutter-xs">
              <q-btn
                flat
                dense
                round
                icon="edit"
                color="primary"
                size="sm"
                :disable="detailLoading"
                @click="openDetailEditDialog(props.row)"
              >
                <q-tooltip>编辑子数据</q-tooltip>
              </q-btn>
              <q-btn
                flat
                dense
                round
                icon="delete"
                color="negative"
                size="sm"
                :disable="detailLoading"
                @click="confirmDetailDelete(props.row)"
              >
                <q-tooltip>删除子数据</q-tooltip>
              </q-btn>
            </q-td>
          </template>

          <template v-slot:no-data>
            <div class="full-width row flex-center q-pa-lg">
              <div class="column items-center">
                <q-icon
                  :name="detailLoadError ? 'cloud_off' : 'inbox'"
                  color="grey-5"
                  size="48px"
                />
                <div v-if="detailLoadError" class="text-negative q-mb-sm">
                  {{ detailLoadErrorMessage || t('generalization.loadFailed') }}
                </div>
                <div v-else class="text-grey-7 q-mb-sm">
                  {{ selectedMasterRow ? t('generalization.noData') : '请选择主表数据' }}
                </div>
                <q-btn
                  outline
                  color="primary"
                  size="sm"
                  :disable="detailLoading || !selectedMasterRow"
                  @click="fetchDetailData"
                >
                  {{ t('generalization.retry') }}
                </q-btn>
              </div>
            </div>
          </template>
        </q-table>
      </template>

      <template #detail-footer>
        <div class="generalization-md-footer">
          <q-space />
          <table-pagination
            v-model:page="detailQuery.page"
            v-model:pageSize="detailQuery.num"
            :total="detailTotal"
          />
        </div>
      </template>
    </master-detail-page>

    <q-table
      v-else
      class="fit sticky-header-table"
      color="primary"
      selection="multiple"
      v-model:selected="selected"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat
      bordered
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      row-key="id"
      v-model:pagination="pagination"
      :loading="loading"
    >
      <template
        v-for="field in fileListFields"
        :key="`file-${field.field_code}`"
        #[`body-cell-${field.field_code}`]="props"
      >
        <q-td :props="props">
          <file-display
            :model-value="props.row[field.field_code]"
            :table-code="currentTable?.table_code || query.table_code || ''"
            :record-id="props.row.id"
            :menu-id="resolveMenuId() || 0"
            dense
          />
        </q-td>
      </template>

      <template v-slot:top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #quick-search>
            <q-input
              dense
              outlined
              debounce="300"
              v-model="query.quick_query!.keyword"
              placeholder="搜索关键词"
            >
              <template v-slot:append>
                <q-icon name="search" />
              </template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
            <q-btn
              outline
              icon="tune"
              color="primary"
              class="q-ml-xs"
              :aria-label="
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
              "
              @click="showAdvancedQuery = true"
            >
              <q-badge v-if="activeFilterCount > 0" floating color="red">{{
                activeFilterCount
              }}</q-badge>
              <q-tooltip>{{
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
              }}</q-tooltip>
            </q-btn>
          </template>

          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>

          <template #right-actions>
            <q-btn
              v-for="btn in topButtons"
              :key="btn.id || btn.code"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              :disable="loading || isButtonDisabled(btn)"
              @click="handleMenuButtonClick(btn)"
            />
          </template>
        </standard-table-toolbar>
      </template>

      <template v-slot:body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="btn in lineButtons"
            :key="btn.id || btn.code"
            flat
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            size="sm"
            :disable="loading || isButtonDisabled(btn, props.row)"
            @click="handleMenuButtonClick(btn, props.row)"
          >
            <q-tooltip>{{ btn.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template v-slot:bottom>
        <div class="row q-gutter-xs">
          <q-btn
            v-for="btn in bottomButtons"
            :key="btn.id || btn.code"
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            :disable="loading || isButtonDisabled(btn)"
            @click="handleMenuButtonClick(btn)"
          />
        </div>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>

      <template v-slot:no-data>
        <div class="full-width row flex-center q-pa-lg">
          <div class="column items-center">
            <q-icon :name="loadError ? 'cloud_off' : 'inbox'" color="grey-5" size="48px" />
            <div v-if="loadError" class="text-negative q-mb-sm">
              {{ loadErrorMessage || t('generalization.loadFailed') }}
            </div>
            <div v-else class="text-grey-7 q-mb-sm">{{ t('generalization.noData') }}</div>
            <q-btn outline color="primary" size="sm" :disable="loading" @click="fetchData">
              {{ t('generalization.retry') }}
            </q-btn>
          </div>
        </div>
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="table_fields_advanced"
      :menu-id="resolveMenuId()"
      @search="handleAdvancedSearch"
    />

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="formDialogTitle"
      :fields="tableFields"
      :form-buttons="formButtons"
      :menu-id="resolveMenuId()"
      :table-code="currentTable?.table_code || query.table_code || ''"
      :readonly="formReadonly"
      :submit-btn-text="currentEditData?.id ? '保存' : '创建'"
      @submit="handleFormSubmit"
      @button-click="handleFormButtonClick"
    />

    <dynamic-form-dialog
      v-model="showDetailFormDialog"
      :edit-data="detailEditData"
      :title="detailFormDialogTitle"
      :fields="detailTableFields"
      :menu-id="resolveDetailMenuId()"
      :table-code="detailTable?.table_code || ''"
      :readonly="detailFormReadonly"
      :submit-btn-text="detailEditData?.id ? '保存' : '创建'"
      @submit="handleDetailFormSubmit"
    />

    <dynamic-form-dialog
      v-model="showParamsDialog"
      :edit-data="null"
      :title="paramsDialogTitle"
      :fields="paramsFields"
      :menu-id="resolveMenuId()"
      :table-code="currentTable?.table_code || query.table_code || ''"
      submit-btn-text="执行"
      @submit="handleParamsSubmit"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'generalization_page' })

import BaseContent from 'components/BaseContent/BaseContent.vue'
import MasterDetailPage from 'src/components/MasterDetail/MasterDetailPage.vue'
import StandardTableToolbar from 'components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'components/Table/TableColumnSelector.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import FileDisplay from 'src/components/FileUpload/FileDisplay.vue'
import {
  MasterDetailDisplayMode,
  resolveMasterDetailDisplayMode,
} from 'src/components/MasterDetail/types'

import { computed, ref, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Ajv from 'ajv'

import type { Query } from 'src/types/global'
import {
  useTableApi,
  type RuntimeTableMetadata,
  type Table,
  type TableField,
  type TableRelation,
} from 'src/api/services/sys-table'
import { useGeneralizationApi } from 'src/api/services/generalization'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import { useDictStore } from 'src/stores/dict'
import { useUserStore } from 'src/stores/user'
import {
  ExpressionType,
  SysFormOpenMode,
  SysMasterDetailMode,
  SysDetailOpenMode,
  SysMenuButtonEventAction,
  SysMenuButtonPosition,
  SysTableFieldInputType,
  SysTableRelationType,
} from 'src/types/enum'
import { useMenuApi, type MenuButton } from 'src/api/services/sys-menu'
import {
  evaluateButtonDisabled,
  executeButtonAction,
  runBeforeHooks,
  runAfterHooks,
} from 'src/utils/button-handlers'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import {
  buildRelationLookups,
  buildColumnFormat,
  hydrateRelationLookups,
  type LookupMap,
} from 'src/utils/column-format'
import {
  countEffectiveQueryRules,
  hasEffectiveQueryRules,
  sanitizeQueryExpressions,
} from 'src/utils/query-state'
import { findMenuById, findMenuByTableCode, toPositiveMenuId } from 'src/utils/menu-context'
import { isPageButton } from 'src/utils/menu-button'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { parseParamsSchema } from 'src/utils/params-schema'

const $q = useQuasar()
const { confirmDanger } = useConfirmDialog($q)
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const dictStore = useDictStore()
const userStore = useUserStore()

const tableApi = useTableApi()
const generalizationApi = useGeneralizationApi()
const menuApi = useMenuApi()

const rows = ref<Array<Record<string, any>>>([])
const total = ref(0)
const selected = ref<Array<Record<string, any>>>([])

const showAdvancedQuery = ref(false)
const showFormDialog = ref(false)
const showParamsDialog = ref(false)
const formReadonly = ref(false)
const currentEditData = ref<Record<string, any> | null>(null)
const pendingActionButton = ref<MenuButton | null>(null)
const pendingActionRow = ref<Record<string, any> | null>(null)
const paramsFields = ref<TableField[]>([])
const loadError = ref(false)
const loadErrorMessage = ref('')

const columns = ref<QTableProps['columns']>([])
const table_fields_advanced = ref<TableField[]>([])
const visibleColumns = ref<string[]>([])
const tableFields = ref<TableField[]>([])
const currentTable = ref<RuntimeTableMetadata | null>(null)
const menuButtons = ref<MenuButton[]>([])
const relationLookups = ref<Record<string, LookupMap>>({})

const ajv = new Ajv({ allErrors: true })

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

const query = ref<Query>({
  page: 1,
  num: 20,
  order: {
    field: '',
    is_asc: false,
  },
  table_code: '',
  expressions: [],
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const detailRows = ref<Array<Record<string, any>>>([])
const detailTotal = ref(0)
const detailColumns = ref<QTableProps['columns']>([])
const detailVisibleColumns = ref<string[]>([])
const detailTableFields = ref<TableField[]>([])
const detailTable = ref<Table | null>(null)
const detailRelationLookups = ref<Record<string, LookupMap>>({})
const detailLoading = ref(false)
const detailLoadError = ref(false)
const detailLoadErrorMessage = ref('')
const selectedMasterRow = ref<Record<string, any> | null>(null)
const showDetailFormDialog = ref(false)
const detailFormReadonly = ref(false)
const detailEditData = ref<Record<string, any> | null>(null)
const detailQuery = ref<Query>({
  page: 1,
  num: 20,
  order: {
    field: '',
    is_asc: false,
  },
  table_code: '',
  expressions: [],
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref(cloneDeep(emptyAdvancedQuery()))

const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: false,
})

const visibleMenuButtons = computed(() => {
  return menuButtons.value
    .filter((btn) => isPageButton(btn) && !btn.is_hidden)
    .slice()
    .sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
})

const topButtons = computed(() =>
  visibleMenuButtons.value.filter((btn) => btn.position === SysMenuButtonPosition.TOP),
)

const lineButtons = computed(() =>
  visibleMenuButtons.value.filter((btn) => btn.position === SysMenuButtonPosition.LINE),
)

// 根据是否有行按钮动态显示/隐藏操作列
watch(lineButtons, (btns) => {
  const hasActions = visibleColumns.value.includes('actions')
  if (btns.length > 0 && !hasActions) {
    visibleColumns.value.push('actions')
  } else if (btns.length === 0 && hasActions) {
    visibleColumns.value = visibleColumns.value.filter((c) => c !== 'actions')
  }
})

const bottomButtons = computed(() =>
  visibleMenuButtons.value.filter((btn) => btn.position === SysMenuButtonPosition.BOTTOM),
)

const formButtons = computed(() =>
  visibleMenuButtons.value.filter(
    (btn) =>
      btn.position === SysMenuButtonPosition.FORM_TOP ||
      btn.position === SysMenuButtonPosition.FORM_BOTTOM,
  ),
)

const paramsDialogTitle = computed(
  () => pendingActionButton.value?.name || t('generalization.paramsDialogTitle'),
)
const formDialogTitle = computed(() => {
  if (formReadonly.value) return '查看数据'
  return currentEditData.value?.id ? '编辑数据' : '新增数据'
})

const activeMasterRelation = computed<TableRelation | null>(() => {
  const relation = currentTable.value?.table_relations?.find((item) => {
    return (
      item.relation_type === SysTableRelationType.ONE_TO_MANY &&
      Number(item.related_table_id) > 0 &&
      !!item.reference_key &&
      !!item.foreign_key
    )
  })
  return relation || null
})

const masterListFields = computed(() =>
  tableFields.value
    .filter((field) => field.is_list_show)
    .slice()
    .sort((a, b) => (a.sequence || 0) - (b.sequence || 0)),
)

const isFileField = (field: TableField) => field.input_type === SysTableFieldInputType.FILE_PICKER

const fileListFields = computed(() => masterListFields.value.filter(isFileField))

const detailFileListFields = computed(() =>
  detailTableFields.value.filter((field) => field.is_list_show && isFileField(field)),
)

const visibleMasterFieldCount = computed(() => masterListFields.value.length)

const masterDetailMode = computed(() =>
  resolveMasterDetailDisplayMode(
    visibleMasterFieldCount.value,
    currentTable.value?.master_detail_mode || SysMasterDetailMode.AUTO,
  ),
)

const isMasterDetailEnabled = computed(
  () => !!currentTable.value && !!activeMasterRelation.value && !!detailTable.value,
)

const masterDetailMasterWidth = computed(() => {
  if (masterDetailMode.value === MasterDetailDisplayMode.TABLE) return 'minmax(560px, 44%)'
  if (masterDetailMode.value === MasterDetailDisplayMode.STACKED) return 'auto'
  return '380px'
})

const masterTitleField = computed(() => {
  return (
    masterListFields.value.find((field) => /name|title|名称|标题/i.test(field.field_code)) ||
    masterListFields.value[0]
  )
})

const masterSubtitleField = computed(() => {
  const titleCode = masterTitleField.value?.field_code
  return (
    masterListFields.value.find(
      (field) => field.field_code !== titleCode && /code|编码|编号/i.test(field.field_code),
    ) || masterListFields.value.find((field) => field.field_code !== titleCode)
  )
})

const masterSummaryMetaFields = computed(() => {
  const excluded = new Set(
    [masterTitleField.value?.field_code, masterSubtitleField.value?.field_code].filter(Boolean),
  )
  return masterListFields.value.filter((field) => !excluded.has(field.field_code)).slice(0, 3)
})

const detailRelationLabel = computed(() => {
  const relation = activeMasterRelation.value
  if (!relation) return ''
  return `${relation.reference_key} -> ${relation.foreign_key}`
})

const detailFormDialogTitle = computed(() => {
  if (detailFormReadonly.value) return '查看子数据'
  return detailEditData.value?.id ? '编辑子数据' : '新增子数据'
})

const detailCanWrite = computed(() => !!detailTable.value?.table_code)

const initTempQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
  if (!tempAdvancedQuery.value.expressions.length) {
    tempAdvancedQuery.value.expressions = emptyAdvancedQuery().expressions
  }
}

const resetToFirstPageOrFetch = () => {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  fetchData()
}

const handleBasicSearch = () => {
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
}

const handleAdvancedSearch = () => {
  query.value.expressions = sanitizeQueryExpressions(cloneDeep(tempAdvancedQuery.value.expressions))
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
  showAdvancedQuery.value = false
}

const buildTableColumns = (
  fields: TableField[],
  lookups: Record<string, LookupMap>,
  includeActions = true,
) => {
  const nextColumns: QTableProps['columns'] = []
  fields.forEach((field) => {
    if (!field.is_list_show) return
    const column: NonNullable<QTableProps['columns']>[number] = {
      name: field.field_code,
      label: field.field_name,
      field: field.field_code,
      sortable: field.is_sort,
    }
    const fmt = buildColumnFormat(field, {
      getDictLabel: dictStore.getDictLabel,
      relationLookups: lookups,
    })
    if (fmt) {
      column.format = fmt
    }
    nextColumns?.push(column)
  })
  if (includeActions) {
    nextColumns?.push({
      name: 'actions',
      align: 'center',
      label: '操作',
      field: 'actions',
      sortable: false,
    })
  }
  return nextColumns
}

const formatFieldValue = (
  row: Record<string, any>,
  field: TableField | undefined,
  lookups: Record<string, LookupMap>,
) => {
  if (!field) return ''
  const fmt = buildColumnFormat(field, {
    getDictLabel: dictStore.getDictLabel,
    relationLookups: lookups,
  })
  const value = fmt ? fmt(row[field.field_code], row) : row[field.field_code]
  if (value === null || value === undefined || value === '') return '-'
  return String(value)
}

const getMasterRowTitle = (row: Record<string, any>) => {
  return formatFieldValue(row, masterTitleField.value, relationLookups.value) || `#${row.id || ''}`
}

const getMasterRowSubtitle = (row: Record<string, any>) => {
  return formatFieldValue(row, masterSubtitleField.value, relationLookups.value)
}

const resolveDetailMenuId = () => {
  const tableCode = detailTable.value?.table_code || ''
  if (!tableCode) return 0
  return findMenuByTableCode(userStore.menus, tableCode)?.id || 0
}

const buildDetailFilterQuery = () => {
  const relation = activeMasterRelation.value
  const table = detailTable.value
  const masterRow = selectedMasterRow.value
  if (!relation || !table || !masterRow) return null

  const referenceValue = masterRow[relation.reference_key]
  if (referenceValue === undefined || referenceValue === null || referenceValue === '') return null

  const foreignField = detailTableFields.value.find(
    (field) => field.field_code === relation.foreign_key,
  )
  const relationRule: NonNullable<Query['expressions']>[number]['rules'][number] = {
    field: relation.foreign_key,
    expression_type: ExpressionType.EQ,
    value: referenceValue,
  }
  if (foreignField?.field_type) {
    relationRule.type = foreignField.field_type
  }

  const nextQuery = cloneDeep(detailQuery.value)
  nextQuery.table_code = table.table_code
  nextQuery.menu_id = resolveDetailMenuId()
  nextQuery.expressions = [
    {
      rules: [relationRule],
      nested: [],
    },
  ]
  return nextQuery
}

const fetchData = async () => {
  if (!currentTable.value?.table_code) return
  loadError.value = false
  loadErrorMessage.value = ''
  try {
    query.value.menu_id = resolveMenuId() || 0
    query.value.table_code = currentTable.value.table_code
    const res = await generalizationApi.queryGeneralizationByCode(
      currentTable.value.table_code,
      query.value,
    )
    const dataRows = Array.isArray(res.data) ? res.data : []
    total.value = res.total || 0
    relationLookups.value = await hydrateRelationLookups(
      tableFields.value,
      dataRows,
      relationLookups.value,
      resolveMenuId() || 0,
    )
    rows.value = dataRows
    syncMasterSelection(dataRows)
  } catch (error) {
    rows.value = []
    total.value = 0
    selectedMasterRow.value = null
    detailRows.value = []
    detailTotal.value = 0
    loadError.value = true
    loadErrorMessage.value = error instanceof Error ? error.message : t('generalization.loadFailed')
  }
}

const fetchDetailTableMeta = async () => {
  const relation = activeMasterRelation.value
  if (!relation) {
    detailTable.value = null
    detailTableFields.value = []
    detailColumns.value = []
    detailVisibleColumns.value = []
    detailRelationLookups.value = {}
    detailRows.value = []
    detailTotal.value = 0
    return
  }

  if (detailTable.value?.id === relation.related_table_id) return

  const res = await tableApi.queryTableById(relation.related_table_id)
  if (!res.data?.table_fields) {
    detailTable.value = null
    detailTableFields.value = []
    detailColumns.value = []
    detailVisibleColumns.value = []
    return
  }

  detailTable.value = res.data as Table
  detailTableFields.value = res.data.table_fields
  detailQuery.value.table_code = res.data.table_code
  const dictCodes = res.data.table_fields
    .map((field) => field.dict_code)
    .filter((code): code is string => !!code)
  const [, initialRelationLookups] = await Promise.all([
    dictStore.loadDicts(dictCodes),
    buildRelationLookups(res.data.table_fields),
  ])
  detailRelationLookups.value = initialRelationLookups
  detailColumns.value = buildTableColumns(res.data.table_fields, detailRelationLookups.value, true)
  detailVisibleColumns.value = detailColumns.value?.map((column) => column.name) || []
}

const fetchDetailData = async () => {
  const detailRequest = buildDetailFilterQuery()
  if (!detailRequest || !detailTable.value?.table_code) {
    detailRows.value = []
    detailTotal.value = 0
    return
  }

  detailLoading.value = true
  detailLoadError.value = false
  detailLoadErrorMessage.value = ''
  try {
    const res = await generalizationApi.queryGeneralizationByCode(
      detailTable.value.table_code,
      detailRequest,
    )
    const dataRows = Array.isArray(res.data) ? res.data : []
    detailTotal.value = res.total || 0
    detailRelationLookups.value = await hydrateRelationLookups(
      detailTableFields.value,
      dataRows,
      detailRelationLookups.value,
      resolveDetailMenuId(),
    )
    detailRows.value = dataRows
  } catch (error) {
    detailRows.value = []
    detailTotal.value = 0
    detailLoadError.value = true
    detailLoadErrorMessage.value =
      error instanceof Error ? error.message : t('generalization.loadFailed')
  } finally {
    detailLoading.value = false
  }
}

const syncMasterSelection = (dataRows: Array<Record<string, any>>) => {
  if (!isMasterDetailEnabled.value) return
  const currentId = selectedMasterRow.value?.id
  const nextRow = dataRows.find((row) => row.id === currentId) || dataRows[0] || null
  selectedMasterRow.value = nextRow
  selected.value = nextRow ? [nextRow] : []
  if (nextRow) {
    fetchDetailData()
  } else {
    detailRows.value = []
    detailTotal.value = 0
  }
}

const selectMasterRow = (row: Record<string, any>) => {
  if (!row || selectedMasterRow.value?.id === row.id) return
  selectedMasterRow.value = row
  selected.value = [row]
  if (detailQuery.value.page !== 1) {
    detailQuery.value.page = 1
    return
  }
  fetchDetailData()
}

const handleDetailSearch = () => {
  if (detailQuery.value.page !== 1) {
    detailQuery.value.page = 1
    return
  }
  fetchDetailData()
}

const openDetailAddDialog = () => {
  if (!detailCanWrite.value) return
  const relation = activeMasterRelation.value
  const masterRow = selectedMasterRow.value
  if (!relation || !masterRow) return
  detailFormReadonly.value = false
  detailEditData.value = {
    [relation.foreign_key]: masterRow[relation.reference_key],
  }
  showDetailFormDialog.value = true
}

const openDetailEditDialog = (row: Record<string, any>) => {
  if (!detailCanWrite.value) return
  detailFormReadonly.value = false
  detailEditData.value = cloneDeep(row)
  showDetailFormDialog.value = true
}

const confirmDetailDelete = (row: Record<string, any>) => {
  if (!detailCanWrite.value) return
  confirmDanger({
    message: `确定要删除该子数据吗？`,
    loading: detailLoading.value,
    disable: detailLoading.value,
  }).onOk(() => {
    void (async () => {
      if (!detailTable.value?.table_code || !row.id) return
      const result = await generalizationApi.deleteGeneralization({
        id: Number(row.id),
        table_code: detailTable.value.table_code,
        menu_id: resolveDetailMenuId(),
      })
      if (result.success) {
        fetchDetailData()
      }
    })()
  })
}

const handleDetailFormSubmit = async (formPayload: {
  data: Record<string, any>
  isEdit: boolean
  id?: number
}) => {
  if (detailFormReadonly.value) return
  if (!detailTable.value?.table_code) return
  try {
    if (formPayload.isEdit && formPayload.id) {
      const result = await generalizationApi.updateGeneralization({
        id: formPayload.id,
        table_code: detailTable.value.table_code,
        data: formPayload.data,
        menu_id: resolveDetailMenuId(),
      })
      if (result.success) {
        showDetailFormDialog.value = false
        fetchDetailData()
      }
    } else {
      const result = await generalizationApi.createGeneralization({
        table_code: detailTable.value.table_code,
        data: formPayload.data,
        menu_id: resolveDetailMenuId(),
      })
      if (result.success) {
        showDetailFormDialog.value = false
        fetchDetailData()
      }
    }
  } catch (error) {
    console.error('通用子表提交失败', error)
  }
}

const openAddDialog = () => {
  if (shouldOpenRecordFormPage('create')) {
    void openRecordFormPage('create')
    return
  }
  formReadonly.value = false
  currentEditData.value = null
  showFormDialog.value = true
}

const openEditDialog = (row: Record<string, any>) => {
  if (shouldOpenRecordFormPage('edit')) {
    void openRecordFormPage('edit', row)
    return
  }
  formReadonly.value = false
  currentEditData.value = cloneDeep(row)
  showFormDialog.value = true
}

const shouldOpenRecordFormPage = (mode: 'create' | 'edit' | 'copy') => {
  const openMode = currentTable.value?.form_open_mode || SysFormOpenMode.AUTO
  if (openMode === SysFormOpenMode.PAGE) return true
  if (openMode === SysFormOpenMode.DIALOG) return false

  const fields = tableFields.value.filter((field) =>
    mode === 'edit' ? field.is_update_show : field.is_insert_show,
  )
  const hasComplexField = fields.some(
    (field) =>
      [
        SysTableFieldInputType.FILE_PICKER,
        SysTableFieldInputType.JSON_EDITOR,
        SysTableFieldInputType.ARRAY_INPUT,
        SysTableFieldInputType.KEY_VALUE_EDITOR,
        SysTableFieldInputType.CASCADER,
        SysTableFieldInputType.RICH_TEXT,
      ].includes(field.input_type) || Boolean(field.linkage_config),
  )
  return fields.length > 14 || hasComplexField
}

const shouldOpenRecordDetailPage = () => {
  const openMode = currentTable.value?.detail_open_mode || SysDetailOpenMode.AUTO
  return openMode !== SysDetailOpenMode.DIALOG
}

const openRecordFormPage = async (
  mode: 'create' | 'edit' | 'copy',
  row?: Record<string, any> | null,
) => {
  const tableCode = currentTable.value?.table_code || query.value.table_code
  if (!tableCode) {
    $q.notify({ type: 'warning', position: 'top-right', message: '缺少表编码，无法打开表单' })
    return
  }
  if ((mode === 'edit' || mode === 'copy') && !row?.id) {
    $q.notify({ type: 'warning', position: 'top-right', message: '缺少记录ID，无法打开表单' })
    return
  }

  await router.push({
    name: 'record_form',
    params: {
      mode,
      table_code: tableCode,
      ...(row?.id ? { id: row.id } : {}),
    },
    state: {
      recordFormFrom: route.fullPath,
    },
  })
}

const confirmDelete = (row: Record<string, any>) => {
  confirmDanger({
    message: `确定要删除该数据吗？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      if (!currentTable.value?.table_code || !row.id) return
      const result = await generalizationApi.deleteGeneralization({
        id: Number(row.id),
        table_code: currentTable.value.table_code,
        menu_id: resolveMenuId() || 0,
      })
      if (result.success) {
        fetchData()
      }
    })()
  })
}

const handleFormSubmit = async (formPayload: {
  data: Record<string, any>
  isEdit: boolean
  id?: number
}) => {
  if (formReadonly.value) return
  if (!currentTable.value?.table_code) return
  try {
    if (formPayload.isEdit && formPayload.id) {
      const result = await generalizationApi.updateGeneralization({
        id: formPayload.id,
        table_code: currentTable.value.table_code,
        data: formPayload.data,
        menu_id: resolveMenuId() || 0,
      })
      if (result.success) {
        showFormDialog.value = false
        fetchData()
      }
    } else {
      const result = await generalizationApi.createGeneralization({
        table_code: currentTable.value.table_code,
        data: formPayload.data,
        menu_id: resolveMenuId() || 0,
      })
      if (result.success) {
        showFormDialog.value = false
        fetchData()
      }
    }
  } catch (error) {
    console.error('通用表单提交失败', error)
  }
}

const resolveMenuId = () => {
  const candidates = [
    route.meta.menuId,
    findMenuByTableCode(userStore.menus, resolveTableCode())?.id,
  ]
  for (const raw of candidates) {
    const id = toPositiveMenuId(raw)
    if (id > 0) return id
  }
  return 0
}

const fetchMenuButtons = async () => {
  const menuId = resolveMenuId()
  menuButtons.value = []
  if (!menuId) return
  try {
    const myRes = await menuApi.queryMyMenu()
    if (myRes.success && myRes.data) {
      const menu = findMenuById(myRes.data, menuId)
      if (menu?.menu_buttons) {
        menuButtons.value = menu.menu_buttons
        return
      }
    }
  } catch (error) {
    console.warn('获取我的菜单按钮失败', error)
  }
}

const resetPageState = () => {
  rows.value = []
  total.value = 0
  selected.value = []
  selectedMasterRow.value = null
  columns.value = []
  visibleColumns.value = []
  table_fields_advanced.value = []
  tableFields.value = []
  currentTable.value = null
  relationLookups.value = {}
  menuButtons.value = []
  loadError.value = false
  loadErrorMessage.value = ''
  currentEditData.value = null
  formReadonly.value = false
  showFormDialog.value = false
  showParamsDialog.value = false
  pendingActionButton.value = null
  pendingActionRow.value = null
  paramsFields.value = []

  detailRows.value = []
  detailTotal.value = 0
  detailColumns.value = []
  detailVisibleColumns.value = []
  detailTableFields.value = []
  detailTable.value = null
  detailRelationLookups.value = {}
  detailLoadError.value = false
  detailLoadErrorMessage.value = ''
  detailFormReadonly.value = false
  detailEditData.value = null
  showDetailFormDialog.value = false

  query.value.page = 1
  query.value.num = 20
  query.value.order = { field: '', is_asc: false }
  query.value.expressions = []
  query.value.quick_query = { keyword: '' }
  query.value.include_deleted = false

  detailQuery.value.page = 1
  detailQuery.value.num = 20
  detailQuery.value.order = { field: '', is_asc: false }
  detailQuery.value.expressions = []
  detailQuery.value.quick_query = { keyword: '' }
  detailQuery.value.include_deleted = false
}

const getJsonSchema = (schemaText: string) => {
  if (!schemaText) return null
  try {
    const parsed = JSON.parse(schemaText)
    if (parsed && typeof parsed === 'object' && (parsed.type || parsed.properties)) {
      return parsed
    }
    return null
  } catch (error) {
    console.error('解析JSON Schema失败', error)
    return null
  }
}

const validateParamsWithSchema = (schemaText: string, data: Record<string, any>) => {
  const paramFields = parseParamsSchema(schemaText)
  if (paramFields.length > 0) {
    const missingField = paramFields.find((field) => {
      if (field.is_null) return false
      const value = data[field.field_code]
      if (value === undefined || value === null) return true
      if (typeof value === 'string') return value.trim() === ''
      if (Array.isArray(value)) return value.length === 0
      return false
    })
    if (missingField) {
      $q.notify({
        type: 'negative',
        position: 'top-right',
        message: `${missingField.field_name}不能为空`,
      })
      return false
    }
  }

  const schema = getJsonSchema(schemaText)
  if (!schema) return true
  try {
    const validate = ajv.compile(schema)
    const valid = validate(data)
    if (!valid) {
      const message = validate.errors?.[0]?.message || t('generalization.paramsInvalid')
      $q.notify({ type: 'negative', position: 'top-right', message })
    }
    return !!valid
  } catch (error) {
    console.error('参数校验失败', error)
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: t('generalization.paramsInvalid'),
    })
    return false
  }
}

const isButtonDisabled = (button: MenuButton, row?: Record<string, any>) => {
  return evaluateButtonDisabled(button, {
    row: row || null,
    selection: selected.value,
    selectionCount: selected.value.length,
    query: query.value,
    params: {},
  })
}

const executeMenuButtonAction = async (
  button: MenuButton,
  row?: Record<string, any> | null,
  params?: Record<string, any>,
) => {
  const actionName = (button.event_action || '').trim()
  if (!actionName) {
    $q.notify({
      type: 'warning',
      message: `按钮 ${button.name || button.code || ''} 未配置事件动作`,
    })
    return
  }
  const ctx: Record<string, any> = {
    selection: selected.value,
    onCreate: () => openAddDialog(),
    onUpdate: (dataRow?: Record<string, any>) => {
      if (dataRow) {
        openEditDialog(dataRow)
      }
    },
    onDelete: (dataRow?: Record<string, any>) => {
      if (dataRow) {
        confirmDelete(dataRow)
      }
    },
    onRefresh: () => fetchData(),
    onClearSelection: () => {
      selected.value = []
    },
    onCloseDialog: () => {
      showFormDialog.value = false
    },
    onBatchDelete: async (rows: Array<Record<string, any>>) => {
      if (!currentTable.value?.table_code) return
      if (button.api_path) {
        const method = (button.http_method || 'DELETE').toUpperCase()
        await generalizationApi.executeRuntimeAction({
          path: button.api_path,
          method,
          payload: {
            table_code: currentTable.value.table_code,
            ids: rows.map((item) => Number(item.id)).filter((id) => Number.isFinite(id) && id > 0),
            menu_id: resolveMenuId() || 0,
          },
        })
        selected.value = []
        $q.notify({
          type: 'positive',
          position: 'top-right',
          message: t('generalization.actionSuccess'),
        })
        fetchData()
        return
      }
      for (const r of rows) {
        if (r.id) {
          await generalizationApi.deleteGeneralization({
            id: Number(r.id),
            table_code: currentTable.value.table_code,
            menu_id: resolveMenuId() || 0,
          })
        }
      }
      selected.value = []
      $q.notify({
        type: 'positive',
        position: 'top-right',
        message: t('generalization.actionSuccess'),
      })
      fetchData()
    },
    onCopy: (sourceRow: Record<string, any>) => {
      if (shouldOpenRecordFormPage('copy')) {
        void openRecordFormPage('copy', sourceRow)
        return
      }
      const copied = cloneDeep(sourceRow)
      delete copied.id
      delete copied.created_at
      delete copied.updated_at
      formReadonly.value = false
      currentEditData.value = copied
      showFormDialog.value = true
    },
    onExport: async () => {
      if (!button.api_path) {
        // 默认导出：构造 CSV
        const csvRows = [columns.value?.map((c) => c.label).join(',')]
        for (const r of rows.value) {
          csvRows.push(columns.value?.map((c) => String(r[c.name] ?? '')).join(',') || '')
        }
        const blob = new Blob([csvRows.join('\n')], { type: 'text/csv;charset=utf-8;' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `${currentTable.value?.table_code || 'export'}.csv`
        a.click()
        URL.revokeObjectURL(url)
        return
      }
      // 有 api_path 时走后端导出
      const method = (button.http_method || 'POST').toUpperCase()
      const payload = {
        ...query.value,
        page: query.value.page || 1,
        num: query.value.num || pagination.value.rowsPerPage || 10000,
        table_code: currentTable.value?.table_code || '',
        menu_id: resolveMenuId() || 0,
        params,
      }
      const res = await generalizationApi.executeRuntimeAction<Blob>({
        path: button.api_path,
        method,
        responseType: 'blob',
        payload,
      })
      const blob = new Blob([res.data])
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${currentTable.value?.table_code || 'export'}.csv`
      a.click()
      URL.revokeObjectURL(url)
    },
    onNavigate: async (path: string) => {
      const target = button.api_path || path
      if (target) {
        await router.push(target)
      }
    },
    onOpenDetail: (dataRow: Record<string, any>) => {
      openRecordDetail(dataRow)
    },
    onCustom: async () => {
      if (!button.api_path) {
        $q.notify({
          type: 'warning',
          position: 'top-right',
          message: t('generalization.apiPathMissing'),
        })
        return
      }
      const method = (button.http_method || 'POST').toUpperCase()
      const payload = {
        table_code: currentTable.value?.table_code,
        row,
        selection: selected.value,
        params,
      }
      const res = await generalizationApi.executeRuntimeAction<{
        success?: boolean
      }>({
        path: button.api_path,
        method,
        payload,
      })
      if (res?.data?.success) {
        await fetchData()
      }
    },
  }

  if (currentTable.value?.table_code) {
    ctx.table_code = currentTable.value.table_code
  }

  if (row) {
    ctx.row = row
  }

  if (params) {
    ctx.params = params
  }

  // 执行 before hooks，返回 false 则中止
  const shouldProceed = await runBeforeHooks(button.before_hooks, ctx)
  if (!shouldProceed) {
    $q.notify({ type: 'warning', position: 'top-right', message: t('generalization.hookAborted') })
    return
  }

  await executeButtonAction(actionName, ctx)

  // 执行 after hooks
  await runAfterHooks(button.after_hooks, ctx)
}

const handleMenuButtonClick = (button: MenuButton, row?: Record<string, any>) => {
  if (isButtonDisabled(button, row)) return

  const proceed = () => {
    if (button.params_schema) {
      const fields = parseParamsSchema(button.params_schema)
      if (fields.length > 0) {
        paramsFields.value = fields
        pendingActionButton.value = button
        pendingActionRow.value = row || null
        showParamsDialog.value = true
        return
      }
    }
    void executeMenuButtonAction(button, row)
  }

  if (button.confirm_text) {
    $q.dialog({
      title: t('generalization.confirmTitle'),
      message: button.confirm_text,
      persistent: true,
      ok: {
        label: t('generalization.confirmOk'),
        color: 'primary',
      },
      cancel: {
        label: t('generalization.confirmCancel'),
        color: 'grey-7',
        flat: true,
      },
    }).onOk(() => proceed())
    return
  }

  proceed()
}

const handleParamsSubmit = async (formPayload: { data: Record<string, any> }) => {
  const button = pendingActionButton.value
  if (!button) return
  if (!validateParamsWithSchema(button.params_schema, formPayload.data)) {
    return
  }
  const row = pendingActionRow.value
  showParamsDialog.value = false
  await executeMenuButtonAction(button, row, formPayload.data)
  pendingActionButton.value = null
  pendingActionRow.value = null
  paramsFields.value = []
}

const openRecordDetail = (dataRow: Record<string, any>) => {
  if (shouldOpenRecordDetailPage()) {
    void openRecordDetailTab(dataRow)
    return
  }
  formReadonly.value = true
  currentEditData.value = cloneDeep(dataRow)
  showFormDialog.value = true
}

const openRecordDetailTab = async (dataRow: Record<string, any>) => {
  const id = dataRow.id
  const tableCode = currentTable.value?.table_code || query.value.table_code
  if (!id || !tableCode) {
    $q.notify({
      type: 'warning',
      position: 'top-right',
      message: '缺少记录ID或表编码，无法打开详情',
    })
    return
  }

  await router.push({
    name: 'record_detail',
    params: {
      source: 'generalization',
      table_code: tableCode,
      id,
    },
  })
}

const handleFormButtonClick = (button: MenuButton, formData: Record<string, any>) => {
  // 表单内按钮点击，将当前表单数据作为 row 上下文传递
  handleMenuButtonClick(button, { ...formData, ...currentEditData.value })
}

const fetchTableMeta = async (tableCode: string) => {
  const res = await tableApi.queryRuntimeTableByCode(tableCode)
  if (res.data && res.data.table_fields) {
    currentTable.value = res.data
    tableFields.value = res.data.table_fields
    const dictCodes = res.data.table_fields
      .map((field) => field.dict_code)
      .filter((code): code is string => !!code)
    // 并行加载字典和关联表查找映射
    const [, initialRelationLookups] = await Promise.all([
      dictStore.loadDicts(dictCodes),
      buildRelationLookups(res.data.table_fields),
    ])
    relationLookups.value = initialRelationLookups
    columns.value = buildTableColumns(res.data.table_fields, relationLookups.value, true)
    table_fields_advanced.value = []
    res.data.table_fields.forEach((field) => {
      if (field.is_advanced_search) {
        table_fields_advanced.value.push(field)
      }
    })
    visibleColumns.value = columns.value!.map((column) => column.name)
    await fetchDetailTableMeta()
    await fetchData()
  }
}

const resolveTableCode = () => {
  const code = (route.meta.tableCode ||
    route.params.table_code ||
    route.query.table_code ||
    '') as string
  return String(code || '').trim()
}

const initPage = () => {
  const code = resolveTableCode()
  resetPageState()
  if (!code) {
    $q.notify({
      type: 'warning',
      position: 'top-right',
      message: t('generalization.missingTableCode'),
    })
    return
  }
  query.value.table_code = code
  query.value.menu_id = resolveMenuId() || 0
  fetchTableMeta(code)
  fetchMenuButtons()
}

const initialized = ref(false)
const routeTableKey = computed(() => resolveTableCode())

onMounted(() => {
  initPage()
  initialized.value = true
})

watch(routeTableKey, (nextKey, prevKey) => {
  if (!initialized.value || nextKey === prevKey) return
  initPage()
})

watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    if (!initialized.value) return
    pagination.value.page = page
    fetchData()
  },
)

watch(
  () => [detailQuery.value.page, detailQuery.value.num] as const,
  () => {
    if (!initialized.value || !isMasterDetailEnabled.value) return
    fetchDetailData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], [prevSortBy, prevDescending]) => {
    if (!initialized.value) return
    if (sortBy === prevSortBy && descending === prevDescending) return

    query.value.order = query.value.order ?? { field: '', is_asc: false }
    query.value.order.field = sortBy || ''
    query.value.order.is_asc = sortBy ? !descending : false

    if (query.value.page !== 1) {
      query.value.page = 1
      return
    }

    fetchData()
  },
)

watch(
  () => showAdvancedQuery.value,
  (isOpen) => {
    if (isOpen) {
      initTempQuery()
    }
  },
)
</script>

<style scoped lang="scss">
.generalization-md-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid #e3e8f2;
  background: #ffffff;
}

.generalization-md-search {
  min-width: 220px;
  flex: 1 1 280px;
}

.generalization-md-table {
  border-right: 0;
  border-bottom: 0;
  border-left: 0;
  border-radius: 0;
}

.generalization-master-scroll {
  height: 100%;
}

.generalization-master-list {
  display: flex;
  flex-direction: column;
}

.generalization-master-item {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 6px 8px;
  padding: 13px 14px 12px 18px;
  border-bottom: 1px solid #e7ebf3;
  color: #172033;
  cursor: pointer;
  transition:
    background-color 0.16s ease,
    box-shadow 0.16s ease;

  &::before {
    position: absolute;
    inset: 0 auto 0 0;
    width: 4px;
    content: '';
    background: transparent;
  }

  &:hover {
    background: #f8f9ff;
  }
}

.generalization-master-item--active {
  background: #f1f0ff;

  &::before {
    background: var(--q-primary);
  }
}

.generalization-master-main {
  min-width: 0;
}

.generalization-master-title {
  overflow: hidden;
  font-size: 14px;
  font-weight: 800;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generalization-master-code {
  margin-top: 3px;
  overflow: hidden;
  color: #69758d;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generalization-master-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.generalization-master-meta {
  grid-column: 1 / -1;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.generalization-master-meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  min-height: 22px;
  padding: 2px 7px;
  overflow: hidden;
  color: #526079;
  font-size: 12px;
  line-height: 1.2;
  background: #f5f7fb;
  border-radius: 6px;
}

.generalization-master-meta-label {
  color: #8a94a8;
}

.generalization-detail-context {
  min-height: 60px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.generalization-detail-title-wrap {
  min-width: 0;
}

.generalization-detail-title {
  overflow: hidden;
  color: #172033;
  font-size: 16px;
  font-weight: 800;
  line-height: 1.3;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generalization-detail-subtitle {
  margin-top: 4px;
  overflow: hidden;
  color: #657189;
  font-size: 12px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.generalization-detail-relation {
  max-width: 360px;
  padding: 5px 9px;
  overflow: hidden;
  color: #64748b;
  font-size: 12px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: #f4f6fb;
  border-radius: 6px;
}

.generalization-detail-toolbar {
  border-top: 0;
}

.generalization-md-footer {
  min-height: 52px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-top: 1px solid #e3e8f2;
  background: #ffffff;
}

.generalization-empty-state {
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
</style>
