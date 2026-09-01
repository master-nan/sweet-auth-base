import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Menu } from 'src/api/services/sys-menu'
import type { Route } from 'src/types'

const userStore = vi.hoisted(() => ({
  menu_names: [] as string[],
  menus: [] as Menu[],
}))

vi.mock('src/stores/user', () => ({
  useUserStore: () => userStore,
}))

import { asyncRoutesChildren } from 'src/router/routes'
import constructionRouters from 'src/router/utils'

const integrationMenus = [
  {
    id: 1200,
    name: 'integration',
    title: '集成中心',
    path: 'integration',
    component: 'src/components/Layout/Layout.vue',
    page_type: 'directory',
    is_hidden: false,
    children: [
      {
        id: 1201,
        name: 'integration_external_system',
        title: '外部系统',
        path: 'external-system',
        component: 'pages/integration/external-system/Index.vue',
        page_type: 'fixed',
        is_hidden: false,
      },
      {
        id: 1202,
        name: 'integration_interface_definition',
        title: '接口定义',
        path: 'interface-definition',
        component: 'pages/integration/interface-definition/Index.vue',
        page_type: 'fixed',
        is_hidden: false,
      },
      {
        id: 1203,
        name: 'integration_credential',
        title: '集成凭证',
        path: 'credential',
        component: 'pages/integration/credential/Index.vue',
        page_type: 'fixed',
        is_hidden: false,
      },
    ],
  },
] as Menu[]

const reportMenus = [
  {
    id: 1300,
    name: 'report',
    title: '报表',
    path: 'report',
    component: 'src/components/Layout/Layout.vue',
    page_type: 'directory',
    is_hidden: false,
    children: [
      {
        id: 1301,
        name: 'report_sales_summary',
        title: '销售汇总',
        path: 'report/runtime/sales-summary',
        component: 'pages/report/runtime/ReportRuntimePage.vue',
        page_type: 'report',
        table_code: 'sales_order',
        option: JSON.stringify({ report_id: 901, report_code: 'sales_summary' }),
        is_hidden: false,
      },
    ],
  },
] as Menu[]

function integrationRoutes(): Route[] {
  const route = asyncRoutesChildren.find((item) => item.name === 'integration')
  if (!route) throw new Error('integration route is missing')
  return [cloneRoute(route)]
}

function reportRoutes(): Route[] {
  const route = asyncRoutesChildren.find((item) => item.name === 'report')
  if (!route) throw new Error('report route is missing')
  return [cloneRoute(route)]
}

function cloneRoute(route: Route): Route {
  const result: Route = {
    name: route.name,
    path: route.path,
    component: route.component,
  }
  if (route.meta) result.meta = { ...route.meta }
  if (route.children) result.children = route.children.map(cloneRoute)
  if (route.redirect) result.redirect = route.redirect
  if (route.props !== undefined) result.props = route.props
  return result
}

describe('permission route construction', () => {
  beforeEach(() => {
    userStore.menu_names = []
    userStore.menus = []
  })

  it('renders only Integration pages returned by the backend permission menu', () => {
    userStore.menu_names = ['integration', 'integration_external_system']

    const routes = constructionRouters(integrationRoutes(), integrationMenus)

    expect(routes).toHaveLength(1)
    expect(routes[0]?.name).toBe('integration')
    expect(routes[0]?.meta?.title).toBe('router.integration.default')
    expect(routes[0]?.children?.map((item) => item.name)).toEqual([
      'integration_external_system',
    ])
    expect(routes[0]?.children?.[0]?.meta?.title).toBe(
      'router.integration.externalSystem',
    )
  })

  it('removes the Integration directory when the backend grants no child page', () => {
    userStore.menu_names = ['integration']

    expect(constructionRouters(integrationRoutes(), integrationMenus)).toEqual([])
  })

  it('renders the credential page only when returned by backend permissions', () => {
    userStore.menu_names = ['integration', 'integration_credential']

    const routes = constructionRouters(integrationRoutes(), integrationMenus)

    expect(routes[0]?.children?.map((item) => item.name)).toEqual(['integration_credential'])
    expect(routes[0]?.children?.[0]?.meta?.title).toBe('router.integration.credential')
  })

  it('keeps only the formal Report entry set', () => {
    const report = asyncRoutesChildren.find((item) => item.name === 'report')

    expect(report?.children?.map((item) => item.name)).toEqual([
      'report_center',
      'report_manage',
      'report_design',
    ])
    expect(asyncRoutesChildren.some((item) => item.name === 'report_v2')).toBe(false)
  })

  it('creates published Report routes with the unified Sheet runtime component', () => {
    userStore.menu_names = ['report']

    const routes = constructionRouters(reportRoutes(), reportMenus)
    const runtimeRoute = routes[0]?.children?.find((item) => item.name === 'report_sales_summary')

    expect(runtimeRoute?.name).toBe('report_sales_summary')
    expect(runtimeRoute?.path).toBe('/admin/report/runtime/sales-summary')
    expect(runtimeRoute?.meta).toMatchObject({
      pageType: 'report',
      reportId: 901,
      reportCode: 'sales_summary',
      menuId: 1301,
      permissionTableCode: 'sales_order',
    })
  })

  it('does not keep the removed report-v2 runtime as a dynamic route fallback', () => {
    userStore.menu_names = ['report']
    const menus = structuredClone(reportMenus)
    if (menus[0]?.children?.[0]) {
      menus[0].children[0].component = 'pages/report-v2/runtime/ReportRuntimePage.vue'
    }
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => undefined)

    const routes = constructionRouters(reportRoutes(), menus)
    expect(routes[0]?.children?.some((item) => item.name === 'report_sales_summary')).toBe(false)
    expect(warn).toHaveBeenCalled()
    warn.mockRestore()
  })
})
