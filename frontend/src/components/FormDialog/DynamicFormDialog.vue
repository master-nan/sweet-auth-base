<template>
  <form-dialog-shell
    v-model="show"
    :title="title"
    :subtitle="dialogSubtitle"
    :submit-text="submitBtnText"
    :loading="loading"
    :readonly="isReadonly"
    :show-preview="showFormPreview"
    :embedded="embedded"
    :width="dialogWidth"
    :max-height="dialogMaxHeight"
    @submit="submitForm"
  >
    <div class="form-dialog-content">
      <!-- 加载状态 -->
      <div v-if="loadingData" class="full-width row flex-center q-my-xl">
        <q-spinner color="primary" size="3em" />
        <div class="q-ml-md text-primary text-subtitle1">加载中...</div>
      </div>

      <!-- 表单内容 -->
      <q-form v-else ref="formRef" @submit="submitForm" class="q-gutter-md">
        <!-- 表单顶部按钮 -->
        <div v-if="!isReadonly && formTopButtons.length" class="row q-gutter-xs q-mb-sm">
          <q-btn
            v-for="btn in formTopButtons"
            :key="btn.id || btn.code"
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            :disable="loading || btn.is_disabled"
            @click="emit('button-click', btn, formData)"
          />
        </div>

        <!-- 没有字段时显示提示 -->
        <div v-if="displayFields.length === 0" class="text-center q-pa-md text-grey">
          未找到可编辑字段，请检查表结构配置
        </div>

        <!-- 表单字段 -->
        <div v-else class="dynamic-form-sections" :class="{ 'is-simple': !isFieldMetadataForm }">
          <section
            v-for="group in displayFieldGroups"
            :key="group.key"
            class="dynamic-form-section"
          >
            <div v-if="isFieldMetadataForm" class="dynamic-form-section__head">
              <div>
                <div class="dynamic-form-section__title">{{ group.title }}</div>
                <div class="dynamic-form-section__desc">{{ group.description }}</div>
              </div>
            </div>

            <div class="row q-col-gutter-md">
              <div
                v-for="field in group.fields"
                :key="field.field_code"
                :class="getFieldColClass(field)"
              >
                <dynamic-field-control
                  v-model="formData[field.field_code]"
                  :field="field"
                  :context="getFieldControlContext(field)"
                  @field-input="handleFieldInput(field.field_code)"
                  @touched="markTouched(field.field_code)"
                  @organization-change="updateOrganizationSelectorValue(field, $event)"
                  @number-change="updateNumberFieldValue(field, $event)"
                  @number-step="adjustNumberField(field, $event)"
                  @number-keydown="(direction, event) => handleNumberArrow(field, direction, event)"
                  @dict-filter="filterDictCodeOptions"
                  @relation-filter="
                    (text, update, abort) => filterRelationSelect(field, text, update, abort)
                  "
                  @relation-preload="preloadRelationSelect(field)"
                  @relation-load-more="loadMoreRelationSelect(field, $event)"
                />
              </div>
            </div>
          </section>
        </div>

        <!-- 自定义内容插槽 -->
        <slot name="custom-fields" :form-data="formData"></slot>

        <!-- 隐藏的提交按钮，用于表单提交 -->
        <button type="submit" class="hidden" />
      </q-form>
    </div>

    <template #footer-left>
      <q-btn
        v-for="btn in isReadonly ? [] : formBottomButtons"
        :key="btn.id || btn.code"
        v-bind="menuButtonDisplayProps(btn)"
        :color="btn.color || 'grey-7'"
        :disable="loading || btn.is_disabled"
        flat
        class="q-mr-sm"
        @click="emit('button-click', btn, formData)"
      />
    </template>

    <template #footer-status>
      {{ formCompletionText }}
    </template>

    <template #preview>
      <div class="form-preview">
        <div class="form-preview__title">填写预览</div>
        <div class="form-preview__metrics">
          <div>
            <strong>{{ filledFieldCount }}/{{ displayFields.length }}</strong>
            <span>已填写</span>
          </div>
          <div>
            <strong>{{ missingRequiredFields.length }}</strong>
            <span>待完善</span>
          </div>
        </div>

        <div class="form-preview__panel">
          <div class="form-preview__panel-title">关键字段</div>
          <div v-for="item in previewItems" :key="item.code" class="form-preview__row">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>

        <div class="form-preview__panel">
          <div class="form-preview__panel-title">校验提示</div>
          <div v-if="previewHints.length === 0" class="form-preview__empty">暂无明显问题</div>
          <div v-for="hint in previewHints" :key="hint" class="form-preview__hint">
            <q-icon name="info" size="16px" color="primary" />
            <span>{{ hint }}</span>
          </div>
        </div>
      </div>
    </template>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick, type PropType } from 'vue'
import { QForm } from 'quasar'
import { type TableField } from 'src/api/services/sys-table'
import { type MenuButton } from 'src/api/services/sys-menu'
import { SysMenuButtonPosition } from 'src/types/enum'
import { SysTableFieldType, SysTableFieldInputType } from 'src/types/enum'
import { useDictStore } from 'src/stores/dict'
import { useDictApi } from 'src/api/services/sys-dict'
import { queryRuntimeRelationOptions } from 'src/api/services/runtime-relation'

import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import DynamicFieldControl, {
  type DynamicFieldControlContext,
} from 'src/components/FormDialog/DynamicFieldControl.vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import {
  coerceFieldValue,
  coerceDictOptions,
  compareExactDecimal,
  decodeHtmlEntities,
  defaultInputTypeForFieldType,
  defaultValueForField,
  getFieldControlType,
  inputTypesAllowingDictionary,
  isExactDecimalFieldType,
  isBooleanFieldMetadata,
  metadataDictDefault,
  parseLinkageConfig,
  resolveOrganizationSelectorConfig,
  selectLikeInputTypes,
} from 'src/utils/field-metadata'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { getFieldFormGridClass } from 'src/utils/field-layout'

// 使用泛型T扩展默认实体接口
interface BaseEntity {
  id?: number
  [key: string]: any
}

type FieldGroup = {
  key: string
  title: string
  description: string
  fields: TableField[]
}

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  editData: {
    type: Object as PropType<BaseEntity | null>,
    default: null,
  },
  // 表单相关配置
  title: {
    type: String,
    default: '表单',
  },
  fields: {
    type: Array as PropType<TableField[]>,
    default: () => [],
  },
  submitBtnText: {
    type: String,
    default: '保存',
  },
  formButtons: {
    type: Array as PropType<MenuButton[]>,
    default: () => [],
  },
  menuId: {
    type: Number,
    default: 0,
  },
  tableCode: {
    type: String,
    default: '',
  },
  readonly: {
    type: Boolean,
    default: false,
  },
  embedded: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue', 'submit', 'button-click'])

const formTopButtons = computed(() =>
  props.formButtons.filter((btn) => btn.position === SysMenuButtonPosition.FORM_TOP),
)
const formBottomButtons = computed(() =>
  props.formButtons.filter((btn) => btn.position === SysMenuButtonPosition.FORM_BOTTOM),
)

