<template>
  <linkage-config-editor
    v-if="context.role === 'linkage-config'"
    v-model="value"
    :disable="context.readonly"
    @update:model-value="notifyInput"
  />

  <organization-select
    v-else-if="context.organization"
    :model-value="value"
    :selector-type="context.organization.selectorType"
    :multiple="context.organization.multiple"
    :include-history="context.organization.includeHistory"
    :disabled="context.readonly || context.organization.disabled"
    :label="field.field_name"
    :clearable="!!field.is_null"
    :rules="context.rules"
    :hint="context.hint"
    :ref="context.fieldRef"
    @update:model-value="emit('organization-change', $event)"
  />

  <json-editor
    v-else-if="context.controlType === 'json-editor'"
    v-model="value"
    :label="field.field_name"
    :rules="context.rules"
    :disabled="context.readonly"
    :ref="context.fieldRef"
    @update:model-value="notifyInput"
  />

  <array-input
    v-else-if="context.controlType === 'array-input'"
    v-model="value"
    :label="field.field_name"
    :rules="context.rules"
    :disabled="context.readonly"
    :ref="context.fieldRef"
    @update:model-value="notifyInput"
  />

  <key-value-editor
    v-else-if="context.controlType === 'key-value-editor'"
    v-model="value"
    :label="field.field_name"
    :rules="context.rules"
    :disabled="context.readonly"
    :ref="context.fieldRef"
    @update:model-value="notifyInput"
  />

  <cascader-select
    v-else-if="context.controlType === 'cascader'"
    v-model="value"
    :label="field.field_name"
    :options="context.options"
    :rules="context.rules"
    :disable="context.readonly"
    :ref="context.fieldRef"
    value-mode="value"
    :clearable="!!field.is_null"
    :selectable="context.cascaderSelectable"
    :show-path="context.cascaderShowPath"
    @change="notifyInput"
  />

  <q-select
    v-else-if="context.role === 'dict-code'"
    v-model="value"
    :label="field.field_name"
    outlined
    dense
    clearable
    clear-icon="close"
    emit-value
    map-options
    use-input
    input-debounce="0"
    :options="context.dictCodeOptions"
    :rules="context.rules"
    :hint="context.hint"
    :disable="context.readonly"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @filter="(text, update) => emit('dict-filter', text, update)"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @update:model-value="notifyInput"
  />

  <q-input
    v-else-if="context.controlType === 'input'"
    v-model="value"
    :label="field.field_name"
    :placeholder="context.placeholder"
    outlined
    dense
    :disable="context.readonly || field.field_code === 'id'"
    :maxlength="context.maxLength"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    :type="passwordHidden ? 'password' : 'text'"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @update:model-value="notifyInput"
  >
    <template v-if="context.hideContent" #append>
      <q-icon
        :name="passwordHidden ? 'visibility_off' : 'visibility'"
        class="cursor-pointer"
        @click="passwordHidden = !passwordHidden"
      />
    </template>
  </q-input>

  <q-input
    v-else-if="context.controlType === 'number'"
    :model-value="value"
    class="number-input-field"
    :label="field.field_name"
    :placeholder="context.placeholder"
    outlined
    dense
    type="text"
    :inputmode="context.numberInputMode"
    :disable="context.readonly"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @keydown.up="emit('number-keydown', 1, $event)"
    @keydown.down="emit('number-keydown', -1, $event)"
    @update:model-value="emit('number-change', $event)"
  >
    <template v-if="!context.exactDecimal" #append>
      <div class="number-input-field__actions">
        <q-btn
          flat
          dense
          unelevated
          icon="remove"
          aria-label="减少"
          class="number-input-field__step"
          :tabindex="-1"
          :disable="context.readonly"
          @mousedown.prevent
          @click.stop="emit('number-step', -1)"
        />
        <q-btn
          flat
          dense
          unelevated
          icon="add"
          aria-label="增加"
          class="number-input-field__step"
          :tabindex="-1"
          :disable="context.readonly"
          @mousedown.prevent
          @click.stop="emit('number-step', 1)"
        />
      </div>
    </template>
  </q-input>

  <q-input
    v-else-if="context.controlType === 'textarea'"
    v-model="value"
    :label="field.field_name"
    :placeholder="context.placeholder"
    outlined
    type="textarea"
    autogrow
    :disable="context.readonly"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @update:model-value="notifyInput"
  />

  <q-select
    v-else-if="context.controlType === 'select'"
    v-model="value"
    :label="field.field_name"
    :placeholder="context.placeholder"
    outlined
    dense
    :clearable="!!field.is_null"
    clear-icon="close"
    emit-value
    map-options
    :disable="context.readonly"
    :options="context.options"
    :use-input="context.relationSelect"
    input-debounce="250"
    :loading="context.relationLoading"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @filter="(text, update, abort) => emit('relation-filter', text, update, abort)"
    @popup-show="emit('relation-preload')"
    @virtual-scroll="emit('relation-load-more', $event)"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @update:model-value="notifyInput"
  >
    <template #no-option>
      <q-item>
        <q-item-section class="text-grey-6">
          {{
            context.relationSelect
              ? context.relationLoading
                ? '正在加载选项...'
                : '暂无选项，可输入关键字搜索'
              : '暂无选项'
          }}
        </q-item-section>
      </q-item>
    </template>
    <template #after-options>
      <q-item v-if="context.relationHasMore" dense>
        <q-item-section class="text-grey-6 text-caption">向下滚动加载更多</q-item-section>
      </q-item>
    </template>
  </q-select>

  <div v-else-if="context.controlType === 'boolean'" class="boolean-toggle-field">
    <div class="boolean-toggle-field__label">{{ field.field_name }}</div>
    <q-btn-toggle
      v-model="value"
      :options="context.booleanOptions"
      dense
      unelevated
      no-caps
      toggle-color="primary"
      color="grey-2"
      text-color="grey-8"
      :disable="context.readonly || field.field_code === 'id'"
      @update:model-value="notifyInput"
    />
  </div>

  <sweet-date-time-picker
    v-else-if="dateTimeControl"
    v-model="value"
    :label="field.field_name"
    :type="dateTimeControl"
    :disable="context.readonly"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @change="notifyInput"
  />

  <file-upload
    v-else-if="context.controlType === 'file'"
    v-model="value"
    :label="field.field_name"
    :rules="context.rules"
    :table-code="context.tableCode"
    :menu-id="context.menuId"
    :row-id="context.rowId"
    :field-code="field.field_code"
    :readonly="context.readonly"
    :multiple="context.file.multiple"
    :accept="context.file.accept"
    :max-size="context.file.maxSize"
    :chunk-threshold="context.file.chunkThreshold"
    :concurrency="context.file.concurrency"
  />

  <rich-text-editor
    v-else-if="context.controlType === 'rich-text'"
    v-model="value"
    :label="field.field_name"
    :rules="context.rules"
    :disabled="context.readonly || field.field_code === 'id'"
    :table-code="context.tableCode"
    :menu-id="context.menuId"
    :row-id="context.rowId"
    :field-code="field.field_code"
    :ref="context.fieldRef"
  />

  <q-input
    v-else
    v-model="value"
    :label="field.field_name"
    :placeholder="context.placeholder"
    outlined
    dense
    :disable="context.readonly"
    :rules="context.rules"
    :hint="context.hint"
    :lazy-rules="context.lazyRules"
    :ref="context.fieldRef"
    @focus="notifyTouched"
    @blur="notifyTouched"
    @update:model-value="notifyInput"
  />
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, ref, watch } from 'vue'
import type { TableField } from 'src/api/services/sys-table'
import CascaderSelect from 'src/components/Cascader/CascaderSelect.vue'
import SweetDateTimePicker from 'src/components/DateTime/SweetDateTimePicker.vue'
import FileUpload from 'src/components/FileUpload/FileUpload.vue'
import LinkageConfigEditor from 'src/components/FormDialog/LinkageConfigEditor.vue'
import ArrayInput from 'src/components/JsonEditor/ArrayInput.vue'
import JsonEditor from 'src/components/JsonEditor/JsonEditor.vue'
import KeyValueEditor from 'src/components/JsonEditor/KeyValueEditor.vue'
import OrganizationSelect from 'src/components/Select/OrganizationSelect.vue'
import type { resolveOrganizationSelectorConfig } from 'src/utils/field-metadata'

