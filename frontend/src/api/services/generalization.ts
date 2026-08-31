import { translate as t } from 'src/i18n/runtime/instance'
import type { Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'
import type { AxiosRequestConfig, AxiosResponse, Method } from 'axios'

export interface GeneralizationListResult {
  data: Array<Record<string, any>>
  total: number
}

export interface GeneralizationCreateReq {
  table_code: string
  data: Record<string, any>
  menu_id?: number
}

export interface GeneralizationUpdateReq {
  id: number
  table_code: string
  data: Record<string, any>
  menu_id?: number
}

export interface GeneralizationDeleteReq {
  id: number
  table_code: string
  menu_id?: number
}

export type RuntimeActionMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface RuntimeActionRequest<TPayload = Record<string, unknown>> {
  path: string
  method?: string
  payload?: TPayload
  responseType?: AxiosRequestConfig['responseType']
}

const allowedRuntimeActionMethods = new Set<RuntimeActionMethod>([
  'GET',
  'POST',
  'PUT',
  'PATCH',
  'DELETE',
])

export const assertControlledRuntimePath = (path: string): string => {
  const value = path.trim()
  if (!value.startsWith('/admin/') || value.startsWith('//') || value.includes('\\')) {
    throw new Error(t('ui.invalidActionPathWhenRunning'))
  }
  const parsed = new URL(value, 'https://runtime.invalid')
  if (
    parsed.origin !== 'https://runtime.invalid' ||
    !parsed.pathname.startsWith('/admin/') ||
    parsed.pathname.includes('..')
  ) {
    throw new Error(t('ui.invalidActionPathWhenRunning'))
  }
  return `${parsed.pathname}${parsed.search}`
}

export const executeControlledRuntimeAction = async <
  TResponse = unknown,
  TPayload = Record<string, unknown>,
>(
  request: RuntimeActionRequest<TPayload>,
): Promise<AxiosResponse<TResponse>> => {
  const method = String(request.method || 'POST').toUpperCase() as RuntimeActionMethod
  if (!allowedRuntimeActionMethods.has(method)) {
    throw new Error(t('ui.runtimeActionMethodNotSupported'))
  }
  const path = assertControlledRuntimePath(request.path)
  const config: AxiosRequestConfig = {
    url: path,
    method: method as Method,
    ...(method === 'GET' ? { params: request.payload } : { data: request.payload }),
  }
  if (request.responseType) config.responseType = request.responseType
  const response = await instance.request(config)
  return response as AxiosResponse<TResponse>
}

export const useGeneralizationApi = () => {
  const queryGeneralizationByCode = async (tableCode: string, params: Query) => {
    const data: Query = {
      table_code: tableCode,
      ...params,
    }
    return instance
      .post<
        ResponseData<Array<Record<string, any>>>
      >(`/admin/generalization/query/code/${tableCode}`, data)
      .then((res) => res.data)
  }

  const getGeneralizationDetailByCode = async (tableCode: string, id: number, menuId?: number) => {
    return instance
      .get<ResponseData<Record<string, any>>>(
        `/admin/generalization/detail/code/${tableCode}/${id}`,
        {
          params: menuId ? { menu_id: menuId } : undefined,
        },
      )
      .then((res) => res.data)
  }

  const createGeneralization = async (req: GeneralizationCreateReq) => {
    return instance.post<ResponseData<boolean>>('/admin/generalization/create', req).then((res) => {
      return res.data
    })
  }

  const updateGeneralization = async (req: GeneralizationUpdateReq) => {
    return instance.put<ResponseData<boolean>>('/admin/generalization/update', req).then((res) => {
      return res.data
    })
  }

  const deleteGeneralization = async (req: GeneralizationDeleteReq) => {
    return instance
      .delete<ResponseData<boolean>>('/admin/generalization/delete', { data: req })
      .then((res) => {
        return res.data
      })
  }

  return {
    queryGeneralizationByCode,
    getGeneralizationDetailByCode,
    createGeneralization,
    updateGeneralization,
    deleteGeneralization,
    executeRuntimeAction: executeControlledRuntimeAction,
  }
}
