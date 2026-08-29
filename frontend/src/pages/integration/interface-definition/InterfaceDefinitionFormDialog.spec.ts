import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { describe, expect, it, vi } from 'vitest'

vi.mock('quasar', () => ({}))

import InterfaceDefinitionFormDialog from './InterfaceDefinitionFormDialog.vue'

const QInputStub = defineComponent({
  name: 'QInput',
  props: { label: String, max: [String, Number], rules: Array, modelValue: [String, Number] },
  template: '<div />',
})
const QSelectStub = defineComponent({
  name: 'QSelect',
  props: { label: String, clearable: Boolean, modelValue: [String, Number] },
  template: '<div />',
})

describe('interface definition runtime limits', () => {
  it('uses the transport timeout and response-size limits for create and edit forms', () => {
    const wrapper = mount(InterfaceDefinitionFormDialog, {
      props: { modelValue: true, editData: null, systems: [], credentials: [], loading: false },
      global: { stubs: {
        FormDialogShell: { template: '<div><slot /></div>' }, QForm: { template: '<form><slot /></form>' },
        QSelect: QSelectStub, QInput: QInputStub,
      } },
    })
    const inputs = wrapper.findAllComponents(QInputStub)
    const timeout = inputs.find((input) => input.props('label') === '请求超时（秒） *')
    const response = inputs.find((input) => input.props('label') === '响应大小限制（KiB） *')
    if (!timeout || !response) throw new Error('runtime limit inputs were not rendered')
    const timeoutRule = (timeout.props('rules') as Array<(value: number) => true | string>)[0]
    const responseRule = (response.props('rules') as Array<() => true | string>)[0]
    if (!timeoutRule || !responseRule) throw new Error('runtime limit rules were not rendered')
    expect(timeout?.props('max')).toBe(120)
    expect(timeoutRule(120)).toBe(true)
    expect(timeoutRule(121)).toBeTypeOf('string')
    expect(response?.props('max')).toBe(65536)

    const vm = wrapper.vm as unknown as {
      form: { response_limit: number }
      updateResponseLimitKiB: (value: number) => void
    }
    vm.updateResponseLimitKiB(65536)
    expect(vm.form.response_limit).toBe(64 * 1024 * 1024)
    vm.updateResponseLimitKiB(65537)
    expect(responseRule()).toBeTypeOf('string')
  })

  it('uses explicit empty selections and builds a structured request contract', () => {
    const wrapper = mount(InterfaceDefinitionFormDialog, {
      props: {
        modelValue: true,
        editData: null,
        systems: [],
        credentials: [],
        retryPolicies: [],
        loading: false,
      },
      global: { stubs: {
        FormDialogShell: { template: '<div><slot /></div>' },
        QForm: { template: '<form><slot /></form>' },
        QSelect: QSelectStub,
        QInput: QInputStub,
        QBtn: true,
        QBanner: true,
        QToggle: true,
        QTooltip: true,
      } },
    })
    const vm = wrapper.vm as unknown as {
      credentialOptions: Array<{ label: string; value: number | null }>
      retryPolicyOptions: Array<{ label: string; value: number | null }>
      form: { input_contract: { version: number; parameters: Array<Record<string, unknown>> } }
      addParameter: () => void
    }
    expect(vm.credentialOptions).toEqual([{ label: '不使用认证凭证', value: null }])
    expect(vm.retryPolicyOptions).toEqual([{ label: '不自动重试', value: null }])
    const selectors = wrapper.findAllComponents(QSelectStub)
    expect(selectors.find((item) => item.props('label') === '认证凭证')?.props('clearable')).toBe(false)
    expect(selectors.find((item) => item.props('label') === '重试策略')?.props('clearable')).toBe(false)
    vm.addParameter()
    expect(vm.form.input_contract).toEqual({
      version: 1,
      parameters: [
        expect.objectContaining({
          code: '',
          location: 'query',
          data_type: 'string',
          required: false,
          allow_multiple: false,
          sensitive: false,
        }),
      ],
    })
  })
})
