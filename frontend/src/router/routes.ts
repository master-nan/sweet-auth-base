import layout from 'src/components/Layout/Layout.vue'
import type { Route } from 'src/types/index'

declare module 'vue-router' {
  interface RouteMeta {
    roles?: string[]
    title: string
    icon?: string
    itemLabel?: string
    keepAlive?: boolean
    isOpen?: boolean
    isHidden?: boolean
    tableCode?: string
    menuId?: number
    pageType?: string
    reportId?: number
    reportCode?: string
    permissionTableCode?: string
    showTag?: boolean
    fullscreen?: boolean
  }
}

const asyncRoutesChildren: Route[] = [
  {
    component: () => import('pages/query-scheme/Index.vue'),
    path: 'query-schemes',
    name: 'query_scheme_manager',
    meta: {
      title: '查询方案管理',
      icon: 'manage_search',
      keepAlive: false,
      isHidden: true,
      showTag: true,
    },
  },
  {
    component: () => import('pages/detail/RecordDetail.vue'),
    path: 'detail/:source/:table_code/:id',
    name: 'record_detail',
    meta: {
      title: '详情',
      icon: 'article',
      keepAlive: false,
      isHidden: true,
      showTag: true,
    },
  },
  {
    component: () => import('pages/detail/RecordForm.vue'),
    path: 'form/:mode/:table_code/:id?',
    name: 'record_form',
    meta: {
      title: '表单',
      icon: 'edit_note',
      keepAlive: false,
      isHidden: true,
      showTag: true,
    },
  },
  {
    component: () => import('pages/integration/execution/Detail.vue'),
    path: 'detail/integration/execution/:id',
    name: 'integration_execution_detail_page',
    meta: {
      title: 'router.integration.executionDetail',
      icon: 'play_circle',
      keepAlive: false,
      isHidden: true,
      showTag: true,
    },
  },
  {
    component: () => import('pages/dashboard/Dashboard.vue'),
    path: 'home',
    name: 'home',
    meta: {
      title: 'router.home',
      icon: 'home',
    },
  },
  {
    component: layout,
    path: 'system',
    name: 'system',
    meta: {
      title: 'router.system.default',
      icon: 'settings',
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/system/application/Index.vue'),
        path: 'application',
        name: 'system_application',
        meta: {
          title: 'router.system.application',
          icon: 'apps',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/sms/Index.vue'),
        path: 'sms',
        name: 'system_sms',
        meta: {
          title: 'router.system.sms',
          icon: 'sms',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/menu/Index.vue'),
        path: 'menu',
        name: 'system_menu',
        meta: {
          title: 'router.system.menu',
          icon: 'menu',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/role/Index.vue'),
        path: 'role',
        name: 'system_role',
        meta: {
          title: 'router.system.role',
          icon: 'admin_panel_settings',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/user/Index.vue'),
        path: 'user',
        name: 'system_user',
        meta: {
          title: 'router.system.user',
          icon: 'person',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/data-permission/Index.vue'),
        path: 'data-permission',
        name: 'system_data_permission',
        meta: {
          title: 'router.system.dataPermission',
          icon: 'rule',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/system/audit/Index.vue'),
        path: 'audit',
        name: 'system_audit',
        meta: {
          title: 'router.system.audit',
          icon: 'manage_search',
          keepAlive: true,
        },
      },
    ],
  },
  {
    component: layout,
    path: 'develop',
    name: 'develop',
    meta: {
      title: 'router.develop.default',
      icon: 'developer_mode',
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/develop/configure/Index.vue'),
        path: 'configure',
        name: 'develop_configure',
        meta: {
          title: 'router.develop.configure',
          icon: 'tune',
          keepAlive: true,
        },
      },
      // {
      //   component: () => import('pages/develop/generalization/Index.vue'),
      //   path: 'generalization/:table_code',
      //   name: 'develop_generalization',
      //   meta: {
      //     title: 'router.develop.generalization',
      //     icon: 'dynamic_form',
      //     keepAlive: true,
      //   },
      // },
      {
        component: () => import('pages/develop/database/Index.vue'),
        path: 'database',
        name: 'develop_database',
        meta: {
          title: 'router.develop.database',
          icon: 'storage',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/develop/dictionary/Index.vue'),
        path: 'dictionary',
        name: 'develop_dictionary',
        meta: {
          title: 'router.develop.dictionary',
          icon: 'menu_book',
          keepAlive: true,
        },
      },
    ],
  },
  {
    component: layout,
    path: 'organization',
    name: 'organization',
    meta: {
      title: 'router.organization.default',
      icon: 'account_tree',
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/organization/structure/Index.vue'),
        path: 'structure',
        name: 'organization_structure',
        meta: {
          title: 'router.organization.structure',
          icon: 'lan',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/organization/employee/Index.vue'),
        path: 'employee',
        name: 'organization_employee',
        meta: {
          title: 'router.organization.employee',
          icon: 'badge',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/organization/position/Index.vue'),
        path: 'position',
        name: 'organization_position',
        meta: {
          title: 'router.organization.position',
          icon: 'work',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/organization/sync-batch/Index.vue'),
        path: 'sync-batch',
        name: 'organization_sync_batch',
        meta: {
          title: 'router.organization.syncBatch',
          icon: 'sync',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/organization/sync-error/Index.vue'),
        path: 'sync-error',
        name: 'organization_sync_error',
        meta: {
          title: 'router.organization.syncError',
          icon: 'error_outline',
          keepAlive: true,
        },
      },
    ],
  },
  {
    component: layout,
    path: 'integration',
    name: 'integration',
    meta: {
      title: 'router.integration.default',
      icon: 'hub',
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/integration/external-system/Index.vue'),
        path: 'external-system',
        name: 'integration_external_system',
        meta: {
          title: 'router.integration.externalSystem',
          icon: 'dns',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/interface-definition/Index.vue'),
        path: 'interface-definition',
        name: 'integration_interface_definition',
        meta: {
          title: 'router.integration.interfaceDefinition',
          icon: 'api',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/credential/Index.vue'),
        path: 'credential',
        name: 'integration_credential',
        meta: {
          title: 'router.integration.credential',
          icon: 'key',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/retry-policy/Index.vue'),
        path: 'retry-policy',
        name: 'integration_retry_policy',
        meta: {
          title: 'router.integration.retryPolicy',
          icon: 'autorenew',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/sync-task/Index.vue'),
        path: 'sync-task',
        name: 'integration_sync_task',
        meta: {
          title: 'router.integration.syncTask',
          icon: 'sync_alt',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/sync-batch/Index.vue'),
        path: 'sync-batch',
        name: 'integration_sync_batch',
        meta: {
          title: 'router.integration.syncBatch',
          icon: 'view_timeline',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/execution/Index.vue'),
        path: 'execution',
        name: 'integration_execution',
        meta: {
          title: 'router.integration.execution',
          icon: 'play_circle',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/integration/log/Index.vue'),
        path: 'log',
        name: 'integration_log',
        meta: {
          title: 'router.integration.log',
          icon: 'history',
          keepAlive: true,
        },
      },
    ],
  },
  {
    component: layout,
    path: 'report',
    name: 'report',
    meta: {
      title: 'router.report.default',
      icon: 'assessment',
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/report/center/Index.vue'),
        path: 'center',
        name: 'report_center',
        meta: {
          title: 'router.report.center',
          icon: 'table_chart',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/report/manage/Index.vue'),
        path: 'manage',
        name: 'report_manage',
        meta: {
          title: 'router.report.manage',
          icon: 'build',
          keepAlive: true,
        },
      },
      {
        component: () => import('pages/report/design/Index.vue'),
        path: 'design',
        name: 'report_design',
        meta: {
          title: 'router.report.design',
          icon: 'design_services',
          keepAlive: false,
          isHidden: true,
          fullscreen: true,
          showTag: true,
        },
      },
    ],
  },
  {
    component: layout,
    path: 'report-v2',
    name: 'report_v2',
    meta: {
      title: 'Report',
      icon: 'assessment',
      isHidden: true,
      isOpen: false,
    },
    children: [
      {
        component: () => import('pages/report-v2/runtime/ReportRuntimePage.vue'),
        path: 'runtime/:id',
        name: 'report_v2_runtime',
        meta: {
          title: 'Report V2 Runtime',
          icon: 'play_circle',
          keepAlive: false,
          isHidden: true,
          showTag: true,
        },
      },
    ],
  },
]

const asyncRootRoute: Route[] = [
  {
    component: () => import('layouts/MainLayout.vue'),
    path: '/admin',
    name: 'admin',
    redirect: '/admin/home',
    children: asyncRoutesChildren,
  },
]

export { asyncRootRoute, asyncRoutesChildren }
