<template>
  <q-layout :view="layoutView" class="app-layout" :class="`app-layout--${layoutMode}`">
    <q-header v-if="!isFullscreenRoute" class="app-header">
      <q-toolbar class="app-header__toolbar">
        <q-btn
          class="app-header__menu"
          flat
          dense
          round
          :aria-label="drawerMenuLabel"
          :icon="drawerMenuIcon"
          @click="toggleLeftDrawer()"
        >
          <q-tooltip>{{ drawerMenuLabel }}</q-tooltip>
        </q-btn>
        <breadcrumbs
          class="app-header__breadcrumbs text-weight-bold"
          :show-icon="false"
          v-if="$q.screen.gt.sm"
        />
        <q-space />
        <toolbar-item @open-settings="openThemeSettings" />
      </q-toolbar>
      <div class="app-header__tabs">
        <tag-view />
      </div>
    </q-header>
    <drawer
      v-if="!isFullscreenRoute"
      ref="drawerRef"
      v-model="isDrawerOpen"
      :title="systemName"
      :subtitle="systemDescription"
      :logo="systemLogo"
    />
    <theme-setting v-if="!isFullscreenRoute" ref="settingRef" />
    <q-page-container
      class="app-main"
      :class="{ 'app-main--fullscreen': isFullscreenRoute }"
      style="height: 100vh"
    >
      <router-view v-if="appStore.reload_flag" v-slot="{ Component }">
        <keep-alive :include="keepAliveStore.getKeepAliveList">
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </q-page-container>
    <q-footer v-if="!isFullscreenRoute" class="q-pa-sm">
      <div>Copyright © 2026 {{ systemName }}</div>
    </q-footer>
  </q-layout>
</template>

<script setup lang="ts">
defineOptions({ name: 'MainLayout' })

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import Drawer from '@/components/Drawer/Drawer.vue'
import Breadcrumbs from '@/components/Breadcrumbs/Breadcrumbs.vue'
import ToolbarItem from '@/components/Toolbar/ToolbarItem.vue'
import ThemeSetting from '@/components/Setting/ThemeSetting.vue'
const drawerRef = ref<typeof Drawer | null>(null)
const settingRef = ref<InstanceType<typeof ThemeSetting> | null>(null)
const isDrawerOpen = ref<boolean>(false)
const toggleLeftDrawer = () => {
  drawerRef.value?.toggleDrawer()
}
const openThemeSettings = () => {
  settingRef.value?.toggleSettingPanel()
}
import { useAppStore } from '@/stores/app'
import { useKeepAliveStore } from '@/stores/keep-alive'
import { useThemeStore } from '@/stores/theme'
import { useConfigureStore } from '@/stores/configure'
import TagView from '@/components/TagView/TagView.vue'
import { useNotificationStore } from '@/stores/notification'
import { useUserStore } from '@/stores/user'
import { useSessionRuntimeStore } from '@/stores/session-runtime'
import { useI18n } from 'vue-i18n'

const appStore = useAppStore()
const themeStore = useThemeStore()
const keepAliveStore = useKeepAliveStore()
const configureStore = useConfigureStore()
const notificationStore = useNotificationStore()
const userStore = useUserStore()
const sessionRuntimeStore = useSessionRuntimeStore()
const { t } = useI18n({ useScope: 'global' })
const { layoutMode } = storeToRefs(themeStore)
themeStore.applyPreferences()
const route = useRoute()
const isFullscreenRoute = computed(() => route.meta.fullscreen === true)
const layoutView = computed(() => (layoutMode.value === 'full' ? 'hHh LpR lFr' : 'lHr LpR lFr'))
const drawerMenuIcon = computed(() => {
  if (!isDrawerOpen.value) return 'menu'
  return appStore.is_drawer_mini ? 'keyboard_double_arrow_right' : 'menu_open'
})
const drawerMenuLabel = computed(() => {
  if (!isDrawerOpen.value) return t('layout.expandSidebar')
  return appStore.is_drawer_mini ? t('layout.expandSidebar') : t('layout.collapseSidebar')
})
const systemName = computed(() => configureStore.getSystemName || 'Sweet Admin')
const systemDescription = computed(
  () => configureStore.getSystemDescription || t('layout.defaultDescription'),
)
const systemLogo = computed(() => configureStore.getSystemLogo || '')

const $q = useQuasar()
onMounted(() => {
  void configureStore.fetchConfigure()
  notificationStore.startPolling()
  sessionRuntimeStore.start()
})
watch(
  () => userStore.session_generation,
  () => {
    notificationStore.reset()
    sessionRuntimeStore.reset()
    if (userStore.isLogin) {
      notificationStore.startPolling()
      sessionRuntimeStore.start()
    }
  },
)
onBeforeUnmount(() => {
  notificationStore.reset()
  sessionRuntimeStore.reset()
})
// const rightDrawerOpen = ref(false)

// function headerClassActive (path: string) {
//   if (route.path.startsWith(path)) {
//     return {
//       'navigation-item': true,
//       'text-primary': true
//     }
//   }
//   return { 'navigation-item': true }
// }

// function toggleRightDrawer () {
//   rightDrawerOpen.value = !rightDrawerOpen.value
// }
</script>

<style scoped lang="scss">
:global(:root) {
  // Theme Store 加载用户偏好后会覆盖这些默认颜色。
  --app-primary-soft: rgba(115, 103, 240, 0.08);
  --app-primary-soft-strong: rgba(115, 103, 240, 0.16);
  --app-primary-border: rgba(115, 103, 240, 0.28);
  --app-primary-shadow: rgba(115, 103, 240, 0.2);
}

.app-header {
  --app-header-bg: #ffffff;
  --app-header-surface: #fbfcff;
  --app-header-text: #172033;
  --app-header-muted: #6f7d96;
  --app-header-border: #e2e7f1;
  --app-header-control-bg: #f7f8fc;
  --app-header-control-hover: var(--app-primary-soft);

  color: var(--app-header-text);
  border-top: 3px solid var(--q-primary);
  border-bottom: 1px solid var(--app-header-border);
  background: var(--app-header-bg);
  box-shadow: 0 6px 18px rgba(31, 38, 54, 0.08);
}

.app-header__toolbar {
  min-height: 48px;
  padding: 0 14px 0 10px;
  align-items: center;
  background: var(--app-header-bg);
}

.app-header__menu {
  width: 34px;
  height: 34px;
  margin-right: 6px;
  color: #ffffff;
  border-radius: 8px;
  background: var(--q-primary);

  &:hover {
    background: var(--q-primary);
    box-shadow: 0 5px 12px var(--app-primary-shadow);
  }
}

.app-header__breadcrumbs {
  min-width: 0;
  color: var(--app-header-text);
}

.app-header__tabs {
  min-height: 43px;
  padding: 0 10px;
  display: flex;
  align-items: center;
  border-top: 1px solid var(--app-header-border);
  background: var(--app-header-surface);
}

:global(.body--dark .app-header) {
  --app-header-bg: #171d2b;
  --app-header-surface: #1d2434;
  --app-header-text: #edf1f8;
  --app-header-muted: #9da9bd;
  --app-header-border: rgba(148, 163, 184, 0.22);
  --app-header-control-bg: rgba(255, 255, 255, 0.05);
  --app-header-control-hover: var(--app-primary-soft-strong);

  box-shadow: 0 7px 20px rgba(0, 0, 0, 0.28);
}
</style>
