<template>
  <q-dialog v-model="dialogVisible" :maximized="$q.screen.lt.md" persistent>
    <q-card class="advanced-search-dialog">
      <!-- 固定标题区域 -->
      <q-card-section class="advanced-search-header row items-center">
        <div class="advanced-search-title">
          <q-icon name="manage_search" size="22px" />
          <span>{{ displayTitle }}</span>
        </div>
        <q-space />
        <q-btn
          icon="close"
          flat
          round
          dense
          size="sm"
          :aria-label="t('ui.closeTheQueryConditionWindow')"
          v-close-popup
        >
          <q-tooltip>{{ t('ui.close') }}</q-tooltip>
        </q-btn>
      </q-card-section>

      <q-banner
        v-if="sourceName"
        dense
        class="bg-primary-1 text-primary q-mx-md q-mt-sm rounded-borders"
      >
        {{ t('ui.currentProgramme') }}{{ sourceName
        }}<span v-if="dirty">{{ t('ui.modified') }}</span>
      </q-banner>
      <q-banner
        v-if="readOnlyDepth"
        dense
        class="bg-warning text-dark q-mx-md q-mt-sm rounded-borders"
      >
        {{ t('ui.theCurrentSchemeContainsAThirdLayerOfConditionsThis') }}
      </q-banner>

      <!-- 内容区域（可滚动） -->
      <q-card-section class="advanced-search-content q-pa-md">
        <q-form v-if="!readOnlyDepth" ref="form" greedy>
          <div class="row justify-center">
            <div class="col-12" :style="{ maxWidth: maxWidth + 'px' }">
              <!-- 表达式组循环 -->
              <div
                v-for="(expression, eIndex) in queryModel.expressions"
                :key="eIndex"
                class="expression-group"
              >
                <q-card flat class="expression-card">
                  <q-card-section class="expression-card-head">
                    <div class="row items-center">
                      <div class="text-subtitle2 text-weight-medium text-primary">
                        {{ t('ui.expressionGroup') }} {{ eIndex + 1 }}
                      </div>
                      <q-space />
                      <q-btn
                        v-if="queryModel.expressions.length > 1"
                        flat
                        round
                        dense
                        size="sm"
                        icon="cancel"
                        color="primary"
                        @click="removeExpression(eIndex)"
                      />
                      <q-btn
                        flat
                        dense
                        size="sm"
                        icon="add_to_photos"
                        :label="t('ui.addExpression')"
                        color="primary"
                        @click="addExpression(eIndex)"
                      />
                    </div>
                  </q-card-section>
                  <q-separator />
                  <q-card-section class="expression-card-body">
                    <!-- 规则列表 - 每行包含逻辑关系+字段+操作符+值 -->
                    <advanced-query-rule-row
                      v-for="(rule, rIndex) in expression.rules"
                      :key="rIndex"
                      v-model:logic="expression.logic"
                      :rule="rule"
                      :is-first="rIndex === 0"
                      :can-remove="expression.rules.length > 1"
                      :show-logic="true"
                      :binding-labels="
                        bindingLabelsForRule(`/expressions/${eIndex}/rules/${rIndex}/value`)
                      "
                      :fields="fields"
                      :field-label-key="fieldLabelKey"
                      :field-value-key="fieldValueKey"
                      :expression-logic-options="expressionLogicOptions"
                      :expression-type-options-for-rule="expressionTypeOptionsForRule"
                      :boolean-options="booleanOptions"
                      :organization-selector-config-for-rule="organizationSelectorConfigForRule"
                      :update-organization-selector-value="updateOrganizationSelectorValue"
                      :is-null-operator="isNullOperator"
                      :has-dict-rule="hasDictRule"
                      :has-relation-rule="hasRelationRule"
                      :is-boolean-rule="isBooleanRule"
                      :is-multi-value-rule="isMultiValueRule"
                      :is-free-input-multi-value-rule="isFreeInputMultiValueRule"
                      :is-range-rule="isRangeRule"
                      :dict-options-for-rule="dictOptionsForRule"
                      :relation-options-for-rule="relationOptionsForRule"
                      :is-relation-loading="isRelationLoading"
                      :has-more-relation-options="hasMoreRelationOptions"
                      :value-rules="valueRules"
                      :input-type-for-rule="inputTypeForRule"
                      :value-placeholder-for-rule="valuePlaceholderForRule"
                      :range-placeholder-for-rule="rangePlaceholderForRule"
                      :filter-relation-options="filterRelationOptions"
                      :preload-relation-options="preloadRelationOptions"
                      :load-more-relation-options="loadMoreRelationOptions"
                      @update-field="
                        (value) =>
                          updateRuleFieldAt(
                            rule,
                            value,
                            `/expressions/${eIndex}/rules/${rIndex}/value`,
                          )
                      "
                      @update-expression-type="
                        () =>
                          updateRuleExpressionTypeAt(
                            rule,
                            `/expressions/${eIndex}/rules/${rIndex}/value`,
                          )
                      "
                      @remove="() => removeRule(eIndex, rIndex)"
                      @add="() => addRule(eIndex)"
                      @clear-bindings="
                        () => clearBindingsForRule(`/expressions/${eIndex}/rules/${rIndex}/value`)
                      "
                    />

                    <!-- 嵌套查询区域 -->
                    <div v-if="enableNested" class="nested-section">
                      <q-card flat class="nested-card">
                        <q-card-section class="nested-section-head">
                          <div class="row items-center">
                            <div class="text-subtitle2 text-primary">
                              {{ t('ui.embeddedCondition') }}
                            </div>
                            <q-space />
                            <q-btn
                              color="primary"
                              icon="fa-regular fa-square-plus"
                              :label="t('ui.addEmbeddedGroup')"
                              dense
                              flat
                              size="sm"
                              @click="addNestedGroup(eIndex)"
                            />
                          </div>
                        </q-card-section>

                        <q-separator
                          color="primary"
                          v-if="expression.nested && expression.nested.length > 0"
                        />

                        <!-- 嵌套组 -->
                        <q-card-section
                          v-if="expression.nested && expression.nested.length > 0"
                          class="q-pa-sm"
                        >
                          <div class="nested-groups">
                            <q-card
                              v-for="(nest, nIndex) in expression.nested"
                              :key="`nest-${nIndex}`"
                              flat
                              bordered
                              class="nested-group q-mb-xs"
                            >
                              <q-card-section class="nested-header q-py-xs">
                                <div class="row items-center">
                                  <div class="text-subtitle2 text-primary">
                                    {{ t('ui.embeddedGroup') }} {{ nIndex + 1 }}
                                  </div>
                                  <q-space />
                                  <q-btn
                                    flat
                                    round
                                    dense
                                    size="sm"
                                    icon="close"
                                    color="primary"
                                    @click="() => removeNest(eIndex, nIndex)"
                                  />
                                </div>
                              </q-card-section>
                              <q-separator color="primary" />
                              <q-card-section class="q-pa-sm">
                                <!-- 嵌套规则列表 - 每行包含逻辑关系+字段+操作符+值 -->
                                <advanced-query-rule-row
                                  v-for="(rule, rIndex) in nest.rules"
                                  :key="rIndex"
                                  v-model:logic="nest.logic"
                                  :rule="rule"
                                  :is-first="rIndex === 0"
                                  :can-remove="nest.rules.length > 1"
                                  :show-logic="true"
                                  :binding-labels="
                                    bindingLabelsForRule(
                                      `/expressions/${eIndex}/nested/${nIndex}/rules/${rIndex}/value`,
                                    )
                                  "
                                  :fields="fields"
                                  :field-label-key="fieldLabelKey"
                                  :field-value-key="fieldValueKey"
                                  :expression-logic-options="expressionLogicOptions"
                                  :expression-type-options-for-rule="expressionTypeOptionsForRule"
                                  :boolean-options="booleanOptions"
                                  :organization-selector-config-for-rule="
                                    organizationSelectorConfigForRule
                                  "
                                  :update-organization-selector-value="
                                    updateOrganizationSelectorValue
                                  "
                                  :is-null-operator="isNullOperator"
                                  :has-dict-rule="hasDictRule"
                                  :has-relation-rule="hasRelationRule"
                                  :is-boolean-rule="isBooleanRule"
                                  :is-multi-value-rule="isMultiValueRule"
                                  :is-free-input-multi-value-rule="isFreeInputMultiValueRule"
                                  :is-range-rule="isRangeRule"
                                  :dict-options-for-rule="dictOptionsForRule"
                                  :relation-options-for-rule="relationOptionsForRule"
                                  :is-relation-loading="isRelationLoading"
                                  :has-more-relation-options="hasMoreRelationOptions"
                                  :value-rules="valueRules"
                                  :input-type-for-rule="inputTypeForRule"
                                  :value-placeholder-for-rule="valuePlaceholderForRule"
                                  :range-placeholder-for-rule="rangePlaceholderForRule"
                                  :filter-relation-options="filterRelationOptions"
                                  :preload-relation-options="preloadRelationOptions"
                                  :load-more-relation-options="loadMoreRelationOptions"
                                  @update-field="
                                    (value) =>
                                      updateRuleFieldAt(
                                        rule,
                                        value,
                                        `/expressions/${eIndex}/nested/${nIndex}/rules/${rIndex}/value`,
                                      )
                                  "
                                  @update-expression-type="
                                    () =>
                                      updateRuleExpressionTypeAt(
                                        rule,
                                        `/expressions/${eIndex}/nested/${nIndex}/rules/${rIndex}/value`,
                                      )
                                  "
                                  @remove="() => removeNestRule(eIndex, nIndex, rIndex)"
                                  @add="() => addNestRule(eIndex, nIndex)"
                                  @clear-bindings="
                                    () =>
                                      clearBindingsForRule(
                                        `/expressions/${eIndex}/nested/${nIndex}/rules/${rIndex}/value`,
                                      )
                                  "
                                />
                              </q-card-section>
                            </q-card>
                          </div>
                        </q-card-section>
                      </q-card>
                    </div>
                  </q-card-section>
                </q-card>
              </div>
            </div>
          </div>
        </q-form>
        <q-separator v-if="!readOnlyDepth" class="q-my-sm" />
        <section class="advanced-query-preview" :aria-label="t('ui.conditionalPreview')">
          <div class="advanced-query-preview__title">{{ t('ui.conditionalPreview') }}</div>
          <query-scheme-preview
            :payload="previewPayload"
            :fields="previewFields"
            :menu-id="currentMenuId"
          />
        </section>
      </q-card-section>

      <!-- 固定底部按钮区域 -->
      <q-card-actions align="right" class="advanced-search-footer">
        <q-btn
          v-if="isSchemeConditionEditor"
          flat
          :label="t('ui.cancel')"
          @click="cancelConditionEdit"
        />
        <q-btn v-else-if="!readOnlyDepth" outline color="secondary" @click="resetFilter">
          <q-icon left size="sm" name="restart_alt" /> {{ t('ui.reset') }}
        </q-btn>
        <q-btn v-if="!readOnlyDepth" color="primary" @click="completeQueryAction">
          <q-icon left size="sm" :name="isSchemeConditionEditor ? 'check' : 'search'" />
          {{ isSchemeConditionEditor ? t('ui.sure') : t('ui.search') }}
        </q-btn>
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'AdvancedQuery' })
import { ref, computed, watch, type PropType } from 'vue'
import { useQuasar, type QForm } from 'quasar'
import {
  ExpressionLogic,
  ExpressionLogicMap,
  ExpressionType,
  ExpressionTypeMap,
  SysTableFieldType,
} from 'src/types/enum'
import type { Query, QueryRule } from 'src/types/global'
import { useDictStore } from 'src/stores/dict'
import { useUserStore } from 'src/stores/user'
import { queryRuntimeRelationOptions } from 'src/api/services/runtime-relation'
import {
  coerceFieldValue,
  isBooleanFieldMetadata,
  parseLinkageConfig,
  queryValueHtmlInputType,
  resolveOrganizationSelectorConfig,
} from 'src/utils/field-metadata'
import {
  isIncompleteQueryRule,
  isMultiValueExpressionType,
  isRangeExpressionType,
  isTextMultiKeywordExpressionType,
  splitMultiValueText,
  normalizeQuerySchemePayload,
  queryExpressionDepth,
} from 'src/utils/query-state'
import type { TableField } from 'src/api/services/sys-table'
import {
  QUERY_SCHEME_BINDING_LABELS,
  type QuerySchemeBinding,
} from 'src/modules/query-scheme/types'
import { findMenuByName, toPositiveMenuId } from 'src/utils/menu-context'
import AdvancedQueryRuleRow from './AdvancedQueryRuleRow.vue'
import QuerySchemePreview from 'src/components/QueryScheme/QuerySchemePreview.vue'
import { Router } from 'src/router'

