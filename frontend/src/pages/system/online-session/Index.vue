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
      :no-data-label="loadError || '暂无符合条件的登录设备'"
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
              在线设备 <strong class="text-dark">{{ onlineDevices }}</strong>
            </div>
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
            <q-chip v-if="props.row.current" dense square color="primary" text-color="white"
              >当前设备</q-chip
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
          <q-chip
            dense
            square
            :color="sessionStatus(props.row).color"
            :text-color="sessionStatus(props.row).textColor"
          >
            {{ sessionStatus(props.row).label }}
          </q-chip>
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
            :disable="loading || props.row.status !== 'active'"
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
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'system_online_session' })

import { onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableProps } from 'quasar'
import BaseContent from 'components/BaseContent/BaseContent.vue'
import StandardTableToolbar from 'components/Table/StandardTableToolbar.vue'
import TableColumnSelector from 'components/Table/TableColumnSelector.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { usePageButtons } from 'src/composables/page-buttons'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import {
  useUserSessionApi,
  type UserSession,
  type UserSessionStatusFilter,
} from 'src/api/services/user-session'

const $q = useQuasar()
const { confirmAction, confirmDanger } = useConfirmDialog($q)
const api = useUserSessionApi()
const { line_buttons } = usePageButtons('system_online_session')

const rows = ref<UserSession[]>([])
const loading = ref(false)
const loadError = ref('')
const keyword = ref('')
const status = ref<UserSessionStatusFilter>('online')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const onlineUsers = ref(0)
const onlineDevices = ref(0)

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

const sessionStatus = (row: UserSession) => {
  if (row.online) return { label: '在线', color: 'positive', textColor: 'white' }
  if (row.status === 'active')
    return { label: '离线，令牌仍有效', color: 'orange-2', textColor: 'orange-10' }
  const labels: Record<string, string> = {
    logged_out: '已退出',
    forced_offline: '已强制下线',
    password_changed: '密码变更失效',
    account_disabled: '账号停用',
    account_deleted: '账号删除',
    expired: '已过期',
  }
  return { label: labels[row.status] || '已结束', color: 'grey-3', textColor: 'grey-9' }
}

const fetchData = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const response = await api.query({
      keyword: keyword.value.trim(),
      status: status.value,
      page: page.value,
      num: pageSize.value,
    })
    rows.value = response.data?.items || []
    total.value = response.data?.total || 0
    onlineUsers.value = response.data?.online_users || 0
    onlineDevices.value = response.data?.online_devices || 0
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

const handleAction = (action: string, row: UserSession) => {
  if (action === 'revoke') {
    confirmAction({
      title: '下线此设备',
      message: `确定让 ${row.user_name} 的这台设备立即退出吗？`,
    }).onOk(() => {
      void revokeSession(row)
    })
  }
  if (action === 'revoke_user') {
    confirmDanger({
      title: '下线全部设备',
      message: `确定让 ${row.user_name} 的全部登录设备立即退出吗？`,
    }).onOk(() => {
      void revokeUser(row)
    })
  }
}

const revokeSession = async (row: UserSession) => {
  loading.value = true
  try {
    await api.revoke(row.id)
    $q.notify({ type: 'positive', position: 'top-right', message: '该设备已下线' })
    await fetchData()
  } finally {
    loading.value = false
  }
}

const revokeUser = async (row: UserSession) => {
  loading.value = true
  try {
    await api.revokeUser(row.user_id)
    $q.notify({ type: 'positive', position: 'top-right', message: '该用户的全部设备已下线' })
    await fetchData()
  } finally {
    loading.value = false
  }
}

watch([page, pageSize], () => void fetchData())
onMounted(() => void fetchData())
</script>
