<template>
  <base-content class="q-pa-sm report-workbench-page">
    <section class="workbench-hero">
      <div>
        <div class="text-h6 text-weight-bold">报表中心</div>
        <div class="text-body2 text-grey-7">
          统一管理报表定义、发布版本、菜单挂载和运行入口。列表区域保留平台表格、分页、查询和操作规范。
        </div>
      </div>
    </section>

    <section class="workbench-stats">
      <div v-for="stat in workbenchStats" :key="stat.label" class="stat-card">
        <q-icon :name="stat.icon" :color="stat.color" size="24px" />
        <div>
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
        </div>
      </div>
    </section>

    <section class="workbench-layout">
      <aside class="workbench-sidebar">
        <q-card flat bordered class="side-card">
          <q-card-section>
            <div class="side-title">报表分类</div>
            <q-list dense>
              <q-item
                v-for="item in categoryNavItems"
                :key="item.value || 'all'"
                clickable
                :active="categoryFilter === item.value"
                active-class="side-active"
                @click="selectCategory(item.value)"
              >
                <q-item-section>{{ item.label }}</q-item-section>
                <q-item-section side>
                  <q-chip dense square color="grey-2" text-color="grey-8" :label="`${item.count}`" />
                </q-item-section>
              </q-item>
            </q-list>
          </q-card-section>
        </q-card>

        <q-card flat bordered class="side-card">
          <q-card-section>
            <div class="side-title">使用说明</div>
            <div class="usage-list">
              <div><q-icon name="play_circle" /> 已发布报表可直接运行。</div>
              <div><q-icon name="account_tree" /> 发布到菜单后进入通用运行页。</div>
              <div><q-icon name="verified_user" /> 运行和导出继承菜单数据权限。</div>
              <div><q-icon name="design_services" /> 草稿修改不影响线上版本。</div>
            </div>
          </q-card-section>
        </q-card>
      </aside>

      <div class="workbench-list">
        <q-table
          v-model:pagination="pagination"
          class="fit sticky-header-table report-workbench-table"
          color="primary"
          :dense="$q.screen.lt.md"
          separator="cell"
          flat
          bordered
          row-key="id"
          :rows="rows"
          :columns="columns"
          :loading="loading"
          :rows-per-page-options="[0]"
        >
          <template #top>
            <div class="row q-gutter-xs full-width items-center">
              <div class="col-grow row q-gutter-xs items-center">
                <q-input
                  v-model="query.quick_query!.keyword"
                  class="report-keyword-input"
                  dense
                  outlined
                  debounce="300"
                  placeholder="搜索报表名称、编码、分类"
                  clearable
                  :disable="loading"
                  @keyup.enter="handleBasicSearch"
                >
                  <template #append>
                    <q-icon name="search" />
                  </template>
                </q-input>

                <q-btn color="primary" label="查询" :disable="loading" @click="handleBasicSearch" />

                <q-btn
                  outline
                  color="secondary"
                  icon="restart_alt"
                  label="重置"
                  :disable="loading"
                  @click="resetFilters"
                />

                <q-btn
                  outline
                  icon="tune"
                  color="primary"
                  :aria-label="
                    hasAppliedAdvancedFilters
                      ? `高级查询，已启用 ${activeFilterCount} 个条件`
                      : '高级查询'
                  "
                  @click="openAdvancedQuery"
                >
                  <q-badge v-if="activeFilterCount > 0" floating color="red">
                    {{ activeFilterCount }}
                  </q-badge>
                  <q-tooltip>
                    {{
                      hasAppliedAdvancedFilters
                        ? `高级查询，已启用 ${activeFilterCount} 个条件`
                        : '高级查询'
                    }}
                  </q-tooltip>
                </q-btn>
              </div>

              <q-space />

              <div class="row q-gutter-xs">
                <q-btn
                  color="primary"
                  icon="add"
                  label="新建报表"
                  :disable="!canTopAction('create') || loading"
                  @click="handleTopAction('create')"
                >
                  <q-tooltip>{{ topActionTooltip('create') }}</q-tooltip>
                </q-btn>

                <q-btn
                  outline
                  color="primary"
                  icon="refresh"
                  label="刷新"
                  :disable="loading"
                  @click="fetchReports"
                />
              </div>
            </div>
          </template>

      <template #body-cell-report_name="props">
        <q-td :props="props">
          <div class="column no-wrap">
            <span class="text-weight-medium">{{ reportName(props.row) }}</span>
            <span v-if="props.row.description" class="text-caption text-grey-6 ellipsis">
              {{ props.row.description }}
            </span>
          </div>
        </q-td>
      </template>

      <template #body-cell-report_kind="props">
        <q-td :props="props">
          <q-chip
            dense
            square
            :color="kindMeta(props.value).color"
            :text-color="kindMeta(props.value).textColor"
            :label="kindMeta(props.value).label"
          />
        </q-td>
      </template>

      <template #body-cell-category="props">
        <q-td :props="props">
          {{ categoryLabel(props.value) }}
        </q-td>
      </template>

      <template #body-cell-status="props">
        <q-td :props="props" class="text-center">
          <q-chip
            dense
            square
            :color="statusMeta(props.row.status).color"
            :text-color="statusMeta(props.row.status).textColor"
            :icon="statusMeta(props.row.status).icon"
            :label="statusMeta(props.row.status).label"
          />
        </q-td>
      </template>

      <template #body-cell-published_version_no="props">
        <q-td :props="props" class="text-center">
          <q-chip
            v-if="versionLabel(props.row)"
            dense
            square
            color="positive"
            text-color="white"
            :label="versionLabel(props.row)"
          />
          <span v-else class="text-grey-6">-</span>
        </q-td>
      </template>

      <template #body-cell-menu_status="props">
        <q-td :props="props" class="text-center">
          <q-chip
            dense
            square
            :color="menuStatusMeta(props.row).color"
            :text-color="menuStatusMeta(props.row).textColor"
            :icon="menuStatusMeta(props.row).icon"
            :label="menuStatusMeta(props.row).label"
          />
        </q-td>
      </template>

      <template #body-cell-menu_title="props">
        <q-td :props="props">
          <div class="column no-wrap">
            <span>{{ menuTitle(props.row) }}</span>
            <span v-if="menuPath(props.row) !== '-'" class="text-caption text-grey-6 ellipsis">
              {{ menuPath(props.row) }}
            </span>
            <span v-else class="text-caption text-grey-6">-</span>
          </div>
        </q-td>
      </template>

      <template #body-cell-updated_at="props">
        <q-td :props="props">
          {{ formatDateTime(props.value) || '-' }}
        </q-td>
      </template>

      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs text-right action-cell">
          <q-btn
            v-for="action in visibleLineActions(props.row)"
            :key="action.action"
            flat
            round
            dense
            size="sm"
            :color="action.color"
            :icon="action.icon"
            :aria-label="action.label"
            :disable="isActionDisabled(action.action, props.row)"
            @click.stop="handleLineAction(action.action, props.row)"
          >
            <q-tooltip>{{ actionTooltip(action.action, props.row) }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #no-data>
        <div class="full-width row flex-center q-gutter-sm text-grey-6 q-pa-xl">
          <q-icon name="inbox" size="28px" />
          <span>{{ loading ? '正在加载报表定义' : '暂无报表定义' }}</span>
        </div>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination
          v-model:page="query.page"
          v-model:pageSize="query.num"
          :total="total"
        />
      </template>
        </q-table>
      </div>
    </section>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      title="报表高级查询"
      :fields="tableFieldsAdvanced"
      :enable-nested="false"
      @search="handleAdvancedSearch"
    />

    <report-publish-menu-dialog
      v-model="publishMenuVisible"
      :report="publishMenuReport"
      @success="fetchReports"
    />
  </base-content>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useQuasar, type QTableProps } from 'quasar'
