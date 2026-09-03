import { instance } from '@/boot/axios'
import { localLoadingRequestConfig } from '@/api/request-config'
import type { ResponseData } from '@/types/global'
import type {
  PersonalSchemeUpdate,
  PersonalSchemeWrite,
  QueryScopeConfig,
  QuerySchemeDetail,
  QuerySchemeListItem,
  QuerySchemeManagementQuery,
  QuerySchemeResolveResult,
  QuerySchemeSummary,
  SharedSchemeUpdate,
  SharedSchemeWrite,
} from '@/modules/query-scheme/types'

export const useQuerySchemeApi = () => ({
  getScopeConfig: (scopeCode: string) =>
    instance
      .get<ResponseData<QueryScopeConfig>>(
        `/admin/runtime/query-scopes/${encodeURIComponent(scopeCode)}`,
        localLoadingRequestConfig,
      )
      .then((response) => response.data),
  available: (scopeCode: string) =>
    instance
      .get<ResponseData<QuerySchemeSummary[]>>('/admin/runtime/query-schemes/available', {
        ...localLoadingRequestConfig,
        params: { scope_code: scopeCode },
      })
      .then((response) => response.data),
  resolve: (id: number, scopeCode: string, expectedRevision?: number) =>
    instance
      .post<ResponseData<QuerySchemeResolveResult>>(
        `/admin/runtime/query-schemes/${id}/resolve`,
        {
          scope_code: scopeCode,
          ...(expectedRevision ? { expected_revision: expectedRevision } : {}),
        },
        localLoadingRequestConfig,
      )
      .then((response) => response.data),
  list: (query: QuerySchemeManagementQuery) =>
    instance
      .post<ResponseData<QuerySchemeListItem[]>>(
        '/admin/query-schemes/query',
        query,
        localLoadingRequestConfig,
      )
      .then((response) => response.data),
  detail: (id: number) =>
    instance
      .get<ResponseData<QuerySchemeDetail>>(
        `/admin/query-schemes/${id}`,
        localLoadingRequestConfig,
      )
      .then((response) => response.data),
  createPersonal: (payload: PersonalSchemeWrite) =>
    instance
      .post<ResponseData<QuerySchemeDetail>>('/admin/query-schemes/personal', payload)
      .then((response) => response.data),
  updatePersonal: (id: number, payload: PersonalSchemeUpdate) =>
    instance
      .put<ResponseData<QuerySchemeDetail>>(`/admin/query-schemes/personal/${id}`, payload)
      .then((response) => response.data),
  deletePersonal: (id: number, revision: number) =>
    instance
      .delete<ResponseData<null>>(`/admin/query-schemes/personal/${id}`, {
        params: { revision },
      })
      .then((response) => response.data),
  setPersonalDefault: (id: number, isDefault: boolean, revision: number) =>
    instance
      .put<ResponseData<QuerySchemeDetail>>(`/admin/query-schemes/personal/${id}/default`, {
        is_default: isDefault,
        revision,
      })
      .then((response) => response.data),
  copyToPersonal: (id: number, scopeCode: string, name: string, isDefault = false) =>
    instance
      .post<ResponseData<QuerySchemeDetail>>(`/admin/query-schemes/${id}/copy-to-personal`, {
        scope_code: scopeCode,
        name,
        is_default: isDefault,
      })
      .then((response) => response.data),
  createShared: (payload: SharedSchemeWrite) =>
    instance
      .post<ResponseData<QuerySchemeDetail>>('/admin/query-schemes/shared', payload)
      .then((response) => response.data),
  updateShared: (id: number, payload: SharedSchemeUpdate) =>
    instance
      .put<ResponseData<QuerySchemeDetail>>(`/admin/query-schemes/shared/${id}`, payload)
      .then((response) => response.data),
  deleteShared: (id: number, revision: number) =>
    instance
      .delete<ResponseData<null>>(`/admin/query-schemes/shared/${id}`, {
        params: { revision },
      })
      .then((response) => response.data),
  setSharedEnabled: (id: number, enabled: boolean, revision: number) =>
    instance
      .put<ResponseData<QuerySchemeDetail>>(`/admin/query-schemes/shared/${id}/enabled`, {
        enabled,
        revision,
      })
      .then((response) => response.data),
})
