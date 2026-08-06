<template>
  <base-content class="q-pa-sm">
    <q-table
      v-model:pagination="pagination"
      class="fit sticky-header-table"
      color="primary"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
    >
      <template #top
        ><div class="row q-gutter-xs full-width items-center">
          <q-input
            v-model="query.quick_query!.keyword"
            dense
            outlined
            debounce="300"
            placeholder="搜索执行编号、系统或接口"
            @keyup.enter="search"
            ><template #append><q-icon name="search" /></template></q-input
          ><q-select
            v-model="query.status"
            dense
            outlined
            clearable
            emit-value
            map-options
            :options="statusOptions"
            label="状态"
            style="min-width: 150px"
          /><q-btn
            color="primary"
            icon="search"
            label="查询"
            :disable="loading"
            @click="search"
          /><q-space /><q-btn
            v-for="button in top_buttons"
            :key="button.id"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            @click="search"
          /></div
      ></template>
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><q-chip
            dense
            square
            outline
            :color="statusMeta[props.row.status]?.color || 'grey'"
            :label="statusMeta[props.row.status]?.label || props.row.status" /></q-td
      ></template>
      <template #body-cell-result_certainty="props"
        ><q-td :props="props"
          ><q-chip
            dense
            square
            :color="props.row.result_certainty === 'confirmed' ? 'positive' : 'warning'"
            text-color="white"
            :label="props.row.result_certainty === 'confirmed' ? '结果已确认' : '结果未知'" /></q-td
      ></template>
      <template #body-cell-actions="props"
        ><q-td :props="props" class="q-gutter-xs no-wrap"
          ><q-btn
            v-for="button in detailButtons"
            :key="button.id"
            flat
            dense
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            @click="openDetail(props.row)"
            ><q-tooltip>{{ button.name }}</q-tooltip></q-btn
          ></q-td
        ></template
      >
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
    </q-table>
    <q-dialog v-model="showDetail"
      ><q-card style="min-width: min(720px, 92vw)"
        ><q-card-section class="row items-center"
          ><div class="text-h6">调用日志详情</div>
          <q-space /><q-btn v-close-popup flat round icon="close" /></q-card-section
        ><q-separator /><q-card-section v-if="detail"
          ><div class="row q-col-gutter-lg">
            <div class="col-6">
              <div class="text-caption text-grey-7">执行编号</div>
              <div>{{ detail.execution_no }}</div>
            </div>
            <div class="col-6">
              <div class="text-caption text-grey-7">Attempt</div>
              <div>#{{ detail.attempt_no }}</div>
            </div>
            <div class="col-6">
              <div class="text-caption text-grey-7">凭证编码</div>
              <div>{{ detail.credential_code || '-' }}</div>
            </div>
            <div class="col-6">
              <div class="text-caption text-grey-7">凭证版本摘要</div>
              <div>{{ detail.credential_version || '-' }}</div>
            </div>
            <div class="col-6">
              <div class="text-caption text-grey-7">响应类型</div>
              <div>{{ detail.response_content_type || '-' }}</div>
            </div>
            <div class="col-6">
              <div class="text-caption text-grey-7">响应大小</div>
              <div>{{ detail.result_size_bytes }} bytes</div>
            </div>
            <div class="col-12">
              <div class="text-caption text-grey-7">安全错误消息</div>
              <div>{{ detail.result_summary || detail.error_code || '-' }}</div>
            </div>
          </div></q-card-section
        ></q-card
      ></q-dialog
    >
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_log' })
import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps } from 'quasar'
import { useRoute } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import {
  useIntegrationApi,
  type IntegrationLogDetail,
  type IntegrationLogListItem,
  type IntegrationLogQuery,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useLoadingStore } from 'src/stores/loading'
import { useUserStore } from 'src/stores/user'
import { storeToRefs } from 'pinia'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'

const route = useRoute()
const api = useIntegrationApi()
const userStore = useUserStore()
const { loading } = storeToRefs(useLoadingStore())
const { line_buttons, top_buttons } = usePageButtons('integration_log')
const detailButtons = computed(() =>
  line_buttons.value.filter((button) => button.event_action === 'detail'),
)
const canViewDetail = computed(() => detailButtons.value.length > 0)
const canQueryLogs = computed(() => userStore.buttons.includes('integration_log_query'))
const rows = ref<IntegrationLogListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const showDetail = ref(false)
const detail = ref<IntegrationLogDetail | null>(null)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const query = ref<IntegrationLogQuery>({
  page: 1,
  num: 15,
  order: { field: 'started_at', is_asc: false },
  quick_query: { keyword: '' },
  expressions: [],
})
if (typeof route.query.execution_no === 'string')
  query.value.execution_no = route.query.execution_no
const statusOptions = [
  { label: '执行中', value: 'running' },
  { label: '成功', value: 'succeeded' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
]
const statusMeta: Record<string, { label: string; color: string }> = {
  running: { label: '执行中', color: 'primary' },
  succeeded: { label: '成功', color: 'positive' },
  failed: { label: '失败', color: 'negative' },
  cancelled: { label: '已取消', color: 'grey-6' },
}
const columns: QTableProps['columns'] = [
  { name: 'execution_no', label: '执行编号', field: 'execution_no', align: 'left' },
  { name: 'attempt_no', label: 'Attempt', field: 'attempt_no', align: 'center' },
  { name: 'system_code', label: '系统', field: 'system_code', align: 'left' },
  { name: 'interface_code', label: '接口', field: 'interface_code', align: 'left' },
  { name: 'worker_id_summary', label: 'Worker', field: 'worker_id_summary', align: 'left' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'http_status', label: 'HTTP状态', field: 'http_status', align: 'center' },
  { name: 'duration_ms', label: '耗时（毫秒）', field: 'duration_ms', align: 'right' },
  { name: 'result_certainty', label: '结果确定性', field: 'result_certainty', align: 'center' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const fetchData = async () => {
  if (!canQueryLogs.value) return
  const response = await api.queryLogs(query.value)
  rows.value = response.data || []
  total.value = response.total || 0
}
const search = () => {
  query.value.page = 1
  void fetchData()
}
const openDetail = async (row: IntegrationLogListItem) => {
  const response = await api.getLog(row.id)
  detail.value = response.data || null
  showDetail.value = true
}
const openDetailFromRoute = async () => {
  if (!canViewDetail.value) return
  const logId = Number(route.query.log_id)
  if (logId > 0) {
    const response = await api.getLog(logId)
    detail.value = response.data || null
    showDetail.value = Boolean(detail.value)
  }
}
onMounted(async () => {
  if (!canQueryLogs.value) {
    initialized.value = true
    return
  }
  await fetchData()
  await openDetailFromRoute()
  initialized.value = true
})
watch(
  () => [query.value.page, query.value.num] as const,
  () => {
    if (initialized.value && canQueryLogs.value) void fetchData()
  },
)
</script>
