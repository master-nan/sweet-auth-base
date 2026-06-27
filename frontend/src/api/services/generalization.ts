import type { Query, ResponseData } from 'src/types/global'
import { instance } from 'boot/axios'

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

export const useGeneralizationApi = () => {
  const queryGeneralizationByCode = async (tableCode: string, params: Query) => {
    const data: Query = {
      table_code: tableCode,
      ...params,
    }
    return instance
      .post<ResponseData<Array<Record<string, any>>>>(
        `/admin/generalization/query/code/${tableCode}`,
        data,
      )
      .then((res) => res.data)
  }

  const getGeneralizationDetailByCode = async (tableCode: string, id: number, menuId?: number) => {
    return instance
      .get<ResponseData<Record<string, any>>>(`/admin/generalization/detail/code/${tableCode}/${id}`, {
        params: menuId ? { menu_id: menuId } : undefined,
      })
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
    return instance.delete<ResponseData<boolean>>('/admin/generalization/delete', { data: req }).then((res) => {
      return res.data
    })
  }

  return {
    queryGeneralizationByCode,
    getGeneralizationDetailByCode,
    createGeneralization,
    updateGeneralization,
    deleteGeneralization,
  }
}
