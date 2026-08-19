import { instance } from 'boot/axios'
import { localLoadingRequestConfig } from 'src/api/request-config'
import type { Basic, Query, ResponseData } from 'src/types/global'

export interface Dict extends Basic {
  dict_name: string
  dict_code: string
  dict_items?: Array<DictItem>
}

export interface DictItem extends Basic {
  dict_id: number
  item_name: string
  item_code: string
  item_value: string
}

export interface RuntimeDict {
  dict_name: string
  dict_code: string
  dict_items: RuntimeDictItem[]
}

export interface RuntimeDictItem {
  item_name: string
  item_code: string
  item_value: string
}

export interface DictCreateReq {
  dict_name: string
  dict_code: string
}

export interface DictUpdateReq extends DictCreateReq {
  id: number
}

export interface DictItemCreateReq {
  item_name: string
  item_code: string
  item_value: string
  dict_id: number
}

export interface DictItemUpdateReq extends DictItemCreateReq {
  id: number
}

export const useDictApi = () => {
  // 字典相关API
  const queryDict = async (params: Query) => {
    return instance.post<ResponseData<Array<Dict>>>('/admin/dict/query', params).then((res) => {
      return res.data
    })
  }

  const queryDictById = async (id: number) => {
    return instance.get<ResponseData<Dict>>(`/admin/dict/id/${id}`).then((res) => {
      return res.data
    })
  }

  const queryDictByCode = async (code: string) => {
    return instance
      .get<ResponseData<Dict>>(`/admin/dict/code/${code}`, localLoadingRequestConfig)
      .then((res) => res.data)
  }

  const queryRuntimeDictByCode = async (code: string) => {
    return instance
      .get<ResponseData<RuntimeDict>>(`/admin/runtime/dict/${code}`, localLoadingRequestConfig)
      .then((res) => res.data)
  }

  const createDict = async (req: DictCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/dict', req).then((res) => {
      return res.data
    })
  }

  const updateDict = async (req: DictUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/dict/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteDict = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/dict/${id}`).then((res) => {
      return res.data
    })
  }

  // 字典项相关API
  const queryDictItemsByDictId = async (dictId: number) => {
    return instance.get<ResponseData<Array<DictItem>>>(`/admin/dict/items/${dictId}`).then((res) => {
      return res.data
    })
  }

  const queryDictItemById = async (id: number) => {
    return instance.get<ResponseData<DictItem>>(`/admin/dict/item/${id}`).then((res) => {
      return res.data
    })
  }

  const createDictItem = async (req: DictItemCreateReq) => {
    return instance.post<ResponseData<number>>('/admin/dict/item', req).then((res) => {
      return res.data
    })
  }

  const updateDictItem = async (req: DictItemUpdateReq) => {
    return instance.put<ResponseData<number>>(`/admin/dict/item/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteDictItem = async (id: number) => {
    return instance.delete<ResponseData<number>>(`/admin/dict/item/${id}`).then((res) => {
      return res.data
    })
  }

  return {
    queryDict,
    queryDictById,
    queryDictByCode,
    queryRuntimeDictByCode,
    createDict,
    updateDict,
    deleteDict,
    queryDictItemsByDictId,
    queryDictItemById,
    createDictItem,
    updateDictItem,
    deleteDictItem
  }
}
