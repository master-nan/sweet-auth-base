import { translate as t } from 'src/i18n/runtime/instance'
import { defineStore } from 'pinia'
import { useBasicApi } from 'src/api/services/basic'

let inflightFetch: Promise<void> | null = null

interface Configure {
  enable_captcha: boolean
  password_length: number
  password_complexity: number
  password_expire_time: number
  password_error_count: number
  password_lock_minutes: number
  password_policy: string
  // 系统基本信息
  system_name?: string
  system_version?: string
  system_logo?: string
  system_description?: string
  // 邮件配置
  enable_email?: boolean
  smtp_server?: string
  smtp_port?: number
  sender_email?: string
  sender_password?: string
}

type ConfigureState = Configure & {
  _lastFetchedAt: number
}

export const useConfigureStore = defineStore('configure', {
  state: (): ConfigureState => ({
    enable_captcha: false,
    password_length: 8,
    password_complexity: 2,
    password_expire_time: 90,
    password_error_count: 5,
    password_lock_minutes: 15,
    password_policy: 'medium',
    // 系统基本信息默认值
    system_name: 'Sweet Admin',
    system_version: '0.1',
    system_logo: '',
    get system_description() {
      return t('ui.generalPurposeLowCodeFoundation')
    },
    // 邮件配置默认值
    enable_email: false,
    smtp_server: '',
    smtp_port: 465,
    sender_email: '',
    sender_password: '',

    _lastFetchedAt: 0,
  }),

  getters: {
    getEnableCaptcha(state) {
      return state.enable_captcha
    },
    getPasswordLength(state) {
      return state.password_length
    },
    getPasswordComplexity(state) {
      return state.password_complexity
    },
    getPasswordExpireTime(state) {
      return state.password_expire_time
    },
    getPasswordErrorCount(state) {
      return state.password_error_count
    },
    getPasswordLockMinutes(state) {
      return state.password_lock_minutes
    },
    getPasswordPolicy(state) {
      return state.password_policy
    },
    // 系统基本信息的getter
    getSystemName(state) {
      return state.system_name
    },
    getSystemVersion(state) {
      return state.system_version
    },
    getSystemLogo(state) {
      return state.system_logo
    },
    getSystemDescription(state) {
      return state.system_description
    },
    // 邮件配置的getter
    getEnableEmail(state) {
      return state.enable_email
    },
    getSmtpServer(state) {
      return state.smtp_server
    },
    getSmtpPort(state) {
      return state.smtp_port
    },
    getSenderEmail(state) {
      return state.sender_email
    },
  },

  actions: {
    setConfigure(partial: Partial<Configure>) {
      return this.$patch(partial)
    },
    async fetchConfigure(options?: { force?: boolean }) {
      const force = options?.force === true
      const maxAgeMs = 5 * 60 * 1000
      const now = Date.now()

      if (!force && this._lastFetchedAt > 0 && now - this._lastFetchedAt < maxAgeMs) {
        return
      }

      if (inflightFetch != null) {
        return inflightFetch
      }

      inflightFetch = (async () => {
        const { configure } = useBasicApi()
        const response = await configure()
        if (response.success) {
          this.setConfigure(response.data)
          this._lastFetchedAt = Date.now()
        }
      })().finally(() => {
        inflightFetch = null
      })

      return inflightFetch
    },
  },
})
