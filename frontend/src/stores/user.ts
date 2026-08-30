import { defineStore } from 'pinia'
import { LocalStorage } from 'quasar'
import { useTagViewStore } from './tagView'
import { useConfigureStore } from './configure'
import { useRouterStore } from './permission'
import { useKeepAliveStore } from './keep-alive'
import { useBreadcrumbsStore } from './breadcrumbs'
import { Router as router } from 'src/router/index'
import type { Role } from 'src/api/services/sys-role'
import type { Menu } from 'src/api/services/sys-menu'

const authSessionStorageKeys = [
  'access_token',
  'must_change_password',
  'password_change_reason',
] as const

const clearAuthSessionStorage = () => {
  authSessionStorageKeys.forEach((key) => LocalStorage.remove(key))
}

const tokenSessionIdentity = (value: string) => {
  try {
    const payload = value.split('.')[1]
    if (!payload) return ''
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/')
    const normalized = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=')
    const decoded = JSON.parse(atob(normalized)) as { sub?: string; sid?: string }
    const subject = String(decoded.sub || '')
    const sessionID = String(decoded.sid || '')
    return subject && sessionID ? `${subject}:${sessionID}` : ''
  } catch {
    return ''
  }
}

export function isStaleSessionSnapshot(
  requestToken: string | undefined,
  requestGeneration: number | undefined,
  currentToken: string,
  currentGeneration: number,
  storedToken: string,
) {
  return (
    requestToken !== currentToken ||
    requestToken !== storedToken ||
    requestGeneration !== currentGeneration
  )
}

interface User {
  roles: Role[]
  user_name: string
  email: string
  phone_number: string
  gmt_last_login: string | null
  is_reset: boolean
  language: string
  access_token: string
  must_change_password: boolean
  password_change_reason: string
  buttons: string[]
  menu_names: string[]
  menus: Menu[]
  session_generation: number
}

export const useUserStore = defineStore('user', {
  state: (): User => ({
    user_name: '',
    roles: [],
    email: '',
    phone_number: '',
    gmt_last_login: '',
    is_reset: false,
    language: '',
    access_token: LocalStorage.getItem('access_token') || '',
    must_change_password: LocalStorage.getItem('must_change_password') === true,
    password_change_reason: String(LocalStorage.getItem('password_change_reason') || ''),
    buttons: [],
    menu_names: [],
    menus: [],
    session_generation: 0,
  }),

  getters: {
    getUserName(state) {
      return state.user_name
    },
    getEmail(state) {
      return state.email
    },
    getPhoneNumber(state) {
      return state.phone_number
    },
    getLanguage(state) {
      return state.language
    },
    getGmtLastLogin(state) {
      return state.gmt_last_login
    },
    getUserRoles(state) {
      return state.roles
    },
    getFirstCharacterOfUserName(state) {
      return state.user_name ? state.user_name.charAt(0).toUpperCase() : ''
    },
    getLoginToken(state) {
      return state.access_token
    },
    isLogin(state) {
      return !!state.access_token
    },
  },

  actions: {
    setLoginToken(access_token: string) {
      if (this.access_token !== access_token) {
        clearAuthSessionStorage()
        this.resetUserScopedState()
      }
      LocalStorage.set('access_token', access_token)
      this.access_token = access_token
      this.session_generation += 1
    },
    setPasswordChangeRequirement(required: boolean, reason = '') {
      this.must_change_password = required
      this.password_change_reason = required ? reason : ''
      LocalStorage.set('must_change_password', required)
      if (required && reason) {
        LocalStorage.set('password_change_reason', reason)
      } else {
        LocalStorage.remove('password_change_reason')
      }
    },
    replaceAccessToken(accessToken: string) {
      if (!accessToken || accessToken === this.access_token) return
      LocalStorage.set('access_token', accessToken)
      this.access_token = accessToken
      this.session_generation += 1
    },
    setUserInfo(partial: Partial<User>) {
      if (partial.roles) {
        this.roles = partial.roles as Role[]
      }
      if (partial.is_reset) {
        this.setPasswordChangeRequirement(true, 'initial_reset')
      }
      return this.$patch(partial)
    },
    setUserRoles(roles: Role[]) {
      this.roles = roles
    },
    resetUserScopedState() {
      this.roles = []
      this.user_name = ''
      this.email = ''
      this.phone_number = ''
      this.gmt_last_login = null
      this.is_reset = false
      this.language = ''
      this.must_change_password = false
      this.password_change_reason = ''
      this.buttons = []
      this.menu_names = []
      this.menus = []

      useTagViewStore().removeAllTagView()
      useConfigureStore().$reset()
      useRouterStore().$reset()
      useKeepAliveStore().$reset()
      useBreadcrumbsStore().$reset()
      if (router.hasRoute('admin')) {
        router.removeRoute('admin')
      }
    },
    syncPersistedSession(accessToken: string) {
      if (accessToken === this.access_token) return
      if (!accessToken) {
        this.setLogout()
        return
      }
      if (
        tokenSessionIdentity(accessToken) &&
        tokenSessionIdentity(accessToken) === tokenSessionIdentity(this.access_token)
      ) {
        this.replaceAccessToken(accessToken)
        return
      }
      this.setLoginToken(accessToken)
    },
    async logout() {
      const token = this.access_token
      const baseURL = String(import.meta.env.VITE_API_URL || '/sweet_admin').replace(/\/$/, '')
      try {
        await fetch(`${baseURL}/admin/logout`, {
          method: 'POST',
          credentials: 'include',
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        })
      } catch {
        // 服务不可达时仍清除本地登录状态，服务端会话由过期时间兜底。
      } finally {
        this.setLogout()
      }
    },
    setLogout() {
      clearAuthSessionStorage()
      this.access_token = ''
      this.resetUserScopedState()
      this.session_generation += 1

      router.push({ name: 'Login' })
    },
  },
})
