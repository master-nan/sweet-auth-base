<template>
  <q-scroll-area :thumb-style="thumbStyle">
    <div v-if="!is_drawer_mini" class="q-px-sm q-pb-sm">
      <q-input
        v-model="menuKeyword"
        dense
        outlined
        clearable
        debounce="150"
        :placeholder="t('ui.searchMenus')"
        :aria-label="t('ui.searchMenus')"
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
        <q-item-section class="text-grey-6 text-caption">{{
          t('ui.noMatchingMenus')
        }}</q-item-section>
      </q-item>
    </q-list>
  </q-scroll-area>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useRouterStore } from '@/stores/permission'
import { useAppStore } from '@/stores/app'
import type { Route } from '@/types/index'
import BaseMenuItem from './BaseMenuItem.vue'
import { resolveRouteTitle } from '@/utils/route-title'
import { primitiveText } from '@/utils/primitive-text'

defineOptions({ name: 'BaseMenu' })

const thumbStyle = {
  right: '3px',
  borderRadius: '999px',
  backgroundColor: 'var(--app-primary-border)',
  width: '4px',
}

const store = useRouterStore()
const appStore = useAppStore()
const { is_drawer_mini } = storeToRefs(appStore)
const { t } = useI18n()

const menuKeyword = ref('')
const router = computed(() => store.getPermissionRoutes[0]?.children || [])
const hasMenuKeyword = computed(() => menuKeyword.value.trim().length > 0)

const normalizeKeyword = (value: unknown) => primitiveText(value).trim().toLowerCase()

const routeMatches = (route: Route, keyword: string) => {
  const title = String(route.meta?.title || '')
  const candidates = [
    title,
    resolveRouteTitle(title, t),
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

const filteredRouter = computed(() =>
  filterRoutes(router.value, normalizeKeyword(menuKeyword.value)),
)

watch(is_drawer_mini, (mini) => {
  if (mini) menuKeyword.value = ''
})
</script>
