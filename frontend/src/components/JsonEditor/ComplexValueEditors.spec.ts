import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import JsonEditor from './JsonEditor.vue'
import ArrayInput from './ArrayInput.vue'
import KeyValueEditor from './KeyValueEditor.vue'

const required = (value: unknown) => {
  if (Array.isArray(value)) return value.length > 0 || '必填'
  if (value && typeof value === 'object') return Object.keys(value).length > 0 || '必填'
  return !!value || '必填'
}

describe('complex value editors', () => {
  it.each([
    [JsonEditor, {}],
    [ArrayInput, []],
    [KeyValueEditor, {}],
  ])('exposes required validation and readonly state', (component, modelValue) => {
    const wrapper = shallowMount(component, {
      props: { modelValue, rules: [required], disabled: true },
    })
    expect(wrapper.props('disabled')).toBe(true)
    expect((wrapper.vm as unknown as { validate: () => boolean }).validate()).toBe(false)
  })
})
