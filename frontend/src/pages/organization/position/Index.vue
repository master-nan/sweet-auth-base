<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <q-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
      class="fit sticky-header-table"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      :loading="loading"
      v-model:pagination="pagination"
      hide-pagination
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #scheme-selector>
            <query-scheme-selector
              :schemes="schemePage.runtime.schemes.value"
              :current-label="schemePage.runtime.currentLabel.value"
              :loading="schemePage.runtime.loading.value"
              :dirty="queryState.dirty.value"
              :load-error="schemePage.runtime.error.value"
              @select="schemePage.selectScheme"
              @restore-current="schemePage.restoreCurrent"
              @reset-default="schemePage.resetDefault"
              @retry="schemePage.runtime.loadAvailable"
              @manage="schemePage.openManager"
            />
          </template>
          <template #quick-presets>
            <query-quick-presets
              :config="schemePage.runtime.scope.config.value"
              @apply="schemePage.applyPreset"
            />
          </template>
          <template #quick-search>
            <q-input
              v-model="keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索关键词"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="search" />
          </template>
          <template #advanced-trigger>
            <q-btn
              outline
              color="primary"
              icon="tune"
              aria-label="高级查询"
              @click="openAdvancedQuery"
            >
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </template>
          <template #save-scheme>
            <q-btn
              outline
              color="primary"
              icon="bookmark_add"
              label="保存方案"
              @click="schemePage.showSaveDialog.value = true"
            />
          </template>
          <template #column-selector>
            <q-select
              v-model="visibleColumns"
              multiple
              outlined
              dense
              options-dense
              emit-value
              map-options
              :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')"
              :options="columns"
              option-value="name"
              options-cover
            />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-position_type="props">
        <q-td :props="props">
          {{ dictLabel('org_position_type', props.row.position_type) }}
        </q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :color="organizationStatusColor(props.row.status)"
            :label="dictLabel('org_object_status', props.row.status)"
          />
        </q-td>
      </template>
      <template #body-cell-validity="props">
        <q-td :props="props">
          {{ formatOrganizationDate(props.row.valid_from) }}
          <span class="text-grey-6 q-mx-xs">至</span>
          {{ formatOrganizationDate(props.row.valid_to, '长期') }}
        </q-td>
      </template>
      <template #body-cell-is_manager_position="props">
        <q-td :props="props">{{ props.row.is_manager_position ? '是' : '否' }}</q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="button in rowButtons"
            :key="button.id || button.code"
            flat
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            size="sm"
            @click="handleRowAction(button, props.row)"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>
      <template #no-data>
        <div class="full-width row flex-center q-pa-xl text-grey-7">
          {{ emptyMessage }}
        </div>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      v-model:bindings="queryState.bindings.value"
      :fields="advancedFields"
      :source-name="queryState.schemeSource.value?.name || ''"
      :dirty="queryState.dirty.value"
      title="岗位高级查询"
      @search="applyAdvancedQuery"
    />
    <query-scheme-save-dialog
      v-model="schemePage.showSaveDialog.value"
      :source="queryState.schemeSource.value"
      :loading="schemePage.saving.value"
      @save="schemePage.savePersonal"
    />

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      :title="positionDetail?.name || '岗位详情'"
      :subtitle="positionDetail?.code || ''"
      :sections="detailSections"
      icon="work"
      :status-label="positionDetail ? dictLabel('org_object_status', positionDetail.status) : ''"
      :status-color="positionDetail ? organizationStatusColor(positionDetail.status) : 'positive'"
      :loading="detailLoading"
      :error="detailError"
      :mode="detailMode"
      :top-buttons="record_detail_top_buttons"
      :bottom-buttons="record_detail_bottom_buttons"
      :record-context="positionDetail"
      @button-click="handleDetailAction"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'organization_position' })

import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeSelector from 'src/components/QueryScheme/QuerySchemeSelector.vue'
import QueryQuickPresets from 'src/components/QueryScheme/QueryQuickPresets.vue'
import QuerySchemeSaveDialog from 'src/components/QueryScheme/QuerySchemeSaveDialog.vue'
import {
  getPositionDetail,
  queryPositions,
  type PositionDetail,
  type PositionListItem,
  type PositionQueryRequest,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import { useRuntimeTableMetadata } from 'src/composables/runtime-table-metadata'
import { useTableQueryState } from 'src/composables/table-query-state'
import { useQuerySchemePage } from 'src/composables/query-scheme-page'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailSection } from 'src/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from 'src/pages/organization/use-organization-detail-mode'
import {
  createOrganizationQuery,
  formatOrganizationDate,
  formatOrganizationValue,
  organizationStatusColor,
  referenceLabel,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { resolveRuntimeColumns } from 'src/utils/column-format'
import { resolveTableEmptyMessage } from 'src/utils/table-state'
import type { TableColumn } from 'src/types/global'
import { compactSelectionDisplay } from 'src/utils/select-display'

const router = useRouter()
const dictStore = useDictStore()
const {
  line_buttons,
  has_line_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
  hasGrantedCapability,
} = usePageButtons('organization_position')
const detailMode = useOrganizationDetailMode('organization_position', 'dialog')

const rows = ref<PositionListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const canQueryPositions = computed(() => hasGrantedCapability('organization_position_query'))
const queryState = useTableQueryState<PositionQueryRequest>({
  createInitialQuery: () => ({
    ...createOrganizationQuery('org_position'),
    only_effective: true,
  }),
})
const { query, keyword, draftAdvanced: tempAdvancedQuery } = queryState
const showAdvancedQuery = ref(false)
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const positionDetail = ref<PositionDetail | null>(null)
const resetAndFetch = () => {
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}
const schemePage = useQuerySchemePage('organization_position', queryState, resetAndFetch)
const initialized = ref(false)

const rowButtons = computed(() =>
  line_buttons.value.filter((button) => button.event_action === 'detail'),
)
const columns = ref<TableColumn<PositionListItem>[]>([])
const visibleColumns = ref<string[]>([])
const sortableFields = ref<ReadonlySet<string>>(new Set())
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: false })
const {
  fields: metadataFields,
  advancedSearchFields: advancedFields,
  loadMetadata,
} = useRuntimeTableMetadata('org_position')
const emptyMessage = computed(() =>
  resolveTableEmptyMessage({
    canRead: canQueryPositions.value,
    error: loadError.value,
    hasQuery: !!keyword.value,
  }),
)

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || formatOrganizationValue(value)

