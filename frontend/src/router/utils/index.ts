import { useUserStore } from 'src/stores/user'
import { type Route } from 'src/types/index'
import layout from 'src/components/Layout/Layout.vue'
import type { Menu } from 'src/api/services/sys-menu'

export default function constructionRouters(router: Route[], backendMenus?: Menu[]) {
  const userStore = useUserStore()
  const sourceMenus = backendMenus ?? userStore.menus

  const temp = router.filter((item) => {
    // 无子路由的叶子节点 → 检查后端菜单权限
    if (!item.children || item.children.length === 0) {
      const routeName = item.name as string
      if (!routeName) return true
      if (item.meta?.isHidden === true) return true
      // home 路由所有人可访问
      if (routeName === 'home') return true
      return userStore.menu_names.includes(routeName)
    }
    // 有子路由的布局节点 → 先递归过滤子路由，有子路由则保留
    return true
  })

  temp.forEach((item) => {
    if (item.children) {
      const matchedBackendMenu = sourceMenus.find((menu) => menu.name === item.name)
      item.children = constructionRouters(item.children, matchedBackendMenu?.children || [])
    }
  })

  appendLowCodeRoutes(temp, sourceMenus)

  // 过滤掉子路由全部被移除的布局节点
  return temp.filter((item) => {
    if (item.children && item.children.length === 0) return false
    return true
  })
}

const componentMap: Record<string, any> = {
  'src/components/Layout/Layout.vue': layout,
  'pages/develop/generalization/Index.vue': () => import('pages/develop/generalization/Index.vue'),
  'pages/system/data-permission/Index.vue': () => import('pages/system/data-permission/Index.vue'),
}

function appendLowCodeRoutes(routes: Route[], menus: Menu[]) {
  if (!menus?.length) return
  const byName = new Map(routes.map((route) => [route.name, route]))
  for (const menu of menus) {
    const route = byName.get(menu.name)
    if (route?.children?.length && menu.children?.length) {
      appendLowCodeRoutes(route.children, menu.children)
    }
    if (route || !isLowCodeMenu(menu)) continue
    routes.push(menuToRoute(menu))
  }
}

function isLowCodeMenu(menu: Menu) {
  return menu.page_type === 'low_code' && !!menu.table_code
}

function menuToRoute(menu: Menu): Route {
  const tableCode = menu.table_code || ''
  return {
    path: menu.path,
    name: menu.name,
    component: componentMap[menu.component],
    meta: {
      title: menu.title,
      ...(menu.icon ? { icon: menu.icon } : {}),
      keepAlive: true,
      isHidden: menu.is_hidden,
      tableCode,
      menuId: menu.id,
    },
  }
}
