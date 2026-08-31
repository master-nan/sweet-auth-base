import { translate as t } from 'src/boot/i18n'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    meta: {
      title: 'login',
    },
    component: () => import('pages/Login.vue'),
  },
  {
    path: '/change-password',
    name: 'ChangePassword',
    meta: {
      get title() {
        return t('ui.changePassword')
      },
    },
    component: () => import('pages/ChangePassword.vue'),
  },
  {
    path: '/',
    name: 'main',
    redirect: '/admin/home',
  },
  {
    path: '/admin',
    redirect: '/admin/home',
  },
  {
    path: '/404',
    component: () => import('pages/ErrorNotFound.vue'),
  },
  {
    path: '/:catchAll(.*)*',
    component: () => import('pages/ErrorNotFound.vue'),
  },
]

export default routes
