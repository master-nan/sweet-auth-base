import { defineComponent, h } from 'vue'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const dictApi = vi.hoisted(() => ({
  queryDict: vi.fn(),
  queryDictItemsByDictId: vi.fn(),
  createDict: vi.fn(),
  updateDict: vi.fn(),
  deleteDict: vi.fn(),
  createDictItem: vi.fn(),
  updateDictItem: vi.fn(),
  deleteDictItem: vi.fn(),
}))
const dictStore = vi.hoisted(() => ({
  loadDicts: vi.fn(),
  clearDict: vi.fn(),
  getDictLabel: vi.fn(),
}))

vi.mock('quasar', async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>()
  return { ...actual, useQuasar: () => ({ dark: { isActive: false } }) }
})
vi.mock('@/boot/axios', () => ({ instance: {} }))
vi.mock('@/api/services/sys-dict', () => ({ useDictApi: () => dictApi }))
vi.mock('@/stores/dict', () => ({ useDictStore: () => dictStore }))
vi.mock('@/composables/runtime-table-metadata', async () => {
  const { ref } = await import('vue')
  return {
    useRuntimeTableMetadata: () => ({
      fields: ref([]),
      formFields: ref([]),
      loadMetadata: vi.fn().mockResolvedValue(true),
    }),
  }
})
vi.mock('@/composables/page-buttons', async () => {
  const { ref } = await import('vue')
  return {
    useMasterDetailPageButtons: () => ({
      master_top_buttons: ref([]),
      master_line_buttons: ref([]),
      detail_top_buttons: ref([]),
      detail_line_buttons: ref([]),
    }),
  }
})
vi.mock('@/composables/confirm-dialog', () => ({
  useConfirmDialog: () => ({ confirmDanger: vi.fn(() => ({ onOk: vi.fn() })) }),
}))

import DictionaryPage from './Index.vue'

const SlotStub = defineComponent({
  setup(_, { slots }) {
    return () => h('div', slots.default?.())
  },
})
const MasterDetailStub = defineComponent({
  name: 'MasterDetailPage',
  props: { masterWidth: String },
  setup(props, { slots }) {
    return () =>
      h(
        'section',
        { 'data-testid': 'master-detail', 'data-master-width': props.masterWidth },
        Object.values(slots).flatMap((slot) => slot?.() || []),
      )
  },
})

const mountPage = () =>
  shallowMount(DictionaryPage, {
    global: {
      stubs: {
        BaseContent: SlotStub,
        MasterDetailPage: MasterDetailStub,
        StandardTableToolbar: SlotStub,
        TablePagination: true,
        DynamicFormDialog: true,
        QList: SlotStub,
        QItem: SlotStub,
        QItemSection: SlotStub,
        QItemLabel: SlotStub,
        QInput: true,
        QBtn: true,
        QIcon: true,
        QTooltip: true,
        QSpinner: true,
        QTable: SlotStub,
        QTd: SlotStub,
      },
    },
  })

describe('Dictionary master-detail workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    dictApi.queryDict.mockResolvedValue({
      data: [{ id: 7, dict_name: '状态', dict_code: 'status' }],
      total: 1,
    })
    dictApi.queryDictItemsByDictId.mockResolvedValue({
      data: [
        { id: 71, dict_id: 7, item_name: '启用', item_code: 'enabled', item_value: '1' },
        { id: 72, dict_id: 7, item_name: '停用', item_code: 'disabled', item_value: '0' },
      ],
    })
  })

  it('keeps the narrow master workspace exempt from Query Scheme controls', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-testid="master-detail"]').attributes('data-master-width')).toBe(
      '372px',
    )
    expect(wrapper.find('query-scheme-controls-stub').exists()).toBe(false)
    expect(dictApi.queryDict).toHaveBeenCalledOnce()
  })

  it('loads and filters detail items independently from the master query', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(dictApi.queryDictItemsByDictId).toHaveBeenCalledWith(7)
    const vm = wrapper.vm as unknown as {
      itemSearchText: string
      filteredDictItems: Array<{ item_name: string }>
      fetchData: () => Promise<void>
    }
    vm.itemSearchText = '启用'
    await wrapper.vm.$nextTick()
    expect(vm.filteredDictItems.map((item) => item.item_name)).toEqual(['启用'])

    await vm.fetchData()
    expect(dictApi.queryDict).toHaveBeenCalledTimes(2)
    expect(dictApi.queryDictItemsByDictId).toHaveBeenCalledTimes(1)
  })
})
