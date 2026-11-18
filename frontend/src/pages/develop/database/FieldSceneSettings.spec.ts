import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent } from 'vue'
import FieldSceneSettings from './FieldSceneSettings.vue'

const select = defineComponent({ props: ['modelValue', 'label'], emits: ['update:modelValue'], template: '<select />' })
const checkbox = defineComponent({ props: ['modelValue', 'label'], emits: ['update:modelValue'], template: '<input type="checkbox" />' })

describe('字段维护的场景配置', () => {
  it('分别保存两种行模式和独立的详情开关，列表显示不兜底', () => {
    const form = mount(FieldSceneSettings, {
      props: { draft: { is_list_show: true, is_insert_show: false, is_update_show: true, is_update_readonly: true, detail_visible: false } },
      global: { stubs: { QSelect: select, QCheckbox: checkbox } },
    })
    const [insert, update] = form.findAllComponents(select)
    const [list, detail] = form.findAllComponents(checkbox)
    expect([insert!.props('modelValue'), update!.props('modelValue')]).toEqual(['hidden', 'readonly'])
    expect([list!.props('modelValue'), detail!.props('modelValue')]).toEqual([true, false])
    insert!.vm.$emit('update:modelValue', 'readonly')
    update!.vm.$emit('update:modelValue', 'hidden')
    detail!.vm.$emit('update:modelValue', true)
    expect(form.emitted('patch')).toEqual([
      [{ is_insert_show: true, is_insert_readonly: true }],
      [{ is_update_show: false, is_update_readonly: false }],
      [{ detail_visible: true }],
    ])
  })
})
