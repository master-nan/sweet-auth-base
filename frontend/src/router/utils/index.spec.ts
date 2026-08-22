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

function integrationRoutes(): Route[] {
  const route = asyncRoutesChildren.find((item) => item.name === 'integration')
  if (!route) throw new Error('integration route is missing')
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

  it('keeps one formal Report entry set and only the published runtime hidden route', () => {
    const report = asyncRoutesChildren.find((item) => item.name === 'report')
    const reportRuntime = asyncRoutesChildren.find((item) => item.name === 'report_v2')

    expect(report?.children?.map((item) => item.name)).toEqual([
      'report_center',
      'report_manage',
      'report_design',
    ])
    expect(reportRuntime?.children?.map((item) => item.name)).toEqual(['report_v2_runtime'])
  })
})
