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
      :pagination="{ rowsPerPage: 0 }"
      hide-pagination
      :no-data-label="loadError || t('ui.unqualifiedLoginSession')"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
          <template #query-controls>
            <q-input
              v-model="keyword"
              dense
              outlined
              debounce="300"
              :placeholder="t('ui.searchForUsernameIpBrowserOrSystem')"
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
                  <div class="text-subtitle2 q-mb-sm">{{ t('ui.loginTimeRange') }}</div>
                  <div class="row q-col-gutter-md">
                    <div class="col-12 col-sm-6">
                      <sweet-date-time-picker
                        v-model="loginStartedAt"
                        type="datetime"
                        :label="t('ui.startTime')"
                      />
                    </div>
                    <div class="col-12 col-sm-6">
                      <sweet-date-time-picker
                        v-model="loginEndedAt"
                        type="datetime"
                        :label="t('ui.endTime')"
                      />
                    </div>
                  </div>
                  <div v-if="dateRangeInvalid" class="text-negative text-caption q-mt-xs">
                    {{ t('ui.theEndTimeCannotBeEarlierThanTheBeginning') }}
                  </div>
                  <div class="row justify-end q-gutter-sm q-mt-md">
                    <q-btn flat color="grey-7" :label="t('ui.clear')" @click="clearDateRange" />
                    <q-btn
                      v-close-popup
                      unelevated
                      color="primary"
                      :label="t('ui.apply')"
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
              :label="t('ui.query')"
              :disable="loading"
              @click="runSearch"
            />
            <q-separator vertical inset class="q-mx-sm" />
            <div class="text-body2 text-grey-7">
              {{ t('ui.onlineUsers') }} <strong class="text-dark">{{ onlineUsers }}</strong>
              <span class="q-mx-xs">·</span> {{ t('ui.onlineSession') }}
              <strong class="text-dark">{{ onlineSessions }}</strong>
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
            <q-chip v-if="props.row.user_deleted" dense square color="grey-4" text-color="grey-9">{{
              t('ui.accountDeleted')
            }}</q-chip>
            <q-chip v-if="props.row.current" dense square color="primary" text-color="white">{{
              t('ui.currentSession')
            }}</q-chip>
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
            size="sm"
            v-bind="menuButtonDisplayProps(button)"
            :color="button.color || 'primary'"
            :disable="
              loading || (isRevokeAction(button.event_action) && props.row.status !== 'active')
            "
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
      :title="t('ui.loginSessionDetails')"
      :subtitle="
        selectedSession
          ? t('ui.sessionSubtitle', { user: selectedSession.user_name, id: selectedSession.id })
          : ''
      "
      icon="devices"
      readonly
      width="min(900px, calc(100vw - 48px))"
    >
      <div v-if="selectedSession" class="session-detail">
        <section>
          <div class="session-detail__title">{{ t('ui.sessionStatus') }}</div>
          <detail-field-grid :items="statusDetailItems" />
        </section>
        <q-separator />
        <section>
          <div class="session-detail__title">{{ t('ui.clientInformation') }}</div>
          <detail-field-grid :items="clientDetailItems" />
        </section>
        <template v-if="selectedSession.logout_at || selectedSession.logout_reason">
          <q-separator />
          <section>
            <div class="session-detail__title">{{ t('ui.endOfRecord') }}</div>
            <detail-field-grid :items="closureDetailItems" />
          </section>
        </template>
      </div>
    </form-dialog-shell>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

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

const { t } = useI18n({ useScope: 'global' })

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
  {
    get label() {
      return t('ui.currentOnline')
    },
    value: 'online',
  },
  {
    get label() {
      return t('ui.validSession')
    },
    value: 'active',
  },
  {
    get label() {
      return t('ui.completed')
    },
    value: 'closed',
  },
  {
    get label() {
      return t('ui.allSessions')
    },
    value: 'all',
  },
]

const columns: NonNullable<QTableProps['columns']> = [
  {
    name: 'user_name',
    get label() {
      return t('ui.user')
    },
    field: 'user_name',
    align: 'left',
  },
  {
    name: 'device',
    get label() {
      return t('ui.equipment')
    },
    field: 'device_type',
    align: 'left',
  },
  {
    name: 'ip_address',
    get label() {
      return t('ui.ipAddress')
    },
    field: 'ip_address',
    align: 'left',
  },
  {
    name: 'status',
    get label() {
      return t('ui.status')
    },
    field: 'status',
    align: 'center',
  },
  {
    name: 'login_at',
    get label() {
      return t('ui.signInTime')
    },
    field: 'login_at',
    align: 'left',
  },
  {
    name: 'last_seen_at',
    get label() {
      return t('ui.lastActive')
    },
    field: 'last_seen_at',
    align: 'left',
  },
  {
    name: 'expires_at',
    get label() {
      return t('ui.refreshableUntil')
    },
    field: 'expires_at',
    align: 'left',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'actions',
    align: 'center',
  },
]
const visibleColumns = ref(columns.map((column) => column.name))

