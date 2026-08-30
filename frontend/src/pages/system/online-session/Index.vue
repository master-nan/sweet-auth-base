<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      color="primary"
      separator="cell"
      flat
      bordered
      row-key="id"
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      :loading="loading"
      :no-data-label="loadError || '暂无符合条件的登录会话'"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <q-input
              v-model="keyword"
              dense
              outlined
              debounce="300"
              placeholder="搜索用户名、IP、浏览器或系统"
              @keyup.enter="runSearch"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-select
              v-model="status"
              dense
              outlined
              emit-value
              map-options
              :options="statusOptions"
              style="width: 180px"
              @update:model-value="runSearch"
            />
            <q-btn outline color="primary" icon="date_range" :label="loginTimeFilterLabel">
              <q-menu anchor="bottom left" self="top left">
                <div class="q-pa-md" style="width: min(560px, calc(100vw - 32px))">
                  <div class="text-subtitle2 q-mb-sm">登录时间范围</div>
                  <div class="row q-col-gutter-md">
                    <div class="col-12 col-sm-6">
                      <sweet-date-time-picker
                        v-model="loginStartedAt"
                        type="datetime"
                        label="开始时间"
                      />
                    </div>
                    <div class="col-12 col-sm-6">
                      <sweet-date-time-picker
                        v-model="loginEndedAt"
                        type="datetime"
                        label="结束时间"
                      />
                    </div>
                  </div>
                  <div v-if="dateRangeInvalid" class="text-negative text-caption q-mt-xs">
                    结束时间不能早于开始时间
                  </div>
                  <div class="row justify-end q-gutter-sm q-mt-md">
                    <q-btn flat color="grey-7" label="清空" @click="clearDateRange" />
                    <q-btn
                      v-close-popup
                      unelevated
                      color="primary"
                      label="应用"
                      :disable="dateRangeInvalid"
                      @click="runSearch"
                    />
                  </div>
                </div>
              </q-menu>
            </q-btn>
            <q-btn
              color="primary"
              icon="search"
              label="查询"
              :disable="loading"
              @click="runSearch"
            />
            <q-separator vertical inset class="q-mx-sm" />
            <div class="text-body2 text-grey-7">
              在线用户 <strong class="text-dark">{{ onlineUsers }}</strong>
              <span class="q-mx-xs">·</span>
              在线会话 <strong class="text-dark">{{ onlineSessions }}</strong>
            </div>
          </template>
          <template #right-actions>
            <q-btn
              v-for="button in top_buttons"
              :key="button.id"
              v-bind="menuButtonDisplayProps(button)"
              :color="button.color || 'primary'"
              :loading="exporting && button.event_action === 'export'"
              :disable="loading || exporting"
              @click="handleTopAction(button.event_action)"
            />
          </template>
          <template #column-selector>
            <table-column-selector v-model="visibleColumns" :columns="columns" />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body-cell-user_name="props">
        <q-td :props="props">
          <div class="row items-center no-wrap q-gutter-xs">
            <span class="text-weight-medium">{{ props.row.user_name }}</span>
            <q-chip v-if="props.row.user_deleted" dense square color="grey-4" text-color="grey-9"
              >账号已删除</q-chip
            >
            <q-chip v-if="props.row.current" dense square color="primary" text-color="white"
              >当前会话</q-chip
            >
          </div>
        </q-td>
      </template>

      <template #body-cell-device="props">
        <q-td :props="props">
          <div>{{ props.row.device_type }} · {{ props.row.browser }}</div>
          <div class="text-caption text-grey-7">{{ props.row.operating_system }}</div>
        </q-td>
      </template>

      <template #body-cell-status="props">
        <q-td :props="props">
          <status-chip
            :label="sessionStatus(props.row).label"
            :color="sessionStatus(props.row).color"
            :text-color="sessionStatus(props.row).textColor"
            :outline="sessionStatus(props.row).outline"
          />
        </q-td>
      </template>

      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="button in line_buttons"
            :key="button.id"
            flat
            dense
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            :disable="loading || (isRevokeAction(button.event_action) && props.row.status !== 'active')"
            @click="handleAction(button.event_action, props.row)"
          >
            <q-tooltip>{{ button.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template #bottom>
        <q-space />
        <table-pagination v-model:page="page" v-model:pageSize="pageSize" :total="total" />
      </template>
    </q-table>

    <form-dialog-shell
      v-model="detailVisible"
      title="登录会话详情"
      :subtitle="selectedSession ? `${selectedSession.user_name} · 会话 ${selectedSession.id}` : ''"
      icon="devices"
      readonly
      width="min(900px, calc(100vw - 48px))"
    >
      <div v-if="selectedSession" class="session-detail">
        <section>
          <div class="session-detail__title">会话状态</div>
          <detail-field-grid :items="statusDetailItems" />
        </section>
        <q-separator />
        <section>
          <div class="session-detail__title">客户端信息</div>
          <detail-field-grid :items="clientDetailItems" />
        </section>
        <template v-if="selectedSession.logout_at || selectedSession.logout_reason">
          <q-separator />
          <section>
            <div class="session-detail__title">结束记录</div>
            <detail-field-grid :items="closureDetailItems" />
          </section>
        </template>
      </div>
    </form-dialog-shell>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_online_session' })

import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import BaseContent from 'components/BaseContent/BaseContent.vue'
import StandardTableToolbar from 'components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'components/Table/TableColumnSelector.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import SweetDateTimePicker from 'components/DateTime/SweetDateTimePicker.vue'
import FormDialogShell from 'components/FormDialog/FormDialogShell.vue'
import DetailFieldGrid from 'components/Detail/DetailFieldGrid.vue'
import StatusChip from 'components/Display/StatusChip.vue'
import type { DetailFieldItem } from 'components/Detail/types'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { downloadBlob, parseContentDispositionFilename } from 'src/utils/download'
import {
  useUserSessionApi,
  type UserSession,
  type UserSessionQuery,
  type UserSessionStatusFilter,
} from 'src/api/services/user-session'

const $q = useQuasar()
const { confirmWithReason } = useConfirmDialog($q)
const api = useUserSessionApi()
const { line_buttons, top_buttons } = usePageButtons('system_online_session')

const rows = ref<UserSession[]>([])
const loading = ref(false)
const exporting = ref(false)
const loadError = ref('')
const keyword = ref('')
const status = ref<UserSessionStatusFilter>('online')
const loginStartedAt = ref('')
const loginEndedAt = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const onlineUsers = ref(0)
const onlineSessions = ref(0)
const detailVisible = ref(false)
const selectedSession = ref<UserSession | null>(null)

const statusOptions = [
  { label: '当前在线', value: 'online' },
  { label: '令牌仍有效', value: 'active' },
  { label: '已结束', value: 'closed' },
  { label: '全部会话', value: 'all' },
]

const columns: NonNullable<QTableProps['columns']> = [
  { name: 'user_name', label: '用户', field: 'user_name', align: 'left' },
  { name: 'device', label: '设备', field: 'device_type', align: 'left' },
  { name: 'ip_address', label: 'IP 地址', field: 'ip_address', align: 'left' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'login_at', label: '登录时间', field: 'login_at', align: 'left' },
  { name: 'last_seen_at', label: '最后活动', field: 'last_seen_at', align: 'left' },
  { name: 'expires_at', label: '可刷新至', field: 'expires_at', align: 'left' },
  { name: 'actions', label: '操作', field: 'actions', align: 'center' },
]
const visibleColumns = ref(columns.map((column) => column.name))

const dateRangeInvalid = computed(
  () =>
    Boolean(loginStartedAt.value) &&
    Boolean(loginEndedAt.value) &&
    loginStartedAt.value > loginEndedAt.value,
)
const loginTimeFilterLabel = computed(() =>
  loginStartedAt.value || loginEndedAt.value ? '登录时间已筛选' : '登录时间',
)

const sessionStatus = (row: UserSession) => {
  if (row.online)
    return { label: '在线', color: 'positive', textColor: 'white', outline: false }
  if (row.status === 'active')
    return { label: '离线，令牌仍有效', color: 'orange-9', textColor: '', outline: true }
  const closedStates: Record<string, { label: string; color: string }> = {
    logged_out: { label: '已退出', color: 'grey-7' },
    forced_offline: { label: '已强制下线', color: 'negative' },
    password_changed: { label: '密码变更失效', color: 'negative' },
    account_disabled: { label: '账号停用', color: 'negative' },
    account_deleted: { label: '账号删除', color: 'negative' },
    expired: { label: '已过期', color: 'blue-grey-7' },
  }
  const state = closedStates[row.status] || { label: '已结束', color: 'grey-7' }
  return { ...state, textColor: '', outline: true }
}

const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.query(buildQuery())
    rows.value = response.data?.items || []
    total.value = response.data?.total || 0
    onlineUsers.value = response.data?.online_users || 0
    onlineSessions.value = response.data?.online_sessions || response.data?.online_devices || 0
  } catch {
    rows.value = []
    total.value = 0
    loadError.value = '登录设备加载失败'
  } finally {
    loading.value = false
  }
}