// 使用Pinia字典存储
const dictStore = useDictStore()
const dictApi = useDictApi()

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const show = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
const loadingData = ref(false)
const controlResetVersion = ref(0)
const isEdit = computed(() => !!props.editData?.id)
const isReadonly = computed(() => props.readonly)
const currentRecordId = computed(() => Number(formData.value.id || props.editData?.id || 0))
const formRef = ref<QForm>()
const formData = ref<Record<string, any>>({})
const validationActive = ref(false)
const isInitializing = ref(false)
const lazyRulesValue = computed(() => (validationActive.value ? false : 'ondemand'))
const touchedMap = ref<Record<string, boolean>>({})
const fieldRefs = ref<Record<string, any>>({})
// 存储联动字段的选项数据（响应式）
const fieldOptionsMap = ref<Record<string, any[]>>({})
const relationSelectStateMap = ref<
  Record<
    string,
    {
      page: number
      pageSize: number
      keyword: string
      hasMore: boolean
      loading: boolean
    }
  >
>({})
const dictCodeOptions = ref<Array<{ label: string; value: string }>>([])
const filteredDictCodeOptions = ref<Array<{ label: string; value: string }>>([])

const booleanToggleOptions = [
  { label: '否', value: false },
  { label: '是', value: true },
]

type OrganizationSelectorConfig = NonNullable<ReturnType<typeof resolveOrganizationSelectorConfig>>

const organizationSelectorConfigMap = computed<Record<string, OrganizationSelectorConfig>>(() => {
  const configs: Record<string, OrganizationSelectorConfig> = {}
  props.fields.forEach((field) => {
    const config = resolveOrganizationSelectorConfig(field)
    if (config) configs[field.field_code] = config
  })
  return configs
})

const getOrganizationSelectorConfig = (field: TableField) => {
  return organizationSelectorConfigMap.value[field.field_code] || null
}

const normalizeOrganizationSelectorId = (value: unknown): number | null => {
  const numericValue =
    typeof value === 'number'
      ? value
      : typeof value === 'string' && /^\d+$/.test(value.trim())
        ? Number(value)
        : Number.NaN
  return Number.isSafeInteger(numericValue) && numericValue > 0 ? numericValue : null
}

const normalizeOrganizationSelectorValue = (value: unknown, config: OrganizationSelectorConfig) => {
  if (!config.multiple) return normalizeOrganizationSelectorId(value)
  if (!Array.isArray(value)) return []

  return value
    .map(normalizeOrganizationSelectorId)
    .filter((item): item is number => item !== null)
    .filter((item, index, values) => values.indexOf(item) === index)
}

const normalizeBooleanFormValue = (value: unknown, fallback = false): boolean => {
  if (value === undefined || value === null || value === '') return fallback
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') return value !== 0
  if (typeof value !== 'string') return fallback
  const normalized = String(value).trim().toLowerCase()
  if (['true', '1', 't', 'yes', 'y', '是'].includes(normalized)) return true
  if (['false', '0', 'f', 'no', 'n', '否'].includes(normalized)) return false
  return fallback
}

const normalizeBooleanFormFields = () => {
  props.fields.forEach((field) => {
    if (isEdit.value ? !field.is_update_show : !field.is_insert_show) return
    if (!isBooleanFieldMetadata(field)) return
    const fallback = defaultValueForField(field) === true
    formData.value[field.field_code] = normalizeBooleanFormValue(
      formData.value[field.field_code],
      fallback,
    )
  })
}

const setFieldRef = (code: string) => (el: any) => {
  if (el) {
    fieldRefs.value[code] = el
  }
}

const markTouched = async (code: string) => {
  touchedMap.value[code] = true
  if (validationActive.value) {
    await nextTick()
    fieldRefs.value[code]?.validate?.()
  }
}

const handleFieldInput = async (code: string) => {
  if (!touchedMap.value[code]) {
    touchedMap.value[code] = true
  }
  if (!isInitializing.value) {
    applyFieldMetadataInputLinkage(code)
  }
  if (!validationActive.value && !touchedMap.value[code]) return
  await nextTick()
  fieldRefs.value[code]?.validate?.()
}

const updateOrganizationSelectorValue = (field: TableField, value: unknown) => {
  const config = getOrganizationSelectorConfig(field)
  if (!config) return
  formData.value[field.field_code] = normalizeOrganizationSelectorValue(value, config)
  void handleFieldInput(field.field_code)
}

const normalizeOrganizationSelectorFormFields = () => {
  props.fields.forEach((field) => {
    const config = getOrganizationSelectorConfig(field)
    if (!config) return
    formData.value[field.field_code] = normalizeOrganizationSelectorValue(
      formData.value[field.field_code],
      config,
    )
  })
}

const isFieldMetadataForm = computed(() => {
  const codes = new Set(props.fields.map((field) => field.field_code))
  return (
    codes.has('field_code') &&
    codes.has('field_type') &&
    codes.has('input_type') &&
    codes.has('dict_code')
  )
})

const isDictCodeField = (field: TableField) =>
  isFieldMetadataForm.value && field.field_code === 'dict_code'

const isLinkageConfigField = (field: TableField) =>
  isFieldMetadataForm.value && field.field_code === 'linkage_config'

const hasActiveMetadataLinkage = () =>
  !!parseLinkageConfig({
    linkage_config: formData.value.linkage_config,
  } as Partial<TableField>)

const filterDictCodeOptions = (value: string, update: (callback: () => void) => void) => {
  const keyword = String(value || '')
    .toLowerCase()
    .trim()
  update(() => {
    filteredDictCodeOptions.value = keyword
      ? dictCodeOptions.value.filter(
          (item) =>
            item.label.toLowerCase().includes(keyword) ||
            item.value.toLowerCase().includes(keyword),
        )
      : dictCodeOptions.value
  })
}

const applyFieldMetadataInputLinkage = (changedCode: string) => {
  if (!isFieldMetadataForm.value) return
  const fieldType = Number(formData.value.field_type) as SysTableFieldType
  const inputType = Number(formData.value.input_type) as SysTableFieldInputType
  const fieldCode = String(formData.value.field_code || '').trim()

  if (changedCode === 'field_type') {
    formData.value.input_type = defaultInputTypeForFieldType(fieldType)
    const defaultDict = metadataDictDefault(fieldCode, fieldType, props.tableCode)
    if (defaultDict && !formData.value.dict_code) {
      formData.value.dict_code = defaultDict
    }
  }

  if (changedCode === 'field_code') {
    const defaultDict = metadataDictDefault(fieldCode, fieldType, props.tableCode)
    if (defaultDict && !formData.value.dict_code) {
      formData.value.dict_code = defaultDict
    }
  }

  if (
    changedCode === 'dict_code' &&
    formData.value.dict_code &&
    !selectLikeInputTypes.has(inputType)
  ) {
    formData.value.input_type = SysTableFieldInputType.SELECT
  }

  if (
    (changedCode === 'linkage_config' || changedCode === 'submit') &&
    hasActiveMetadataLinkage() &&
    !selectLikeInputTypes.has(inputType)
  ) {
    formData.value.input_type = SysTableFieldInputType.SELECT
  }

  const nextInputType = Number(formData.value.input_type) as SysTableFieldInputType
  if (
    (changedCode === 'input_type' || changedCode === 'submit') &&
    !inputTypesAllowingDictionary.has(nextInputType) &&
    !hasActiveMetadataLinkage()
  ) {
    formData.value.dict_code = ''
    formData.value.linkage_config = ''
  }

  if (nextInputType === SysTableFieldInputType.BOOLEAN && !formData.value.dict_code) {
    formData.value.dict_code = 'whether'
  }
}

