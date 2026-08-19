import { instance } from 'boot/axios'
import { localLoadingRequestConfig } from 'src/api/request-config'
// import qs from 'qs'
import type { Basic, Query, ResponseData } from 'src/types/global'
import type {
  SysTableFieldType,
  SysTableType,
  SysTableFieldInputType,
  SysTableRelationType,
  SysMasterDetailMode,
  SysFormOpenMode,
  SysDetailOpenMode,
} from 'src/types/enum'

export interface TableField extends Basic {
  table_id: number
  field_name: string
  field_code: string
  field_type: SysTableFieldType
  field_length: number
  field_decimal_length: number
  input_type: SysTableFieldInputType
  form_span?: number
  detail_span?: number
  default_value: string
  dict_code: string
  is_primary_key: boolean
  is_index: boolean
  is_quick_search: boolean
  is_advanced_search: boolean
  is_sort: boolean
  is_null: boolean
  is_list_show: boolean
  is_insert_show: boolean
  is_update_show: boolean
  sequence: number
  original_field_id: number
  binding: string
  field_category?: string
  expression?: string
  tag?: string
  linkage_config?: string
}

export interface TableIndex extends Basic {
  table_id: number
  index_name: string
  is_unique: boolean
  index_fields: Array<TableField>
}

export interface TableIndexFieldReq {
  table_id: number
  field_id: number
  field_code: string
}

export interface TableIndexCreateReq {
  table_id: number
  index_name: string
  is_unique: boolean
  index_fields: Array<TableIndexFieldReq>
}

export interface TableIndexUpdateReq extends TableIndexCreateReq {
  id: number
}

export interface TableRelation extends Basic {
  table_id: number
  related_table_id: number
  reference_key: string
  foreign_key: string
  relation_type: SysTableRelationType
  many_table_code: string
}

export interface TableRelationCreateReq {
  table_id: number
  related_table_id: number
  reference_key: string
  foreign_key: string
  relation_type: SysTableRelationType
  manyTableCode?: string
}

export interface TableRelationUpdateReq extends TableRelationCreateReq {
  id: number
}

export interface Table extends Basic {
  table_name: string
  table_code: string
  table_type: SysTableType
  master_detail_mode: SysMasterDetailMode
  form_open_mode: SysFormOpenMode
  detail_open_mode: SysDetailOpenMode
  parent_id: number
  table_fields: Array<TableField>
  sql: string
  table_indexes: Array<TableIndex>
  table_relations: Array<TableRelation>
}

export interface RuntimeTableMetadata extends Basic {
  table_name: string
  table_code: string
  table_type: SysTableType
  master_detail_mode: SysMasterDetailMode
  form_open_mode: SysFormOpenMode
  detail_open_mode: SysDetailOpenMode
  table_fields: Array<TableField>
  table_relations: Array<TableRelation>
}

export interface TablePublishReq {
  parent_id?: number
}

export interface TableCreateReq {
  table_name: string
  table_code: string
  table_type: SysTableType
  master_detail_mode?: SysMasterDetailMode
  form_open_mode?: SysFormOpenMode
  detail_open_mode?: SysDetailOpenMode
  parent_id: number
  sql?: string
}

export interface TableUpdateReq {
  table_name: string
  table_code: string
  table_type: SysTableType
  master_detail_mode?: SysMasterDetailMode
  form_open_mode?: SysFormOpenMode
  detail_open_mode?: SysDetailOpenMode
  parent_id: number
  sql?: string
  id: number
}

export interface TableFieldCreateReq {
  table_id: number
  field_name: string
  field_code: string
  type: SysTableFieldType
  field_length: number
  field_decimal_length: number
  input_type: SysTableFieldInputType
  form_span?: number
  detail_span?: number
  default_value: string
  dict_code: string
  is_primary_key: boolean
  is_index: boolean
  is_quick_search: boolean
  is_advanced_search: boolean
  is_sort: boolean
  is_null: boolean
  is_list_show: boolean
  is_insert_show: boolean
  is_update_show: boolean
  sequence: number
  original_field_id: number
  binding: string
  field_category?: string
  expression?: string
  linkage_config?: string
}

export interface TableFieldUpdateReq {
  table_id: number
  field_name: string
  field_code: string
  type: SysTableFieldType
  field_length: number
  field_decimal_length: number
  input_type: SysTableFieldInputType
  form_span?: number
  detail_span?: number
  default_value: string
  dict_code: string
  is_primary_key: boolean
  is_index: boolean
  is_quick_search: boolean
  is_advanced_search: boolean
  is_sort: boolean
  is_null: boolean
  is_list_show: boolean
  is_insert_show: boolean
  is_update_show: boolean
  sequence: number
  original_field_id: number
  binding: string
  field_category?: string
  expression?: string
  linkage_config?: string
  id: number
}

