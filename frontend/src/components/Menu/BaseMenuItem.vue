<template>
  <template v-for="(item, index) in myRouter">
    <div class="base-menu-item" :key="index" v-if="item.meta?.isHidden !== true">
      <q-item-label
        v-if="item.meta?.itemLabel"
        header
        class="text-weight-bold text-uppercase"
        :key="item.meta.itemLabel as string"
        >{{ item.meta.itemLabel }}</q-item-label
      >
      <q-item
        v-if="!item.children"
        :exact="item.path === '/'"
        clickable
        v-ripple
        :inset-level="menuInsetLevel"
        active-class="baseItemActive"
        :active="isMenuItemActive(basePath as string, item)"
        @click="handleMenuClick(basePath as string, item)"
      >
        <q-item-section avatar>
          <q-icon :name="item.meta?.icon as string" size="xs" />
        </q-item-section>
        <q-item-section v-if="!isMiniMode">
          {{ item.meta ? displayTitle(item.meta?.title) : '' }}
        </q-item-section>
        <q-item-section v-if="!isMiniMode && handleLink(basePath as string, item) === '#'" side>
          <q-icon name="fa-solid fa-up-right-from-square" size="10px" />
        </q-item-section>
        <q-tooltip v-if="isMiniMode" anchor="center right" self="center left" :offset="[10, 0]">
          {{ item.meta ? displayTitle(item.meta?.title) : item.name }}
        </q-tooltip>
      </q-item>
      <template v-else>
        <q-expansion-item
          v-if="!isMiniMode"
          class="base-menu-expansion"
          :class="{
            'base-menu-expansion--active': isRouteInBranch(basePath as string, item),
          }"
          :duration="duration"
          :default-opened="item.meta?.isOpen as boolean"
          :header-style="expansionHeaderStyle(basePath as string, item)"
          :header-inset-level="menuInsetLevel"
          :icon="item.meta?.icon as string"
          :label="item.meta ? displayTitle(item.meta?.title) : ''"
          :model-value="expansionItemOpen(basePath as string, item)"
        >
          <!-- Quasar 用小数 inset 层级控制子菜单缩进；递归时同时拼接父路由。 -->
          <base-menu-item
            :my-router="item.children"
            :init-level="(initLevel as number) + 0.2"
            :layer-level="(layerLevel as number) + 1"
            :base-path="basePath === '' ? item.path : basePath + '/' + item.path"
            :duration="duration"
            :force-expand="forceExpand"
          />
        </q-expansion-item>
        <q-item v-else :style="expansionHeaderStyle(basePath as string, item)" clickable v-ripple>
          <q-item-section avatar>
            <q-icon :name="item.meta?.icon as string" size="xs" />
          </q-item-section>
          <q-item-section v-if="!isMiniMode">
            {{ item.meta ? displayTitle(item.meta?.title) : '' }}
          </q-item-section>
          <q-item-section v-if="!isMiniMode && handleLink(basePath as string, item) === '#'" side>
            <q-icon name="fa-solid fa-up-right-from-square" size="10px" />
          </q-item-section>
          <q-tooltip anchor="center right" self="center left" :offset="[10, 0]">
            {{ item.meta ? displayTitle(item.meta?.title) : item.name }}
          </q-tooltip>
          <q-menu
            class="base-menu-popup base-menu-popup--mini"
            anchor="top right"
            self="top left"
            transition-show="scale"
            transition-hide="scale"
            :offset="[6, 0]"
          >
            <base-menu-item
              :my-router="item.children"
              :layer-level="(layerLevel as number) + 1"
              :base-path="basePath === '' ? item.path : basePath + '/' + item.path"
              :duration="duration"
              force-mini
              :force-expand="forceExpand"
            />
          </q-menu>
        </q-item>
      </template>
    </div>
  </template>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Route } from 'src/types/index'
import { QMenu, openURL } from 'quasar'
import { useThemeStore } from 'src/stores/theme'
import { useAppStore } from 'src/stores/app'
import { useUserStore } from 'src/stores/user'
import { findMenuPathByTableCode } from 'src/utils/menu-context'
import { storeToRefs } from 'pinia'
import { resolveRouteTitle } from 'src/utils/route-title'

