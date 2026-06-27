<template>
  <div class="rule-item">
    <div class="row q-col-gutter-xs items-center">
      <div class="col-12 col-md-2" v-if="isFirst">
        <q-select
          :model-value="logic"
          outlined
          dense
          options-dense
          emit-value
          map-options
          option-label="label"
          option-value="value"
          :options="expressionLogicOptions"
          label="逻辑关系"
          class="logic-select"
          :rules="[(val: any) => hasValue(val) || '请选择逻辑关系']"
          hide-bottom-space
          @update:model-value="(value) => emit('update:logic', value)"
        />
      </div>
      <div class="col-12 col-md-2" v-else></div>

      <div class="col-12 col-md-4">
        <q-select
          v-model="rule.field"
          outlined
          dense
          options-dense
          emit-value
          map-options
          :options="filteredFields"
          :option-label="fieldLabelKey"
          :option-value="fieldValueKey"
          label="字段"
          class="field-select"
          use-input
          input-debounce="120"
          :rules="[(val: any) => hasValue(val) || '请选择字段']"
          hide-bottom-space
          clearable
          @filter="filterFields"
          @popup-show="resetFieldOptions"
          @update:model-value="(value) => emit('update-field', value)"
        >
          <template #option="scope">
            <q-item v-bind="scope.itemProps">
              <q-item-section>
                <q-item-label>{{ fieldOptionLabel(scope.opt) }}</q-item-label>
                <q-item-label caption>{{ fieldOptionCaption(scope.opt) }}</q-item-label>
              </q-item-section>
            </q-item>
          </template>
          <template #no-option>
            <q-item>
              <q-item-section class="text-grey-6">暂无匹配字段</q-item-section>
            </q-item>
          </template>
        </q-select>
      </div>

      <div class="col-12 col-md-2">
        <q-select
          v-model="rule.expression_type"
          outlined
          dense
          options-dense
          emit-value
          map-options
          :options="expressionTypeOptionsForRule(rule)"
          label="操作符"
          class="field-select"
          :rules="[(val: any) => hasValue(val) || '请选择操作符']"
          hide-bottom-space
          @update:model-value="() => emit('update-expression-type')"
        />
      </div>

      <div class="col-12 col-md-3">
        <q-input
          v-if="isNullOperator(rule)"
          dense
          outlined
          disable
          model-value="无需填写"
          label="值"
          class="field-input"
          hide-bottom-space
        />
        <q-select
          v-else-if="hasDictRule(rule)"
          dense
          outlined
          options-dense
          v-model="rule.value"
          :options="dictOptionsForRule(rule)"
          :multiple="isMultiValueRule(rule)"
          :display-value="
            isMultiValueRule(rule) ? multiValueDisplay(rule.value, dictOptionsForRule(rule)) : undefined
          "
          emit-value
          map-options
          clearable
          label="值"
          class="field-input"
          :rules="valueRules(rule)"
          hide-bottom-space
        >
          <q-tooltip v-if="isMultiValueRule(rule) && multiValueTooltip(rule.value, dictOptionsForRule(rule))">
            {{ multiValueTooltip(rule.value, dictOptionsForRule(rule)) }}
          </q-tooltip>
        </q-select>
        <q-select
          v-else-if="hasRelationRule(rule)"
          dense
          outlined
          options-dense
          v-model="rule.value"
          :options="relationOptionsForRule(rule)"
          :multiple="isMultiValueRule(rule)"
          :display-value="
            isMultiValueRule(rule)
              ? multiValueDisplay(rule.value, relationOptionsForRule(rule))
              : undefined
          "
          :loading="isRelationLoading(rule)"
          use-input
          input-debounce="150"
          emit-value
          map-options
          clearable
          label="值"
          class="field-input"
          :rules="valueRules(rule)"
          hide-bottom-space
          @filter="(val, update, abort) => filterRelationOptions(rule, val, update, abort)"
          @popup-show="() => preloadRelationOptions(rule)"
          @virtual-scroll="(details) => loadMoreRelationOptions(rule, details)"
        >
          <q-tooltip
            v-if="isMultiValueRule(rule) && multiValueTooltip(rule.value, relationOptionsForRule(rule))"
          >
            {{ multiValueTooltip(rule.value, relationOptionsForRule(rule)) }}
          </q-tooltip>
          <template #no-option>
            <q-item>
              <q-item-section class="text-grey-6">
                {{ isRelationLoading(rule) ? '正在加载选项...' : '暂无选项，可输入关键字搜索' }}
              </q-item-section>
            </q-item>
          </template>
          <template #after-options>
            <q-item v-if="hasMoreRelationOptions(rule)" dense>
              <q-item-section class="text-grey-6 text-caption"> 向下滚动加载更多 </q-item-section>
            </q-item>
          </template>
        </q-select>
        <div v-else-if="isRangeRule(rule)" class="range-inputs">
          <q-input
            dense
            outlined
            :model-value="rangeValue(rule, 0)"
            :type="inputTypeForRule(rule)"
            :label="rangePlaceholderForRule(rule, 0)"
            class="field-input range-input"
            :rules="[rangeBoundaryRule]"
            hide-bottom-space
            clearable
            @update:model-value="(value) => updateRangeValue(rule, 0, value)"
          />
          <q-input
            dense
            outlined
            :model-value="rangeValue(rule, 1)"
            :type="inputTypeForRule(rule)"
            :label="rangePlaceholderForRule(rule, 1)"
            class="field-input range-input"
            :rules="[rangeBoundaryRule]"
            hide-bottom-space
            clearable
            @update:model-value="(value) => updateRangeValue(rule, 1, value)"
          />
        </div>
        <q-select
          v-else-if="isBooleanRule(rule)"
          dense
          outlined
          options-dense
          v-model="rule.value"
          :options="booleanOptions"
          :multiple="isMultiValueRule(rule)"
          :display-value="
            isMultiValueRule(rule) ? multiValueDisplay(rule.value, booleanOptions) : undefined
          "
          emit-value
          map-options
          clearable
          label="值"
          class="field-input"
          :rules="valueRules(rule)"
          hide-bottom-space
        >
          <q-tooltip v-if="isMultiValueRule(rule) && multiValueTooltip(rule.value, booleanOptions)">
            {{ multiValueTooltip(rule.value, booleanOptions) }}
          </q-tooltip>
        </q-select>
        <q-input
          v-else
          dense
          outlined
          v-model="rule.value"
          :type="isFreeInputMultiValueRule(rule) ? 'text' : inputTypeForRule(rule)"
          label="值"
          class="field-input"
          :placeholder="valuePlaceholderForRule(rule)"
          :rules="valueRules(rule)"
          hide-bottom-space
        />
      </div>

      <div class="col-12 col-md-1 rule-actions">
        <q-btn
          v-if="canRemove"
          flat
          round
          size="sm"
          icon="horizontal_rule"
          color="primary"
          @click="emit('remove')"
        />
        <q-btn flat round size="sm" icon="add" color="primary" @click="emit('add')" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, toRefs, watch } from 'vue'