import cloneDeep from 'lodash/cloneDeep'

import AdvancedQuery from 'components/Query/AdvancedQuery.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { useReportApi, type Report, type ReportKind, type ReportStatus } from 'src/api/services/report'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import { usePageButtons } from 'src/composables/page-buttons'
import type { Query } from 'src/types/global'
import { formatDateTime } from 'src/utils/column-format'
import { countEffectiveQueryRules, hasEffectiveQueryRules } from 'src/utils/query-state'
import ReportPublishMenuDialog from '../components/ReportPublishMenuDialog.vue'

const router = useRouter()
const $q = useQuasar()
const reportApi = useReportApi()
const tableApi = useTableApi()
const { all_buttons: pageButtons } = usePageButtons('report_v2_workbench')

const rows = ref<Report[]>([])
const total = ref(0)
const loading = ref(false)
const publishMenuVisible = ref(false)
const publishMenuReport = ref<Report | null>(null)
const showAdvancedQuery = ref(false)
const tableFieldsAdvanced = ref<TableField[]>([])
const tableMetadataLoaded = ref(false)
const categoryFilter = ref('')
const fallbackCategories = ['system_audit', 'finance', 'operation', 'tms']

const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 15,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})

const query = ref<Query>({
  page: 1,
  num: 15,
  order: { field: 'gmt_modify', is_asc: false },
  table_code: 'report_definition',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: { keyword: '' },
  include_deleted: false,
  filters: {},
})

