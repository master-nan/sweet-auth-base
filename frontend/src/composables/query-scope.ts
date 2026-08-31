import { translate as t } from 'src/boot/i18n'
import { computed, ref, type MaybeRefOrGetter, toValue } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from 'src/stores/user'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import type { Menu } from 'src/api/services/sys-menu'
import type { QueryScopeConfig } from 'src/modules/query-scheme/types'

const findMenu = (menus: Menu[], routeName: string): Menu | undefined => {
  for (const menu of menus) {
    if (menu.name === routeName) return menu
    const nested = findMenu(menu.children || [], routeName)
    if (nested) return nested
  }
}

export const collectQueryScopes = (menus: Menu[]) => {
  const scopes: Array<{ scope_code: string; scope_label: string; route_name: string }> = []
  const visit = (items: Menu[]) => {
    items.forEach((menu) => {
      if (menu.query_scope_code) {
        scopes.push({
          scope_code: menu.query_scope_code,
          scope_label: menu.title,
          route_name: menu.name,
        })
      }
      visit(menu.children || [])
    })
  }
  visit(menus)
  return scopes.sort((left, right) => left.scope_label.localeCompare(right.scope_label, 'zh-CN'))
}

export function useQueryScope(routeName?: MaybeRefOrGetter<string>) {
  const route = routeName ? null : useRoute()
  const userStore = useUserStore()
  const api = useQuerySchemeApi()
  const config = ref<QueryScopeConfig | null>(null)
  const loading = ref(false)
  const error = ref('')

  const currentRouteName = computed(() =>
    String(routeName ? toValue(routeName) : route?.name || ''),
  )
  const menu = computed(() => findMenu(userStore.menus, currentRouteName.value))
  const scopeCode = computed(() => String(menu.value?.query_scope_code || '').trim())
  const available = computed(() => !!scopeCode.value && !!config.value)

  const loadScope = async () => {
    config.value = null
    error.value = ''
    if (!scopeCode.value) {
      error.value = t('ui.theCurrentPageDoesNotConfigureTheQueryRange')
      return null
    }
    loading.value = true
    try {
      const response = await api.getScopeConfig(scopeCode.value)
      config.value = response.data || null
      if (!config.value) error.value = t('ui.queryScopeNotAvailable')
      return config.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : t('ui.failedToLoadQueryProjectRange')
      return null
    } finally {
      loading.value = false
    }
  }

  return { scopeCode, config, loading, error, available, loadScope }
}