// 获取当前需要显示的字段（根据编辑或新增模式）
const displayFields = computed(() => {
  const baseFields = isEdit.value
    ? props.fields.filter((field) => field.is_update_show)
    : props.fields.filter((field) => field.is_insert_show)

  // 字段类型联动：根据当前表单的 field_type 隐藏长度相关字段
  const fieldType = formData.value.field_type
  if (
    fieldType === SysTableFieldType.DATE ||
    fieldType === SysTableFieldType.DATETIME ||
    fieldType === SysTableFieldType.TIME ||
    fieldType === SysTableFieldType.BOOLEAN
  ) {
    return baseFields.filter(
      (field) => field.field_code !== 'field_length' && field.field_code !== 'field_decimal_length',
    )
  }

  if (fieldType === SysTableFieldType.TEXT || fieldType === SysTableFieldType.JSON) {
    return baseFields.filter((field) => field.field_code !== 'field_decimal_length')
  }

  return baseFields
})

/**
 * 获取字典选项并将 value 转换为与字段数据库类型一致的类型
 * 字典 item_value 在数据库中是 VARCHAR，始终为字符串；
 * 整数转为 number；Decimal 保持字符串，避免 JavaScript Number 丢失精度。
 * 这样 q-select 的 emit-value + map-options 严格相等才能匹配上。
 */
const getTypedDictOptions = (field: TableField) => {
  if (!field.dict_code) return []
  const options = dictStore.getDictOptions(field.dict_code)
  return coerceDictOptions(options, field.field_type)
}

const getSelectOptions = (field: TableField) => {
  const linkedOptions = fieldOptionsMap.value[field.field_code]
  if (Array.isArray(linkedOptions) && linkedOptions.length > 0) return linkedOptions

  const schemaOptions = (field as any).options
  if (Array.isArray(schemaOptions) && schemaOptions.length > 0) return schemaOptions

  return getTypedDictOptions(field)
}

const fieldMetadataBaseCodes = new Set([
  'state',
  'table_id',
  'field_name',
  'field_code',
  'field_type',
  'field_length',
  'field_decimal_length',
  'input_type',
  'default_value',
  'dict_code',
])

const fieldMetadataBehaviorCodes = new Set([
  'is_primary_key',
  'is_index',
  'is_quick_search',
  'is_advanced_search',
  'is_sort',
  'is_null',
  'is_list_show',
  'is_insert_show',
  'is_update_show',
])

const makeFieldGroup = (
  key: string,
  title: string,
  description: string,
  fields: TableField[],
): FieldGroup | null => (fields.length > 0 ? { key, title, description, fields } : null)

const displayFieldGroups = computed<FieldGroup[]>(() => {
  const fields = displayFields.value
  if (!isFieldMetadataForm.value) {
    return [
      {
        key: 'base',
        title: '基础信息',
        description: isEdit.value ? '调整当前记录字段' : '填写新记录字段',
        fields,
      },
    ]
  }

  const baseFields = fields.filter((field) => fieldMetadataBaseCodes.has(field.field_code))
  const behaviorFields = fields.filter((field) => fieldMetadataBehaviorCodes.has(field.field_code))
  const advancedFields = fields.filter(
    (field) =>
      !fieldMetadataBaseCodes.has(field.field_code) &&
      !fieldMetadataBehaviorCodes.has(field.field_code),
  )

  return [
    makeFieldGroup('base', '基础信息', '字段名称、编码、类型和默认值', baseFields),
    makeFieldGroup(
      'behavior',
      '页面与查询',
      '控制列表、新增、编辑、查询和排序能力',
      behaviorFields,
    ),
    makeFieldGroup('advanced', '约束与高级配置', '字典、联动、表达式和扩展规则', advancedFields),
  ].filter((group): group is FieldGroup => Boolean(group))
})

const dialogSubtitle = computed(() => {
  if (isReadonly.value) return '查看记录详情'
  if (isFieldMetadataForm.value) {
    return isEdit.value ? '调整字段结构、页面能力和高级配置' : '配置字段结构、页面能力和高级配置'
  }
  return isEdit.value ? '编辑当前记录' : '创建记录'
})

const showFormPreview = computed(() => false)

const dialogWidth = computed(() =>
  isFieldMetadataForm.value ? 'min(980px, calc(100vw - 48px))' : 'min(760px, calc(100vw - 48px))',
)

const dialogMaxHeight = computed(() =>
  isFieldMetadataForm.value ? 'min(88vh, 860px)' : 'min(84vh, 760px)',
)

const isFilledValue = (value: any): boolean => {
  if (value === undefined || value === null) return false
  if (typeof value === 'string') return value.trim() !== ''
  if (Array.isArray(value)) return value.length > 0
  return true
}

const filledFieldCount = computed(
  () =>
    displayFields.value.filter((field) => isFilledValue(formData.value[field.field_code])).length,
)

const missingRequiredFields = computed(() =>
  displayFields.value.filter(
    (field) => !field.is_null && !isFilledValue(formData.value[field.field_code]),
  ),
)

const resolveFieldDisplayValue = (field: TableField) => {
  const value = formData.value[field.field_code]
  if (!isFilledValue(value)) return '-'
  if (getFieldInputType(field) === 'boolean') return value ? '是' : '否'
  const options = getSelectOptions(field)
  const option = Array.isArray(options) ? options.find((item) => item.value === value) : null
  if (option) return option.label
  if (Array.isArray(value)) return value.join('、')
  if (typeof value === 'object') return '已配置'
  return String(value)
}

const previewItems = computed(() =>
  displayFields.value.slice(0, 6).map((field) => ({
    code: field.field_code,
    label: field.field_name,
    value: resolveFieldDisplayValue(field),
  })),
)

const previewHints = computed(() => {
  const hints: string[] = []
  if (missingRequiredFields.value.length > 0) {
    hints.push(`还有 ${missingRequiredFields.value.length} 个必填字段未填写`)
  }
  if (isFieldMetadataForm.value && formData.value.input_type === SysTableFieldInputType.SELECT) {
    if (!formData.value.dict_code && !formData.value.linkage_config) {
      hints.push('下拉字段建议配置字典或关联字段')
    }
  }
  if (isFieldMetadataForm.value && formData.value.field_code) {
    const code = String(formData.value.field_code)
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(code)) {
      hints.push('字段编码建议使用字母、数字和下划线，且不要以数字开头')
    }
  }
  return hints
})

