import { LocalStorage } from 'quasar'

export const AUTH_SESSION_STORAGE_KEYS = [
  'access_token',
  'must_change_password',
  'password_change_reason',
] as const

export const clearAuthSessionStorage = () => {
  AUTH_SESSION_STORAGE_KEYS.forEach((key) => LocalStorage.remove(key))
}