defineOptions({ name: 'BaseMenuItem' })

interface Props {
  myRouter?: Route[] | undefined
  initLevel?: number | undefined
  layerLevel?: number | undefined
  duration?: number | undefined
  basePath?: string | undefined
  forceMini?: boolean | undefined
  forceExpand?: boolean | undefined
}

const props = withDefaults(defineProps<Props>(), {
  myRouter: () => [] as Route[],
  layerLevel: 0,
  initLevel: 0,
  duration: 150,
  basePath: '',
  forceMini: false,
  forceExpand: false,
})
const { t } = useI18n()
const displayTitle = (title: unknown) => resolveRouteTitle(title, t)
const route = useRoute()
const router = useRouter()
const themeStore = useThemeStore()
const appStore = useAppStore()
const userStore = useUserStore()

const { primaryColor, activeTextColor, activeBgColor } = storeToRefs(themeStore)
const { is_drawer_mini } = storeToRefs(appStore)

const isMiniMode = computed(() => is_drawer_mini.value || props.forceMini)
const menuInsetLevel = computed(() => (isMiniMode.value ? undefined : props.initLevel))

const buildRawLink = (basePath: string, itemPath: string) => {
  return basePath === '' ? itemPath : basePath + '/' + itemPath
}

const isExternalLink = (path: string) => path.indexOf('http') !== -1

const normalizeMenuPath = (value: string) => {
  const cleanValue = (value || '/').split(/[?#]/)[0] || '/'
  if (isExternalLink(cleanValue)) return cleanValue
  const withLeadingSlash = cleanValue.startsWith('/') ? cleanValue : `/${cleanValue}`
  return withLeadingSlash.replace(/\/+/g, '/').replace(/\/$/, '') || '/'
}

const currentRoutePath = computed(() => {
  if (route.name === 'record_detail') {
    const tableCode = String(route.params.table_code || '')
    if (tableCode) {
      const menuPath = findMenuPathByTableCode(userStore.menus, tableCode)
      return menuPath
        ? normalizeMenuPath(menuPath)
        : normalizeMenuPath(`/admin/develop/generalization/${tableCode}`)
    }
  }
  if (route.name === 'record_form') {
    const tableCode = String(route.params.table_code || '')
    if (tableCode) {
      const menuPath = findMenuPathByTableCode(userStore.menus, tableCode)
      return menuPath
        ? normalizeMenuPath(menuPath)
        : normalizeMenuPath(`/admin/develop/generalization/${tableCode}`)
    }
  }
  return normalizeMenuPath(route.path)
})

const itemRoutePath = (basePath: string, item: Route) => {
  return normalizeMenuPath(buildRawLink(basePath, item.path))
}

const handleLink = (basePath: string, item: Route) => {
  const link = buildRawLink(basePath, item.path)
  if (isExternalLink(link)) {
    return '#'
  }
  return link
}

const handleMenuClick = async (basePath: string, item: Route) => {
  const rawLink = buildRawLink(basePath, item.path)
  const link = handleLink(basePath, item)
  const externalLink = rawLink.indexOf('http')
  if (externalLink !== -1) {
    openURL(rawLink.slice(externalLink))
    return false
  }
  const i = link.indexOf('http')
  if (i !== -1) {
    openURL(link.slice(i))
    return false
  }
  try {
    await router.push(link)
  } catch (error) {
    console.error('Navigation failed:', error)
  }
}

const isMenuItemActive = (basePath: string, item: Route) => {
  const path = itemRoutePath(basePath, item)
  if (isExternalLink(path)) return false
  return currentRoutePath.value === path
}

const isRouteInBranch = (basePath: string, item: Route) => {
  const path = itemRoutePath(basePath, item)
  if (isExternalLink(path)) return false
  if (path === '/') return currentRoutePath.value === '/'
  return currentRoutePath.value === path || currentRoutePath.value.startsWith(`${path}/`)
}

const expansionHeaderStyle = computed(() => {
  return (basePath: string, item: Route) => {
    const active = isRouteInBranch(basePath, item)
    let cssString = ''
    cssString += active ? `color: ${activeTextColor.value};` : ''
    if (isMiniMode.value) {
      cssString += active ? `background: ${activeBgColor.value};` : ''
    }
    return cssString
  }
})
const expansionItemOpen = (basePath: string, item: Route) => {
  return props.forceExpand || isRouteInBranch(basePath, item)
}
</script>

<style lang="scss" scoped>
.base-menu-item {
  :deep(.base-menu-expansion .q-item__section--avatar > .q-icon) {
    font-size: 20px;
  }

  .base-menu-expansion--active {
    :deep(.q-expansion-item__container > .q-item) {
      color: v-bind(primaryColor) !important;
      background: transparent !important;
    }
  }

  .baseItemActive {
    color: v-bind(primaryColor) !important;
    background: color-mix(in srgb, v-bind(primaryColor) 18%, transparent) !important;
    box-shadow: inset 3px 0 0 v-bind(primaryColor);
  }
}

:global(
  .body--dark .app-drawer .base-menu-expansion--active > .q-expansion-item__container > .q-item
) {
  color: #c7d2fe !important;
  background: transparent !important;
}

:global(.body--dark .app-drawer .baseItemActive) {
  color: #ffffff !important;
  background: var(--app-primary-soft-strong) !important;
  box-shadow: inset 3px 0 0 var(--q-primary);
}

:global(.body--dark .app-drawer .base-menu-expansion .q-expansion-item__content) {
  background: transparent !important;
}

:global(.app-drawer--mini .base-menu-item .q-item) {
  box-sizing: border-box;
  justify-content: center;
  width: 42px !important;
  min-width: 42px !important;
  max-width: 42px !important;
  height: 42px !important;
  min-height: 42px !important;
  margin: 4px auto;
  padding: 0;
  border-radius: 12px;
  overflow: visible;
}

:global(.app-drawer--mini .base-menu-item) {
  display: flex;
  justify-content: center;
}

:global(.app-drawer--mini .base-menu-item .q-item__section--avatar) {
  flex: 0 0 100%;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-width: 0;
  margin: 0;
  padding-right: 0;

  .q-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    min-width: 24px;
    text-align: center;
  }
}

:global(.base-menu-popup) {
  box-sizing: border-box;
  width: 56px !important;
  min-width: 56px !important;
  max-width: 56px !important;
  padding: 7px !important;
  border: 1px solid var(--app-primary-border);
  border-radius: 12px;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.16);
}

