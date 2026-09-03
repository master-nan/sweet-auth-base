import { defineStore } from 'pinia'
import type { Route } from '@/types/index'

export const useRouterStore = defineStore('routes', {
  state: () => ({
    permissionRoutes: [] as Route[],
  }),

  getters: {
    getPermissionRoutes(state): Route[] {
      return state.permissionRoutes
    },
  },

  actions: {
    setRoutes(routes: Route[]) {
      this.permissionRoutes = routes
    },
  },
})