const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))
const appliedAdvancedQuery = ref(cloneDeep(emptyAdvancedQuery()))

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: false,
})

const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

type WorkbenchTopAction = 'create'
type WorkbenchLineAction =
  | 'design'
  | 'run'
  | 'publish'
  | 'publish_menu'
  | 'unpublish_menu'
  | 'version'
  | 'disable'
  | 'delete'

interface WorkbenchLineActionConfig {
  action: WorkbenchLineAction
  label: string
  icon: string
  color: string
}

const lineActionConfigs: WorkbenchLineActionConfig[] = [
  { action: 'design', label: '设计', icon: 'design_services', color: 'primary' },
  { action: 'run', label: '运行', icon: 'play_circle', color: 'primary' },
  { action: 'publish', label: '发布版本', icon: 'publish', color: 'primary' },
  { action: 'publish_menu', label: '发布到菜单', icon: 'account_tree', color: 'primary' },
  { action: 'unpublish_menu', label: '取消发布菜单', icon: 'remove_from_queue', color: 'warning' },
  { action: 'version', label: '版本', icon: 'history', color: 'primary' },
  { action: 'disable', label: '停用', icon: 'block', color: 'negative' },
  { action: 'delete', label: '删除', icon: 'delete', color: 'negative' },
]

const actionPermissionMap: Record<WorkbenchLineAction | WorkbenchTopAction, string[]> = {
  create: ['create'],
  design: ['update', 'navigate'],
  run: ['run', 'query'],
  publish: ['publish'],
  publish_menu: ['publish_menu'],
  unpublish_menu: ['unpublish_menu'],
  version: ['version', 'detail'],
  disable: ['disable', 'update'],
  delete: ['delete'],
}

const hasSeededWorkbenchButtons = computed(() => pageButtons.value.length > 0)

const workbenchStats = computed(() => {
  const published = rows.value.filter((item) => item.status === 'published').length
  const draft = rows.value.filter((item) => item.status === 'draft').length
  const categoryCount = new Set(rows.value.map((item) => item.category || '').filter(Boolean)).size
  const sourceCount = new Set(rows.value.map((item) => item.source_code || item.data_source_id || '').filter(Boolean)).size
  const permissionCount = rows.value.filter((item) => item.permission_table_code).length
  return [
    { label: '已发布报表', value: published, icon: 'verified', color: 'positive' },
    { label: '草稿报表', value: draft, icon: 'edit_note', color: 'grey-8' },
    { label: '报表分类', value: categoryCount || '-', icon: 'category', color: 'primary' },
    { label: '可用数据集', value: sourceCount || '-', icon: 'storage', color: 'indigo' },
    { label: '继承数据权限', value: permissionCount, icon: 'verified_user', color: 'teal' },
  ]
})

const categoryNavItems = computed(() => {
  const counts = new Map<string, number>()
  rows.value.forEach((item) => {
    const category = item.category || '未分类'
    counts.set(category, (counts.get(category) || 0) + 1)
  })
  const categories = Array.from(new Set([...fallbackCategories, ...counts.keys()]))
  return [
    { label: '全部报表', value: '', count: total.value || rows.value.length },
    ...categories.map((category) => ({
      label: categoryLabel(category),
      value: category === '未分类' ? '' : category,
      count: counts.get(category) || 0,
    })),
  ]
})

