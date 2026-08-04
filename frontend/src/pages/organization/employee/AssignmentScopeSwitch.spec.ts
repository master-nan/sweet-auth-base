import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AssignmentScopeSwitch from './AssignmentScopeSwitch.vue'

describe('AssignmentScopeSwitch', () => {
  it('renders all time scopes and marks the current selection', () => {
    const wrapper = mount(AssignmentScopeSwitch, {
      props: { modelValue: 'history' },
    })

    expect(wrapper.findAll('[role="tab"]')).toHaveLength(4)
    expect(wrapper.get('[aria-selected="true"]').text()).toBe('历史')
    expect(wrapper.attributes('style')).toContain('--assignment-scope-index: 1')
  })

  it('emits a scope change only when a different enabled option is selected', async () => {
    const wrapper = mount(AssignmentScopeSwitch, {
      props: { modelValue: 'current' },
    })

    await wrapper.findAll('[role="tab"]')[2]?.trigger('click')
    await wrapper.findAll('[role="tab"]')[0]?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toEqual([['future']])
  })

  it('blocks selection while assignment data is loading', async () => {
    const wrapper = mount(AssignmentScopeSwitch, {
      props: { modelValue: 'current', loading: true },
    })

    await wrapper.findAll('[role="tab"]')[1]?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    expect(wrapper.findAll('button').every((button) => button.attributes('disabled') !== undefined)).toBe(true)
  })
})