const { t } = useI18n({ useScope: 'global' })

const $q = useQuasar()
const form = ref<QForm>()
const dictStore = useDictStore()
const userStore = useUserStore()

type FieldRecord = Record<string, any> & Partial<TableField>
type RelationOption = {
  label: string
  value: unknown
}
type RelationOptionState = {
  page: number
  pageSize: number
  keyword: string
  hasMore: boolean
}
type RelationLoadOptions = {
  keyword?: string
  page?: number
  append?: boolean
  force?: boolean
}
type QSelectFilterUpdate = (callback: () => void) => void
type QSelectFilterAbort = () => void
type VirtualScrollDetails = {
  to?: number
}
// 业务查询会立即触发列表查询；方案条件编辑只确认表达式并返回上层保存流程。
type AdvancedQueryUsage = 'business-query' | 'scheme-condition-editor'

const props = defineProps({
  // 第一个v-model，控制对话框显示/隐藏
  modelValue: {
    type: Boolean,
    default: false,
  },
  // 第二个v-model，查询条件
  queryModel: {
    type: Object as () => Query,
    required: true,
  },
  title: {
    type: String,
    default: '',
  },
  maxWidth: {
    type: Number,
    default: 1040,
  },
  fields: {
    type: Array,
    required: true,
  },
  fieldLabelKey: {
    type: String,
    default: 'field_name',
  },
  fieldValueKey: {
    type: String,
    default: 'field_code',
  },
  enableNested: {
    type: Boolean,
    default: true,
  },
  menuId: {
    type: Number,
    default: 0,
  },
  bindings: {
    type: Array as PropType<QuerySchemeBinding[]>,
    default: () => [],
  },
  sourceName: {
    type: String,
    default: '',
  },
  dirty: {
    type: Boolean,
    default: false,
  },
  usage: {
    type: String as PropType<AdvancedQueryUsage>,
    default: 'business-query',
  },
})

