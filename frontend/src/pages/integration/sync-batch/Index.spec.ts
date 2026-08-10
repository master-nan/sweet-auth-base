import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const api = vi.hoisted(() => ({ querySyncBatches: vi.fn(), getSyncBatch: vi.fn() }))
vi.mock('quasar', () => ({ useQuasar: () => ({ screen: { lt: { md: false } } }) }))
vi.mock('boot/axios', () => ({ instance: {} }))
vi.mock('src/api/services/integration', () => ({ useIntegrationApi: () => api }))
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: ['integration_sync_batch_detail'] }) }))
vi.mock('src/stores/loading', () => ({ useLoadingStore: () => ({ loading: false }) }))
vi.mock('src/components/BaseContent/BaseContent.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('src/components/Table/TablePagination.vue', () => ({ default: { template: '<div />' } }))
vi.mock('src/components/FormDialog/FormDialogShell.vue', () => ({ default: { template: '<div><slot /></div>' } }))
import Page from './Index.vue'

const Slot = defineComponent({ setup(_, { slots }) { return () => h('div', slots.default?.()) } })
const Table = defineComponent({ props: { rows: Array }, setup(props, { slots }) { return () => h('div', { 'data-count': props.rows?.length }, [slots.top?.(), slots.bottom?.()]) } })

describe('sync batch query page', () => {
  it('is query-only and exposes no run, cancel or checkpoint mutation command', async () => {
    api.querySyncBatches.mockResolvedValue({ data: [], total: 0 })
    const wrapper = shallowMount(Page, { global: { stubs: { BaseContent: Slot, QTable: Table, QInput: true, QSelect: true, QBtn: true, QIcon: true, QChip: true, QTd: Slot, QSpace: true, TablePagination: true, FormDialogShell: true } } })
    await flushPromises()
    expect(api.querySyncBatches).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toMatch(/运行|取消|修改 Checkpoint|补数|Dry Run/)
  })
})
