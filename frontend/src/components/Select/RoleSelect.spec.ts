import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const roleApi = vi.hoisted(() => ({
  queryRole: vi.fn(),
  queryRoleById: vi.fn(),
}))
vi.mock('src/api/services/sys-role', () => ({ useRoleApi: () => roleApi }))

import RoleSelect from './RoleSelect.vue'

const SweetSelectStub = defineComponent({
  name: 'SweetSelect',
  inheritAttrs: false,
  props: { modelValue: Array, options: Array, loading: Boolean },
  emits: ['update:modelValue', 'filter', 'virtual-scroll'],
  setup(props, { slots }) {
    return () => h('div', { 'data-testid': 'role-select' }, [
      JSON.stringify(props.options || []),
      slots['no-option']?.(),
    ])
  },
})

const mountSelector = (modelValue: number[] = []) =>
  shallowMount(RoleSelect, {
    props: { modelValue },
    global: { stubs: { SweetSelect: SweetSelectStub } },
  })

describe('RoleSelect', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    roleApi.queryRole.mockResolvedValue({ data: [], total: 0 })
    roleApi.queryRoleById.mockImplementation((id: number) =>
      Promise.resolve({ data: { id, name: `角色${id}` } }),
    )
  })

  it('uses the official paged role API and hydrates selected roles by id', async () => {
    const wrapper = mountSelector([101])
    await flushPromises()

    expect(roleApi.queryRole).toHaveBeenCalledWith(expect.objectContaining({ page: 1, num: 20 }))
    expect(roleApi.queryRoleById).toHaveBeenCalledWith(101)
    expect(wrapper.findComponent(SweetSelectStub).props('options')).toContainEqual({
      label: '角色101',
      value: 101,
    })
  })

  it('searches remotely and appends later result pages', async () => {
    const wrapper = mountSelector()
    await flushPromises()
    roleApi.queryRole.mockClear()
    roleApi.queryRole
      .mockResolvedValueOnce({ data: [{ id: 1, name: '审计角色' }], total: 2 })
      .mockResolvedValueOnce({ data: [{ id: 2, name: '只读角色' }], total: 2 })

    const update = vi.fn((callback: () => void) => callback())
    const abort = vi.fn()
    wrapper.findComponent(SweetSelectStub).vm.$emit('filter', ' 审计 ', update, abort)
    await flushPromises()
    wrapper.findComponent(SweetSelectStub).vm.$emit('virtual-scroll', { to: 0 })
    await flushPromises()

    expect(roleApi.queryRole).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ page: 1, num: 20, quick_query: { keyword: '审计' } }),
    )
    expect(roleApi.queryRole).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ page: 2, num: 20, quick_query: { keyword: '审计' } }),
    )
    expect(wrapper.findComponent(SweetSelectStub).props('options')).toEqual([
      { label: '审计角色', value: 1 },
      { label: '只读角色', value: 2 },
    ])
    expect(abort).not.toHaveBeenCalled()
  })
})
