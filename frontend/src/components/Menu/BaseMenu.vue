<template>
  <q-scroll-area :thumb-style="thumbStyle">
    <div v-if="!is_drawer_mini" class="q-px-sm q-pb-sm">
      <q-input
        v-model="menuKeyword"
        dense
        outlined
        clearable
        debounce="150"
        placeholder="搜索菜单"
        aria-label="搜索菜单"
      >
        <template #prepend>
          <q-icon name="search" />
        </template>
      </q-input>
    </div>
    <q-list>
      <template v-if="filteredRouter.length">
        <base-menu-item
          :my-router="filteredRouter"
          :basePath="'/admin'"
          :force-expand="hasMenuKeyword"
        />
      </template>
      <q-item v-else-if="hasMenuKeyword" dense>
        <q-item-section class="text-grey-6 text-caption">没有匹配的菜单</q-item-section>
      </q-item>
    </q-list>
  </q-scroll-area>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouterStore } from 'src/stores/permission'
import { useAppStore } from 'src/stores/app'
import type { Route } from 'src/types/index'
import BaseMenuItem from './BaseMenuItem.vue'

defineOptions({ name: 'BaseMenu' })

const thumbStyle = {
  right: '3px',
  borderRadius: '999px',
  backgroundColor: 'rgba(115, 103, 240, 0.35)',
  width: '4px',
}

const store = useRouterStore()
const appStore = useAppStore()
const { is_drawer_mini } = storeToRefs(appStore)
const { t } = useI18n()

const menuKeyword = ref('')
const router = computed(() => store.getPermissionRoutes[0]?.children || [])
const hasMenuKeyword = computed(() => menuKeyword.value.trim().length > 0)

const normalizeKeyword = (value: unknown) => String(value || '').trim().toLowerCase()

const routeMatches = (route: Route, keyword: string) => {
  const title = String(route.meta?.title || '')
  const candidates = [
    title,
    t(title),
    route.name,
    route.path,
    route.meta?.tableCode,
  ]
  return candidates.some((item) => normalizeKeyword(item).includes(keyword))
}

const filterRoutes = (routes: Route[], keyword: string): Route[] => {
  if (!keyword) return routes
  return routes.reduce<Route[]>((items, route) => {
    if (route.meta?.isHidden === true) return items
    const children = route.children ? filterRoutes(route.children, keyword) : []
    if (routeMatches(route, keyword) || children.length > 0) {
      items.push({
        ...route,
        ...(children.length > 0 ? { children } : {}),
      })
    }
    return items
  }, [])
}

const filteredRouter = computed(() => filterRoutes(router.value, normalizeKeyword(menuKeyword.value)))

watch(is_drawer_mini, (mini) => {
  if (mini) menuKeyword.value = ''
})
</script>