import type { QueryRule } from 'src/types/global'
import { SysTableFieldInputTypeMap, SysTableFieldTypeMap } from 'src/types/enum'
import { compactSelectionDisplay, compactSelectionTooltip } from 'src/utils/select-display'

defineOptions({ name: 'AdvancedQueryRuleRow' })

type SelectOption = {
  label: string
  value: unknown
  [key: string]: unknown
}

type QSelectFilterUpdate = (callback: () => void) => void
type QSelectFilterAbort = () => void
type VirtualScrollDetails = {
  to?: number
}
type FieldOption = Record<string, any>
type HtmlInputType =
  | 'number'
  | 'text'
  | 'file'
  | 'textarea'
  | 'time'
  | 'search'
  | 'password'
  | 'date'
  | 'email'
  | 'tel'
  | 'url'
  | 'datetime-local'

const props = defineProps<{
  rule: QueryRule
  logic: number | undefined
  isFirst: boolean
  canRemove: boolean
  fields: unknown[]
  fieldLabelKey: string
  fieldValueKey: string
  expressionLogicOptions: SelectOption[]
  expressionTypeOptionsForRule: (rule: QueryRule) => SelectOption[]
  booleanOptions: SelectOption[]
  isNullOperator: (rule: QueryRule) => boolean
  hasDictRule: (rule: QueryRule) => boolean
  hasRelationRule: (rule: QueryRule) => boolean
  isBooleanRule: (rule: QueryRule) => boolean
  isMultiValueRule: (rule: QueryRule) => boolean
  isFreeInputMultiValueRule: (rule: QueryRule) => boolean
  isRangeRule: (rule: QueryRule) => boolean
  dictOptionsForRule: (rule: QueryRule) => SelectOption[]
  relationOptionsForRule: (rule: QueryRule) => SelectOption[]
  isRelationLoading: (rule: QueryRule) => boolean
  hasMoreRelationOptions: (rule: QueryRule) => boolean
  valueRules: (rule: QueryRule) => Array<(val: unknown) => boolean | string>
  inputTypeForRule: (rule: QueryRule) => HtmlInputType
  valuePlaceholderForRule: (rule: QueryRule) => string
  rangePlaceholderForRule: (rule: QueryRule, index: 0 | 1) => string
  filterRelationOptions: (
    rule: QueryRule,
    val: string,
    update: QSelectFilterUpdate,
    abort?: QSelectFilterAbort,
  ) => void
  preloadRelationOptions: (rule: QueryRule) => void
  loadMoreRelationOptions: (rule: QueryRule, details: VirtualScrollDetails) => void
}>()

