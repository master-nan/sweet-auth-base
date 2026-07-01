<template>
  <q-layout view="lHr LpR lFr">
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
        <toolbar-item />
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
import Drawer from 'src/components/Drawer/Drawer.vue'
import Breadcrumbs from 'src/components/Breadcrumbs/Breadcrumbs.vue'
import ToolbarItem from 'src/components/Toolbar/ToolbarItem.vue'
import ThemeSetting from 'src/components/Setting/ThemeSetting.vue'
const drawerRef = ref<typeof Drawer | null>(null)
const isDrawerOpen = ref<boolean>(false)
const toggleLeftDrawer = () => {
  drawerRef.value?.toggleDrawer()
}
import { useAppStore } from 'src/stores/app'
import { useKeepAliveStore } from 'src/stores/keep-alive'
import { useConfigureStore } from 'stores/configure'
import TagView from 'components/TagView/TagView.vue'

const appStore = useAppStore()
const keepAliveStore = useKeepAliveStore()
const configureStore = useConfigureStore()
const route = useRoute()
const isFullscreenRoute = computed(() => route.meta.fullscreen === true)
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
.app-header {
  color: #fff;
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.08), rgba(255, 255, 255, 0) 44%),
    linear-gradient(90deg, $primary 0%, #665be7 62%, #5d55db 100%);
  border-bottom: 1px solid rgba(255, 255, 255, 0.18);
  box-shadow: 0 10px 28px rgba(64, 54, 180, 0.18);
}

.app-header__toolbar {
  min-height: 48px;
  padding: 0 14px 0 10px;
}

.app-header__menu {
  width: 34px;
  height: 34px;
  margin-right: 6px;
  color: #fff;
  border-radius: 10px;

  &:hover {
    background: rgba(255, 255, 255, 0.12);
  }
}

.app-header__breadcrumbs {
  min-width: 0;
  color: rgba(255, 255, 255, 0.96);

  :deep(.q-breadcrumbs__el) {
    color: inherit;
    font-size: 14px;
    font-weight: 800;
    letter-spacing: 0;
  }

  :deep(.q-breadcrumbs__separator) {
    color: rgba(255, 255, 255, 0.48);
    margin: 0 9px;
  }
}

.app-header__tabs {
  min-height: 47px;
  padding: 7px 10px 8px;
  display: flex;
  align-items: center;
  border-top: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(44, 35, 170, 0.1);
}
</style>