const runSearch = () => {
  if (page.value !== 1) page.value = 1
  else void fetchData()
}

const clearDateRange = () => {
  loginStartedAt.value = ''
  loginEndedAt.value = ''
}

const buildQuery = (overrides: Partial<UserSessionQuery> = {}): UserSessionQuery => ({
  keyword: keyword.value.trim(),
  status: status.value,
  ...(loginStartedAt.value ? { login_started_at: loginStartedAt.value } : {}),
  ...(loginEndedAt.value ? { login_ended_at: loginEndedAt.value } : {}),
  page: page.value,
  num: pageSize.value,
  ...overrides,
})

const isRevokeAction = (action: string) => action === 'revoke' || action === 'revoke_user'

const handleAction = (action: string, row: UserSession) => {
  if (action === 'detail') {
    selectedSession.value = row
    detailVisible.value = true
    return
  }
  if (action === 'revoke') {
    confirmWithReason({
      title: '下线此会话',
      message: `确定让 ${row.user_name} 的当前会话立即退出吗？`,
      reasonLabel: '下线原因',
      defaultReason: '管理员手动下线',
    }).onOk((reason: string) => {
      void revokeSession(row, reason)
    })
  }
  if (action === 'revoke_user') {
    confirmWithReason({
      title: '下线该用户全部会话',
      message: `确定让 ${row.user_name} 的全部登录会话立即退出吗？`,
      reasonLabel: '下线原因',
      defaultReason: '管理员手动下线全部会话',
      color: 'negative',
    }).onOk((reason: string) => {
      void revokeUser(row, reason)
    })
  }
}