const emit = defineEmits<{
  (event: 'update:logic', value: number): void
  (event: 'update-field', value: unknown): void
  (event: 'update-expression-type'): void
  (event: 'remove'): void
  (event: 'add'): void
}>()

const hasValue = (value: unknown) => {
  return value !== null && value !== undefined && value !== ''
}

const multiValueTooltip = (value: unknown, options: SelectOption[]) => {
  return compactSelectionTooltip(value, options)
}

const multiValueDisplay = (value: unknown, options: SelectOption[]) => {
  return compactSelectionDisplay(value, options, 2)
}

const filteredFields = ref<unknown[]>([])

const fieldOptionLabel = (field: unknown) => {
  const item = field as FieldOption
  return String(item?.[props.fieldLabelKey] ?? item?.label ?? item?.field_name ?? '')
}

const fieldOptionCode = (field: unknown) => {
  const item = field as FieldOption
  return String(item?.[props.fieldValueKey] ?? item?.value ?? item?.field_code ?? '')
}

const fieldOptionCaption = (field: unknown) => {
  const item = field as FieldOption
  const parts = [fieldOptionCode(item)]
  const fieldTypeLabel =
    SysTableFieldTypeMap[item?.field_type as keyof typeof SysTableFieldTypeMap] ||
    (item?.field_type ? `类型 ${item.field_type}` : '')
  const inputTypeLabel =
    SysTableFieldInputTypeMap[item?.input_type as keyof typeof SysTableFieldInputTypeMap] ||
    (item?.input_type ? `控件 ${item.input_type}` : '')
  if (fieldTypeLabel) parts.push(fieldTypeLabel)
  if (inputTypeLabel) parts.push(inputTypeLabel)
  if (item?.dict_code) parts.push(`字典 ${item.dict_code}`)
  if (item?.linkage_config) parts.push('关联')
  return parts.filter(Boolean).join(' / ')
}

const resetFieldOptions = () => {
  filteredFields.value = [...props.fields]
}

const filterFields = (value: string, update: QSelectFilterUpdate) => {
  const keyword = String(value || '')
    .trim()
    .toLowerCase()
  update(() => {
    filteredFields.value = keyword
      ? props.fields.filter((field) => {
          const text = [
            fieldOptionLabel(field),
            fieldOptionCode(field),
            fieldOptionCaption(field),
          ]
            .join(' ')
            .toLowerCase()
          return text.includes(keyword)
        })
      : [...props.fields]
  })
}

const rangeValue = (rule: QueryRule, index: 0 | 1) => {
  if (!Array.isArray(rule.value)) return null
  return rule.value[index] ?? null
}

const updateRangeValue = (rule: QueryRule, index: 0 | 1, value: unknown) => {
  const next = Array.isArray(rule.value) ? [...rule.value] : [null, null]
  next[index] = value
  rule.value = [next[0] ?? null, next[1] ?? null]
}

const rangeBoundaryRule = (value: unknown) => {
  return hasValue(value) || '请填写'
}

const {
  rule,
  logic,
  isFirst,
  canRemove,
  fieldLabelKey,
  fieldValueKey,
  expressionLogicOptions,
  expressionTypeOptionsForRule,
  booleanOptions,
  isNullOperator,
  hasDictRule,
  hasRelationRule,
  isBooleanRule,
  isMultiValueRule,
  isFreeInputMultiValueRule,
  isRangeRule,
  dictOptionsForRule,
  relationOptionsForRule,
  isRelationLoading,
  hasMoreRelationOptions,
  valueRules,
  inputTypeForRule,
  valuePlaceholderForRule,
  rangePlaceholderForRule,
  filterRelationOptions,
  preloadRelationOptions,
  loadMoreRelationOptions,
} = toRefs(props)

watch(
  () => props.fields,
  () => resetFieldOptions(),
  { immediate: true },
)
</script>

<style scoped lang="scss">
.rule-item {
  position: relative;
  padding: 10px 12px;
  border-radius: 8px;
  background: #ffffff;
}

.rule-item + .rule-item {
  margin-top: 8px;
}

.rule-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 4px;
}

.range-inputs {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
}

.range-input {
  min-width: 0;
}

.logic-select :deep(.q-field__native) {
  font-weight: 500;
  color: $primary;
}

.field-select :deep(.q-field__native),
.field-input :deep(.q-field__native) {
  font-weight: 500;
  color: $primary;
}

@media (max-width: 1023px) {
  .rule-actions {
    justify-content: flex-start;
  }

  .range-inputs {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