const formCompletionText = computed(() => {
  if (displayFields.value.length === 0) return ''
  if (missingRequiredFields.value.length > 0) {
    return `已填写 ${filledFieldCount.value}/${displayFields.value.length}，待完善 ${missingRequiredFields.value.length} 项`
  }
  return `已填写 ${filledFieldCount.value}/${displayFields.value.length}`
})

const buildRuntimeRelationOptions = (
  field: TableField,
  items: Array<{ value: string; label: string; parent_value?: string }>,
) =>
  items.map((item) => ({
    value: coerceFieldValue(item.value, field.field_type),
    label: item.label || item.value,
    ...(item.parent_value ? { parent_value: item.parent_value } : {}),
  }))

const relationPageSizeForLinkage = (cfg: any) => {
  const configuredSize = Number(cfg?.searchPageSize || cfg?.pageSize || 50)
  if (!Number.isFinite(configuredSize) || configuredSize <= 0) return 50
  return Math.min(Math.max(configuredSize, 20), 200)
}

const mergeOptions = (base: any[], incoming: any[]) => {
  const optionMap = new Map<string, any>()
  ;[...base, ...incoming].forEach((option) => {
    if (option?.value === undefined || option?.value === null) return
    optionMap.set(String(option.value), option)
  })
  return Array.from(optionMap.values())
}

const currentFieldValues = (field: TableField) => {
  const value = formData.value[field.field_code]
  const values = Array.isArray(value) ? value : [value]
  return values.filter((item) => item !== undefined && item !== null && item !== '')
}

const updateRelationSelectState = (
  fieldCode: string,
  state: Partial<(typeof relationSelectStateMap.value)[string]>,
) => {
  relationSelectStateMap.value = {
    ...relationSelectStateMap.value,
    [fieldCode]: {
      page: 1,
      pageSize: 50,
      keyword: '',
      hasMore: false,
      loading: false,
      ...(relationSelectStateMap.value[fieldCode] || {}),
      ...state,
    },
  }
}

// 从扁平数据构建级联树结构
const buildTreeFromFlat = (
  rows: Array<Record<string, any>>,
  cfg: any,
): Array<Record<string, any>> => {
  const labelKey = cfg?.labelKey || 'label'
  const valueKey = cfg?.valueKey || 'value'
  const configuredParentKey = cfg?.parentKey || 'parent_id'
  const parentKey =
    configuredParentKey === valueKey && rows.some((row) => row.parent_id !== undefined)
      ? 'parent_id'
      : configuredParentKey
  const childrenKey = cfg?.childrenKey || 'children'
  const rootValue = cfg?.rootValue ?? 0
  const normalizeTreeKey = (value: unknown) =>
    typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
      ? String(value)
      : ''

  // 先构建所有节点
  const nodeMap = new Map<any, Record<string, any>>()
  const allNodes = rows.map((row) => {
    const rawLabel = row[labelKey]
    const rawValue = row[valueKey]
    const node: Record<string, any> = {
      ...row,
      label: rawLabel ?? row.label ?? row.name ?? row.title ?? '',
      value: rawValue ?? row.value ?? row.id,
      [childrenKey]: [],
    }
    nodeMap.set(normalizeTreeKey(node.value), node)
    return node
  })

  // 构建树
  const tree: Array<Record<string, any>> = []
  allNodes.forEach((node) => {
    const parentVal = node[parentKey]
    // 根节点判断：parentKey 为 rootValue 或 null/undefined/空串，或找不到父节点
    const isRoot =
      normalizeTreeKey(parentVal) === normalizeTreeKey(rootValue) ||
      parentVal === null ||
      parentVal === undefined ||
      parentVal === ''
    if (isRoot) {
      tree.push(node)
    } else {
      const parent = nodeMap.get(normalizeTreeKey(parentVal))
      if (parent) {
        parent[childrenKey].push(node)
      } else {
        // 父节点不存在，作为根节点
        tree.push(node)
      }
    }
  })

  // 清理空 children
  const cleanEmpty = (nodes: Array<Record<string, any>>) => {
    nodes.forEach((n) => {
      if (n[childrenKey]?.length === 0) {
        delete n[childrenKey]
      } else if (n[childrenKey]?.length > 0) {
        cleanEmpty(n[childrenKey])
      }
    })
  }
  cleanEmpty(tree)

  return tree
}

const fetchRelationOptions = async (field: TableField, cfg: any, page: number, keyword: string) => {
  if (!field.id || props.menuId <= 0) return { options: [], total: 0 }
  const pageSize = relationPageSizeForLinkage(cfg)
  const result = await queryRuntimeRelationOptions(field.id, {
    menu_id: props.menuId,
    page,
    num: pageSize,
    keyword,
    source_values: formData.value,
  })
  return { options: buildRuntimeRelationOptions(field, result.items), total: result.total }
}

const loadSelectedRelationOptions = async (field: TableField, cfg: any, knownOptions: any[]) => {
  const selectedValues = currentFieldValues(field)
  if (selectedValues.length === 0) return []

  const knownValues = new Set(knownOptions.map((option) => String(option.value)))
  const missingValues = selectedValues.filter((value) => !knownValues.has(String(value)))
  if (missingValues.length === 0) return []

  if (!field.id || props.menuId <= 0) return []
  const result = await queryRuntimeRelationOptions(field.id, {
    menu_id: props.menuId,
    page: 1,
    num: Math.max(missingValues.length, 20),
    selected_values: missingValues.map(String),
    source_values: formData.value,
  })
  return buildRuntimeRelationOptions(field, result.items)
}

const loadRelationOptions = async (
  field: TableField,
  cfg: any,
  options: { page?: number; keyword?: string; append?: boolean; force?: boolean } = {},
) => {
  const fieldCode = field.field_code
  const page = options.page || 1
  const keyword = options.keyword || ''
  const append = !!options.append
  const state = relationSelectStateMap.value[fieldCode]
  if (
    !options.force &&
    page === 1 &&
    state?.keyword === keyword &&
    fieldOptionsMap.value[fieldCode]
  ) {
    return fieldOptionsMap.value[fieldCode]
  }
  if (state?.loading) return fieldOptionsMap.value[fieldCode] || []

  updateRelationSelectState(fieldCode, {
    page,
    keyword,
    pageSize: relationPageSizeForLinkage(cfg),
    loading: true,
  })

  try {
    const result = await fetchRelationOptions(field, cfg, page, keyword)
    const incomingOptions = result.options
    const cachedSelectedOptions = append ? fieldOptionsMap.value[fieldCode] || [] : []
    const selectedOptions = append
      ? []
      : await loadSelectedRelationOptions(field, cfg, incomingOptions)
    const optionsData = append
      ? mergeOptions(fieldOptionsMap.value[fieldCode] || [], incomingOptions)
      : mergeOptions(mergeOptions(cachedSelectedOptions, selectedOptions), incomingOptions)

    fieldOptionsMap.value = {
      ...fieldOptionsMap.value,
      [fieldCode]: optionsData,
    }
    updateRelationSelectState(fieldCode, {
      page,
      keyword,
      hasMore: page * relationPageSizeForLinkage(cfg) < result.total,
      loading: false,
    })
    return optionsData
  } catch (error) {
    console.warn('联动选项加载失败', error)
    if (!append) {
      fieldOptionsMap.value = {
        ...fieldOptionsMap.value,
        [fieldCode]: [],
      }
    }
    updateRelationSelectState(fieldCode, {
      page,
      keyword,
      hasMore: false,
      loading: false,
    })
    return fieldOptionsMap.value[fieldCode] || []
  }
}

