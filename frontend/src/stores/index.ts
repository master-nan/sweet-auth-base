import { defineStore } from '#q-app'
import { createPinia } from 'pinia'
import type { Router } from 'vue-router'
// Store可通过类型化扩展访问Router，避免各Store自行导入并创建路由实例。
declare module 'pinia' {
  export interface PiniaCustomProperties {
    readonly router: Router
  }
}

export default defineStore((/* { ssrContext } */) => {
  const pinia = createPinia()

  return pinia
})