const displayTitle = computed(() => props.title || t('ui.advancedQuery'))

const currentMenuId = computed(() => {
  const explicit = toPositiveMenuId(props.menuId)
  if (explicit > 0) return explicit
  return findMenuByName(userStore.menus, String(Router?.currentRoute.value.name || ''))?.id || 0
})

const emit = defineEmits([
  'update:modelValue',
  'update:queryModel',
  'update:bindings',
  'search',
  'confirm',
  'cancel',
])

const booleanOptions = [
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

const readOnlyDepth = computed(() => queryExpressionDepth(props.queryModel.expressions) > 2)
const isSchemeConditionEditor = computed(() => props.usage === 'scheme-condition-editor')
const previewPayload = computed(() => normalizeQuerySchemePayload(props.queryModel, props.bindings))
const previewFields = computed(() => props.fields as TableField[])
const bindingLabelsForRule = (pointer: string) =>
  props.bindings
    .filter((binding) => binding.pointer === pointer || binding.pointer.startsWith(`${pointer}/`))
    .map((binding) => QUERY_SCHEME_BINDING_LABELS[binding.kind])
const clearBindingsForRule = (pointer: string) => {
  emit(
    'update:bindings',
    props.bindings.filter(
      (binding) => binding.pointer !== pointer && !binding.pointer.startsWith(`${pointer}/`),
    ),
  )
}
const clearAllBindings = () => {
  if (props.bindings.length) emit('update:bindings', [])
}
const updateRuleFieldAt = (rule: QueryRule, value: unknown, pointer: string) => {
  clearBindingsForRule(pointer)
  updateRuleField(rule, value)
}
const updateRuleExpressionTypeAt = (rule: QueryRule, pointer: string) => {
  clearBindingsForRule(pointer)
  updateRuleExpressionType(rule)
}

const relationOptionsMap = ref<Record<string, RelationOption[]>>({})
const filteredRelationOptionsMap = ref<Record<string, RelationOption[]>>({})
const relationOptionsLoading = ref<Record<string, boolean>>({})
const relationOptionsStateMap = ref<Record<string, RelationOptionState>>({})

// 控制对话框显示
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => {
    emit('update:modelValue', val)
  },
})

