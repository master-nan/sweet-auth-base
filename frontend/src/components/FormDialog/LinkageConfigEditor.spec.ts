import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const darkState = vi.hoisted(() => ({ isActive: false }))

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: darkState }),
}))

import LinkageConfigEditor from './LinkageConfigEditor.vue'

describe('LinkageConfigEditor', () => {
  beforeEach(() => {
    darkState.isActive = false
  })

  it('applies the explicit dark surface state in dark mode', () => {
    darkState.isActive = true

    const wrapper = shallowMount(LinkageConfigEditor)

    expect(wrapper.classes()).toContain('linkage-editor--dark')
  })

  it('keeps the light surface state in light mode', () => {
    const wrapper = shallowMount(LinkageConfigEditor)

    expect(wrapper.classes()).not.toContain('linkage-editor--dark')
  })
})