:global(.base-menu-popup--mini .base-menu-item) {
  display: flex;
  justify-content: center;
}

:global(.base-menu-popup--mini .q-item) {
  box-sizing: border-box;
  justify-content: center;
  width: 40px !important;
  min-width: 40px !important;
  max-width: 40px !important;
  height: 40px !important;
  min-height: 40px !important;
  margin: 2px auto;
  padding: 0;
  border-radius: 12px;
}

:global(.base-menu-popup--mini .q-item__section--avatar) {
  flex: 0 0 100%;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  min-width: 0;
  margin: 0;
  padding-right: 0;

  .q-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    min-width: 24px;
    text-align: center;
  }
}

:global(.app-drawer--mini .baseItemActive),
:global(.base-menu-popup--mini .baseItemActive) {
  color: #fff !important;
  background: v-bind(primaryColor) !important;
  box-shadow:
    inset 0 0 0 1px rgba(255, 255, 255, 0.2),
    0 10px 20px var(--app-primary-shadow);
}

:global(.app-drawer--mini .baseItemActive .q-icon),
:global(.app-drawer--mini .baseItemActive .q-item__section--avatar),
:global(.base-menu-popup--mini .baseItemActive .q-icon),
:global(.base-menu-popup--mini .baseItemActive .q-item__section--avatar) {
  color: #fff !important;
}

:global(.body--dark .base-menu-popup) {
  border-color: var(--app-dark-border);
  background: var(--app-dark-surface);
  color: var(--app-dark-text);
  box-shadow: 0 16px 42px rgba(0, 0, 0, 0.42);
}
</style>
