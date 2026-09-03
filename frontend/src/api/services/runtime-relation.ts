import { instance } from '@/boot/axios'
import { localLoadingRequestConfig } from '@/api/request-config'
import type { ResponseData } from '@/types/global'

export interface RuntimeRelationOption {
  value: string
  label: string
  parent_value?: string
}

export interface RuntimeRelationOptionsRequest {
  menu_id: number
  keyword?: string
  page?: number
  num?: number
  selected_values?: string[]
  source_values?: Record<string, unknown>
}

export const queryRuntimeRelationOptions = async (
  fieldId: number,
  request: RuntimeRelationOptionsRequest,
) => {
  const response = await instance.post<ResponseData<RuntimeRelationOption[]>>(
    `/admin/runtime/relation-fields/${fieldId}/options`,
    request,
    localLoadingRequestConfig,
  )
  return {
    items: response.data.data,
    total: response.data.total || 0,
  }
}
