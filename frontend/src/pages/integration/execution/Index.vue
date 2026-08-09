<template>
  <base-content class="q-pa-sm">
    <div class="row q-col-gutter-sm q-mb-sm">
      <div class="col-12 col-md-8">
        <q-card flat bordered class="runtime-status-card">
          <q-card-section class="row items-center q-gutter-md">
            <q-icon name="monitor_heart" size="32px" color="primary" />
            <div class="col">
              <div class="text-subtitle1 text-weight-bold">当前实例状态</div>
              <div class="text-caption text-grey-7">
                {{ workerStatus.worker_id || '未启用 Worker' }}
              </div>
            </div>
            <q-chip
              dense
              square
              :color="workerStatus.running ? 'positive' : 'grey-6'"
              text-color="white"
            >
              {{ workerStatus.running ? '运行中' : workerStatus.enabled ? '已停止' : '未启用' }}
            </q-chip>
            <div class="text-caption">活动 {{ workerStatus.active_execution_count }}</div>
            <div class="text-caption">已完成 {{ workerStatus.completed_total }}</div>
          </q-card-section>
        </q-card>
      </div>
      <div class="col-12 col-md-4">
        <q-card flat bordered class="runtime-status-card fit">
          <q-card-section class="row items-center justify-between">
            <span class="text-caption text-grey-7">最近轮询</span>
            <span>{{ formatDate(workerStatus.last_poll_at) }}</span>
            <q-btn
              flat
              round
              dense
              icon="refresh"
              color="primary"
              :loading="loading"
              @click="refresh"
            />
          </q-card-section>
        </q-card>
      </div>
    </div>

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
      <template #top>
        <div class="row q-gutter-xs full-width items-center">
          <q-input
            v-model="query.quick_query!.keyword"
            dense
            outlined
            debounce="300"
            placeholder="搜索执行编号、系统或接口"
            @keyup.enter="search"
          >
            <template #append><q-icon name="search" /></template>
          </q-input>
          <q-select
            v-model="query.status"
            dense
            outlined
            clearable
            emit-value
            map-options
            :options="statusOptions"
            label="状态"
            style="min-width: 150px"
          />
          <q-btn color="primary" icon="search" label="查询" :disable="loading" @click="search" />
          <q-space />
          <q-btn
            v-for="button in top_buttons"
            :key="button.id"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            :disable="loading"
            @click="handleButton(button)"
          />
        </div>
      </template>

      <template #body-cell-execution_no="props">
        <q-td :props="props">
          <q-btn
            v-if="canViewDetail"
            flat
            dense
            color="primary"
            :label="props.row.execution_no"
            @click="openDetail(props.row)"
          />
          <span v-else>{{ props.row.execution_no }}</span>
        </q-td>
      </template>
      <template #body-cell-status="props">
        <q-td :props="props"
          ><q-chip
            dense
            square
            outline
            :color="statusMeta[props.row.status]?.color || 'grey'"
            :label="statusMeta[props.row.status]?.label || props.row.status"
        /></q-td>
      </template>
      <template #body-cell-external_system="props">
        <q-td :props="props"
          ><div class="text-weight-medium">{{ props.row.external_system.name }}</div>
          <div class="text-caption text-grey-7">
            {{ props.row.external_system.system_code }}
          </div></q-td
        >
      </template>
      <template #body-cell-interface="props">
        <q-td :props="props"
          ><div>{{ props.row.interface.interface_code }}</div>
          <div class="text-caption text-grey-7">
            v{{ props.row.interface.version }} · {{ props.row.interface.name }}
          </div></q-td
        >
      </template>
      <template #body-cell-current_attempt="props">
        <q-td :props="props">{{ props.row.current_attempt }} / {{ props.row.max_attempts }}</q-td>
      </template>
      <template #body-cell-next_run_at="props">
        <q-td :props="props">{{ formatDate(props.row.next_run_at) }}</q-td>
      </template>
      <template #body-cell-started_at="props">
        <q-td :props="props">{{ formatDate(props.row.started_at) }}</q-td>
      </template>
      <template #body-cell-completed_at="props">
        <q-td :props="props">{{ formatDate(props.row.completed_at) }}</q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs no-wrap">
          <q-btn
            v-for="button in availableButtons(props.row)"
            :key="button.id"
            flat
            dense
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            @click="handleButton(button, props.row)"
            ><q-tooltip>{{ button.name }}</q-tooltip></q-btn
          >
        </q-td>
      </template>
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="query.page"
          v-model:page-size="query.num"
          :total="total"
      /></template>
    </q-table>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'integration_execution' })

