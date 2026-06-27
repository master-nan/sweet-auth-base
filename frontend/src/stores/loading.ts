import { defineStore } from 'pinia'

export const useLoadingStore = defineStore('loading', {
  state: () => ({
    loading: false,
    loadingCount: 0, // 支持嵌套加载
  }),
  actions: {
    setLoading(status: boolean) {
      if (status) {
        this.loadingCount++
      } else if (this.loadingCount > 0) {
        this.loadingCount--
      }
      this.loading = this.loadingCount > 0
    },
    resetLoading() {
      this.loadingCount = 0
      this.loading = false
    }
  },
})
