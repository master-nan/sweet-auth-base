import { defineStore } from 'pinia'
import type { RouteData } from '@/types/index'
import type { RouteRecordNormalized } from 'vue-router'

export const useBreadcrumbsStore = defineStore('breadCrumbs', {
  state: () => ({
    breadcrumbs: [] as RouteData[],
  }),

  getters: {
    getBreadCrumbs(state) {
      return state.breadcrumbs
    },
  },

  actions: {
    setBreadcrumbs(matched: RouteRecordNormalized[]) {
      const temp = []
      for (let i = 0; i < matched.length; i++) {
        const isLast = i === matched.length - 1
        const breadcrumb: RouteData = {
          title: isLast ? resolveBreadcrumbTitle(matched[i]?.meta.title) : (matched[i]?.meta.title as string),
          name: matched[i]?.name,
          fullPath: matched[i]?.path as string,
          icon: matched[i]?.meta.icon as string,
          keepAlive: matched[i]?.meta.keepAlive as boolean,
        }

        temp.push(breadcrumb)
      }

      const last = temp[temp.length - 1]
      if (last && (String(last.name || '') === 'record_detail' || last.fullPath.includes('/detail/'))) {
        this.breadcrumbs = [last]
        return
      }

      this.breadcrumbs = temp
    },

    updateLastBreadcrumbTitle(title: string) {
      if (!title || !this.breadcrumbs.length) return
      this.breadcrumbs = this.breadcrumbs.map((item, index) =>
        index === this.breadcrumbs.length - 1 ? { ...item, title } : item,
      )
    },

    setBreadcrumbItems(items: RouteData[]) {
      this.breadcrumbs = items.filter((item) => !!item.title)
    },
  },
})

function resolveBreadcrumbTitle(title: unknown) {
  return safeString(title)
}

function safeString(value: unknown) {
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}