type OrganizationSelectorConfig = NonNullable<ReturnType<typeof resolveOrganizationSelectorConfig>>

// DynamicFieldControlContext 由DynamicFormDialog集中解析，单字段控件不重复判断Metadata合同。
export interface DynamicFieldControlContext {
  role: 'standard' | 'dict-code' | 'linkage-config'
  controlType: string
  readonly: boolean
  tableCode: string
  menuId: number
  rowId: number
  lazyRules: boolean | 'ondemand'
  organization: OrganizationSelectorConfig | null
  rules: Array<(value: any) => boolean | string>
  hint: string
  placeholder: string
  options: any[]
  dictCodeOptions: Array<{ label: string; value: string }>
  cascaderSelectable: 'leaf' | 'any' | 'level'
  cascaderShowPath: boolean
  maxLength: number | undefined
  numberInputMode: 'numeric' | 'decimal'
  exactDecimal: boolean
  hideContent: boolean
  visibilityResetToken: number
  relationSelect: boolean
  relationLoading: boolean
  relationHasMore: boolean
  booleanOptions: Array<{ label: string; value: boolean }>
  file: {
    multiple: boolean
    accept: string
    maxSize: number
    chunkThreshold: number
    concurrency: number
  }
  fieldRef: (element: any) => void
}

