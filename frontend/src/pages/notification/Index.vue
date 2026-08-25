<template>
  <base-content class="notification-center q-pa-sm column no-wrap">
    <div class="notification-center__surface column no-wrap">
      <standard-table-toolbar :refreshing="loading" @refresh="loadPage">
        <template #quick-search>
          <q-btn-toggle
            v-model="readStatus"
            dense
            no-caps
            unelevated
            toggle-color="primary"
            :options="readStatusOptions"
            @update:model-value="search"
          />
          <q-input
            v-model="keyword"
            dense
            outlined
            clearable
            debounce="250"
            placeholder="搜索通知标题或内容"
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
            label="通知类型"
            class="notification-center__category"
            @update:model-value="search"
          />
          <q-btn color="primary" icon="search" label="查询" @click="search" />
        </template>
        <template #right-actions>
          <q-btn
            outline
            color="primary"
            icon="done_all"
            label="全部已读"
            :disable="notificationStore.unreadCount === 0"
            @click="markAllRead"
          />
        </template>
      </standard-table-toolbar>

      <q-separator class="q-mt-sm" />
      <div v-if="loading" class="notification-center__state q-pa-md q-gutter-sm">
        <q-skeleton v-for="index in 6" :key="index" type="QToolbar" />
      </div>
      <div v-else-if="error" class="notification-center__state column flex-center q-gutter-md">
        <q-icon name="cloud_off" color="grey-6" size="42px" />
        <div class="text-grey-7">{{ error }}</div>
        <q-btn outline color="primary" icon="refresh" label="重试" @click="loadPage" />
      </div>
      <div
        v-else-if="items.length === 0"
        class="notification-center__state column flex-center q-gutter-sm"
      >
        <q-icon name="notifications_none" color="grey-5" size="48px" />
        <div class="text-body1 text-grey-7">暂无符合条件的通知</div>
      </div>
      <q-list v-else separator class="notification-center__list col">
        <q-item
          v-for="item in items"
          :key="item.id"
          clickable
          class="notification-center__item"
          :class="{ 'notification-center__item--unread': !item.read }"
          @click="openNotification(item)"
        >
          <q-item-section avatar>
            <q-avatar :color="levelColor(item.level)" text-color="white" size="36px">
              <q-icon :name="levelIcon(item.level)" size="20px" />
            </q-avatar>
          </q-item-section>
          <q-item-section>
            <div class="row items-center no-wrap q-gutter-sm">
              <div class="ellipsis text-subtitle2" :class="{ 'text-weight-bold': !item.read }">
                {{ item.title }}
              </div>
              <status-chip
                :label="categoryLabel(item.category)"
                color="primary"
                class="notification-center__category-chip"
              />
            </div>
            <q-item-label caption lines="2" class="q-mt-xs">{{
              item.content_preview
            }}</q-item-label>
          </q-item-section>
          <q-item-section side class="notification-center__meta">
            <span class="text-caption text-grey-7">{{ formatTime(item.created_at) }}</span>
            <q-icon v-if="item.action" name="open_in_new" color="primary" size="18px">
              <q-tooltip>{{
                item.action.available ? '打开相关页面' : '当前无目标页面权限'
              }}</q-tooltip>
            </q-icon>
          </q-item-section>
        </q-item>
      </q-list>

      <div class="notification-center__pagination row justify-end q-pa-sm">
        <table-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-size-options="[15, 20, 30, 50]"
        />
      </div>
    </div>

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
          <q-btn v-close-popup flat round dense icon="close" aria-label="关闭通知详情" />
        </q-card-section>
        <q-separator />
        <q-card-section class="notification-center__content">{{ detail?.content }}</q-card-section>
        <q-separator />
        <q-card-actions align="right">
          <q-btn v-close-popup flat label="关闭" />
          <q-btn
            v-if="detail?.action"
            color="primary"
            icon="open_in_new"
            label="前往相关页面"
            :disable="!detail.action.available || !detail.action.path"
            @click="navigateToAction"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'notification_center' })

import { onMounted, ref, watch } from 'vue'
import { date, useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import {
  NOTIFICATION_CATEGORY_LABELS,
  NOTIFICATION_LEVEL_COLORS,
  useNotificationApi,
  type NotificationCategory,
  type NotificationDetail,
  type NotificationLevel,
  type NotificationReadStatus,
  type NotificationSummary,
} from 'src/api/services/notification'
import { useNotificationStore } from 'src/stores/notification'

const api = useNotificationApi()
const notificationStore = useNotificationStore()
const router = useRouter()
const $q = useQuasar()
const page = ref(1)
const pageSize = ref(15)
const total = ref(0)
const keyword = ref('')
const readStatus = ref<NotificationReadStatus>('ALL')
const category = ref<NotificationCategory | null>(null)
const items = ref<NotificationSummary[]>([])
const loading = ref(false)
const error = ref('')
const detail = ref<NotificationDetail | null>(null)
const showDetail = ref(false)

const readStatusOptions = [
  { label: '全部', value: 'ALL' },
  { label: '未读', value: 'UNREAD' },
  { label: '已读', value: 'READ' },
]
const categoryOptions = Object.entries(NOTIFICATION_CATEGORY_LABELS).map(([value, label]) => ({
  value,
  label,
}))
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
    error.value = caught instanceof Error ? caught.message : '通知列表加载失败'
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
    $q.notify({ type: 'warning', message: '当前无权访问目标页面' })
    return
  }
  const resolved = router.resolve(action.path)
  const missing = resolved.matched.some((route) => route.path.includes(':catchAll'))
  if (!resolved.matched.length || missing) {
    $q.notify({ type: 'warning', message: '目标页面不存在或暂不可用' })
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

.notification-center__surface {
  min-height: 0;
  height: 100%;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.notification-center__surface > :first-child {
  padding: 10px 12px 0;
}

.notification-center__keyword {
  width: 260px;
}

.notification-center__category {
  width: 150px;
}

.notification-center__list {
  min-height: 0;
  overflow-y: auto;
}

.notification-center__item {
  min-height: 84px;
  padding: 12px 16px;
}

.notification-center__item--unread {
  background: var(--app-primary-soft);
}

.notification-center__category-chip {
  flex: none;
}

.notification-center__meta {
  min-width: 150px;
  align-items: flex-end;
  gap: 8px;
}

.notification-center__state {
  min-height: 280px;
}

.notification-center__pagination {
  flex: none;
  border-top: 1px solid var(--app-border);
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
