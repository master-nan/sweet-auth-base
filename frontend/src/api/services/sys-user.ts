import type { Basic, ResponseData, Query } from 'src/types/global'
import { instance } from 'boot/axios'

export interface User extends Basic {
  user_name: string
  email: string
  phone_number: string
  gmt_last_login?: string | null
  password_changed_at?: string | null
  language: string
  is_reset?: boolean
  roles?: Array<{ id: number; name: string; memo?: string }>
}

export interface UserCreateReq {
  user_name: string
  password: string
  email: string
  phone_number: string
}

export interface ResetPasswordRes {
  temporary_password: string
  must_change_password: boolean
  email_sent: boolean
  email_message?: string
}

export interface UserUpdateReq {
  id: number
  user_name: string
  email: string
  phone_number: string
  access_tokens?: string
  gmt_last_login?: string | null
  is_reset?: boolean
}

export const useSysUserApi = () => {
  const queryUser = async (params: Query) => {
    // return instance
    //   .get<ResponseData<Array<User>>>('/admin/user/query', {
    //     params,
    //     paramsSerializer: (params) => {
    //       return qs.stringify(params)
    //     },
    //   })
    //   .then((res) => {
    //     return res.data
    //   })
    return instance.post<ResponseData<Array<User>>>('/admin/user/query', params).then((res) => {
      return res.data
    })
  }
  const queryUserById = async (id: number) => {
    return instance.get<ResponseData<User>>(`/admin/user/${id}`).then((res) => {
      return res.data
    })
  }
  const me = async () => {
    return instance.get<ResponseData<User>>('/admin/user/me').then((res) => {
      return res.data
    })
  }

  const createUser = async (req: UserCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/user', req).then((res) => {
      return res.data
    })
  }

  const updateUser = async (req: UserUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/user/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteUser = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/user/${id}`).then((res) => {
      return res.data
    })
  }

  const resetPassword = async (id: number) => {
    return instance
      .post<ResponseData<ResetPasswordRes>>(`/admin/user/reset_password/${id}`)
      .then((res) => {
        return res.data
      })
  }

  const unlockLogin = async (id: number) => {
    return instance.post<ResponseData<boolean>>(`/admin/user/unlock_login/${id}`).then((res) => {
      return res.data
    })
  }

  const updatePassword = async (password: string) => {
    return instance.post<ResponseData<null>>('/admin/user/password', { password }).then((res) => {
      return res.data
    })
  }

  return {
    queryUserById,
    me,
    queryUser,
    createUser,
    updateUser,
    deleteUser,
    updatePassword,
    resetPassword,
    unlockLogin,
  }
}
