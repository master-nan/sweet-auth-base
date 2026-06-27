import type { Basic, Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'

export interface Application extends Basic {
  name: string
  app_key: string
  expiration: number
  ding_key: string
  ding_app_id: string
  remark: string
}

export interface ApplicationCreateReq {
  name: string
  expiration: number
  ding_key: string
  ding_secret: string
  ding_app_id: string
  remark: string
}

export interface ApplicationUpdateReq {
  name: string
  expiration: number
  ding_key: string
  ding_secret: string
  ding_app_id: string
  remark: string
}

export interface ApplicationSecretRes {
  id: number
  name: string
  app_key: string
  app_secret: string
  expiration: number
}

export const useApplicationApi = () => {
  const queryApplication = async (params: Query) => {
    return instance
      .post<ResponseData<Array<Application>>>('/admin/application/query', params)
      .then((res) => {
        return res.data
      })
  }

  const queryApplicationById = (id: number) => {
    return instance.get<ResponseData<Application>>(`/admin/application/${id}`).then((res) => {
      return res.data
    })
  }

  const createApplication = async (req: ApplicationCreateReq) => {
    return instance.post<ResponseData<ApplicationSecretRes>>('/admin/application', req).then((res) => {
      return res.data
    })
  }

  const updateApplication = async (id: number, req: ApplicationUpdateReq) => {
    return instance.put<ResponseData<BigInteger>>(`/admin/application/${id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteApplication = async (id: number) => {
    return instance.delete<ResponseData<BigInteger>>(`/admin/application/${id}`).then((res) => {
      return res.data
    })
  }

  const rotateApplicationSecret = async (id: number) => {
    return instance
      .post<ResponseData<ApplicationSecretRes>>(`/admin/application/${id}/rotate-secret`)
      .then((res) => {
        return res.data
      })
  }

  return {
    queryApplication,
    queryApplicationById,
    createApplication,
    updateApplication,
    deleteApplication,
    rotateApplicationSecret,
  }
}