const dateRangeInvalid = computed(
  () =>
    Boolean(loginStartedAt.value) &&
    Boolean(loginEndedAt.value) &&
    loginStartedAt.value > loginEndedAt.value,
)
const loginTimeFilterLabel = computed(() =>
  loginStartedAt.value || loginEndedAt.value ? t('ui.loginTimeFiltered') : t('ui.signInTime'),
)

const sessionStatus = (row: UserSession) => {
  if (row.online)
    return {
      get label() {
        return t('ui.online')
      },
      color: 'positive',
      textColor: 'white',
      outline: false,
    }
  if (row.status === 'active')
    return {
      get label() {
        return t('ui.effectiveNotCurrentlyOnline')
      },
      color: 'orange-9',
      textColor: '',
      outline: true,
    }
  const closedStates: Record<string, { label: string; color: string }> = {
    logged_out: {
      get label() {
        return t('ui.quitped')
      },
      color: 'grey-7',
    },
    forced_offline: {
      get label() {
        return t('ui.forcedOffline')
      },
      color: 'negative',
    },
    password_changed: {
      get label() {
        return t('ui.passwordChangeLapsed')
      },
      color: 'negative',
    },
    account_disabled: {
      get label() {
        return t('ui.accountDisabled')
      },
      color: 'negative',
    },
    account_deleted: {
      get label() {
        return t('ui.accountDelete')
      },
      color: 'negative',
    },
    expired: {
      get label() {
        return t('ui.expired')
      },
      color: 'blue-grey-7',
    },
  }
  const state = closedStates[row.status] || {
    get label() {
      return t('ui.completed')
    },
    color: 'grey-7',
  }
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
    loadError.value = t('ui.loginDeviceLoadedFailed')
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
      get title() {
        return t('ui.signOutThisSession')
      },
      get message() {
        return t('ui.areYouSureYouWantToQuitTheCurrentSessionImmediately', {
          value1: row.user_name,
        })
      },
      get reasonLabel() {
        return t('ui.signOutReason')
      },
      get defaultReason() {
        return t('ui.managerManuallyOffline')
      },
    }).onOk((reason: string) => {
      void revokeSession(row, reason)
    })
  }
  if (action === 'revoke_user') {
    confirmWithReason({
      get title() {
        return t('ui.signOutAllSessionsForThisUser')
      },
      get message() {
        return t('ui.signOutAllLoginSessionsForImmediately', { value1: row.user_name })
      },
      get reasonLabel() {
        return t('ui.signOutReason')
      },
      get defaultReason() {
        return t('ui.allSessionsManuallyDownlinedByAdministrator')
      },
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
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.loginSessionRecordExported')
      },
    })
  } finally {
    exporting.value = false
  }
}

const revokeSession = async (row: UserSession, reason: string) => {
  loading.value = true
  try {
    await api.revoke(row.id, reason.trim())
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.theSessionIsOffline')
      },
    })
    await fetchData()
  } finally {
    loading.value = false
  }
}

const revokeUser = async (row: UserSession, reason: string) => {
  loading.value = true
  try {
    await api.revokeUser(row.user_id, reason.trim())
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.allSessionsOfTheUserAreOffline')
      },
    })
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
    {
      get label() {
        return t('ui.user')
      },
      value: row.user_name,
    },
    {
      get label() {
        return t('ui.accountStatus')
      },
      value: row.user_deleted ? t('ui.accountDeleted') : t('ui.normalStatus'),
    },
    {
      get label() {
        return t('ui.sessionStatus')
      },
      value: state.label,
      chip: true,
      color: state.color,
      textColor: state.textColor,
      outline: state.outline,
    },
    {
      get label() {
        return t('ui.sessionNumber')
      },
      value: row.id,
    },
    {
      get label() {
        return t('ui.signInTime')
      },
      value: row.login_at,
    },
    {
      get label() {
        return t('ui.lastActive')
      },
      value: row.last_seen_at,
    },
    {
      get label() {
        return t('ui.refreshableUntil')
      },
      value: row.expires_at,
    },
    {
      get label() {
        return t('ui.loginChannel')
      },
      value: row.login_channel,
    },
  ]
})

const clientDetailItems = computed<DetailFieldItem[]>(() => {
  const row = selectedSession.value
  if (!row) return []
  return [
    {
      get label() {
        return t('ui.ipAddress')
      },
      value: row.ip_address,
    },
    {
      get label() {
        return t('ui.deviceType')
      },
      value: row.device_type,
    },
    {
      get label() {
        return t('ui.browser')
      },
      value: row.browser,
    },
    {
      get label() {
        return t('ui.operatingSystems')
      },
      value: row.operating_system,
    },
    { label: 'User-Agent', value: row.user_agent, fullWidth: true },
  ]
})

const closureDetailItems = computed<DetailFieldItem[]>(() => {
  const row = selectedSession.value
  if (!row) return []
  return [
    {
      get label() {
        return t('ui.endTime')
      },
      value: row.logout_at || '-',
    },
    {
      get label() {
        return t('ui.endOperator')
      },
      value: row.closed_by_user_name || t('ui.system'),
    },
    {
      get label() {
        return t('ui.endReason')
      },
      value: row.logout_reason || '-',
      fullWidth: true,
    },
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
