import { translate as t } from 'src/boot/i18n'
import { ref } from 'vue'
import { useQuasar } from 'quasar'
import { useTableApi, type TableField } from 'src/api/services/sys-table'
import type { Report, ReportDataset, ReportParameter } from 'src/api/services/report'
import { useDictStore } from 'src/stores/dict'
import {
  fieldHtmlInputType,
  getFieldControlType,
  type FieldControlType,
  type HtmlInputType,
} from 'src/utils/field-metadata'
import type { ReportRuntimeParameterValue } from './useReportRuntime'

export type ReportParameterControlType =
  | 'text'
  | 'number'
  | 'date'
  | 'datetime'
  | 'select'
  | 'boolean'

export interface ReportParameterControlOption {
  label: string
  value: string | number | boolean
}

export interface ReportParameterControlMeta {
  id: string
  label: string
  field: string
  controlType: ReportParameterControlType
  htmlInputType: HtmlInputType
  options: ReportParameterControlOption[]
  required: boolean
  placeholder: string
  source: 'field_metadata' | 'parameter_options' | 'parameter' | 'fallback'
  dictCode?: string
  defaultValue?: ReportRuntimeParameterValue
  tableField?: TableField
}

type RuntimeParameterOption =
  | string
  | number
  | boolean
  | {
      label?: string
      name?: string
      text?: string
      value?: string | number | boolean
      code?: string | number | boolean
    }

const tableFieldCache = new Map<string, TableField[]>()

