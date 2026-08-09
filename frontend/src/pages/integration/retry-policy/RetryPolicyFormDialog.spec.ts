import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { describe, expect, it, vi } from 'vitest'

vi.mock('quasar', () => ({}))

import RetryPolicyFormDialog from './RetryPolicyFormDialog.vue'
import type { RetryPolicyDetail } from 'src/api/services/integration'

const FieldStub = defineComponent({
  props: { label: String, modelValue: [String, Number, Boolean, Array], rules: Array },
  emits: ['update:modelValue'],
  template: '<div :data-label="label" />',
})

const mountForm = () => mount(RetryPolicyFormDialog, {
  props: { modelValue: true, editData: null, loading: false },
  global: { stubs: {
    FormDialogShell: { template: '<div><slot /><slot name="footer-status" /></div>' },
    QForm: { template: '<form><slot /></form>' }, QInput: FieldStub, QSelect: FieldStub, QToggle: FieldStub,
  } },
})

describe('retry policy form', () => {
  it('uses frozen V1 defaults and clears inactive backoff and jitter fields', async () => {
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as { form: Record<string, string | number | boolean | string[] | number[]> }
    expect(vm.form.max_attempts).toBe(3)
    expect(vm.form.initial_delay_ms).toBe(5000)
    expect(vm.form.retryable_http_statuses).toEqual([429, 502, 503, 504])

    vm.form.backoff_type = 'fixed'
    vm.form.jitter_type = 'none'
    await nextTick()
    expect(vm.form.backoff_multiplier).toBe(1)
    expect(vm.form.jitter_ratio).toBe(0)
    expect(wrapper.find('[data-label="退避倍数 *"]').exists()).toBe(false)
    expect(wrapper.find('[data-label="抖动比例 *"]').exists()).toBe(false)

    vm.form.backoff_type = 'exponential'
    vm.form.jitter_type = 'full'
    await nextTick()
    expect(vm.form.backoff_multiplier).toBe(2)
    expect(vm.form.jitter_ratio).toBe(1)
  })

  it('contains no delete, immediate retry or replay controls', () => {
    const text = mountForm().text()
    expect(text).not.toContain('删除')
    expect(text).not.toContain('立即重试')
    expect(text).not.toContain('重放')
  })

  it('requires the retry window to cover the complete backoff schedule', () => {
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as {
      form: Record<string, string | number | boolean | string[] | number[]>
      retryWindowRule: (value: number) => true | string
    }
    vm.form.max_attempts = 10
    vm.form.initial_delay_ms = 3600000
    vm.form.max_delay_ms = 3600000
    vm.form.backoff_type = 'fixed'
    vm.form.backoff_multiplier = 1
    expect(vm.retryWindowRule(3600)).toBeTypeOf('string')
    expect(vm.retryWindowRule(32400)).toBe(true)
  })

  it('preserves a valid exponential multiplier when loading a version draft', async () => {
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as { form: Record<string, string | number | boolean | string[] | number[]> }
    vm.form.backoff_type = 'fixed'
    await nextTick()

    await wrapper.setProps({ editData: {
      id: 92, policy_code: 'erp_retry', policy_name: 'ERP 重试', version: 2, status: 'draft',
      description: '', max_attempts: 4, initial_delay_ms: 3000, max_delay_ms: 60000,
      backoff_type: 'exponential', backoff_multiplier: 3, jitter_type: 'full', jitter_ratio: 1,
      retry_window_ms: 3600000, retryable_error_categories: ['network'], retryable_http_statuses: [503],
      respect_retry_after: true, revision: 1, gmt_create: '', gmt_modify: '',
    } satisfies RetryPolicyDetail })
    await nextTick()

    expect(vm.form.backoff_multiplier).toBe(3)
  })
})