// 表达式逻辑选项
const expressionLogicOptions = computed(() => {
  return Object.values(ExpressionLogic)
    .filter((value) => typeof value === 'number')
    .map((value) => ({
      label: ExpressionLogicMap[value],
      value,
    }))
})

// 表达式类型选项
const expressionTypeOptions = computed(() => {
  return Object.values(ExpressionType)
    .filter((value) => typeof value === 'number')
    .map((value) => ({
      label: ExpressionTypeMap[value],
      value,
    }))
})

const nullableExpressionTypes = [ExpressionType.IS_NULL, ExpressionType.IS_NOT_NULL]
const organizationSelectorExpressionTypes = [ExpressionType.EQ, ExpressionType.IN]
const equalityExpressionTypes = [
  ExpressionType.EQ,
  ExpressionType.NE,
  ExpressionType.IN,
  ExpressionType.NOT_IN,
  ...nullableExpressionTypes,
]
const expressionTypeOptionMap = computed(() => {
  return new Map(expressionTypeOptions.value.map((option) => [option.value, option]))
})

const findField = (fieldCode: string) => {
  return (props.fields as FieldRecord[]).find((field) => field[props.fieldValueKey] === fieldCode)
}

const organizationSelectorConfigForField = (field?: FieldRecord) => {
  return field ? resolveOrganizationSelectorConfig(field) : null
}

const organizationSelectorConfigForRule = (rule: QueryRule) => {
  return organizationSelectorConfigForField(findField(rule.field))
}

const updateRuleField = (rule: QueryRule, fieldCode: unknown) => {
  if (!fieldCode) {
    delete rule.type
    delete rule.expression_type
    rule.value = null
    return
  }
  if (typeof fieldCode !== 'string' && typeof fieldCode !== 'number') return
  const field = findField(String(fieldCode))
  if (field?.field_type !== undefined) {
    rule.type = field.field_type
  } else {
    delete rule.type
  }
  delete rule.expression_type
  rule.value = null
  void loadRelationOptionsForField(field)
}

const nullExpressionTypes = new Set([ExpressionType.IS_NULL, ExpressionType.IS_NOT_NULL])
const isNullOperator = (rule: QueryRule) => {
  return nullExpressionTypes.has(rule.expression_type as ExpressionType)
}

const isMultiValueRule = (rule: QueryRule) => {
  return isMultiValueExpressionType(rule.expression_type)
}

const isRangeRule = (rule: QueryRule) => {
  return isRangeExpressionType(rule.expression_type)
}

const hasOptionValueControl = (rule: QueryRule) => {
  return (
    !!organizationSelectorConfigForRule(rule) ||
    hasDictRule(rule) ||
    hasRelationRule(rule) ||
    isBooleanRule(rule)
  )
}

const isFreeInputMultiValueRule = (rule: QueryRule) => {
  return (
    (isMultiValueRule(rule) || isTextMultiKeywordExpressionType(rule.expression_type)) &&
    !hasOptionValueControl(rule)
  )
}

const arrayValueToInputText = (values: unknown[]) => {
  return values
    .filter((value) => value !== null && value !== undefined && value !== '')
    .map((value) => String(value).trim())
    .filter(Boolean)
    .join(', ')
}

const updateRuleExpressionType = (rule: QueryRule) => {
  if (isNullOperator(rule)) {
    rule.value = null
    return
  }

  if (isRangeRule(rule)) {
    if (Array.isArray(rule.value)) {
      rule.value = [rule.value[0] ?? null, rule.value[1] ?? null]
      return
    }
    rule.value = hasValue(rule.value) ? [rule.value, null] : [null, null]
    return
  }

  if (isMultiValueRule(rule)) {
    if (hasOptionValueControl(rule)) {
      if (Array.isArray(rule.value)) return
      rule.value = hasValue(rule.value) ? [rule.value] : []
      return
    }

    if (Array.isArray(rule.value)) {
      rule.value = arrayValueToInputText(rule.value)
    }
    return
  }

  if (isTextMultiKeywordExpressionType(rule.expression_type) && Array.isArray(rule.value)) {
    rule.value = arrayValueToInputText(rule.value)
    return
  }

  if (
    !isMultiValueRule(rule) &&
    !isTextMultiKeywordExpressionType(rule.expression_type) &&
    Array.isArray(rule.value)
  ) {
    rule.value = rule.value[0] ?? null
  }
}