const columns: QTableProps['columns'] = [
  {
    name: 'report_name',
    label: '报表名称',
    field: (row: Report) => reportName(row),
    align: 'left',
    style: 'min-width: 190px; max-width: 260px;',
  },
  {
    name: 'report_code',
    label: '编码',
    field: (row: Report) => row.report_code || row.code || '-',
    align: 'left',
    style: 'min-width: 180px;',
  },
  {
    name: 'report_kind',
    label: '类型',
    field: 'report_kind',
    align: 'center',
    style: 'width: 110px;',
  },
  {
    name: 'category',
    label: '分类',
    field: 'category',
    align: 'left',
    style: 'width: 120px;',
  },
  {
    name: 'status',
    label: '状态',
    field: 'status',
    align: 'center',
    style: 'width: 100px;',
  },
  {
    name: 'published_version_no',
    label: '当前版本',
    field: 'published_version_no',
    align: 'center',
    style: 'width: 100px;',
  },
  {
    name: 'menu_status',
    label: '是否挂菜单',
    field: (row: Report) => isPublishedToMenu(row),
    align: 'center',
    style: 'width: 120px;',
  },
  {
    name: 'menu_title',
    label: '菜单名称 / 路径',
    field: (row: Report) => menuTitle(row),
    align: 'left',
    style: 'min-width: 180px;',
  },
  {
    name: 'updated_at',
    label: '更新时间',
    field: (row: Report) => row.updated_at || row.gmt_modify || '',
    align: 'left',
    style: 'width: 170px;',
  },
  {
    name: 'actions',
    label: '操作',
    field: 'actions',
    align: 'right',
    style: 'width: 260px;',
  },
]

onMounted(() => {
  void loadReportDefinitionMetadata()
  void fetchReports()
})

watch(
  () => query.value.page,
  (page) => {
    pagination.value.page = page
    void fetchReports()
  },
)

watch(
  () => query.value.num,
  (num) => {
    query.value.page = 1
    pagination.value.rowsPerPage = num
    void fetchReports()
  },
)

function reportName(row: Report) {
  return row.report_name || row.name || '-'
}

function statusMeta(value: ReportStatus) {
  const map = {
    draft: { label: '草稿', color: 'grey-3', textColor: 'grey-9', icon: 'edit_note' },
    published: { label: '已发布', color: 'positive', textColor: 'white', icon: 'verified' },
    disabled: { label: '已停用', color: 'negative', textColor: 'white', icon: 'block' },
  }
  return map[value] || map.draft
}

function kindMeta(value?: ReportKind) {
  if (value === 'summary') {
    return { label: '汇总报表', color: 'indigo-1', textColor: 'indigo-10' }
  }
  if (value === 'detail') {
    return { label: '明细报表', color: 'blue-1', textColor: 'blue-10' }
  }
  return { label: '版式报表', color: 'teal-1', textColor: 'teal-10' }
}

function categoryLabel(value?: string) {
  const map: Record<string, string> = {
    system_audit: '系统审计',
    finance: '财务报表',
    operation: '运营报表',
    tms: 'TMS 报表',
  }
  return value ? (map[value] || value) : '未分类'
}

function versionLabel(row: Report) {
  const versionNo = row.published_version_no
  return versionNo ? `V${versionNo}` : ''
}

function isPublishedToMenu(row: Report) {
  if (row.published_to_menu !== undefined) return row.published_to_menu
  return Boolean(row.permission_menu_id || row.menu_id)
}

function menuTitle(row: Report) {
  if (!isPublishedToMenu(row)) return '未接入'
  return row.menu_title || row.menu_name || '已接入'
}

function menuPath(row: Report) {
  if (!isPublishedToMenu(row)) return '-'
  return row.menu_path || '-'
}

function menuStatusMeta(row: Report) {
  if (isPublishedToMenu(row)) {
    return { label: '已挂菜单', color: 'positive', textColor: 'white', icon: 'done' }
  }
  return { label: '未接入', color: 'grey-3', textColor: 'grey-8', icon: 'link_off' }
}

function visibleLineActions(row: Report) {
  const publishedToMenu = isPublishedToMenu(row)
  return lineActionConfigs.filter((item) => {
    if (!hasActionPermission(item.action)) return false
    if (item.action === 'publish_menu') return !publishedToMenu
    if (item.action === 'unpublish_menu') return publishedToMenu
    return true
  })
}