// 加载级联树形选项
const loadCascaderOptions = async (field: TableField, cfg: any) => {
  if (!field.id || props.menuId <= 0) return []
  try {
    const rows: Array<{ value: unknown; label: string; parent_value?: string }> = []
    let page = 1
    let total = 0
    do {
      const result = await queryRuntimeRelationOptions(field.id, {
        menu_id: props.menuId,
        page,
        num: 200,
        source_values: formData.value,
      })
      rows.push(...buildRuntimeRelationOptions(field, result.items))
      total = result.total
      page += 1
    } while (rows.length < total)
    return buildTreeFromFlat(rows, {
      ...cfg,
      labelKey: 'label',
      valueKey: 'value',
      parentKey: 'parent_value',
    })
  } catch (error) {
    console.warn('级联选项加载失败', error)
    return []
  }
}

const resolveFieldOptions = async (field: TableField) => {
  if (getOrganizationSelectorConfig(field)) return
  const linkage = parseLinkageConfig(field)
  if (!linkage) return

  if (linkage.mode === 'relation') {
    const options = await loadRelationOptions(field, linkage, { force: true })
    fieldOptionsMap.value[field.field_code] = options
  } else if (linkage.mode === 'cascader') {
    const options = await loadCascaderOptions(field, linkage)
    fieldOptionsMap.value[field.field_code] = options
  }
}

const isRelationSelectField = (field: TableField) => {
  return parseLinkageConfig(field)?.mode === 'relation'
}

const isRelationOptionsLoading = (field: TableField) => {
  return !!relationSelectStateMap.value[field.field_code]?.loading
}

const hasMoreRelationOptions = (field: TableField) => {
  return !!relationSelectStateMap.value[field.field_code]?.hasMore
}

const preloadRelationSelect = (field: TableField) => {
  const linkage = parseLinkageConfig(field)
  if (linkage?.mode !== 'relation') return
  void loadRelationOptions(field, linkage)
}

const filterRelationSelect = (
  field: TableField,
  val: string,
  update: (callback: () => void) => void,
  abort?: () => void,
) => {
  const linkage = parseLinkageConfig(field)
  if (linkage?.mode !== 'relation') {
    update(() => undefined)
    return
  }

  void loadRelationOptions(field, linkage, {
    page: 1,
    keyword: val.trim(),
    force: true,
  })
    .then(() => update(() => undefined))
    .catch(() => abort?.())
}

const loadMoreRelationSelect = (field: TableField, details: { to?: number }) => {
  const linkage = parseLinkageConfig(field)
  if (linkage?.mode !== 'relation') return

  const fieldCode = field.field_code
  const state = relationSelectStateMap.value[fieldCode]
  if (!state?.hasMore || state.loading) return

  const options = fieldOptionsMap.value[fieldCode] || []
  if ((details.to ?? 0) < options.length - 2) return

  void loadRelationOptions(field, linkage, {
    page: state.page + 1,
    keyword: state.keyword,
    append: true,
    force: true,
  })
}

const linkageRefreshing = ref(false)
let linkageDebounceTimer: ReturnType<typeof setTimeout> | null = null
// 记录上次 filterMapping 相关的表单值快照，用于判断是否需要重新请求
let prevFilterSnapshot: Record<string, any> = {}

/** 收集所有联动字段的 filterMapping 中引用的源字段名 */
const collectFilterSourceFields = (): Set<string> => {
  const sources = new Set<string>()
  props.fields.forEach((field) => {
    const mapping = parseLinkageConfig(field)?.filterMapping || {}
    Object.values(mapping).forEach((src) => {
      if (typeof src === 'string') sources.add(src)
    })
  })
  return sources
}

/** 判断字段是否有 filterMapping 依赖 */
const hasFilterDependency = (field: TableField): boolean => {
  const mapping = parseLinkageConfig(field)?.filterMapping || {}
  return Object.keys(mapping).length > 0
}

const refreshLinkageOptions = async (forceAll = false) => {
  if (linkageRefreshing.value) return
  linkageRefreshing.value = true
  try {
    if (forceAll) {
      await Promise.all(props.fields.map((field) => resolveFieldOptions(field)))
    } else {
      // 只刷新有 filterMapping 依赖的字段
      const fieldsToRefresh = props.fields.filter((f) => hasFilterDependency(f))
      if (fieldsToRefresh.length > 0) {
        await Promise.all(fieldsToRefresh.map((field) => resolveFieldOptions(field)))
      }
    }
  } finally {
    linkageRefreshing.value = false
  }
}

/** 只在 filterMapping 相关的源字段值变化时才触发刷新 */
const smartRefreshLinkage = () => {
  const sourceFields = collectFilterSourceFields()
  if (sourceFields.size === 0) return // 没有任何字段有 filterMapping，无需 watch 刷新
  const currentSnapshot: Record<string, any> = {}
  sourceFields.forEach((key) => {
    currentSnapshot[key] = formData.value[key]
  })
  // 对比快照
  const changed = Object.keys(currentSnapshot).some(
    (key) => currentSnapshot[key] !== prevFilterSnapshot[key],
  )
  if (!changed) return
  prevFilterSnapshot = currentSnapshot
  void refreshLinkageOptions(false)
}

const debouncedRefreshLinkage = () => {
  if (linkageDebounceTimer) clearTimeout(linkageDebounceTimer)
  linkageDebounceTimer = setTimeout(() => smartRefreshLinkage(), 300)
}

// 判断是否应该隐藏内容（密码字段）
const shouldHideContent = (field: TableField) => {
  return (
    field.field_name.toLowerCase().includes('secret') ||
    field.field_name.toLowerCase().includes('密钥') ||
    field.field_name.toLowerCase().includes('password') ||
    field.field_name.toLowerCase().includes('密码')
  )
}

