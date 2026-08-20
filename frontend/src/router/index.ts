import { route } from 'quasar/wrappers'
import {
  createMemoryHistory,
  createRouter,
  createWebHashHistory,
  createWebHistory
} from 'vue-router'

// import routes from './routes'

import constantRoutes from './constantRoutes'
/*
 * If not building with SSR mode, you can
 * directly export the Router instantiation;
 *
 * The function below can be async too; either use
 * async/await or return a Promise which resolves
 * with the Router instance.
 */
let Router: ReturnType<typeof createRouter>
const chunkReloadFlag = 'sweet-admin:chunk-reload'

function isDynamicImportError(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === 'string'
        ? error
        : ''
  return /Failed to fetch dynamically imported module|Loading chunk|Importing a module script failed/i.test(message)
}

function reloadOnceForFreshAssets() {
  if (typeof window === 'undefined') return
  if (window.sessionStorage.getItem(chunkReloadFlag) === '1') return
  window.sessionStorage.setItem(chunkReloadFlag, '1')
  window.location.reload()
}

export default route(function (/* { store, ssrContext } */) {
  const createHistory = process.env.SERVER
    ? createMemoryHistory
    : (process.env.VUE_ROUTER_MODE === 'history' ? createWebHistory : createWebHashHistory)

  Router = createRouter({
    scrollBehavior: () => ({
      left: 0,
      top: 0
    }),
    routes: constantRoutes,

    // Leave this as is and make changes in quasar.conf.js instead!
    // quasar.conf.js -> build -> vueRouterMode
    // quasar.conf.js -> build -> publicPath
    history: createHistory(process.env.VUE_ROUTER_BASE)
  })
  Router.onError((error) => {
    if (isDynamicImportError(error)) {
      reloadOnceForFreshAssets()
    }
  })
  Router.afterEach(() => {
    if (typeof window !== 'undefined') {
      window.sessionStorage.removeItem(chunkReloadFlag)
    }
  })

  return Router
})
export { Router }
