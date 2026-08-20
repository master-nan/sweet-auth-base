import { defineBoot } from '#q-app/wrappers'
import { useRouterStore } from 'stores/permission'
import { useUserStore } from 'stores/user'
import cloneDeep from 'lodash/cloneDeep'
import { asyncRoutesChildren, asyncRootRoute } from 'src/router/routes'
import constructionRouters from 'src/router/utils'
import type { RouteRecordRaw } from 'vue-router'
import { useSysUserApi } from 'src/api/services/sys-user'
import { useMenuApi, type Menu } from 'src/api/services/sys-menu'

function collectButtonCodes(menus: Menu[]): string[] {
  const codes: string[] = []
  for (const menu of menus) {
    if (menu.menu_buttons) {
      for (const btn of menu.menu_buttons) {
        if (btn.code) codes.push(btn.code)
      }
    }
    if (menu.children) {
      codes.push(...collectButtonCodes(menu.children))
    }
  }
  return codes
}

function collectMenuNames(menus: Menu[]): string[] {
  const names: string[] = []
  for (const menu of menus) {
    if (menu.name) names.push(menu.name)
    if (menu.children) {
      names.push(...collectMenuNames(menu.children))
    }
  }
  return names
}

// 导出一个 boot 函数，用于初始化路由权限控制
export default defineBoot(({ router }) => {
  // 初始化路由和用户状态管理
  const routerStore = useRouterStore()
  const userStore = useUserStore()
  const { me } = useSysUserApi()
  const menuApi = useMenuApi()

  // 设置全局路由守卫
  router.beforeEach(async (to, from, next) => {
    // 检查用户是否已登录
    const isLogin = userStore.isLogin
    const changePasswordPath = '/change-password'

    if (isLogin) {
      // 已登录用户不能访问登录页面
      if (to.path === '/login') {
        next({ path: userStore.must_change_password ? changePasswordPath : '/admin/home' })
        return
      }

      if (to.path === changePasswordPath) {
        next()
        return
      }

      if (userStore.must_change_password) {
        next({ path: changePasswordPath })
        return
      }

      // 如果已有用户信息和权限路由，直接放行
      if (userStore.getUserName !== '' && routerStore.getPermissionRoutes.length) {
        next()
      } else {
        try {
          // 如果没有用户信息，获取用户信息
          if (userStore.getUserName === '') {
            const res = await me()
            if (res.success) {
              userStore.setUserInfo(res.data)
            }
            if (userStore.must_change_password) {
              next({ path: changePasswordPath })
              return
            }
          }
          if (!userStore.menus.length || !userStore.menu_names.length) {
            // 获取权限菜单，收集按钮编码和菜单名称
            const menuRes = await menuApi.queryMyMenu()
            if (menuRes.success && menuRes.data) {
              userStore.buttons = collectButtonCodes(menuRes.data)
              userStore.menu_names = collectMenuNames(menuRes.data)
              userStore.menus = menuRes.data
            }
          }
          // 根据权限构建动态路由
          const accessRoutes = cloneDeep(asyncRoutesChildren)
          asyncRootRoute[0]!.children = constructionRouters(accessRoutes)
          routerStore.setRoutes(asyncRootRoute)

          // 动态添加路由
          for (const item of asyncRootRoute) {
            router.addRoute(item as RouteRecordRaw)
          }

          // 重新导航到目标路由
          next({
            ...to,
            replace: true,
          })
        } catch (error) {
          console.error('路由权限初始化失败:', error)
          next('/login')
        }
      }
    } else {
      // 处理未登录用户的路由访问
      if (to.path === '/login' || to.path === '/404') {
        // 允许访问不需要权限的路由
        next()
      } else {
        // 重定向到登录页
        next({ path: '/login' })
      }
    }
  })
})