const expressionTypesForField = (field?: FieldRecord) => {
  if (!field) return Object.values(ExpressionType).filter((value) => typeof value === 'number')
  if (organizationSelectorConfigForField(field)) return organizationSelectorExpressionTypes
  return field.allowed_operators?.length ? field.allowed_operators : equalityExpressionTypes
}

const expressionTypeOptionsForRule = (rule: QueryRule) => {
  return expressionTypesForField(findField(rule.field))
    .map((value) => expressionTypeOptionMap.value.get(value))
    .filter((option): option is { label: string; value: ExpressionType } => !!option)
}

const normalizeRuleExpressionType = (rule: QueryRule) => {
  if (!rule.field) return
  const field = findField(rule.field)
  if (!field) return
  if (field.field_type !== undefined) {
    rule.type = field.field_type
  }

  const availableTypes = expressionTypesForField(field)
  if (rule.expression_type === undefined) {
    rule.value = null
    return
  }
  if (!availableTypes.includes(rule.expression_type as ExpressionType)) {
    delete rule.expression_type
    rule.value = null
    return
  }
  updateRuleExpressionType(rule)
  if (organizationSelectorConfigForField(field)) {
    updateOrganizationSelectorValue(rule, rule.value)
  }
}

const normalizeQueryExpressionTypes = () => {
  props.queryModel.expressions.forEach((expression) => {
    expression.logic ??= ExpressionLogic.AND
    expression.rules.forEach(normalizeRuleExpressionType)
    expression.nested?.forEach((nested) => {
      nested.logic ??= ExpressionLogic.AND
      nested.rules.forEach(normalizeRuleExpressionType)
    })
  })
}

const coerceDictValue = (value: unknown, rule: QueryRule) => {
  const field = findField(rule.field)
  const fieldType = rule.type ?? field?.field_type
  return coerceFieldValue(value, fieldType)
}

const hasDictRule = (rule: QueryRule) => {
  return !organizationSelectorConfigForRule(rule) && !!findField(rule.field)?.dict_code
}

const dictOptionsForRule = (rule: QueryRule) => {
  const field = findField(rule.field)
  if (!field?.dict_code) return []
  return dictStore.getDictOptions(field.dict_code).map((option) => ({
    ...option,
    value: coerceDictValue(option.value, rule),
  }))
}

const isBooleanRule = (rule: QueryRule) => isBooleanFieldMetadata(findField(rule.field), rule.type)

const relationLinkageForRule = (rule: QueryRule) => {
  if (organizationSelectorConfigForRule(rule)) return null
  if (hasDictRule(rule)) return null
  const field = findField(rule.field)
  if (!field) return null
  const linkage = parseLinkageConfig(field as TableField)
  if (linkage?.mode === 'relation' || linkage?.mode === 'cascader') {
    return linkage
  }
  return null
}

const hasRelationRule = (rule: QueryRule) => {
  return !!relationLinkageForRule(rule)
}

const relationOptionsForRule = (rule: QueryRule) => {
  return filteredRelationOptionsMap.value[rule.field] ?? relationOptionsMap.value[rule.field] ?? []
}

const isRelationLoading = (rule: QueryRule) => {
  return !!relationOptionsLoading.value[rule.field]
}

const hasMoreRelationOptions = (rule: QueryRule) => {
  return !!relationOptionsStateMap.value[rule.field]?.hasMore
}

const relationPageSizeForLinkage = (linkage: ReturnType<typeof parseLinkageConfig>) => {
  const configuredSize = Number(linkage?.searchPageSize || linkage?.pageSize || 50)
  if (!Number.isFinite(configuredSize) || configuredSize <= 0) return 50
  return Math.min(Math.max(configuredSize, 20), 200)
}

const buildRelationOptions = (items: Array<{ value: string; label: string }>, field: FieldRecord) =>
  items.map((item) => ({
    value: coerceFieldValue(item.value, field.field_type),
    label: item.label || item.value,
  }))

const mergeRelationOptions = (base: RelationOption[], incoming: RelationOption[]) => {
  const optionMap = new Map<string, RelationOption>()
  ;[...base, ...incoming].forEach((option) => {
    optionMap.set(String(option.value), option)
  })
  return Array.from(optionMap.values())
}

const collectSelectedValues = (fieldCode: string) => {
  const values = new Set<string>()
  props.queryModel.expressions.forEach((expression) => {
    const collectRule = (rule: QueryRule) => {
      if (rule.field !== fieldCode) return
      const ruleValues = Array.isArray(rule.value) ? rule.value : [rule.value]
      ruleValues.forEach((value) => {
        if (value !== null && value !== undefined && value !== '') values.add(String(value))
      })
    }
    expression.rules.forEach(collectRule)
    expression.nested?.forEach((nested) => nested.rules.forEach(collectRule))
  })
  return values
}

