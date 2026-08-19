import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const apiMocks = vi.hoisted(() => ({
  getSyncBatchDetail: vi.fn(),
  getSyncBatchError: vi.fn(),
  querySyncRecords: vi.fn(),
}))
const permissions = vi.hoisted(() => ({ values: ['organization_sync_error_query'] }))

vi.mock('src/api/services/org', () => apiMocks)
vi.mock('src/stores/user', () => ({ useUserStore: () => ({ buttons: permissions.values }) }))

vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    record_detail_top_buttons: { value: [] },
    record_detail_bottom_buttons: { value: [] },
    hasGrantedCapability: (code: string) => permissions.values.includes(code),
  }),
}))

vi.mock('src/stores/dict', () => ({
  useDictStore: () => ({
    loadDicts: vi.fn().mockResolvedValue(undefined),
    getDictLabel: (_code: string, value: unknown) =>
      typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean'
        ? String(value)
        : '',
  }),
}))

import SyncBatchDetail from './Detail.vue'

const DetailContentStub = defineComponent({
  name: 'OrganizationRecordDetailContent',
  props: {
    title: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () => h('div', { 'data-testid': 'detail-title' }, props.title)
  },
})

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})

const mountDetail = (recordId: number) =>
  shallowMount(SyncBatchDetail, {
    props: { recordId },
    global: {
      stubs: {
        BaseContent: SlotStub,
        OrganizationRecordDetailContent: DetailContentStub,
        QTable: true,
        QTd: true,
      },
    },
  })

const batch = (id: number) => ({
  id,
  batch_no: `BATCH-${id}`,
  sync_type: 'full',
  object_scope: 'all',
  started_at: '2026-07-27T10:00:00Z',
  completed_at: '2026-07-27T10:05:00Z',
  total_count: 2,
  success_count: 2,
  failed_count: 0,
  skipped_count: 0,
  status: 'success',
  has_error: false,
})

describe('Organization sync batch detail', () => {
  beforeEach(() => {
    permissions.values = ['organization_sync_error_query']
    Object.values(apiMocks).forEach((mock) => mock.mockReset())
    apiMocks.getSyncBatchDetail.mockImplementation((id: number) => Promise.resolve(batch(id)))
    apiMocks.querySyncRecords.mockResolvedValue({
      items: [
        {
          id: 1,
          object_type: 'org_unit',
          source_summary: '0123456789abcdef01234567',
          action: 'update',
          status: 'success',
          error_code: '',
        },
      ],
      total: 1,
    })
  })

  it('loads the batch summary without requesting records when record query is not granted', async () => {
    permissions.values = []
    const wrapper = mountDetail(41)
    await flushPromises()

    expect(apiMocks.getSyncBatchDetail).toHaveBeenCalledWith(41)
    expect(apiMocks.querySyncRecords).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="detail-title"]').text()).toBe('同步批次详情：BATCH-41')
  })

  it('loads detail and records by the route record id', async () => {
    const wrapper = mountDetail(41)
    await flushPromises()

    expect(apiMocks.getSyncBatchDetail).toHaveBeenCalledWith(41)
    expect(apiMocks.querySyncRecords).toHaveBeenCalledWith(
      expect.objectContaining({ batch_id: 41 }),
    )
    expect(wrapper.get('[data-testid="detail-title"]').text()).toBe('同步批次详情：BATCH-41')
    expect(wrapper.emitted('title-change')?.at(-1)).toEqual(['同步批次详情：BATCH-41'])
  })

  it('reloads independently when another detail id is opened', async () => {
    const wrapper = mountDetail(41)
    await flushPromises()

    await wrapper.setProps({ recordId: 42 })
    await flushPromises()

    expect(apiMocks.getSyncBatchDetail).toHaveBeenNthCalledWith(1, 41)
    expect(apiMocks.getSyncBatchDetail).toHaveBeenNthCalledWith(2, 42)
    expect(apiMocks.querySyncRecords).toHaveBeenLastCalledWith(
      expect.objectContaining({ batch_id: 42 }),
    )
    expect(wrapper.get('[data-testid="detail-title"]').text()).toBe('同步批次详情：BATCH-42')
  })
})
