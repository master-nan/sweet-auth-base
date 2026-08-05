<template>
  <base-content :scrollable="showDetailDialog && detailMode === 'page'" class="q-pa-sm">
    <scrollable-table
      v-if="!showDetailDialog || detailMode === 'dialog'"
      class="fit"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      :pagination="{ rowsPerPage: 0 }"
      hide-pagination
    >
      <template #top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              v-model="query.quick_query!.keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索关键词"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="search" />
            <q-btn outline color="primary" icon="tune" @click="openAdvancedQuery">
              <q-tooltip>高级查询</q-tooltip>
            </q-btn>
          </div>
          <q-space />
          <div class="row q-gutter-xs">
            <q-btn
              v-for="button in refreshButtons"
              :key="button.id || button.code"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :disable="loading"
              @click="fetchData"
            />
          </div>
        </div>
      </template>

      <template #body-cell-position_type="props">
        <q-td :props="props">
          {{ dictLabel('org_position_type', props.row.position_type) }}
        </q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props">
          <q-chip dense square outline :color="organizationStatusColor(props.row.status)">
            {{ dictLabel('org_object_status', props.row.status) }}
          </q-chip>
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
          {{ loadError || '暂无岗位数据' }}
        </div>
      </template>
      <template #bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </scrollable-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="advancedFields"
      title="岗位高级查询"
      @search="applyAdvancedQuery"
    />

    <organization-record-detail-dialog
      v-model="showDetailDialog"
      :title="positionDetail?.name || '岗位详情'"
      :subtitle="positionDetail?.code || ''"
      :sections="detailSections"
      icon="work"
      :status-label="
        positionDetail ? dictLabel('org_object_status', positionDetail.status) : ''
      "
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

import cloneDeep from 'lodash/cloneDeep'
import { computed, onMounted, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import ScrollableTable from 'src/components/Table/ScrollableTable.vue'
import {
  getPositionDetail,
  queryPositions,
  type PositionDetail,
  type PositionListItem,
  type PositionQueryRequest,
} from 'src/api/services/org'
import type { MenuButton } from 'src/api/services/sys-menu'
import { usePageButtons } from 'src/composables/page-buttons'
import OrganizationRecordDetailDialog from 'src/pages/organization/components/OrganizationRecordDetailDialog.vue'
import type { OrganizationDetailSection } from 'src/pages/organization/components/organization-record-detail'
import { useOrganizationDetailMode } from 'src/pages/organization/use-organization-detail-mode'
import {
  createOrganizationField,
  createOrganizationQuery,
  formatOrganizationDate,
  organizationStatusColor,
  referenceLabel,
} from 'src/pages/organization/organization-list-page'
import { useDictStore } from 'src/stores/dict'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const router = useRouter()
const dictStore = useDictStore()
const {
  line_buttons,
  top_buttons,
  record_detail_top_buttons,
  record_detail_bottom_buttons,
} = usePageButtons('organization_position')
const detailMode = useOrganizationDetailMode('organization_position', 'dialog')

const rows = ref<PositionListItem[]>([])
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const query = ref<PositionQueryRequest>({
  ...createOrganizationQuery('org_position'),
  only_effective: true,
})
const tempAdvancedQuery = ref<PositionQueryRequest>(cloneDeep(query.value))
const showAdvancedQuery = ref(false)
const showDetailDialog = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const positionDetail = ref<PositionDetail | null>(null)

const refreshButtons = computed(() =>
  top_buttons.value.filter((button) => button.event_action === 'refresh'),
)
const rowButtons = computed(() =>
  line_buttons.value.filter((button) => button.event_action === 'detail'),
)
const columns: QTableProps['columns'] = [
  { name: 'code', field: 'code', label: '岗位编码', align: 'left', sortable: true },
  { name: 'name', field: 'name', label: '岗位名称', align: 'left', sortable: true },
  { name: 'position_type', field: 'position_type', label: '岗位类型', align: 'center' },
  { name: 'job_level', field: 'job_level', label: '职级', align: 'left' },
  {
    name: 'is_manager_position',
    field: 'is_manager_position',
    label: '管理岗位',
    align: 'center',
  },
  { name: 'status', field: 'status', label: '状态', align: 'center' },
  { name: 'validity', field: 'validity', label: '有效期', align: 'left' },
  { name: 'actions', field: 'actions', label: '操作', align: 'center' },
]

const advancedFields = [
  createOrganizationField('岗位编码', 'code'),
  createOrganizationField('岗位名称', 'name'),
  createOrganizationField('岗位类型', 'position_type', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_position_type',
  }),
  createOrganizationField('职级', 'job_level'),
  createOrganizationField('管理岗位', 'is_manager_position', SysTableFieldType.BOOLEAN, {
    inputType: SysTableFieldInputType.BOOLEAN,
  }),
  createOrganizationField('状态', 'status', SysTableFieldType.VARCHAR, {
    inputType: SysTableFieldInputType.SELECT,
    dictCode: 'org_object_status',
  }),
]

const dictLabel = (code: string, value: unknown) =>
  dictStore.getDictLabel(code, value) || String(value || '-')

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
  if (query.value.page !== 1) query.value.page = 1
  else void fetchData()
}

const openAdvancedQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
  showAdvancedQuery.value = true
}

const applyAdvancedQuery = () => {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  showAdvancedQuery.value = false
  search()
}

const fetchData = async () => {
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
  () => void fetchData(),
)

onMounted(async () => {
  await dictStore.loadDicts(['org_position_type', 'org_object_status'])
  await fetchData()
})
</script>