function canTopAction(action: WorkbenchTopAction) {
  if (!hasActionPermission(action)) return false
  if (action === 'create') return false
  return false
}

function topActionTooltip(action: WorkbenchTopAction) {
  if (!hasActionPermission(action)) return '无权限'
  if (action === 'create') return '新建报表流程将在后续设计器阶段完善'
  return '暂不可用'
}

function handleTopAction(action: WorkbenchTopAction) {
  if (!canTopAction(action)) {
    $q.notify({
      type: 'info',
      position: 'top-right',
      message: topActionTooltip(action),
    })
  }
}

function canAction(action: WorkbenchLineAction, row: Report) {
  if (!row.id) return false
  if (!hasActionPermission(action)) return false
  if (action === 'publish_menu') return row.status === 'published' && !isPublishedToMenu(row)
  if (action === 'unpublish_menu') return isPublishedToMenu(row)
  if (action === 'publish' || action === 'version' || action === 'disable' || action === 'delete') {
    return false
  }
  return true
}

function isActionDisabled(action: WorkbenchLineAction, row: Report) {
  return loading.value || !canAction(action, row)
}

function actionTooltip(action: WorkbenchLineAction, row: Report) {
  const config = lineActionConfigs.find((item) => item.action === action)
  if (!hasActionPermission(action)) return '无权限'
  if (action === 'publish') return '发布版本请进入报表设计器执行'
  if (action === 'publish_menu' && row.status !== 'published') return '请先发布报表版本'
  if (action === 'version') return '版本列表入口将在后续接入'
  if (action === 'disable') return '停用操作将在按钮权限接入后开放'
  if (action === 'delete') return '删除操作将在按钮权限接入后开放'
  return config?.label || '操作'
}

function hasActionPermission(action: WorkbenchLineAction | WorkbenchTopAction) {
  if (!hasSeededWorkbenchButtons.value) return true
  const allowedActions = actionPermissionMap[action] || [action]
  return pageButtons.value.some((button) => {
    const eventAction = String(button.event_action || '')
    const code = String(button.code || '')
    return allowedActions.includes(eventAction) || allowedActions.some((item) => code.endsWith(`_${item}`))
  })
}

function handleLineAction(action: WorkbenchLineAction, row: Report) {
  if (!canAction(action, row)) {
    $q.notify({
      type: 'info',
      position: 'top-right',
      message: actionTooltip(action, row),
    })
    return
  }

  const handlers: Record<WorkbenchLineAction, () => void> = {
    design: () => goDesigner(row.id),
    run: () => goRuntime(row.id),
    publish: () => undefined,
    publish_menu: () => openPublishMenu(row),
    unpublish_menu: () => confirmUnpublishMenu(row),
    version: () => undefined,
    disable: () => undefined,
    delete: () => undefined,
  }
  handlers[action]()
}

function currentFilters() {
  const filters: Record<string, string> = {}
  if (categoryFilter.value) filters.category = categoryFilter.value
  return filters
}

function buildQuery(): Query {
  return {
    ...query.value,
    page: query.value.page,
    num: query.value.num,
    table_code: 'report_definition',
    quick_query: { keyword: query.value.quick_query?.keyword?.trim() || '' },
    filters: currentFilters(),
  }
}

async function loadReportDefinitionMetadata() {
  try {
    const res = await tableApi.queryTableByCode('report_definition')
    const fields = res.success ? (res.data?.table_fields || []) : []
    tableFieldsAdvanced.value = fields.filter((field) => field.is_advanced_search)
    tableMetadataLoaded.value = tableFieldsAdvanced.value.length > 0
  } catch (error) {
    tableFieldsAdvanced.value = []
    tableMetadataLoaded.value = false
    console.warn('report_definition 元数据未初始化或加载失败，高级查询仅保留入口', error)
  }
}

async function fetchReports() {
  loading.value = true
  try {
    const res = await reportApi.queryReports(buildQuery())
    rows.value = res.data || []
    total.value = res.total ?? rows.value.length
  } catch (error) {
    rows.value = []
    total.value = 0
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: error instanceof Error && error.message ? error.message : '报表工作台列表加载失败',
    })
  } finally {
    loading.value = false
  }
}

