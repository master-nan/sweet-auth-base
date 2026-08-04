import { instance } from 'boot/axios'
import type { Query, ResponseData } from 'src/types/global'

export type ExternalSystemStatus = 'draft' | 'enabled' | 'disabled'
export type ExternalSystemType = 'hr' | 'erp' | 'tms' | 'wms' | 'other'

export interface ExternalSystemListItem {
  id: number
  system_code: string
  name: string
  system_type: ExternalSystemType
  base_url_summary: string
  owner_identifier: string
  owner_name: string
  status: ExternalSystemStatus
  revision: number
  gmt_modify: string
}

export interface ExternalSystemDetail extends ExternalSystemListItem {
  base_url: string
  description: string
  gmt_create: string
}

export interface ExternalSystemCreateRequest {
  system_code: string
  name: string
  system_type: ExternalSystemType
  base_url: string
  owner_identifier: string
  owner_name: string
  description?: string
}

export interface ExternalSystemUpdateRequest {
  name?: string
  system_type?: ExternalSystemType
  base_url?: string
  owner_identifier?: string
  owner_name?: string
  description?: string
  revision: number
}

export interface ExternalSystemQuery extends Query {
  system_type?: ExternalSystemType | ''
  status?: ExternalSystemStatus | ''
  owner?: string
}

export const useIntegrationApi = () => ({
  queryExternalSystems: (query: ExternalSystemQuery) =>
    instance
      .post<ResponseData<ExternalSystemListItem[]>>('/admin/integration/external-system/query', query)
      .then((response) => response.data),
  getExternalSystem: (id: number) =>
    instance
      .get<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}`)
      .then((response) => response.data),
  createExternalSystem: (request: ExternalSystemCreateRequest) =>
    instance
      .post<ResponseData<ExternalSystemDetail>>('/admin/integration/external-system', request)
      .then((response) => response.data),
  updateExternalSystem: (id: number, request: ExternalSystemUpdateRequest) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}`, request)
      .then((response) => response.data),
  enableExternalSystem: (id: number, revision: number) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}/enable`, {
        revision,
      })
      .then((response) => response.data),
  disableExternalSystem: (id: number, revision: number) =>
    instance
      .put<ResponseData<ExternalSystemDetail>>(`/admin/integration/external-system/${id}/disable`, {
        revision,
      })
      .then((response) => response.data),
})