const selectedOptionsForField = (fieldCode: string) => {
  const selectedValues = collectSelectedValues(fieldCode)
  if (selectedValues.size === 0) return []
  return (relationOptionsMap.value[fieldCode] ?? []).filter((option) =>
    selectedValues.has(String(option.value)),
  )
}

const selectedRawValuesForField = (field: FieldRecord) => {
  const fieldCode = String(field[props.fieldValueKey])
  const selectedValues = new Map<string, unknown>()
  props.queryModel.expressions.forEach((expression) => {
    const collectRule = (rule: QueryRule) => {
      if (rule.field !== fieldCode) return
      const ruleValues = Array.isArray(rule.value) ? rule.value : [rule.value]
      ruleValues.forEach((value) => {
        if (value === null || value === undefined || value === '') return
        const coercedValue = coerceFieldValue(value, field.field_type)
        selectedValues.set(String(coercedValue), coercedValue)
      })
    }
    expression.rules.forEach(collectRule)
    expression.nested?.forEach((nested) => nested.rules.forEach(collectRule))
  })
  return Array.from(selectedValues.values())
}

const loadSelectedRelationOptionsForField = async (
  field: FieldRecord,
  knownOptions: RelationOption[],
) => {
  const knownValues = new Set(knownOptions.map((option) => String(option.value)))
  const missingValues = selectedRawValuesForField(field).filter(
    (value) => !knownValues.has(String(value)),
  )
  if (missingValues.length === 0) return []

  if (!field.id || currentMenuId.value <= 0) return []
  const result = await queryRuntimeRelationOptions(field.id, {
    menu_id: currentMenuId.value,
    page: 1,
    num: Math.max(missingValues.length, 20),
    selected_values: missingValues.map(String),
  })
  return buildRelationOptions(result.items, field)
}

const loadRelationOptionsForField = async (
  field?: FieldRecord,
  loadOptions: RelationLoadOptions = {},
) => {
  if (!field?.[props.fieldValueKey]) return
  if (organizationSelectorConfigForField(field)) return
  if (field.dict_code) return

  const fieldCode = String(field[props.fieldValueKey])

  const linkage = parseLinkageConfig(field as TableField)
  if (linkage?.mode !== 'relation' && linkage?.mode !== 'cascader') return

  const tableCode = linkage.tableCode
  if (!tableCode) return

  const prevState = relationOptionsStateMap.value[fieldCode]
  const keyword = loadOptions.keyword ?? prevState?.keyword ?? ''
  const page = loadOptions.page ?? 1
  const append = !!loadOptions.append
  const force = !!loadOptions.force
  if (
    !force &&
    page === 1 &&
    relationOptionsMap.value[fieldCode] &&
    prevState?.keyword === keyword
  ) {
    return
  }
  if (relationOptionsLoading.value[fieldCode]) return

  relationOptionsLoading.value = { ...relationOptionsLoading.value, [fieldCode]: true }
  try {
    const pageSize = relationPageSizeForLinkage(linkage)
    if (!field.id || currentMenuId.value <= 0) return
    const result = await queryRuntimeRelationOptions(field.id, {
      menu_id: currentMenuId.value,
      page,
      num: pageSize,
      keyword,
      selected_values: append ? [] : selectedRawValuesForField(field).map(String),
    })
    const incomingOptions = buildRelationOptions(result.items, field)
    const cachedSelectedOptions = append
      ? (relationOptionsMap.value[fieldCode] ?? [])
      : selectedOptionsForField(fieldCode)
    const hydratedSelectedOptions = append
      ? []
      : await loadSelectedRelationOptionsForField(
          field,
          mergeRelationOptions(cachedSelectedOptions, incomingOptions),
        )
    const baseOptions = mergeRelationOptions(cachedSelectedOptions, hydratedSelectedOptions)
    const options = mergeRelationOptions(baseOptions, incomingOptions)
    relationOptionsMap.value = { ...relationOptionsMap.value, [fieldCode]: options }
    filteredRelationOptionsMap.value = { ...filteredRelationOptionsMap.value, [fieldCode]: options }
    relationOptionsStateMap.value = {
      ...relationOptionsStateMap.value,
      [fieldCode]: {
        page,
        pageSize,
        keyword,
        hasMore: page * pageSize < result.total,
      },
    }
  } catch (error) {
    console.warn(`加载关联字段 ${fieldCode} 查询选项失败`, error)
    if (!append) {
      relationOptionsMap.value = { ...relationOptionsMap.value, [fieldCode]: [] }
      filteredRelationOptionsMap.value = { ...filteredRelationOptionsMap.value, [fieldCode]: [] }
      relationOptionsStateMap.value = {
        ...relationOptionsStateMap.value,
        [fieldCode]: {
          page: 1,
          pageSize: relationPageSizeForLinkage(linkage),
          keyword,
          hasMore: false,
        },
      }
    }
  } finally {
    relationOptionsLoading.value = { ...relationOptionsLoading.value, [fieldCode]: false }
  }
}

const filterRelationOptions = (
  rule: QueryRule,
  val: string,
  update: QSelectFilterUpdate,
  abort?: QSelectFilterAbort,
) => {
  const field = findField(rule.field)
  if (!field) {
    abort?.()
    return
  }

  void loadRelationOptionsForField(field, {
    keyword: val.trim(),
    page: 1,
    force: true,
  }).then(() => {
    update(() => undefined)
  })
}