export const useTableApi = () => {
  const queryTable = async (params: Query) => {
    return instance.post<ResponseData<Array<Table>>>('/admin/table/query', params).then((res) => {
      return res.data
    })
  }

  const queryTableById = async (id: number) => {
    return instance.get<ResponseData<Table>>(`/admin/table/id/${id}`).then((res) => {
      return res.data
    })
  }

  const queryTableByCode = async (code: string) => {
    return instance
      .get<ResponseData<RuntimeTableMetadata>>(
        `/admin/table/code/${code}`,
        localLoadingRequestConfig,
      )
      .then((res) => res.data)
  }

  const queryRuntimeTableByCode = async (code: string) => {
    return instance
      .get<ResponseData<RuntimeTableMetadata>>(
        `/admin/runtime/table/${code}`,
        localLoadingRequestConfig,
      )
      .then((res) => res.data)
  }

  const createTable = async (req: TableCreateReq) => {
    return instance.post<ResponseData<BigInteger>>('/admin/table', req).then((res) => {
      return res.data
    })
  }

  const updateTable = async (req: TableUpdateReq) => {
    return instance.put<ResponseData<BigInteger>>(`/admin/table/${req.id}`, req).then((res) => {
      return res.data
    })
  }

  const deleteTable = async (id: number) => {
    return instance.delete<ResponseData<BigInteger>>(`/admin/table/${id}`).then((res) => {
      return res.data
    })
  }

  const initTable = async (code: string) => {
    return instance.get<ResponseData<BigInteger>>(`/admin/table/init/${code}`).then((res) => {
      return res.data
    })
  }

  const syncTable = async (code: string) => {
    return instance.post<ResponseData<boolean>>(`/admin/table/sync/${code}`).then((res) => {
      return res.data
    })
  }

  const syncTableIndexes = async (code: string) => {
    return instance.post<ResponseData<boolean>>(`/admin/table/sync/index/${code}`).then((res) => {
      return res.data
    })
  }

  const publishTable = async (code: string, req: TablePublishReq = {}) => {
    return instance.post<ResponseData<boolean>>(`/admin/table/publish/${code}`, req).then((res) => {
      return res.data
    })
  }

  const unpublishTable = async (code: string) => {
    return instance.post<ResponseData<boolean>>(`/admin/table/unpublish/${code}`).then((res) => {
      return res.data
    })
  }

  const queryTableFieldsByTableId = async (id: number) => {
    return instance
      .get<ResponseData<Array<TableField>>>(`/admin/table/fields/${id}`)
      .then((res) => {
        return res.data
      })
  }

  const queryTableFieldById = async (id: number) => {
    return instance.get<ResponseData<TableField>>(`/admin/table/field/${id}`).then((res) => {
      return res.data
    })
  }

  const createTableField = async (req: TableFieldCreateReq) => {
    return instance.post<ResponseData<BigInteger>>('/admin/table/field', req).then((res) => {
      return res.data
    })
  }

  const updateTableField = async (req: TableFieldUpdateReq) => {
    return instance
      .put<ResponseData<BigInteger>>(`/admin/table/field/${req.id}`, req)
      .then((res) => {
        return res.data
      })
  }

  const deleteTableField = async (id: number) => {
    return await instance
      .delete<ResponseData<BigInteger>>(`/admin/table/field/${id}`)
      .then((res) => {
        return res.data
      })
  }

  const queryTableIndexByTableId = async (id: number) => {
    return instance
      .get<ResponseData<Array<TableIndex>>>(`/admin/table/indexes/${id}`)
      .then((res) => {
        return res.data
      })
  }

  const createTableIndex = async (req: TableIndexCreateReq) => {
    return instance.post<ResponseData<BigInteger>>('/admin/table/index', req).then((res) => {
      return res.data
    })
  }

  const updateTableIndex = async (req: TableIndexUpdateReq) => {
    return instance
      .put<ResponseData<BigInteger>>(`/admin/table/index/${req.id}`, req)
      .then((res) => {
        return res.data
      })
  }

  const deleteTableIndex = async (id: number) => {
    return instance.delete<ResponseData<BigInteger>>(`/admin/table/index/${id}`).then((res) => {
      return res.data
    })
  }

  const queryTableRelationsByTableId = async (id: number) => {
    return instance
      .get<ResponseData<Array<TableRelation>>>(`/admin/table/relations/${id}`)
      .then((res) => {
        return res.data
      })
  }

  const createTableRelation = async (req: TableRelationCreateReq) => {
    return instance.post<ResponseData<BigInteger>>('/admin/table/relation', req).then((res) => {
      return res.data
    })
  }

  const updateTableRelation = async (req: TableRelationUpdateReq) => {
    return instance
      .put<ResponseData<BigInteger>>(`/admin/table/relation/${req.id}`, req)
      .then((res) => {
        return res.data
      })
  }

  const deleteTableRelation = async (id: number) => {
    return instance.delete<ResponseData<BigInteger>>(`/admin/table/relation/${id}`).then((res) => {
      return res.data
    })
  }

  return {
    queryTable,
    queryTableById,
    queryTableByCode,
    queryRuntimeTableByCode,
    createTable,
    updateTable,
    deleteTable,
    initTable,
    syncTable,
    syncTableIndexes,
    publishTable,
    unpublishTable,
    queryTableFieldsByTableId,
    queryTableFieldById,
    createTableField,
    updateTableField,
    deleteTableField,
    queryTableIndexByTableId,
    createTableIndex,
    updateTableIndex,
    deleteTableIndex,
    queryTableRelationsByTableId,
    createTableRelation,
    updateTableRelation,
    deleteTableRelation,
  }
}
