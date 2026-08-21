<template>
  <q-layout
    :view="layoutView"
    class="app-layout"
    :class="`app-layout--${layoutMode}`"
  >
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
        <breadcrumbs class="app-header__breadcrumbs text-weight-bold" :show-icon="false" v-if="$q.screen.gt.sm" />
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
    <q-page-container class="app-main" :class="{ 'app-main--fullscreen': isFullscreenRoute }" style="height: 100vh">
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

import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import Drawer from 'src/components/Drawer/Drawer.vue'
import Breadcrumbs from 'src/components/Breadcrumbs/Breadcrumbs.vue'
import ToolbarItem from 'src/components/Toolbar/ToolbarItem.vue'
import ThemeSetting from 'src/components/Setting/ThemeSetting.vue'
const drawerRef = ref<typeof Drawer | null>(null)
const settingRef = ref<InstanceType<typeof ThemeSetting> | null>(null)
const isDrawerOpen = ref<boolean>(false)
const toggleLeftDrawer = () => {
  drawerRef.value?.toggleDrawer()
}
const openThemeSettings = () => {
  settingRef.value?.toggleSettingPanel()
}
import { useAppStore } from 'src/stores/app'
import { useKeepAliveStore } from 'src/stores/keep-alive'
import { useThemeStore } from 'src/stores/theme'
import { useConfigureStore } from 'stores/configure'
import TagView from 'components/TagView/TagView.vue'

const appStore = useAppStore()
const themeStore = useThemeStore()
const keepAliveStore = useKeepAliveStore()
const configureStore = useConfigureStore()
const { layoutMode } = storeToRefs(themeStore)
themeStore.applyPreferences()
const route = useRoute()
const isFullscreenRoute = computed(() => route.meta.fullscreen === true)
const layoutView = computed(() =>
  layoutMode.value === 'full' ? 'hHh LpR lFr' : 'lHr LpR lFr',
)
const drawerMenuIcon = computed(() => {
  if (!isDrawerOpen.value) return 'menu'
  return appStore.is_drawer_mini ? 'keyboard_double_arrow_right' : 'menu_open'
})
const drawerMenuLabel = computed(() => {
  if (!isDrawerOpen.value) return '展开侧栏'
  return appStore.is_drawer_mini ? '展开侧栏' : '收起为图标栏'
})
const systemName = computed(() => configureStore.getSystemName || 'Sweet Admin')
const systemDescription = computed(() => configureStore.getSystemDescription || '通用低代码底座')
const systemLogo = computed(() => configureStore.getSystemLogo || '')

const $q = useQuasar()
onMounted(() => {
  void configureStore.fetchConfigure()
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
  // The theme store overwrites these defaults when preferences are applied.
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
