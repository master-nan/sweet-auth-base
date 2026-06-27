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
    showTag?: boolean
  }
}

const asyncRoutesChildren: Route[] = [
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
      {
        component: () => import('pages/develop/generalization/Index.vue'),
        path: 'generalization/:table_code',
        name: 'develop_generalization',
        meta: {
          title: 'router.develop.generalization',
          icon: 'dynamic_form',
          keepAlive: true,
        },
      },
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
