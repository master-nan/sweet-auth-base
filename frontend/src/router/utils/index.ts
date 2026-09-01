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

  appendDynamicMenuRoutes(temp, sourceMenus)

  // 过滤掉子路由全部被移除的布局节点
  return temp.filter((item) => {
    if (item.children && item.children.length === 0) return false
    return true
  })
}

const reportRuntimeComponent = 'pages/report/runtime/ReportRuntimePage.vue'

const componentMap: Record<string, any> = {
  'src/components/Layout/Layout.vue': layout,
  'pages/develop/generalization/Index.vue': () => import('pages/develop/generalization/Index.vue'),
  'pages/system/data-permission/Index.vue': () => import('pages/system/data-permission/Index.vue'),
  [reportRuntimeComponent]: () => import('pages/report/runtime/ReportRuntimePage.vue'),
}

type ReportMenuOption = {
  report_id: number
  report_code?: string
}

function appendDynamicMenuRoutes(routes: Route[], menus: Menu[]) {
  if (!menus?.length) return
  const byName = new Map(routes.map((route) => [route.name, route]))
  for (const menu of menus) {
    const route = byName.get(menu.name)
    if (route) mergeBackendMenuMeta(route, menu)
    if (route?.children?.length && menu.children?.length) {
      appendDynamicMenuRoutes(route.children, menu.children)
    }
    if (route) continue
    const dynamicRoute = menuToDynamicRoute(menu)
    if (dynamicRoute) routes.push(dynamicRoute)
  }
}

function mergeBackendMenuMeta(route: Route, menu: Menu) {
  const reportOption = isReportMenu(menu) ? parseReportMenuOption(menu.option) : null
  const routeTitle = route.meta?.title
  route.meta = {
    ...(route.meta || {}),
    title:
      typeof routeTitle === 'string' && routeTitle.startsWith('router.')
        ? routeTitle
        : menu.title || routeTitle || String(route.name || ''),
    ...(menu.icon ? { icon: menu.icon } : {}),
    isHidden: menu.is_hidden,
    menuId: menu.id,
    ...(menu.table_code ? { tableCode: menu.table_code } : {}),
    ...(menu.page_type ? { pageType: menu.page_type } : {}),
    ...(reportOption
      ? {
          reportId: reportOption.report_id,
          ...(reportOption.report_code ? { reportCode: reportOption.report_code } : {}),
          ...(menu.table_code ? { permissionTableCode: menu.table_code } : {}),
        }
      : {}),
  }
}

function isLowCodeMenu(menu: Menu) {
  return menu.page_type === 'low_code' && !!menu.table_code
}

function isReportMenu(menu: Menu) {
  return (
    menu.page_type === 'report' &&
    menu.component === reportRuntimeComponent &&
    !!parseReportMenuOption(menu.option)
  )
}

function isDirectoryMenu(menu: Menu) {
  return menu.page_type === 'directory' && !!menu.children?.length
}

function menuToDynamicRoute(menu: Menu): Route | null {
  if (isLowCodeMenu(menu)) return menuToLowCodeRoute(menu)
  if (menu.page_type === 'report') return menuToReportRoute(menu)
  if (!isDirectoryMenu(menu)) return null
  const children = (menu.children || [])
    .map((child) => menuToDynamicRoute(child))
    .filter((route): route is Route => !!route)
  if (!children.length) return null
  return {
    path: menu.path,
    name: menu.name,
    component: componentMap[menu.component] || layout,
    meta: {
      title: menu.title,
      ...(menu.icon ? { icon: menu.icon } : {}),
      isHidden: menu.is_hidden,
      menuId: menu.id,
    },
    children,
  }
}

function parseReportMenuOption(option: unknown): ReportMenuOption | null {
  if (!option) return null
  let source: unknown = option
  if (typeof option === 'string') {
    if (!option.trim()) return null
    try {
      source = JSON.parse(option)
    } catch (error) {
      console.warn('报表菜单 option 不是合法 JSON，已跳过动态路由生成:', option, error)
      return null
    }
  }
  if (!source || typeof source !== 'object') return null
  const raw = source as Record<string, unknown>
  const reportId = Number(raw.report_id)
  if (!Number.isFinite(reportId) || reportId <= 0) return null
  const reportCode = raw.report_code
  return {
    report_id: reportId,
    ...(typeof reportCode === 'string' ||
    typeof reportCode === 'number' ||
    typeof reportCode === 'boolean'
      ? { report_code: String(reportCode) }
      : {}),
  }
}

function menuToReportRoute(menu: Menu): Route | null {
  const option = parseReportMenuOption(menu.option)
  if (!option) {
    console.warn('报表菜单缺少有效 report_id，已跳过动态路由生成:', menu)
    return null
  }
  if (menu.component !== reportRuntimeComponent) {
    console.warn('报表菜单 component 不是正式运行页，已跳过动态路由生成:', menu.component, menu)
    return null
  }
  const component = componentMap[menu.component]
  if (!component) {
    console.warn('报表菜单 component 未注册，已跳过动态路由生成:', menu.component, menu)
    return null
  }
  return {
    path: reportMenuRoutePath(menu.path),
    name: menu.name,
    component,
    meta: {
      title: menu.title,
      ...(menu.icon ? { icon: menu.icon } : {}),
      keepAlive: false,
      isHidden: menu.is_hidden,
      pageType: 'report',
      menuId: menu.id,
      reportId: option.report_id,
      ...(option.report_code ? { reportCode: option.report_code } : {}),
      ...(menu.table_code ? { permissionTableCode: menu.table_code } : {}),
    },
  }
}

function reportMenuRoutePath(path: string) {
  const normalized = path.trim().replace(/^\/+/, '')
  if (!normalized) return '/admin/report/runtime'
  if (normalized.startsWith('admin/')) return `/${normalized}`
  return `/admin/${normalized}`
}

function menuToLowCodeRoute(menu: Menu): Route {
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
