import type { Basic, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'

export interface SignInReq {
  user_name: string
  password: string
  captcha: string
  captcha_id: string
}

export interface SignInRes {
  access_token: string
  refresh_token?: string
  must_change_password: boolean
  password_change_reason?: string
}

export interface Captcha {
  captcha_id: string
  image: string
}

export interface Configure extends Basic {
  // 安全配置
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

export const useBasicApi = () => {
  const login = async (params: SignInReq) => {
    return instance.post<ResponseData<SignInRes>>('/admin/login', params).then((res) => {
      return res.data
    })
  }

  const captchaImg = async () => {
    return instance.get<ResponseData<Captcha>>('/admin/captcha').then((res) => {
      return res.data
    })
  }

  const refresh = async () => {
    return instance.post<ResponseData<SignInRes>>('/admin/refresh').then((res) => res.data)
  }

  const logout = async () => {
    return instance.post<ResponseData<null>>('/admin/logout').then((res) => res.data)
  }

  const configure = async () => {
    return instance.get<ResponseData<Configure>>('/admin/configure').then((res) => {
      return res.data
    })
  }

  const configureDetail = async () => {
    return instance.get<ResponseData<Configure>>('/admin/configure/detail').then((res) => {
      return res.data
    })
  }

  const updateConfigure = async (params: Configure) => {
    return instance
      .put<ResponseData<Configure>>('/admin/configure/' + params.id, params)
      .then((res) => {
        return res.data
      })
  }

  const testConfigureEmail = async (to: string) => {
    return instance
      .post<ResponseData<{ sent: boolean }>>('/admin/configure/test-email', { to })
      .then((res) => {
        return res.data
      })
  }

  return {
    login,
    refresh,
    logout,
    captchaImg,
    configure,
    configureDetail,
    updateConfigure,
    testConfigureEmail,
  }
}