// 初始化表单数据
const initFormData = () => {
  controlResetVersion.value += 1
  loadingData.value = true
  if (isEdit.value && props.editData) {
    // 编辑模式：使用现有数据
    formData.value = { ...props.editData }

    if (typeof formData.value.linkage_config === 'string') {
      formData.value.linkage_config = decodeHtmlEntities(formData.value.linkage_config)
    }
  } else {
    // 新增模式：使用默认空对象
    formData.value = props.editData ? { ...props.editData } : {}
    // 根据字段类型设置合适的默认值
    props.fields.forEach((field) => {
      if (!field.is_insert_show) {
        return
      }
      // 关键修改：只有当字段未被设置或为undefined时才应用默认值
      if (formData.value[field.field_code] === undefined) {
        const selectorConfig = getOrganizationSelectorConfig(field)
        formData.value[field.field_code] = selectorConfig
          ? selectorConfig.multiple
            ? []
            : null
          : defaultValueForField(field)
      }
    })
  }
  normalizeOrganizationSelectorFormFields()
  normalizeBooleanFormFields()
  loadingData.value = false
}

// 获取字段最大长度
const getMaxLength = (field: TableField) => {
  return field.field_type === SysTableFieldType.VARCHAR ? field.field_length : undefined
}

const getNumberInputMode = (field: TableField) => {
  return isExactDecimalFieldType(field.field_type) ? 'decimal' : 'numeric'
}

const getNumberBindingValue = (field: TableField, name: string) => {
  const binding = String(field.binding || '')
    .split(/[|,]/)
    .map((item) => item.trim())
    .find((item) => item.startsWith(name + '='))
  if (!binding) return null
  const value = Number(binding.slice(name.length + 1))
  return Number.isFinite(value) ? value : null
}

type FileUploadFieldConfig = {
  multiple: boolean
  accept: string
  maxSize: number
  chunkThreshold: number
  concurrency: number
}

const parseFileUploadBinding = (field: TableField): Record<string, string | boolean | number> => {
  const binding = String(field.binding || '').trim()
  if (!binding) return {}

  if (binding.startsWith('{')) {
    try {
      const parsed = JSON.parse(binding)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        return parsed
      }
    } catch (error) {
      console.warn('文件上传配置解析失败', error)
    }
  }

  const result: Record<string, string | boolean> = {}
  binding
    .split('|')
    .map((item) => item.trim())
    .filter(Boolean)
    .forEach((item) => {
      const index = item.indexOf('=')
      if (index < 0) {
        result[item] = true
        return
      }
      result[item.slice(0, index).trim()] = item.slice(index + 1).trim()
    })
  return result
}

const getFileUploadConfig = (field: TableField): FileUploadFieldConfig => {
  const config = parseFileUploadBinding(field)
  const readBoolean = (name: string) => {
    const value = config[name]
    return value === true || value === 'true' || value === '1' || value === 'yes'
  }
  const readNumber = (name: string, fallback: number) => {
    const value = Number(config[name])
    return Number.isFinite(value) && value > 0 ? value : fallback
  }

  return {
    multiple: readBoolean('multiple'),
    accept: String(config.accept || '*'),
    maxSize: readNumber('maxSize', 50000),
    chunkThreshold: readNumber('chunkThreshold', 5),
    concurrency: readNumber('concurrency', 10),
  }
}

const adjustNumberField = (field: TableField, direction: 1 | -1) => {
  if (isReadonly.value || isExactDecimalFieldType(field.field_type)) return

  const code = field.field_code
  const currentValue = Number(formData.value[code])
  const current = Number.isFinite(currentValue) ? currentValue : 0
  const min = getNumberBindingValue(field, 'min')
  const max = getNumberBindingValue(field, 'max')

  let next = Math.trunc(current + direction)
  if (min !== null) next = Math.max(min, next)
  if (max !== null) next = Math.min(max, next)

  formData.value[code] = next
  markTouched(code)
  handleFieldInput(code)
}

const handleNumberArrow = (field: TableField, direction: 1 | -1, event: KeyboardEvent) => {
  if (isExactDecimalFieldType(field.field_type)) return
  event.preventDefault()
  adjustNumberField(field, direction)
}

const updateNumberFieldValue = (field: TableField, value: string | number | null) => {
  if (value === null || value === '') {
    formData.value[field.field_code] = value
  } else if (isExactDecimalFieldType(field.field_type)) {
    formData.value[field.field_code] = String(value)
  } else {
    const numeric = Number(value)
    formData.value[field.field_code] = Number.isFinite(numeric) ? numeric : value
  }
  handleFieldInput(field.field_code)
}

// 判断值是否为空
const isValueEmpty = (val: any): boolean => {
  if (val === undefined || val === null) return true
  if (typeof val === 'string' && val.trim() === '') return true
  if (Array.isArray(val) && val.length === 0) return true
  if (typeof val === 'object' && Object.keys(val).length === 0) return true
  return false
}