const preloadRelationOptions = (rule: QueryRule) => {
  void loadRelationOptionsForField(findField(rule.field))
}

const loadMoreRelationOptions = (rule: QueryRule, details: VirtualScrollDetails) => {
  const field = findField(rule.field)
  if (!field) return

  const fieldCode = rule.field
  const state = relationOptionsStateMap.value[fieldCode]
  if (!state?.hasMore || relationOptionsLoading.value[fieldCode]) return

  const options = relationOptionsForRule(rule)
  if ((details.to ?? 0) < options.length - 2) return

  void loadRelationOptionsForField(field, {
    keyword: state.keyword,
    page: state.page + 1,
    append: true,
    force: true,
  })
}

const hasValue = (value: unknown) => {
  if (Array.isArray(value)) return value.length > 0
  return value !== null && value !== undefined && value !== ''
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

const updateOrganizationSelectorValue = (rule: QueryRule, value: unknown) => {
  if (!organizationSelectorConfigForRule(rule)) return
  if (!isMultiValueRule(rule)) {
    rule.value = normalizeOrganizationSelectorId(value)
    return
  }

  const values = Array.isArray(value) ? value : []
  rule.value = values
    .map(normalizeOrganizationSelectorId)
    .filter((item): item is number => item !== null)
    .filter((item, index, items) => items.indexOf(item) === index)
}

const hasMultiValue = (value: unknown) => {
  if (Array.isArray(value)) return value.some(hasValue)
  if (typeof value === 'string') return splitMultiValueText(value).length > 0
  return hasValue(value)
}

const hasRangeValue = (value: unknown) => {
  return Array.isArray(value) && hasValue(value[0]) && hasValue(value[1])
}

const valueRules = (rule: QueryRule) => {
  if (isNullOperator(rule)) return []
  if (isRangeRule(rule)) {
    return [(val: unknown) => hasRangeValue(val) || t('ui.pleaseFillInTheFullSpace')]
  }
  if (isMultiValueRule(rule) || isFreeInputMultiValueRule(rule)) {
    return [(val: unknown) => hasMultiValue(val) || t('ui.pleaseFillInAtLeastOneValue')]
  }
  return [(val: unknown) => hasValue(val) || t('ui.pleaseFillInTheValues')]
}

const valuePlaceholderForRule = (rule: QueryRule) => {
  if (!isFreeInputMultiValueRule(rule)) return ''
  if (isTextMultiKeywordExpressionType(rule.expression_type)) {
    return t('ui.multipleKeywordsSeparatedByCommasSemicolonsOrLineBreaks')
  }
  return t('ui.multipleValuesSeparatedByCommasSemiColonsOrLines')
}

const rangePlaceholderForRule = (rule: QueryRule, index: 0 | 1) => {
  const field = findField(rule.field)
  const label = index === 0 ? t('ui.startValue') : t('ui.endValue')
  switch (field?.field_type) {
    case SysTableFieldType.DATE:
      return index === 0 ? t('ui.startDate') : t('ui.endDate')
    case SysTableFieldType.DATETIME:
      return index === 0 ? t('ui.startTime') : t('ui.endTime')
    case SysTableFieldType.TIME:
      return index === 0 ? t('ui.startMoment') : t('ui.endMoment')
    default:
      return label
  }
}

const inputTypeForRule = (rule: QueryRule) => {
  if (isFreeInputMultiValueRule(rule)) return 'textarea'
  const field = findField(rule.field)
  return queryValueHtmlInputType(field)
}

const hasIncompleteExpressionRules = (expressions: Query['expressions']): boolean => {
  return expressions.some((expression) => {
    if (expression.rules.some(isIncompleteQueryRule)) return true
    return expression.nested ? hasIncompleteExpressionRules(expression.nested) : false
  })
}

const emptyExpressionGroup = () => ({
  logic: ExpressionLogic.AND,
  rules: [
    {
      field: '',
      value: null,
    },
  ],
  nested: [],
})

const submitQueryModel = () => {
  const expressions = normalizeQuerySchemePayload(props.queryModel, props.bindings).expressions
  const nextQuery = {
    ...props.queryModel,
    expressions: expressions.length > 0 ? expressions : [emptyExpressionGroup()],
  }
  emit('update:queryModel', nextQuery)
}

watch(
  () => props.fields,
  (fields) => {
    if (readOnlyDepth.value) return
    const dictCodes = Array.from(
      new Set(
        (fields as FieldRecord[])
          .filter((field) => !organizationSelectorConfigForField(field))
          .map((field) => field.dict_code)
          .filter((dictCode): dictCode is string => !!dictCode),
      ),
    )
    void dictStore.loadDicts(dictCodes)
    normalizeQueryExpressionTypes()
  },
  { immediate: true },
)

watch(
  () => props.modelValue,
  (visible) => {
    if (!visible) return
    if (readOnlyDepth.value) return
    normalizeQueryExpressionTypes()
    props.queryModel.expressions.forEach((expression) => {
      expression.rules.forEach((rule) => {
        void loadRelationOptionsForField(findField(rule.field))
      })
      expression.nested?.forEach((nested) => {
        nested.rules.forEach((rule) => {
          void loadRelationOptionsForField(findField(rule.field))
        })
      })
    })
  },
)

// 移除表达式组
const removeExpression = (eIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions.splice(eIndex, 1)
  emit('update:queryModel', newQuery)
}

// 添加表达式组
const addExpression = (eIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions.splice(eIndex + 1, 0, {
    logic: ExpressionLogic.AND,
    rules: [
      {
        field: '',
        value: null,
      },
    ],
    nested: [],
  })
  emit('update:queryModel', newQuery)
}

// 移除规则
const removeRule = (eIndex: number, rIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions[eIndex]!.rules.splice(rIndex, 1)
  emit('update:queryModel', newQuery)
}

// 移除嵌套组
const removeNest = (eIndex: number, nIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions[eIndex]!.nested!.splice(nIndex, 1)
  emit('update:queryModel', newQuery)
}

// 添加规则
const addRule = (eIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions[eIndex]!.rules.push({
    field: '',
    value: null,
  })
  emit('update:queryModel', newQuery)
}

// 添加嵌套组
const addNestedGroup = (eIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  if (!newQuery.expressions[eIndex]!.nested) {
    newQuery.expressions[eIndex]!.nested = []
  }
  newQuery.expressions[eIndex]!.nested!.push({
    logic: ExpressionLogic.AND,
    rules: [
      {
        field: '',
        value: null,
      },
    ],
  })
  emit('update:queryModel', newQuery)
}

// 添加嵌套规则
const addNestRule = (eIndex: number, nIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions[eIndex]!.nested![nIndex]!.rules.push({
    field: '',
    value: null,
  })
  emit('update:queryModel', newQuery)
}

// 移除嵌套规则
const removeNestRule = (eIndex: number, nIndex: number, rIndex: number) => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  if (newQuery.expressions[eIndex]?.nested?.[nIndex]) {
    if (newQuery.expressions[eIndex]?.nested?.[nIndex].rules.length === 1) {
      removeNest(eIndex, nIndex)
    } else {
      newQuery.expressions[eIndex]?.nested?.[nIndex].rules.splice(rIndex, 1)
    }
  }
  emit('update:queryModel', newQuery)
}

