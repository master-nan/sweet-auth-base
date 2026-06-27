import { defineBoot } from '#q-app/wrappers'
import { useTagViewStore } from 'stores/tagView'
import { useBreadcrumbsStore } from 'stores/breadcrumbs'
import { useKeepAliveStore } from 'stores/keep-alive'
import { useConfigureStore } from 'stores/configure'
import { LocalStorage, LoadingBar } from 'quasar'
import type { RouteLocationNormalized } from 'vue-router'
import constantRoutes from 'src/router/constantRoutes'
import type { RouteData } from 'src/types'

export default defineBoot(async ({ router }) => {
  const tagViewStore = useTagViewStore()
  const breadCrumbsStore = useBreadcrumbsStore()
  const keepAliveStore = useKeepAliveStore()
  const configureStore = useConfigureStore()
  try {
    await configureStore.fetchConfigure({ force: true })
  } catch (error) {
    console.error('Failed to fetch configuration:', error)
  }

  // 让长时间不刷新的页面也能最终更新到最新配置：切回窗口/标签页时做一次“按需刷新”
  const refreshConfigureIfNeeded = async () => {
    try {
      await configureStore.fetchConfigure()
    } catch (error) {
      console.error('Failed to refresh configuration:', error)
    }
  }

  window.addEventListener('focus', () => {
    void refreshConfigureIfNeeded()
  })
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      void refreshConfigureIfNeeded()
    }
  })
  router.beforeEach((to, from) => {
    LoadingBar.stop()
    LoadingBar.start()

    if (to.name != null && to.path.startsWith('/admin')) {
      // is a public route
      if (Array.isArray(constantRoutes)) {
        for (let i = 0; i < constantRoutes.length; i++) {
          if (constantRoutes[i]?.path === to.path) {
            return
          }
        }
      }

      const tagViewOnSessionStorage = (LocalStorage.getItem('tagView') as RouteData[]) ?? []
      if (tagViewStore.getTagView.length === 0 && tagViewOnSessionStorage.length !== 0) {
        tagViewStore.setTagView(tagViewOnSessionStorage)
        keepAliveStore.setKeepAliveList(tagViewOnSessionStorage)
      } else if (from.fullPath !== to.fullPath) {
        tagViewStore.addTagView(to)
        // 同步更新 keepAliveStore
        keepAliveStore.setKeepAliveList(tagViewStore.getTagView)
      }
      breadCrumbsStore.setBreadcrumbs(to.matched)
      handleKeepAlive(to)
    }
  })

  router.afterEach(() => {
    LoadingBar.stop()
  })
})

/**
 * Handle redundant layout: router-view and keep the current component under the first layer index <router-view>
 * This method cannot filter the on-demand loading <layout> used for nested routing
 * @param to
 */
function handleKeepAlive(to: RouteLocationNormalized) {
  if (to.matched && to.matched.length > 2) {
    for (let i = 0; i < to.matched.length; i++) {
      const element = to.matched[i]
      if (
        element!.components &&
        element!.components.default &&
        element!.components.default.name === 'MyLayout'
      ) {
        to.matched.splice(i, 1)
        handleKeepAlive(to)
      }
    }
  }
}
