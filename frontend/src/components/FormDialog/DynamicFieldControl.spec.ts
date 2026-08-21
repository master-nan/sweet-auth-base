import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import DynamicFieldControl, {
  type DynamicFieldControlContext,
} from 'src/components/FormDialog/DynamicFieldControl.vue'
import JsonEditor from 'src/components/JsonEditor/JsonEditor.vue'
import ArrayInput from 'src/components/JsonEditor/ArrayInput.vue'
import KeyValueEditor from 'src/components/JsonEditor/KeyValueEditor.vue'
import type { TableField } from 'src/api/services/sys-table'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'

vi.mock('src/components/FileUpload/FileUpload.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Select/OrganizationSelect.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/FormDialog/LinkageConfigEditor.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/Cascader/CascaderSelect.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/DateTime/SweetDateTimePicker.vue', () => ({ default: { template: '<div />' } }))

const field = {
  id: 1,
  table_id: 1,
  field_name: '配置',
  field_code: 'config',
  field_type: SysTableFieldType.JSON,
  input_type: SysTableFieldInputType.JSON_EDITOR,
  is_null: false,
} as TableField

const context = (controlType: string): DynamicFieldControlContext => ({
  role: 'standard',
  controlType,
  readonly: true,
  tableCode: 'example',
  menuId: 1,
  rowId: 1,
  lazyRules: false,
  organization: null,
  rules: [value => (value ? true : '必填')],
  hint: '',
  placeholder: '',
  options: [],
  dictCodeOptions: [],
  cascaderSelectable: 'leaf',
  cascaderShowPath: false,
  maxLength: undefined,
  numberInputMode: 'decimal',
  exactDecimal: false,
  hideContent: false,
  visibilityResetToken: 0,
  relationSelect: false,
  relationLoading: false,
  relationHasMore: false,
  booleanOptions: [],
  file: { multiple: false, accept: '', maxSize: 1, chunkThreshold: 1, concurrency: 1 },
  fieldRef: vi.fn(),
})

describe('DynamicFieldControl complex input contract', () => {
  it.each([
    ['json-editor', JsonEditor],
    ['array-input', ArrayInput],
    ['key-value-editor', KeyValueEditor],
  ])('passes readonly and validation rules to %s', (controlType, component) => {
    const wrapper = shallowMount(DynamicFieldControl, {
      props: { field, modelValue: null, context: context(controlType) },
    })
    const control = wrapper.findComponent(component)
    expect(control.props('disabled')).toBe(true)
    expect(control.props('rules')).toHaveLength(1)
  })
})
