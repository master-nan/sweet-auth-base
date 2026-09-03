import { defineStore } from 'pinia'
import { nextTick } from 'vue'
import { readUIPreferences, writeUIPreferences } from '@/utils/ui-preferences'

interface APP {
  reload_flag: boolean
  is_drawer_mini: boolean
}

const preferences = readUIPreferences()

export const useAppStore = defineStore('app', {
  state: (): APP => ({
    reload_flag: true,
    is_drawer_mini: preferences.drawerMini,
  }),

  getters: {},

  actions: {
    async reloadPage(duration = 0) {
      this.reload_flag = false
      await nextTick()
      if (duration) {
        setTimeout(() => {
          this.reload_flag = true
        }, duration)
      } else {
        this.reload_flag = true
      }
    },
    setDrawerMini(mini: boolean) {
      this.is_drawer_mini = mini
      writeUIPreferences({ drawerMini: mini })
    },
  },
})