function resetToFirstPageOrFetch() {
  if (query.value.page !== 1) {
    query.value.page = 1
    return
  }
  void fetchReports()
}

function handleBasicSearch() {
  query.value.expressions = emptyAdvancedQuery().expressions
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
}

function handleAdvancedSearch() {
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  showAdvancedQuery.value = false
  resetToFirstPageOrFetch()
}

function openAdvancedQuery() {
  if (!tableMetadataLoaded.value) {
    $q.notify({
      type: 'warning',
      position: 'top-right',
      message: 'report_definition 元数据未初始化，高级查询字段暂不可用',
    })
  }
  tempAdvancedQuery.value = cloneDeep(query.value)
  showAdvancedQuery.value = true
}

function resetFilters() {
  query.value.quick_query = { keyword: '' }
  query.value.expressions = emptyAdvancedQuery().expressions
  categoryFilter.value = ''
  appliedAdvancedQuery.value = cloneDeep(emptyAdvancedQuery())
  tempAdvancedQuery.value = cloneDeep(query.value)
  resetToFirstPageOrFetch()
}

function selectCategory(value: string) {
  categoryFilter.value = value
  handleBasicSearch()
}

function goDesigner(id: number) {
  void router.push({ name: 'report_v2_designer', params: { id } })
}

function goRuntime(id: number) {
  void router.push({ name: 'report_v2_runtime', params: { id } })
}

function openPublishMenu(row: Report) {
  if (row.status !== 'published') return
  publishMenuReport.value = row
  publishMenuVisible.value = true
}

function confirmUnpublishMenu(row: Report) {
  if (!row.id) return
  $q.dialog({
    title: '取消发布菜单',
    message: `确定要取消报表「${reportName(row)}」的菜单发布吗？该操作不会删除报表定义。`,
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void unpublishMenu(row.id)
  })
}

async function unpublishMenu(id: number) {
  try {
    await reportApi.unpublishReportMenu(id)
    $q.notify({
      type: 'positive',
      position: 'top-right',
      message: '已取消发布菜单',
    })
    await fetchReports()
  } catch (error) {
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message: error instanceof Error && error.message ? error.message : '取消发布菜单失败',
    })
  }
}
</script>

<style scoped lang="scss">
.report-workbench-page {
  height: 100%;
  min-height: 0;
  overflow: hidden;
  box-sizing: border-box;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
}

.workbench-hero {
  flex: 0 0 auto;
  margin-bottom: 10px;
  padding: 14px 16px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.workbench-stats {
  flex: 0 0 auto;
  margin-bottom: 10px;
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.stat-card {
  min-height: 72px;
  padding: 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  display: flex;
  align-items: center;
  gap: 10px;
}

.stat-value {
  color: #172033;
  font-size: 20px;
  font-weight: 800;
  line-height: 1.2;
}

.stat-label {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
}

.workbench-layout {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 10px;
  align-items: stretch;
}

.workbench-sidebar {
  min-height: 0;
  overflow-y: auto;
  display: grid;
  gap: 10px;
  align-content: start;
}

.side-card {
  border-radius: 8px;
}

.side-title {
  margin-bottom: 8px;
  color: #172033;
  font-weight: 700;
}

.side-active {
  color: var(--q-primary);
  background: #eef4ff;
}

.usage-list {
  display: grid;
  gap: 8px;
  color: #475467;
  font-size: 13px;

  > div {
    display: flex;
    align-items: flex-start;
    gap: 6px;
  }
}

.workbench-list {
  height: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  display: flex;
}

.report-workbench-table {
  height: 100%;
  min-height: 0;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;

  :deep(.q-table__top) {
    padding: 8px;
    flex: 0 0 auto;
  }

  :deep(.q-table__middle) {
    flex: 1 1 auto;
    min-height: 0;
    overflow: auto;
    overflow: overlay;
  }

  :deep(.q-table__bottom) {
    flex: 0 0 auto;
    min-height: 48px;
  }
}

.report-keyword-input {
  width: 280px;
}

.action-cell {
  white-space: nowrap;
}

@media (max-width: 1280px) {
  .workbench-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .workbench-layout {
    grid-template-columns: 220px minmax(0, 1fr);
  }
}
</style>
