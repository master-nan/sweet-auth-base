<template>
  <base-content class="notification-center q-pa-sm">
    <q-table
      class="notification-center__table fit sticky-header-table"
      :class="{
        'app-table--empty': !loading && (Boolean(error) || items.length === 0),
      }"
      color="primary"
      flat
      bordered
      separator="cell"
      row-key="id"
      :dense="$q.screen.lt.md"
      :rows="error ? [] : items"
      :columns="columns"
      :loading="loading"
      :pagination="tableState"
    >
      <template #top>
        <standard-table-toolbar :refreshing="loading" @refresh="loadPage">
          <template #quick-search>
            <q-tabs
              v-model="readStatus"
              class="notification-center__read-tabs"
              dense
              no-caps
              narrow-indicator
              active-color="primary"
              indicator-color="primary"
              @update:model-value="search"
            >
              <q-tab name="ALL" :label="t('ui.all')" />
              <q-tab name="UNREAD" :label="t('ui.unread')" />
              <q-tab name="READ" :label="t('ui.read')" />
            </q-tabs>
            <q-input
              v-model="keyword"
              dense
              outlined
              clearable
              debounce="250"
              :placeholder="t('ui.searchForNotificationTitleOrContent')"
              class="notification-center__keyword"
              @keyup.enter="search"
            >
              <template #append><q-icon name="search" /></template>
            </q-input>
            <q-select
              v-model="category"
              dense
              outlined
              clearable
              emit-value
              map-options
              :options="categoryOptions"
              :label="t('ui.typeOfNotification')"
              class="notification-center__category"
              @update:model-value="search"
            />
            <q-btn color="primary" icon="search" :label="t('ui.query')" @click="search" />
          </template>
          <template #right-actions>
            <q-btn
              outline
              color="primary"
              icon="done_all"
              :label="t('ui.markAllAsRead')"
              :disable="notificationStore.unreadCount === 0"
              @click="markAllRead"
            />
          </template>
        </standard-table-toolbar>
      </template>

      <template #body="props">
        <q-tr
          :props="props"
          data-notification-row
          class="notification-center__row cursor-pointer"
          :class="{ 'notification-center__row--unread': !props.row.read }"
          tabindex="0"
          @click="openNotification(props.row)"
          @keyup.enter="openNotification(props.row)"
          @keyup.space.prevent="openNotification(props.row)"
        >
          <q-td key="message" :props="props">
            <div class="notification-center__message row items-center no-wrap">
              <q-icon
                :name="levelIcon(props.row.level)"
                :color="levelColor(props.row.level)"
                size="22px"
                class="notification-center__level-icon"
              />
              <div class="notification-center__message-body col">
                <div class="row items-center no-wrap q-gutter-xs">
                  <span
                    v-if="!props.row.read"
                    class="notification-center__unread-dot"
                    :aria-label="t('ui.unread')"
                  />
                  <span
                    class="ellipsis"
                    :class="!props.row.read ? 'text-weight-bold' : 'text-weight-medium'"
                  >
                    {{ props.row.title }}
                  </span>
                  <q-icon v-if="props.row.action" name="open_in_new" color="primary" size="16px">
                    <q-tooltip>
                      {{
                        props.row.action.available
                          ? t('ui.openRelatedPageCapability')
                          : t('ui.currentlyNoTargetPagePermissions')
                      }}
                    </q-tooltip>
                  </q-icon>
                </div>
                <div class="notification-center__preview ellipsis text-caption q-mt-xs">
                  {{ props.row.content_preview }}
                </div>
              </div>
            </div>
          </q-td>
          <q-td key="category" :props="props">
            <status-chip :label="categoryLabel(props.row.category)" color="primary" />
          </q-td>
          <q-td key="read" :props="props">
            <status-chip
              :label="props.row.read ? t('ui.read') : t('ui.unread')"
              :color="props.row.read ? 'grey-7' : 'primary'"
            />
          </q-td>
          <q-td key="created_at" :props="props">
            <span class="text-caption text-no-wrap">{{ formatTime(props.row.created_at) }}</span>
          </q-td>
          <q-td key="actions" :props="props">
            <q-btn
              flat
              round
              dense
              color="primary"
              icon="visibility"
              :aria-label="t('ui.viewNotifications')"
              @click.stop="openNotification(props.row)"
            >
              <q-tooltip>{{ t('ui.viewNotifications') }}</q-tooltip>
            </q-btn>
          </q-td>
        </q-tr>
      </template>

      <template #no-data>
        <div class="app-table-empty-state full-width column flex-center q-gutter-sm">
          <template v-if="error">
            <q-icon name="cloud_off" color="grey-6" size="42px" />
            <div class="text-grey-7">{{ error }}</div>
            <q-btn
              outline
              color="primary"
              icon="refresh"
              :label="t('ui.retry')"
              @click="loadPage"
            />
          </template>
          <template v-else>
            <q-icon name="inbox" color="grey-5" size="48px" />
            <div class="text-body1 text-grey-7">{{ t('ui.notEligibleForNotice') }}</div>
          </template>
        </div>
      </template>

      <template #bottom>
        <div class="full-width row items-center no-wrap">
          <div class="text-caption text-grey-7">
            {{ t('ui.total') }} {{ total }} {{ t('ui.articleNotifications') }}
          </div>
          <q-space />
          <table-pagination v-model:page="page" v-model:page-size="pageSize" :total="total" />
        </div>
      </template>
    </q-table>

    <q-dialog v-model="showDetail">
      <q-card class="notification-center__dialog">
        <q-card-section class="row items-start no-wrap">
          <div class="col">
            <div class="text-h6">{{ detail?.title }}</div>
            <div class="text-caption text-grey-7 q-mt-xs">
              {{ detail ? categoryLabel(detail.category) : '' }} ·
              {{ detail ? formatTime(detail.created_at) : '' }}
            </div>
          </div>
          <q-btn
            v-close-popup
            flat
            round
            dense
            icon="close"
            :aria-label="t('ui.closeNotificationDetails')"
          />
        </q-card-section>
        <q-separator />
        <q-card-section class="notification-center__content">{{ detail?.content }}</q-card-section>
        <q-separator />
        <q-card-actions align="right">
          <q-btn v-close-popup flat :label="t('ui.close')" />
          <q-btn
            v-if="detail?.action"
            color="primary"
            icon="open_in_new"
            :label="t('ui.goToTheRelevantPage')"
            :disable="!detail.action.available || !detail.action.path"
            @click="navigateToAction"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'notification_center' })

