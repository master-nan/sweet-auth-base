<template>
  <div class="notification-trigger">
    <q-btn
      class="notification-trigger__button"
      round
      dense
      flat
      icon="notifications"
      aria-label="通知"
    >
      <q-badge v-if="store.unreadCount > 0" color="red" floating>
        {{ notificationBadge }}
      </q-badge>
      <q-menu
        class="notification-popover"
        anchor="bottom right"
        self="top right"
        :offset="[0, 8]"
        @before-show="openPopover"
      >
        <div class="notification-popover__header row items-center no-wrap">
          <div>
            <div class="text-subtitle1 text-weight-medium">通知</div>
            <div class="text-caption text-grey-7">{{ unreadSummary }}</div>
          </div>
          <q-space />
          <q-btn
            flat
            dense
            no-caps
            color="primary"
            label="全部已读"
            :disable="store.unreadCount === 0"
            @click="markAllRead"
          />
        </div>
        <q-separator />

        <div class="notification-popover__body">
          <div v-if="store.loading" class="q-pa-md q-gutter-sm">
            <q-skeleton v-for="index in 4" :key="index" type="QToolbar" />
          </div>
          <div
            v-else-if="store.error"
            class="notification-popover__state column flex-center q-gutter-sm"
          >
            <q-icon name="cloud_off" size="32px" color="grey-6" />
            <div class="text-caption text-grey-7">{{ store.error }}</div>
            <q-btn outline dense color="primary" icon="refresh" label="重试" @click="retry" />
          </div>
          <div
            v-else-if="store.recentItems.length === 0"
            class="notification-popover__state column flex-center q-gutter-sm"
          >
            <q-icon name="notifications_none" size="36px" color="grey-5" />
            <div class="text-body2 text-grey-7">暂无通知</div>
          </div>
          <q-list v-else separator>
            <q-item
              v-for="item in store.recentItems"
              :key="item.id"
              clickable
              class="notification-popover__item"
              :class="{ 'notification-popover__item--unread': !item.read }"
              @click="openNotification(item)"
            >
              <q-item-section avatar top>
                <q-icon :name="levelIcon(item.level)" :color="levelColor(item.level)" size="22px" />
              </q-item-section>
              <q-item-section>
                <q-item-label class="ellipsis text-weight-medium">{{ item.title }}</q-item-label>
                <q-item-label caption lines="2">{{ item.content_preview }}</q-item-label>
                <div class="row items-center q-gutter-xs q-mt-xs text-caption text-grey-7">
                  <span>{{ categoryLabel(item.category) }}</span>
                  <span>·</span>
                  <span>{{ formatTime(item.created_at) }}</span>
                </div>
              </q-item-section>
              <q-item-section v-if="!item.read" side top>
                <span class="notification-popover__unread-dot" aria-label="未读" />
              </q-item-section>
            </q-item>
          </q-list>
        </div>

        <q-separator />
        <div class="q-pa-sm">
          <q-btn
            v-close-popup
            flat
            no-caps
            class="full-width"
            color="primary"
            label="查看全部通知"
            @click="viewAll"
          />
        </div>
      </q-menu>
      <q-tooltip>通知</q-tooltip>
    </q-btn>

    <q-dialog v-model="showDetail">
      <q-card class="notification-detail-dialog">
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
        <q-card-section class="notification-detail-dialog__content">{{
          detail?.content
        }}</q-card-section>
      </q-card>
    </q-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuasar, date } from 'quasar'
import { useRouter } from 'vue-router'
import {
  NOTIFICATION_CATEGORY_LABELS,
  NOTIFICATION_LEVEL_COLORS,
  useNotificationApi,
  type NotificationCategory,
  type NotificationDetail,
  type NotificationLevel,
  type NotificationSummary,
} from 'src/api/services/notification'
import { useNotificationStore } from 'src/stores/notification'

const store = useNotificationStore()
const api = useNotificationApi()
const router = useRouter()
const $q = useQuasar()
const showDetail = ref(false)
const detail = ref<NotificationDetail | null>(null)

const unreadSummary = computed(() =>
  store.unreadCount > 0 ? `${store.unreadCount} 条未读` : '没有未读通知',
)
const notificationBadge = computed(() =>
  store.unreadCount > 99 ? '99+' : String(store.unreadCount),
)
const categoryLabel = (category: NotificationCategory) => NOTIFICATION_CATEGORY_LABELS[category]
const levelColor = (level: NotificationLevel) => NOTIFICATION_LEVEL_COLORS[level]
const levelIcon = (level: NotificationLevel) =>
  ({ INFO: 'info', SUCCESS: 'check_circle', WARNING: 'warning', ERROR: 'error' })[level]
const formatTime = (value: string) => date.formatDate(value, 'YYYY-MM-DD HH:mm')

const openPopover = () => {
  void Promise.all([store.refreshUnreadCount(), store.loadRecent()])
}
const retry = () => {
  void store.loadRecent()
}
const markAllRead = () => {
  void store.markAllRead()
}
const viewAll = () => {
  void router.push({ name: 'notification_center' })
}

const openDetail = async (id: number, marked?: NotificationDetail | null) => {
  try {
    detail.value = marked || (await api.detail(id)).data || null
    showDetail.value = !!detail.value
  } catch {
    $q.notify({ type: 'negative', message: '通知详情加载失败' })
  }
}

const openNotification = async (item: NotificationSummary) => {
  const marked = await store.markRead(item.id)
  if (item.action?.available && item.action.path) {
    const resolved = router.resolve(item.action.path)
    const missing = resolved.matched.some((route) => route.path.includes(':catchAll'))
    if (resolved.matched.length && !missing) {
      await router.push(item.action.path)
      return
    }
    $q.notify({ type: 'warning', message: '目标页面不存在或暂不可用' })
  } else if (item.action && !item.action.available) {
    $q.notify({ type: 'warning', message: '当前无权访问目标页面' })
  }
  await openDetail(item.id, marked)
}
</script>

<style scoped lang="scss">
.notification-trigger {
  width: 34px;
  height: 34px;
}

.notification-trigger__button {
  width: 34px;
  height: 34px;
  color: var(--app-header-text);
  border: 1px solid var(--app-header-border);
  border-radius: 8px;
  background: var(--app-header-control-bg);

  &:hover {
    color: var(--q-primary);
    background: var(--app-header-control-hover);
  }
}

.notification-popover {
  width: min(400px, calc(100vw - 24px));
  color: var(--app-text-strong);
  background: var(--app-surface);
}

.notification-popover__header {
  min-height: 62px;
  padding: 10px 14px;
}

.notification-popover__body {
  min-height: 220px;
  max-height: 420px;
  overflow-y: auto;
}

.notification-popover__state {
  min-height: 220px;
  padding: 24px;
  text-align: center;
}

.notification-popover__item {
  min-height: 78px;
  padding: 10px 14px;
}

.notification-popover__item--unread {
  background: var(--app-primary-soft);
}

.notification-popover__unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--q-primary);
}

.notification-detail-dialog {
  width: min(560px, calc(100vw - 32px));
}

.notification-detail-dialog__content {
  min-height: 140px;
  max-height: 55vh;
  overflow-y: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