export function useReportParameterControls() {
  const $q = useQuasar()
  const tableApi = useTableApi()
  const dictStore = useDictStore()

  const loading = ref(false)
  const errors = ref<string[]>([])
  const controlMetas = ref<Record<string, ReportParameterControlMeta>>({})
  const defaultParameterValues = ref<Record<string, ReportRuntimeParameterValue>>({})

  async function loadControls(report: Report | null, parameters: ReportParameter[] = []) {
    loading.value = true
    errors.value = []
    try {
      const tableFieldsBySource = await loadTableFields(report)
      const metas = parameters.map((param) => buildControlMeta(report, param, tableFieldsBySource))
      await loadDictOptions(metas)

      const nextMetas: Record<string, ReportParameterControlMeta> = {}
      const nextDefaults: Record<string, ReportRuntimeParameterValue> = {}
      metas.forEach((meta) => {
        nextMetas[meta.id] = meta
        if (
          meta.defaultValue !== undefined &&
          meta.defaultValue !== null &&
          meta.defaultValue !== ''
        ) {
          nextDefaults[meta.id] = meta.defaultValue
        }
      })
      controlMetas.value = nextMetas
      defaultParameterValues.value = nextDefaults
    } finally {
      loading.value = false
    }
  }

  async function loadTableFields(report: Report | null) {
    const result: Record<string, TableField[]> = {}
    const sourceCodes = Array.from(
      new Set(
        resolveDatasets(report)
          .filter((dataset) => dataset.type === 'table' && dataset.source_code)
          .map((dataset) => String(dataset.source_code)),
      ),
    )

    await Promise.all(
      sourceCodes.map(async (sourceCode) => {
        if (tableFieldCache.has(sourceCode)) {
          result[sourceCode] = tableFieldCache.get(sourceCode) || []
          return
        }
        try {
          const res = await tableApi.queryTableByCode(sourceCode)
          const fields = res.data?.table_fields || []
          tableFieldCache.set(sourceCode, fields)
          result[sourceCode] = fields
        } catch {
          const message = t(
            'ui.fieldMetadataLoadedFailedArgumentControlDowngradedToReportParameterType',
            { sourceCode: sourceCode },
          )
          errors.value.push(message)
          result[sourceCode] = []
          $q.notify({ type: 'warning', message })
        }
      }),
    )

    return result
  }

  async function loadDictOptions(metas: ReportParameterControlMeta[]) {
    const dictCodes = Array.from(
      new Set(metas.map((meta) => meta.dictCode).filter((code): code is string => !!code)),
    )
    if (!dictCodes.length) return
    await dictStore.loadDicts(dictCodes)
    metas.forEach((meta) => {
      if (!meta.dictCode) return
      meta.options = dictStore.getDictOptions(meta.dictCode).map((item) => ({
        label: String(item.label || item.dict_label || item.item_name || item.value),
        value: item.value,
      }))
    })
  }

  function buildControlMeta(
    report: Report | null,
    param: ReportParameter,
    tableFieldsBySource: Record<string, TableField[]>,
  ): ReportParameterControlMeta {
    const dataset = resolveParameterDataset(report, param)
    const sourceCode = String(dataset?.source_code || '')
    const tableField = sourceCode
      ? tableFieldsBySource[sourceCode]?.find((field) => matchField(field, param.field))
      : undefined
    const parameterOptions = normalizeOptions(
      (param as { options?: RuntimeParameterOption[] }).options || [],
    )
    const dictCode = tableField?.dict_code || readString(param, ['dict_code', 'dictCode'])
    const controlType = inferControlType(param, tableField, dictCode, parameterOptions)
    const defaultValue = readDefaultValue(param, tableField)
    const meta: ReportParameterControlMeta = {
      id: param.id,
      label: param.label || tableField?.field_name || param.field,
      field: param.field,
      controlType,
      htmlInputType: tableField
        ? fieldHtmlInputType(tableField.input_type)
        : fallbackHtmlInputType(String(param.type || 'text')),
      options: controlType === 'boolean' ? booleanOptions() : parameterOptions,
      required: Boolean(
        readBoolean(param, ['required']) ??
          readBoolean(tableField, ['is_required']) ??
          (tableField ? !tableField.is_null : false),
      ),
      placeholder:
        param.placeholder ||
        t('ui.pleaseEnter', { value1: param.label || tableField?.field_name || param.field }),
      source: tableField
        ? 'field_metadata'
        : parameterOptions.length
          ? 'parameter_options'
          : 'parameter',
    }
    if (dictCode) {
      meta.dictCode = dictCode
      meta.source = 'field_metadata'
    }
    if (defaultValue !== undefined) meta.defaultValue = defaultValue
    if (tableField) meta.tableField = tableField
    return meta
  }

  function inferControlType(
    param: ReportParameter,
    tableField: TableField | undefined,
    dictCode: string,
    parameterOptions: ReportParameterControlOption[],
  ): ReportParameterControlType {
    if (dictCode) return 'select'
    if (parameterOptions.length) return 'select'
    if (tableField) return normalizeFieldControlType(getFieldControlType(tableField))
    return fallbackControlType(String(param.type || 'text'))
  }

  function resolveDatasets(report: Report | null): ReportDataset[] {
    if (!report) return []
    return report.layout_config?.datasets?.length
      ? report.layout_config.datasets
      : report.query_config?.datasets || []
  }

  function resolveParameterDataset(report: Report | null, param: ReportParameter) {
    const datasets = resolveDatasets(report)
    if (param.dataset_id) {
      const matched = datasets.find((dataset) => dataset.id === param.dataset_id)
      if (matched) return matched
    }
    return (
      datasets.find((dataset) =>
        dataset.fields?.some((field) => field.code === param.field || field.name === param.field),
      ) || datasets[0]
    )
  }

  function matchField(field: TableField, fieldCode: string) {
    return field.field_code === fieldCode || field.field_name === fieldCode
  }

  function normalizeFieldControlType(type: FieldControlType): ReportParameterControlType {
    if (type === 'select' || type === 'cascader') return 'select'
    if (type === 'date' || type === 'year' || type === 'year-month') return 'date'
    if (type === 'datetime') return 'datetime'
    if (type === 'number') return 'number'
    if (type === 'boolean') return 'boolean'
    return 'text'
  }

  function fallbackControlType(type: string): ReportParameterControlType {
    if (type === 'select') return 'select'
    if (type === 'date') return 'date'
    if (type === 'datetime') return 'datetime'
    if (type === 'number') return 'number'
    if (type === 'boolean' || type === 'bool') return 'boolean'
    return 'text'
  }

  function fallbackHtmlInputType(type: string): HtmlInputType {
    if (type === 'number') return 'number'
    if (type === 'date') return 'date'
    if (type === 'datetime') return 'datetime-local'
    return 'text'
  }

  function normalizeOptions(options: RuntimeParameterOption[]) {
    return options
      .map((option) => {
        if (
          typeof option === 'string' ||
          typeof option === 'number' ||
          typeof option === 'boolean'
        ) {
          return { label: String(option), value: option }
        }
        const value =
          option.value ?? option.code ?? option.label ?? option.name ?? option.text ?? ''
        return {
          label: String(option.label || option.name || option.text || value),
          value,
        }
      })
      .filter((option) => option.value !== '')
  }

  function booleanOptions(): ReportParameterControlOption[] {
    return [
      {
        get label() {
          return t('ui.yes')
        },
        value: true,
      },
      {
        get label() {
          return t('ui.no')
        },
        value: false,
      },
    ]
  }

  function readDefaultValue(param: ReportParameter, tableField?: TableField) {
    if (
      param.default_value !== undefined &&
      param.default_value !== null &&
      param.default_value !== ''
    ) {
      return param.default_value
    }
    const fieldDefault = tableField?.default_value
    return fieldDefault === '' || fieldDefault === null || fieldDefault === undefined
      ? undefined
      : fieldDefault
  }

  function readString(source: unknown, keys: string[]) {
    const data = source as Record<string, unknown>
    for (const key of keys) {
      const value = data?.[key]
      if (typeof value === 'string' && value) return value
    }
    return ''
  }

  function readBoolean(source: unknown, keys: string[]) {
    const data = source as Record<string, unknown>
    for (const key of keys) {
      const value = data?.[key]
      if (typeof value === 'boolean') return value
    }
    return undefined
  }

  return {
    loading,
    errors,
    controlMetas,
    defaultParameterValues,
    loadControls,
  }
}