import { onMounted, ref, watch } from 'vue'
import { date, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from '@/components/BaseContent/BaseContent.vue'
import StandardTableToolbar from '@/components/Table/StandardTableToolbar.vue'
import TablePagination from '@/components/Table/TablePagination.vue'
import StatusChip from '@/components/Display/StatusChip.vue'
import {
  NOTIFICATION_CATEGORY_LABELS,
  NOTIFICATION_LEVEL_COLORS,
  useNotificationApi,
  type NotificationCategory,
  type NotificationDetail,
  type NotificationLevel,
  type NotificationReadStatus,
  type NotificationSummary,
} from '@/api/services/notification'
import { useNotificationStore } from '@/stores/notification'
import type { TableColumn } from '@/types/global'

const { t } = useI18n({ useScope: 'global' })

const api = useNotificationApi()
const notificationStore = useNotificationStore()
const router = useRouter()
const $q = useQuasar()
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keyword = ref('')
const readStatus = ref<NotificationReadStatus>('ALL')
const category = ref<NotificationCategory | null>(null)
const items = ref<NotificationSummary[]>([])
const loading = ref(false)
const error = ref('')
const detail = ref<NotificationDetail | null>(null)
const showDetail = ref(false)
const tableState = { rowsPerPage: 0 }

const categoryOptions = Object.entries(NOTIFICATION_CATEGORY_LABELS).map(([value, label]) => ({
  value,
  label,
}))
const columns: TableColumn<NotificationSummary>[] = [
  {
    name: 'message',
    get label() {
      return t('ui.contentsOfTheNotice')
    },
    field: 'title',
    align: 'left',
  },
  {
    name: 'category',
    get label() {
      return t('ui.type')
    },
    field: 'category',
    align: 'center',
    headerStyle: 'width: 110px',
  },
  {
    name: 'read',
    get label() {
      return t('ui.status')
    },
    field: 'read',
    align: 'center',
    headerStyle: 'width: 90px',
  },
  {
    name: 'created_at',
    get label() {
      return t('ui.time')
    },
    field: 'created_at',
    align: 'left',
    headerStyle: 'width: 180px',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'id',
    align: 'center',
    headerStyle: 'width: 72px',
  },
]
const categoryLabel = (value: NotificationCategory) => NOTIFICATION_CATEGORY_LABELS[value]
const levelColor = (value: NotificationLevel) => NOTIFICATION_LEVEL_COLORS[value]
const levelIcon = (value: NotificationLevel) =>
  ({ INFO: 'info', SUCCESS: 'check_circle', WARNING: 'warning', ERROR: 'error' })[value]
const formatTime = (value: string) => date.formatDate(value, 'YYYY-MM-DD HH:mm:ss')

const loadPage = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.query({
      page: page.value,
      num: pageSize.value,
      keyword: keyword.value.trim(),
      read_status: readStatus.value,
      ...(category.value ? { category: category.value } : {}),
    })
    items.value = response.data || []
    total.value = response.total || 0
  } catch (caught) {
    error.value =
      caught instanceof Error ? caught.message : t('ui.notificationListLoadFailed')
  } finally {
    loading.value = false
  }
}
const search = () => {
  if (page.value !== 1) page.value = 1
  else void loadPage()
}
const markAllRead = async () => {
  if (await notificationStore.markAllRead()) void loadPage()
}
const openNotification = async (item: NotificationSummary) => {
  detail.value = await notificationStore.markRead(item.id)
  if (!detail.value) return
  showDetail.value = true
  const current = items.value.find((value) => value.id === item.id)
  if (current) Object.assign(current, detail.value)
}
const navigateToAction = async () => {
  const action = detail.value?.action
  if (!action?.available || !action.path) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.youDoNotHaveAccessToTheTargetPage')
      },
    })
    return
  }
  const resolved = router.resolve(action.path)
  const missing = resolved.matched.some((route) => route.path.includes(':catchAll'))
  if (!resolved.matched.length || missing) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.theTargetPageDoesNotExistOrIsTemporarilyUnavailable')
      },
    })
    return
  }
  showDetail.value = false
  await router.push(action.path)
}