// 重置筛选条件
const resetFilter = () => {
  clearAllBindings()
  const newQuery = { ...props.queryModel }
  newQuery.expressions = [emptyExpressionGroup()]
  emit('update:queryModel', newQuery)
  if (form.value) {
    form.value.resetValidation()
  }
}

const completeQueryAction = () => {
  normalizeQueryExpressionTypes()
  if (hasIncompleteExpressionRules(props.queryModel.expressions)) {
    void form.value?.validate()
    $q.notify({
      color: 'negative',
      get message() {
        return isSchemeConditionEditor.value
          ? t('ui.pleaseCompleteTheSearchConditions')
          : t('ui.completeSearchConditions')
      },
      position: 'top-right',
      timeout: 6000,
    })
    return
  }
  form.value?.resetValidation()
  submitQueryModel()
  emit(isSchemeConditionEditor.value ? 'confirm' : 'search')
}

const cancelConditionEdit = () => {
  emit('cancel')
  dialogVisible.value = false
}

// 暴露方法
defineExpose({
  resetFilter,
})
</script>

<style scoped lang="scss">
.advanced-search-dialog {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 1120px;
  max-height: 90vh;
  border-radius: 8px;
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.18);
}

.advanced-search-header {
  padding: 12px 16px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  background: linear-gradient(180deg, rgba(25, 118, 210, 0.05), rgba(255, 255, 255, 0));
}

.advanced-search-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: $primary;
  font-size: 17px;
  font-weight: 600;
}

.advanced-search-content {
  overflow-y: auto;
  flex: 1;
  padding: 12px 14px;
  background: #f8fafc;
}

.advanced-search-footer {
  padding: 10px 14px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
}

.expression-card {
  border-radius: 8px;
  border: 1px solid rgba(25, 118, 210, 0.16);
  box-shadow: none;
  background: #ffffff;
  overflow: hidden;
}

.expression-group + .expression-group {
  margin-top: 8px;
}

.expression-card-head,
.nested-section-head {
  padding: 8px 12px;
  background: rgba(115, 103, 240, 0.04);
}

.expression-card-body {
  padding: 8px 10px;
}

.nested-card {
  border-radius: 8px;
  border: 1px dashed rgba(115, 103, 240, 0.22);
  background: rgba(115, 103, 240, 0.025);
  overflow: hidden;
}

.nested-section {
  margin-top: 8px;
  border-top: 1px dashed rgba(15, 23, 42, 0.12);
  padding-top: 8px;
}

.nested-groups {
  position: relative;
}

.nested-group {
  background-color: #f8fbff;
  border-radius: 8px;
  border: 1px solid rgba(25, 118, 210, 0.24);
}

.advanced-query-preview {
  padding: 2px 4px 4px;
}

.advanced-query-preview__title {
  margin-bottom: 6px;
  color: var(--app-text-strong);
  font-size: 14px;
  font-weight: 600;
}

@media (max-width: 1023px) {
  .advanced-search-dialog {
    max-height: 100vh;
    border-radius: 0;
  }

  .advanced-search-content {
    padding: 12px;
  }
}
</style>