const props = defineProps<{
  field: TableField
  modelValue: any
  context: DynamicFieldControlContext
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
  'field-input': []
  touched: []
  'organization-change': [value: unknown]
  'number-change': [value: string | number | null]
  'number-step': [direction: 1 | -1]
  'number-keydown': [direction: 1 | -1, event: KeyboardEvent]
  'dict-filter': [text: string, update: (callback: () => void) => void]
  'relation-filter': [text: string, update: (callback: () => void) => void, abort: () => void]
  'relation-preload': []
  'relation-load-more': [details: { to?: number }]
}>()

const RichTextEditor = defineAsyncComponent(
  () => import('src/components/RichTextEditor/RichTextEditor.vue'),
)

const value = computed({
  get: () => props.modelValue,
  set: (next) => emit('update:modelValue', next),
})
const passwordHidden = ref(props.context.hideContent)
watch(
  () => props.context.visibilityResetToken,
  () => {
    passwordHidden.value = props.context.hideContent
  },
)
const dateTimeControl = computed(() => {
  const type = props.context.controlType
  return ['date', 'datetime', 'time', 'year', 'year-month'].includes(type)
    ? (type as 'date' | 'datetime' | 'time' | 'year' | 'year-month')
    : null
})

const notifyInput = () => emit('field-input')
const notifyTouched = () => emit('touched')
</script>

<style scoped lang="scss">
.boolean-toggle-field {
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 4px 7px 4px 12px;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  background: transparent;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.boolean-toggle-field:hover {
  border-color: var(--app-primary-border);
}
.boolean-toggle-field:focus-within {
  border-color: $primary;
  box-shadow: inset 0 0 0 1px $primary;
}
.boolean-toggle-field__label {
  min-width: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--app-text-strong);
  white-space: nowrap;
  word-break: keep-all;
}
.boolean-toggle-field :deep(.q-btn-toggle) {
  flex: 0 0 auto;
  overflow: hidden;
  border-radius: 4px;
  background: transparent;
}
.boolean-toggle-field :deep(.q-btn) {
  min-width: 34px;
  min-height: 30px;
  padding: 4px 10px;
  font-size: 13px;
  line-height: 1.2;
  border-radius: 4px;
}
.boolean-toggle-field :deep(.q-btn.bg-grey-2) {
  color: var(--app-text-muted) !important;
  background: transparent !important;
}
.boolean-toggle-field :deep(.q-btn.bg-primary) {
  color: #fff !important;
  background: $primary !important;
}
.boolean-toggle-field :deep(.q-btn::before) {
  box-shadow: none;
}

.number-input-field :deep(.q-field__native) {
  font-variant-numeric: tabular-nums;
}
.number-input-field :deep(input[type='text']) {
  appearance: textfield;
}
.number-input-field :deep(.q-field__append) {
  padding-left: 4px;
}
.number-input-field__actions {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-right: -3px;
}
.number-input-field__step {
  width: 24px;
  min-width: 24px;
  height: 24px;
  min-height: 24px;
  padding: 0;
  border-radius: 6px;
  color: #717d92;
}
.number-input-field__step :deep(.q-focus-helper) {
  display: none;
}
.number-input-field__step :deep(.q-icon) {
  font-size: 16px;
}
.number-input-field__step:not(.disabled):hover,
.number-input-field__step:not(.disabled):focus-visible {
  color: $primary;
  background: rgba($primary, 0.08);
}
.cursor-pointer {
  cursor: pointer;
}
</style>