watch([page, pageSize], () => void loadPage())
onMounted(() => {
  void notificationStore.refreshUnreadCount()
  void loadPage()
})
</script>

<style scoped lang="scss">
.notification-center {
  min-height: 0;
}

.notification-center__table :deep(.q-table__top) {
  padding: 10px 12px;
}

.notification-center__read-tabs {
  flex: none;
  min-width: 184px;
  min-height: 40px;
}

.notification-center__read-tabs :deep(.q-tab) {
  min-height: 40px;
  padding: 0 14px;
  color: var(--app-text-muted);
  font-weight: 500;
}

.notification-center__read-tabs :deep(.q-tab--active) {
  font-weight: 600;
}

.notification-center__keyword {
  width: 280px;
}

.notification-center__category {
  width: 150px;
}

.notification-center__row {
  height: 68px;
}

.notification-center__row--unread > td {
  background: var(--app-primary-soft);
}

.notification-center__message {
  min-width: 0;
  max-width: 760px;
}

.notification-center__message-body {
  min-width: 0;
}

.notification-center__level-icon {
  flex: none;
  margin-right: 12px;
}

.notification-center__unread-dot {
  width: 7px;
  height: 7px;
  flex: none;
  border-radius: 50%;
  background: var(--q-primary);
}

.notification-center__preview {
  color: var(--app-text-muted);
}

.notification-center__dialog {
  width: min(620px, calc(100vw - 32px));
}

.notification-center__content {
  min-height: 180px;
  max-height: 58vh;
  overflow-y: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
