import { defineComponent, h, nextTick } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const queryOrganizationOptionsMock = vi.hoisted(() => vi.fn())

vi.mock('src/api/services/org', () => ({
  queryOrganizationOptions: queryOrganizationOptionsMock,
}))

import OrganizationSelect from 'src/components/Select/OrganizationSelect.vue'
import type { OrganizationSelectorType } from 'src/api/services/org'

const SweetSelectStub = defineComponent({
  name: 'SweetSelect',
  inheritAttrs: false,
  props: {
    modelValue: {
      type: [Number, Array],
      default: null,
    },
    options: {
      type: Array,
      default: () => [],
    },
    multiple: Boolean,
    disable: Boolean,
    clearable: Boolean,
    loading: Boolean,
  },
  emits: ['update:modelValue', 'filter'],
  setup(_, { slots }) {
    return () =>
      h('div', { 'data-testid': 'sweet-select' }, [slots.default?.(), slots['no-option']?.()])
  },
})

const mountSelector = (
  selectorType: OrganizationSelectorType,
  props: Record<string, unknown> = {},
) =>
  shallowMount(OrganizationSelect, {
    props: {
      selectorType,
      ...props,
    },
    global: {
      stubs: {
        SweetSelect: SweetSelectStub,
      },
    },
  })

describe('OrganizationSelect', () => {
  beforeEach(() => {
    queryOrganizationOptionsMock.mockResolvedValue({
      items: [],
      total: 0,
    })
  })

  it.each<OrganizationSelectorType>(['legal_entity', 'org_unit', 'employee', 'position'])(
    'renders and loads %s options through the shared API',
    async (selectorType) => {
      const wrapper = mountSelector(selectorType)

      await flushPromises()

      expect(wrapper.find('[data-testid="sweet-select"]').exists()).toBe(true)
      expect(queryOrganizationOptionsMock).toHaveBeenCalledWith(
        selectorType,
        expect.objectContaining({
          page: 1,
          num: 50,
          only_effective: true,
        }),
      )
    },
  )

  it('keeps internal IDs as the model value and emits change', async () => {
    const wrapper = mountSelector('legal_entity', { modelValue: 7 })
    await flushPromises()

    wrapper.findComponent(SweetSelectStub).vm.$emit('update:modelValue', 11)
    await nextTick()

    expect(wrapper.emitted('update:modelValue')).toEqual([[11]])
    expect(wrapper.emitted('change')).toEqual([[11]])
  })

  it('passes disabled state to the platform select', async () => {
    const wrapper = mountSelector('position', { disabled: true })
    await flushPromises()

    expect(wrapper.findComponent(SweetSelectStub).props('disable')).toBe(true)
  })

  it('exposes loading and empty states through the shared select', async () => {
    let resolveRequest: ((value: { items: []; total: number }) => void) | undefined
    queryOrganizationOptionsMock.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )

    const wrapper = mountSelector('legal_entity')
    expect(wrapper.findComponent(SweetSelectStub).props('loading')).toBe(true)

    resolveRequest?.({ items: [], total: 0 })
    await flushPromises()

    expect(wrapper.findComponent(SweetSelectStub).props('loading')).toBe(false)
    expect(wrapper.text()).toContain('暂无可选数据')
  })

  it('shows a stable error state when the options request fails', async () => {
    queryOrganizationOptionsMock.mockRejectedValueOnce(new Error('network error'))

    const wrapper = mountSelector('position')
    await flushPromises()

    expect(wrapper.text()).toContain('选项加载失败')
  })

  it('loads selected historical IDs and keeps replay options disabled', async () => {
    queryOrganizationOptionsMock.mockResolvedValueOnce({
      items: [
        {
          value: 23,
          label: 'EMP-23 - 历史员工',
          code: 'EMP-23',
          name: '历史员工',
          disabled: true,
        },
      ],
      total: 0,
    })

    const wrapper = mountSelector('employee', {
      modelValue: 23,
      selectedIds: [23, 23],
      includeHistory: true,
    })
    await flushPromises()

    expect(queryOrganizationOptionsMock).toHaveBeenCalledWith(
      'employee',
      expect.objectContaining({
        selected_ids: [23],
        include_history: true,
      }),
    )
    expect(wrapper.findComponent(SweetSelectStub).props('options')).toEqual([
      expect.objectContaining({
        value: 23,
        disabled: true,
      }),
    ])
  })

  it('uses remote keyword search instead of filtering all options locally', async () => {
    const wrapper = mountSelector('org_unit')
    await flushPromises()
    queryOrganizationOptionsMock.mockClear()
    queryOrganizationOptionsMock.mockResolvedValueOnce({
      items: [
        {
          value: 31,
          label: 'OU-31 - 财务中心',
          code: 'OU-31',
          name: '财务中心',
          disabled: false,
        },
      ],
      total: 1,
    })

    const update = vi.fn((callback: () => void) => callback())
    const abort = vi.fn()
    wrapper.findComponent(SweetSelectStub).vm.$emit('filter', ' 财务 ', update, abort)
    await flushPromises()

    expect(queryOrganizationOptionsMock).toHaveBeenCalledWith(
      'org_unit',
      expect.objectContaining({
        keyword: '财务',
      }),
    )
    expect(update).toHaveBeenCalledOnce()
    expect(abort).not.toHaveBeenCalled()
    expect(wrapper.findComponent(SweetSelectStub).props('options')).toEqual([
      expect.objectContaining({
        value: 31,
        label: 'OU-31 - 财务中心',
      }),
    ])
  })
})