// 构建字段规则（内部方法，只在 computed 中调用）
// 注意：把 shouldValidate 判断移入每个 rule 函数内部，
// 保证规则数组结构稳定，避免 Quasar 因检测到 rules 变化而触发不一致的重新验证。
const buildFieldRules = (field: TableField) => {
  const rules: Array<(val: any) => boolean | string> = []

  const bindings = String(field.binding || '')
    .split(/[|,]/)
    .map((item) => item.trim())
    .filter(Boolean)

  const hasBinding = (name: string) =>
    bindings.some((item) => item === name || item.startsWith(name + '='))
  const getBindingValue = (name: string) => {
    const found = bindings.find((item) => item.startsWith(name + '='))
    return found ? found.slice(name.length + 1) : ''
  }

  const shouldSkip = () => !validationActive.value && !touchedMap.value[field.field_code]

  // 必填验证（可空字段不强制必填）
  if (!field.is_null) {
    rules.push((val) => {
      if (shouldSkip()) return true
      return !isValueEmpty(val) || `${field.field_name}不能为空`
    })
  }

  // select / cascader 字段的值由选项决定，跳过数字格式校验
  const fieldInputType = getFieldInputType(field)
  const isSelectLike = fieldInputType === 'select' || fieldInputType === 'cascader'

  const isIntegerField =
    field.field_type === SysTableFieldType.BIGINT ||
    field.field_type === SysTableFieldType.INT ||
    field.field_type === SysTableFieldType.SMALLINT
  const isNumberField = isIntegerField || isExactDecimalFieldType(field.field_type)

  // 根据字段类型添加验证（select/cascader 跳过格式校验）
  if (!isSelectLike && isIntegerField) {
    rules.push((val) => {
      if (shouldSkip()) return true
      return isValueEmpty(val) || /^-?\d+$/.test(String(val)) || `${field.field_name}必须是整数`
    })
  } else if (!isSelectLike && isExactDecimalFieldType(field.field_type)) {
    rules.push((val) => {
      if (shouldSkip()) return true
      if (isValueEmpty(val)) return true
      const match = String(val)
        .trim()
        .match(/^-?(\d+)(?:\.(\d+))?$/)
      if (!match) return `${field.field_name}必须是十进制数字`
      const integerDigits = match[1]!.replace(/^0+(?=\d)/, '').length
      const fractionalDigits = (match[2] || '').length
      const precision = Number(field.numeric_precision || 0)
      const scale = Number(field.numeric_scale || 0)
      if (fractionalDigits > scale) return `${field.field_name}小数位不能超过${scale}位`
      if (precision > 0 && integerDigits + fractionalDigits > precision) {
        return `${field.field_name}总位数不能超过${precision}位`
      }
      return true
    })
  } else if (field.field_type === SysTableFieldType.VARCHAR) {
    rules.push((val) => {
      if (shouldSkip()) return true
      return (
        isValueEmpty(val) ||
        String(val).length <= field.field_length ||
        `${field.field_name}长度不能超过${field.field_length}`
      )
    })
  }

  if (!isSelectLike && field.field_type === SysTableFieldType.SMALLINT) {
    rules.push((val) => {
      if (shouldSkip() || isValueEmpty(val)) return true
      const value = Number(val)
      return (value >= -32768 && value <= 32767) || `${field.field_name}必须在-32768到32767之间`
    })
  }

  if (hasBinding('min')) {
    const minText = getBindingValue('min')
    const minVal = Number(minText)
    if (isExactDecimalFieldType(field.field_type)) {
      rules.push((val) => {
        if (shouldSkip() || isValueEmpty(val)) return true
        const comparison = compareExactDecimal(val, minText)
        return comparison === null || comparison >= 0 || `${field.field_name}不能小于${minText}`
      })
    } else if (!Number.isNaN(minVal)) {
      rules.push((val) => {
        if (shouldSkip()) return true
        if (isValueEmpty(val)) return true
        return isNumberField
          ? Number(val) >= minVal || `${field.field_name}不能小于${minVal}`
          : String(val).length >= minVal || `${field.field_name}长度不能小于${minVal}`
      })
    }
  }

  if (hasBinding('max')) {
    const maxText = getBindingValue('max')
    const maxVal = Number(maxText)
    if (isExactDecimalFieldType(field.field_type)) {
      rules.push((val) => {
        if (shouldSkip() || isValueEmpty(val)) return true
        const comparison = compareExactDecimal(val, maxText)
        return comparison === null || comparison <= 0 || `${field.field_name}不能大于${maxText}`
      })
    } else if (!Number.isNaN(maxVal)) {
      rules.push((val) => {
        if (shouldSkip()) return true
        if (isValueEmpty(val)) return true
        return isNumberField
          ? Number(val) <= maxVal || `${field.field_name}不能大于${maxVal}`
          : String(val).length <= maxVal || `${field.field_name}长度不能超过${maxVal}`
      })
    }
  }

  if (hasBinding('email')) {
    const emailReg = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    rules.push((val) => {
      if (shouldSkip()) return true
      return isValueEmpty(val) || emailReg.test(String(val)) || `${field.field_name}格式不正确`
    })
  }

  if (hasBinding('url')) {
    const urlReg = /^(https?:\/\/)?([\w-]+\.)+[\w-]+(\/[\w-./?%&=]*)?$/
    rules.push((val) => {
      if (shouldSkip()) return true
      return isValueEmpty(val) || urlReg.test(String(val)) || `${field.field_name}格式不正确`
    })
  }

  if (hasBinding('phone') || hasBinding('mobile')) {
    const phoneReg = /^1\d{10}$/
    rules.push((val) => {
      if (shouldSkip()) return true
      return isValueEmpty(val) || phoneReg.test(String(val)) || `${field.field_name}格式不正确`
    })
  }

  if (hasBinding('regex') || hasBinding('regexp')) {
    const rule = getBindingValue('regex') || getBindingValue('regexp')
    if (rule) {
      try {
        const reg = new RegExp(rule)
        rules.push((val) => {
          if (shouldSkip()) return true
          return isValueEmpty(val) || reg.test(String(val)) || `${field.field_name}格式不正确`
        })
      } catch (error) {
        console.warn('无效的正则校验规则', error)
      }
    }
  }

  return rules
}

// 使用 computed 缓存每个字段的规则数组，确保规则引用在 re-render 时保持稳定。
// 若每次 render 都创建新数组，Quasar 检测到 rules prop 引用变化会触发全部字段重新验证，
// 导致选择某个 SELECT 时其他未填写字段也显示错误。
const fieldRulesMap = computed(() => {
  const map: Record<string, Array<(val: any) => boolean | string>> = {}
  for (const field of props.fields) {
    map[field.field_code] = buildFieldRules(field)
  }
  return map
})

// 模板中使用的方法：返回缓存的规则数组（引用稳定）
const getFieldRules = (field: TableField) => {
  return fieldRulesMap.value[field.field_code] || []
}

const getFieldHint = (field: TableField) => {
  if (field.field_code === 'form_span') {
    return '0为自动；表单最多2列，1为半行，2为整行'
  }
  if (field.field_code === 'detail_span') {
    return '0为自动；详情最多4列，1/2/3/4控制宽度，4为整行'
  }
  if (isDictCodeField(field)) {
    return '下拉、级联、布尔字段可配置字典'
  }
  if (field.field_code === 'binding') {
    return '校验：min=1|max=50|email|url|phone；文件：multiple=true|accept=.pdf,.docx|maxSize=500|chunkThreshold=5'
  }
  if (field.field_code === 'linkage_config') {
    return 'JSON示例：{"linkage":{"enabled":true,"mode":"relation","tableCode":"sys_user","labelKey":"user_name","valueKey":"id","filterMapping":{"foreign_key":"main_field"}}}'
  }
  if (field.field_code === 'tag') {
    return '示例：gorm:"size:128;comment:发件人密码" json:"sender_password"'
  }
  return ''
}

const getFieldPlaceholder = (field: TableField) => {
  return (field as any).placeholder || ''
}

const loadDictCodeOptions = async () => {
  if (!isFieldMetadataForm.value) return
  try {
    const res = await dictApi.queryDict({
      page: 1,
      num: 1000,
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
      quick_query: { keyword: '' },
      include_deleted: false,
    })
    const rows = Array.isArray(res.data) ? res.data : []
    dictCodeOptions.value = rows
      .filter((item) => item.dict_code)
      .map((item) => ({
        label: `${item.dict_name} (${item.dict_code})`,
        value: item.dict_code,
      }))
      .sort((a, b) => a.value.localeCompare(b.value))
    filteredDictCodeOptions.value = dictCodeOptions.value
  } catch (error) {
    console.warn('加载字典编码失败', error)
    dictCodeOptions.value = []
    filteredDictCodeOptions.value = []
  }
}

// 预加载所有需要的字典，改用Pinia store的方法
const preloadDictionaries = async () => {
  const dictCodes = new Set<string>()
  // 收集所有需要的字典代码
  props.fields.forEach((field) => {
    if (field.dict_code && !getOrganizationSelectorConfig(field)) {
      dictCodes.add(field.dict_code)
    }
  })

  // 使用Pinia store批量加载字典
  if (dictCodes.size > 0) {
    await dictStore.loadDicts(Array.from(dictCodes))
  }
  await loadDictCodeOptions()
}

// 获取字段列宽类
const getFieldColClass = (field: TableField) => {
  if (isLinkageConfigField(field) || isDictCodeField(field)) {
    return 'col-12'
  }
  return getFieldFormGridClass(field)
}

