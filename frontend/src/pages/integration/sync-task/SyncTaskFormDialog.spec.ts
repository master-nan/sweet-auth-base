import { defineComponent, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({ getInterfaceDefinition: vi.fn() }))
vi.mock('quasar', () => ({}))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => apiMocks }))
import SyncTaskFormDialog from './SyncTaskFormDialog.vue'

const FieldStub = defineComponent({
  props: { label: String, modelValue: [String, Number, Boolean, Array], options: Array },
  emits: ['update:modelValue'],
  template: '<div :data-label="label" />',
})
const interfaceDetail = {
  id: 12,
  input_contract: {
    version: 1,
    parameters: [
      {
        code: 'updated_from',
        name: '更新时间起点',
        location: 'query',
        data_type: 'string',
        required: true,
        allow_multiple: false,
        sensitive: false,
      },
      {
        code: 'updated_to',
        name: '更新时间终点',
        location: 'query',
        data_type: 'string',
        required: true,
        allow_multiple: false,
        sensitive: false,
      },
      {
        code: 'Authorization',
        location: 'header',
        data_type: 'string',
        required: false,
        allow_multiple: false,
        sensitive: true,
      },
      {
        code: 'tenant',
        name: '租户',
        location: 'body',
        data_type: 'string',
        required: false,
        allow_multiple: false,
        sensitive: false,
      },
    ],
  },
}
const mountForm = () =>
  mount(SyncTaskFormDialog, {
    props: {
      modelValue: true,
      editData: null,
      systems: [
        {
          id: 1,
          name: 'HR',
          system_code: 'hr',
          system_type: 'hr',
          base_url_summary: 'https://hr.example',
          owner_identifier: 'platform',
          owner_name: '平台',
          status: 'enabled',
          revision: 1,
          gmt_modify: '2026-08-10T00:00:00Z',
        },
      ],
      interfaces: [
        {
          id: 12,
          external_system: { id: 1, system_code: 'hr', name: 'HR' },
          status: 'enabled',
          effective_status: 'enabled',
          name: '人员',
          interface_code: 'employee',
          version: 1,
          protocol: 'https',
          http_method: 'GET',
          path_summary: '/employees',
          revision: 1,
          gmt_modify: '2026-08-10T00:00:00Z',
        },
      ],
      consumers: [
        {
          code: 'org_employee',
          version: 1,
          name: '人员消费器',
          content_types: ['application/json'],
          max_response_bytes: 1024,
          max_duration_ms: 1000,
          checkpoint_modes: ['timestamp'],
        },
      ],
    },
    global: {
      stubs: {
        FormDialogShell: { template: '<div><slot /><slot name="footer-status" /></div>' },
        QForm: { template: '<form><slot /></form>' },
        QInput: FieldStub,
        QSelect: FieldStub,
        QToggle: FieldStub,
      },
    },
  })

describe('sync task controlled form', () => {
  it('uses empty values for dependent selectors until the user makes a choice', () => {
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as {
      form: { external_system_id: number | null; interface_definition_id: number | null }
      onSystemChanged: () => void
    }

    expect(vm.form.external_system_id).toBeNull()
    expect(vm.form.interface_definition_id).toBeNull()
    vm.form.external_system_id = 1
    vm.form.interface_definition_id = 12
    vm.onSystemChanged()
    expect(vm.form.interface_definition_id).toBeNull()
  })

  it('clears inactive schedule and checkpoint fields', async () => {
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as {
      form: Record<string, unknown>
      windowStartKey: string
      windowEndKey: string
    }
    vm.form.schedule_type = 'cron'
    vm.form.cron_expression = '*/5 * * * *'
    vm.form.checkpoint_mode = 'timestamp'
    vm.form.window_slice_seconds = 3600
    await nextTick()
    vm.form.schedule_type = 'none'
    vm.form.checkpoint_mode = 'none'
    await nextTick()
    expect(vm.form.cron_expression).toBe('')
    expect(vm.form.window_slice_seconds).toBe(0)
    expect(vm.windowStartKey).toBe('')
    expect(vm.windowEndKey).toBe('')
  })

  it('offers only non-sensitive parameters from the interface contract', async () => {
    apiMocks.getInterfaceDefinition.mockResolvedValue({ data: interfaceDetail })
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as {
      onInterfaceChanged: (id: number) => Promise<void>
      staticParameterOptions: Array<{ value: string }>
      windowParameterOptions: Array<{ value: string }>
    }
    await vm.onInterfaceChanged(12)
    await flushPromises()
    expect(vm.staticParameterOptions.map((item) => item.value)).toContain('body:tenant')
    expect(vm.windowParameterOptions.map((item) => item.value)).toEqual(
      expect.arrayContaining(['query:updated_from', 'query:updated_to']),
    )
    expect(vm.staticParameterOptions.map((item) => item.value)).not.toContain(
      'header:Authorization',
    )
  })

  it('does not expose free JSON, run, cancel, checkpoint update or scripts', () => {
    const text = mountForm().text()
    for (const forbidden of ['自由 JSON', '运行一次', '取消批次', '修改 Checkpoint', '脚本', 'SQL'])
      expect(text).not.toContain(forbidden)
  })

  it('builds V2 lower-bound plans without a fake end binding', async () => {
    apiMocks.getInterfaceDefinition.mockResolvedValue({ data: interfaceDetail })
    const wrapper = mountForm()
    const vm = wrapper.vm as unknown as {
      onInterfaceChanged: (id: number) => Promise<void>
      form: Record<string, unknown>
      windowMode: 'bounded_window' | 'lower_bound_only'
      windowStartKey: string
      buildInputPlan: () => Record<string, unknown>
    }
    await vm.onInterfaceChanged(12)
    await flushPromises()
    vm.form.checkpoint_mode = 'timestamp'
    vm.windowMode = 'lower_bound_only'
    vm.windowStartKey = 'query:updated_from'
    const plan = vm.buildInputPlan()
    expect(plan).toMatchObject({
      version: 2,
      window_mode: 'lower_bound_only',
      window_start_binding: { location: 'query', code: 'updated_from' },
    })
    expect(plan).not.toHaveProperty('window_end_binding')
  })
})
