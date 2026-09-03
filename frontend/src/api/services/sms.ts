import type { Basic, Query, ResponseData } from '@/types/global'
import { instance } from '@/boot/axios'

export interface SmsTemplate extends Basic {
  sign_name: string
  template_code: string
  template_name: string
  template_params: Array<string>
}

export interface SmsTemplateCreateReq {
  sign_name: string
  template_code: string
  template_name: string
  template_params: Array<string>
}

export interface SmsTemplateUpdateReq {
  sign_name: string
  template_code: string
  template_name: string
  template_params: Array<string>
}

export const useSmsApi = () => {
  const querySmsTemplate = async (params: Query) => {
    return instance
      .post<ResponseData<Array<SmsTemplate>>>('/admin/sms/template/query', params)
      .then((res) => {
        return res.data
      })
  }

  const querySmsTemplateById = (id: number) => {
    return instance.get<ResponseData<SmsTemplate>>(`/admin/sms/template/${id}`).then((res) => {
      return res.data
    })
  }

  const createSmsTemplate = async (req: SmsTemplateCreateReq) => {
    return instance.post<ResponseData<BigInteger>>('/admin/sms/template', req).then((res) => {
      return res.data
    })
  }

  const updateSmsTemplate = async (id: number, req: SmsTemplateUpdateReq) => {
    return instance.put<ResponseData<BigInteger>>(`/admin/sms/template/${id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteSmsTemplate = async (id: number) => {
    return instance.delete<ResponseData<BigInteger>>(`/admin/sms/template/${id}`).then((res) => {
      return res.data
    })
  }

  return {
    querySmsTemplate,
    querySmsTemplateById,
    createSmsTemplate,
    updateSmsTemplate,
    deleteSmsTemplate,
  }
}