const detailSections = computed<OrganizationDetailSection[]>(() => {
  const detail = positionDetail.value
  if (!detail) return []
  return [
    {
      key: 'basic',
      label: '基本信息',
      caption: '岗位定义与有效状态',
      icon: 'work',
      items: [
        { label: '岗位编码', value: detail.code },
        { label: '岗位名称', value: detail.name },
        { label: '岗位类型', value: dictLabel('org_position_type', detail.position_type) },
        { label: '职级', value: detail.job_level },
        { label: '管理岗位', value: detail.is_manager_position },
        {
          label: '状态',
          value: dictLabel('org_object_status', detail.status),
          chip: true,
          color: organizationStatusColor(detail.status),
        },
        { label: '有效期开始', value: formatOrganizationDate(detail.valid_from) },
        { label: '有效期结束', value: formatOrganizationDate(detail.valid_to, '长期') },
      ],
    },
    {
      key: 'ownership',
      label: '归属信息',
      caption: '法人和组织归属',
      icon: 'account_tree',
      items: [
        { label: '所属组织', value: referenceLabel(detail.org_unit) },
        { label: '所属法人', value: referenceLabel(detail.legal_entity) },
      ],
    },
    {
      key: 'mirror',
      label: '镜像信息',
      caption: '平台扩展信息',
      icon: 'sync',
      items: [{ label: '平台备注', value: detail.local_note, fullWidth: true }],
    },
  ]
})

const search = () => {
  schemePage.runQueryChange(queryState.submitQuickSearch)
}

const openAdvancedQuery = () => {
  queryState.beginAdvancedEdit()
  showAdvancedQuery.value = true
}

const applyAdvancedQuery = () => {
  schemePage.runQueryChange(() => queryState.applyAdvancedQuery(tempAdvancedQuery.value))
  showAdvancedQuery.value = false
}

const fetchData = async () => {
  if (!canQueryPositions.value) return
  loading.value = true
  loadError.value = ''
  try {
    const result = await queryPositions(query.value)
    rows.value = result.items
    total.value = result.total
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '岗位数据加载失败'
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: PositionListItem) => {
  positionDetail.value = null
  detailError.value = ''
  detailLoading.value = true
  showDetailDialog.value = true
  try {
    positionDetail.value = await getPositionDetail(row.id)
  } catch {
    detailError.value = '岗位详情加载失败'
  } finally {
    detailLoading.value = false
  }
}

const handleRowAction = (button: MenuButton, row: PositionListItem) => {
  if (button.event_action === 'detail') void openDetail(row)
  if (button.event_action === 'view_sync') {
    void router.push({
      name: 'organization_sync_error',
      query: { object_type: 'position', local_id: String(row.id) },
    })
  }
}

const handleDetailAction = (button: MenuButton) => {
  if (!positionDetail.value || button.event_action !== 'view_sync') return
  void router.push({
    name: 'organization_sync_error',
    query: { object_type: 'position', local_id: String(positionDetail.value.id) },
  })
}

watch(
  () => [query.value.page, query.value.num],
  () => {
    if (initialized.value) void fetchData()
  },
)

watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], previous) => {
    if (!initialized.value) return
    if (sortBy === previous[0] && descending === previous[1]) return
    if (!queryState.applySorting(sortBy || '', descending, sortableFields.value)) return
    void fetchData()
  },
)

onMounted(async () => {
  await Promise.all([
    dictStore.loadDicts(['org_position_type', 'org_object_status']),
    loadMetadata(),
  ])
  const resolution = resolveRuntimeColumns<PositionListItem>(metadataFields.value, {
    context: { getDictLabel: dictStore.getDictLabel },
    overrides: [
      { fieldCode: 'org_unit_id', visible: false },
      { fieldCode: 'position_type', align: 'center' },
      { fieldCode: 'is_manager_position', align: 'center' },
      { fieldCode: 'status', align: 'center' },
    ],
    virtualColumns: [
      { name: 'validity', field: 'validity', label: '有效期', order: 90 },
      {
        name: 'actions',
        field: 'actions',
        label: '操作',
        align: 'center',
        order: 100,
        defaultVisible: has_line_buttons.value,
      },
    ],
  })
  columns.value = resolution.columns
  visibleColumns.value = resolution.visibleColumns
  sortableFields.value = resolution.sortableFields
  await schemePage.initialize()
  await fetchData()
  initialized.value = true
})
</script>