const handleTopAction = (action: string) => {
  if (action === 'export') void exportSessions()
}

const exportSessions = async () => {
  exporting.value = true
  try {
    const response = await api.export(buildQuery({ page: 1, num: 10000 }))
    const filename =
      parseContentDispositionFilename(response.headers['content-disposition']) ||
      'login-sessions.csv'
    downloadBlob(response.data, filename)
    $q.notify({ type: 'positive', position: 'top-right', message: '登录会话记录已导出' })
  } finally {
    exporting.value = false
  }
}

const revokeSession = async (row: UserSession, reason: string) => {
  loading.value = true
  try {
    await api.revoke(row.id, reason.trim())
    $q.notify({ type: 'positive', position: 'top-right', message: '该会话已下线' })
    await fetchData()
  } finally {
    loading.value = false
  }
}

const revokeUser = async (row: UserSession, reason: string) => {
  loading.value = true
  try {
    await api.revokeUser(row.user_id, reason.trim())
    $q.notify({ type: 'positive', position: 'top-right', message: '该用户的全部会话已下线' })
    await fetchData()
  } finally {
    loading.value = false
  }
}

const statusDetailItems = computed<DetailFieldItem[]>(() => {
  const row = selectedSession.value
  if (!row) return []
  const state = sessionStatus(row)
  return [
    { label: '用户', value: row.user_name },
    { label: '账号状态', value: row.user_deleted ? '账号已删除' : '正常' },
    {
      label: '会话状态',
      value: state.label,
      chip: true,
      color: state.color,
      textColor: state.textColor,
      outline: state.outline,
    },
    { label: '会话编号', value: row.id },
    { label: '登录时间', value: row.login_at },
    { label: '最后活动', value: row.last_seen_at },
    { label: '可刷新至', value: row.expires_at },
    { label: '登录渠道', value: row.login_channel },
  ]
})

const clientDetailItems = computed<DetailFieldItem[]>(() => {
  const row = selectedSession.value
  if (!row) return []
  return [
    { label: 'IP 地址', value: row.ip_address },
    { label: '设备类型', value: row.device_type },
    { label: '浏览器', value: row.browser },
    { label: '操作系统', value: row.operating_system },
    { label: 'User-Agent', value: row.user_agent, fullWidth: true },
  ]
})

const closureDetailItems = computed<DetailFieldItem[]>(() => {
  const row = selectedSession.value
  if (!row) return []
  return [
    { label: '结束时间', value: row.logout_at || '-' },
    { label: '结束操作人', value: row.closed_by_user_name || '系统' },
    { label: '结束原因', value: row.logout_reason || '-', fullWidth: true },
  ]
})

watch([page, pageSize], () => void fetchData())
onMounted(() => void fetchData())
</script>

<style scoped>
.session-detail {
  display: grid;
  gap: 24px;
  padding: 4px 6px 20px;
}

.session-detail__title {
  margin-bottom: 16px;
  font-size: 16px;
  font-weight: 700;
}
</style>