import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import {
  useIntegrationApi,
  type IntegrationExecutionListItem,
  type IntegrationExecutionQuery,
  type IntegrationWorkerStatus,
} from 'src/api/services/integration'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useLoadingStore } from 'src/stores/loading'
import { useUserStore } from 'src/stores/user'
import { storeToRefs } from 'pinia'
import type { MenuButton } from 'src/api/services/sys-menu'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { formatRetryReason, formatRuntimeDateTime } from 'src/pages/integration/runtime-display'

const $q = useQuasar()
const router = useRouter()
const api = useIntegrationApi()
const userStore = useUserStore()
const { loading } = storeToRefs(useLoadingStore())
const { confirmAction } = useConfirmDialog($q)
const { line_buttons, top_buttons } = usePageButtons('integration_execution')
const canViewDetail = computed(() => userStore.buttons.includes('integration_execution_detail'))
const rows = ref<IntegrationExecutionListItem[]>([])
const total = ref(0)
const initialized = ref(false)
const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const query = ref<IntegrationExecutionQuery>({
  page: 1,
  num: 15,
  order: { field: 'gmt_create', is_asc: false },
  quick_query: { keyword: '' },
  expressions: [],
})
const workerStatus = ref<IntegrationWorkerStatus>({
  enabled: false,
  running: false,
  worker_id: '',
  started_at: '',
  last_poll_at: '',
  last_success_at: '',
  last_error_category: '',
  active_execution_count: 0,
  claimed_total: 0,
  completed_total: 0,
  failed_total: 0,
  recovered_total: 0,
})
const statusOptions = [
  { label: '待执行', value: 'created' },
  { label: '执行中', value: 'running' },
  { label: '等待重试', value: 'retry_waiting' },
  { label: '成功', value: 'succeeded' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
]
const statusMeta: Record<string, { label: string; color: string }> = {
  created: { label: '待执行', color: 'grey-7' },
  running: { label: '执行中', color: 'primary' },
  retry_waiting: { label: '等待重试', color: 'warning' },
  succeeded: { label: '成功', color: 'positive' },
  failed: { label: '失败', color: 'negative' },
  cancelled: { label: '已取消', color: 'grey-6' },
}
const columns: QTableProps['columns'] = [
  { name: 'execution_no', label: '执行编号', field: 'execution_no', align: 'left', sortable: true },
  { name: 'external_system', label: '外部系统', field: 'external_system', align: 'left' },
  { name: 'interface', label: '接口', field: 'interface', align: 'left' },
  { name: 'trigger_source', label: '触发来源', field: 'trigger_source', align: 'center' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'current_attempt', label: 'Attempt', field: 'current_attempt', align: 'center' },
  { name: 'next_run_at', label: '下次重试', field: 'next_run_at', align: 'left' },
  { name: 'retry_reason_code', label: '重试原因', field: (row) => formatRetryReason(row.retry_reason_code), align: 'left' },
  { name: 'started_at', label: '开始时间', field: 'started_at', align: 'left' },
  { name: 'completed_at', label: '结束时间', field: 'completed_at', align: 'left' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]

const formatDate = formatRuntimeDateTime
const fetchData = async () => {
  const response = await api.queryExecutions(query.value)
  rows.value = response.data || []
  total.value = response.total || 0
}
const fetchWorker = async () => {
  const response = await api.getWorkerStatus()
  if (response.data) workerStatus.value = response.data
}
const refresh = async () => {
  await fetchData()
  await fetchWorker().catch(() => undefined)
}
const search = () => {
  query.value.page = 1
  void fetchData()
}
const openDetail = (row: IntegrationExecutionListItem) => {
  void router.push({ name: 'integration_execution_detail_page', params: { id: row.id } })
}
const availableButtons = (row: IntegrationExecutionListItem) =>
  line_buttons.value.filter((button) =>
    button.event_action === 'cancel'
      ? row.status === 'created' || row.status === 'retry_waiting'
      : true,
  )
const cancel = (row: IntegrationExecutionListItem) => {
  confirmAction({
    title: '取消执行',
    message: `确认取消执行“${row.execution_no}”？`,
    loading: loading.value,
    disable: loading.value,
  }).onOk(() => {
    void (async () => {
      const result = await api.cancelExecution(row.id, row.revision)
      if (result.success) await fetchData()
    })()
  })
}
const handleButton = (button: MenuButton, row?: IntegrationExecutionListItem) => {
  if (button.event_action === 'detail' && row) openDetail(row)
  if (button.event_action === 'cancel' && row) cancel(row)
}
onMounted(async () => {
  await refresh()
  initialized.value = true
})
watch(
  () => [query.value.page, query.value.num] as const,
  () => {
    if (initialized.value) void fetchData()
  },
)
</script>

<style scoped>
.runtime-status-card {
  min-height: 76px;
}
</style>