// 获取字段输入类型
const getFieldInputType = (field: TableField) => {
  return getFieldControlType(field)
}

/** 从 linkage_config 读取 cascader 的 selectable 配置 */
const getCascaderSelectable = (field: TableField): 'leaf' | 'any' | 'level' => {
  return parseLinkageConfig(field)?.selectable || 'any'
}

/** 从 linkage_config 读取是否显示完整路径 */
const getCascaderShowPath = (field: TableField): boolean => {
  return parseLinkageConfig(field)?.showPath !== false
}

const getFieldControlContext = (field: TableField): DynamicFieldControlContext => ({
  role: isLinkageConfigField(field)
    ? 'linkage-config'
    : isDictCodeField(field)
      ? 'dict-code'
      : 'standard',
  controlType: getFieldInputType(field),
  readonly: isReadonly.value,
  tableCode: props.tableCode,
  menuId: props.menuId,
  rowId: currentRecordId.value,
  lazyRules: lazyRulesValue.value,
  organization: getOrganizationSelectorConfig(field),
  rules: getFieldRules(field),
  hint: getFieldHint(field),
  placeholder: getFieldPlaceholder(field),
  options:
    getFieldInputType(field) === 'cascader'
      ? fieldOptionsMap.value[field.field_code] || []
      : getSelectOptions(field),
  dictCodeOptions: filteredDictCodeOptions.value,
  cascaderSelectable: getCascaderSelectable(field),
  cascaderShowPath: getCascaderShowPath(field),
  maxLength: getMaxLength(field),
  numberInputMode: getNumberInputMode(field),
  exactDecimal: isExactDecimalFieldType(field.field_type),
  hideContent: shouldHideContent(field),
  visibilityResetToken: controlResetVersion.value,
  relationSelect: isRelationSelectField(field),
  relationLoading: isRelationOptionsLoading(field),
  relationHasMore: hasMoreRelationOptions(field),
  booleanOptions: booleanToggleOptions,
  file: getFileUploadConfig(field),
  fieldRef: setFieldRef(field.field_code),
})

// 提交表单
const submitForm = async () => {
  if (isReadonly.value) {
    return
  }
  validationActive.value = true
  if (isFieldMetadataForm.value) {
    applyFieldMetadataInputLinkage('submit')
  }
  await nextTick()
  const ok = await formRef.value?.validate()
  const fieldResults = Object.values(fieldRefs.value).map((ref) => ref?.validate?.())
  const fieldOk = fieldResults.every((res) => res !== false)
  if (!ok || !fieldOk) return

  // linkage_config：确保提交时始终为字符串（JsonEditor 会将其解析为对象）
  if (formData.value.linkage_config !== undefined && formData.value.linkage_config !== null) {
    if (typeof formData.value.linkage_config === 'object') {
      formData.value.linkage_config = JSON.stringify(formData.value.linkage_config)
    } else if (typeof formData.value.linkage_config === 'string') {
      formData.value.linkage_config = decodeHtmlEntities(formData.value.linkage_config)
    }
  }
  normalizeBooleanFormFields()
  // 通过事件发射表单数据，让父组件决定如何处理
  emit('submit', {
    data: formData.value,
    isEdit: isEdit.value,
    id: isEdit.value && props.editData ? props.editData.id : undefined,
  })
}

// 监听对话框显示状态
watch(
  () => show.value,
  (val) => {
    if (val) {
      // 加载字典数据并初始化表单
      validationActive.value = false
      touchedMap.value = {}
      preloadDictionaries().then(async () => {
        isInitializing.value = true
        initFormData()
        await nextTick()
        isInitializing.value = false
        prevFilterSnapshot = {}
        await refreshLinkageOptions(true)
        formRef.value?.resetValidation()
      })
    }
  },
)

watch(
  () => formData.value,
  () => {
    if (isInitializing.value) return
    debouncedRefreshLinkage()
  },
  { deep: true },
)

// 监听编辑数据变化
watch(
  () => props.editData,
  () => {
    if (props.editData && show.value) {
      initFormData()
    }
  },
)

watch(
  () =>
    props.fields.map((field) =>
      [field.id, field.field_code, field.input_type, field.dict_code, field.linkage_config].join(
        ':',
      ),
    ),
  async () => {
    if (!show.value || props.fields.length === 0) return
    await preloadDictionaries()
    isInitializing.value = true
    initFormData()
    await nextTick()
    isInitializing.value = false
    prevFilterSnapshot = {}
    await refreshLinkageOptions(true)
    formRef.value?.resetValidation()
  },
)

// 组件挂载时初始化
onMounted(() => {
  if (show.value) {
    validationActive.value = false
    touchedMap.value = {}
    preloadDictionaries().then(async () => {
      isInitializing.value = true
      initFormData()
      await nextTick()
      isInitializing.value = false
      prevFilterSnapshot = {}
      await refreshLinkageOptions(true)
      formRef.value?.resetValidation()
    })
  }
})
</script>

<style scoped lang="scss">
.form-dialog-content {
  min-height: 0;
}

.dynamic-form-sections {
  display: grid;
  gap: 12px;
}

.dynamic-form-sections.is-simple {
  display: block;
}

.dynamic-form-section {
  padding: 16px;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.dynamic-form-sections.is-simple .dynamic-form-section {
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
}

.dynamic-form-section__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.dynamic-form-section__title {
  font-size: 15px;
  line-height: 1.3;
  font-weight: 800;
  color: var(--app-text-strong);
}

.dynamic-form-section__desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.form-preview {
  display: grid;
  gap: 14px;
}

.form-preview__title {
  font-size: 15px;
  font-weight: 800;
  color: var(--app-text-strong);
}

.form-preview__metrics {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.form-preview__metrics > div {
  min-width: 0;
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background: var(--app-surface-muted);
}

.form-preview__metrics strong {
  display: block;
  font-size: 20px;
  line-height: 1.1;
  color: var(--app-text-strong);
}

.form-preview__metrics span {
  display: block;
  margin-top: 5px;
  font-size: 12px;
  color: #7b879b;
}

.form-preview__panel {
  padding: 12px;
  border-radius: 8px;
  border: 1px solid var(--app-border);
  background: var(--app-surface);
}

.form-preview__panel-title {
  margin-bottom: 9px;
  font-size: 13px;
  font-weight: 800;
  color: #26334d;
}

.form-preview__row {
  display: grid;
  grid-template-columns: 78px minmax(0, 1fr);
  gap: 8px;
  padding: 7px 0;
  border-top: 1px solid #edf1f8;
}

.form-preview__row:first-of-type {
  border-top: 0;
}

.form-preview__row span {
  color: #8290a6;
  font-size: 12px;
}

.form-preview__row strong {
  min-width: 0;
  font-size: 12px;
  color: var(--app-text-strong);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.form-preview__hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: #56657c;
}

.form-preview__empty {
  font-size: 12px;
  color: #8a96a9;
}
</style>
