import { defineComponent, h, ref } from 'vue'
import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import QuerySchemeControls from './QuerySchemeControls.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import QuerySchemeSelector from 'src/components/QueryScheme/QuerySchemeSelector.vue'
import type { QuerySchemePageController } from 'src/composables/query-scheme-page'
import type { TableQueryState } from 'src/composables/table-query-state'
import { QuerySchemeType, QuerySchemeValidationStatus } from 'src/modules/query-scheme/types'
import type { Query } from 'src/types/global'

vi.mock('src/boot/axios', () => ({
  instance: { get: vi.fn(), post: vi.fn() },
}))

const QBtnStub = defineComponent({
  props: { icon: String },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h('button', { 'data-icon': props.icon, onClick: () => emit('click') }, slots.default?.())
  },
})

const createHarness = (props: { advancedEnabled?: boolean } = {}) => {
  const applyAdvancedQuery = vi.fn()
  const runQueryChange = vi.fn((change: () => void) => change())
  const controller = {
    runtime: {
      schemes: ref([]),
      currentLabel: ref('查询方案'),
      loading: ref(false),
      error: ref(''),
      scope: { config: ref(null) },
      loadAvailable: vi.fn(),
    },
    showSaveDialog: ref(false),
    saving: ref(false),
    runQueryChange,
    selectScheme: vi.fn(),
    applyPreset: vi.fn(),
    restoreCurrent: vi.fn(),
    resetDefault: vi.fn(),
    openManager: vi.fn(),
    savePersonal: vi.fn(),
  } as unknown as QuerySchemePageController<Query>
  const queryState = {
    draftAdvanced: ref<Query>({ page: 1, num: 15, expressions: [] }),
    appliedAdvanced: ref<Query>({ page: 1, num: 15, expressions: [] }),
    bindings: ref([]),
    schemeSource: ref(null),
    dirty: ref(false),
    beginAdvancedEdit: vi.fn(),
    applyAdvancedQuery,
  } as unknown as TableQueryState<Query>

  const wrapper = shallowMount(QuerySchemeControls, {
    props: { controller, queryState, fields: [], ...props },
    slots: { 'quick-search': '<input data-testid="quick-search" />' },
    global: {
      stubs: {
        QBtn: QBtnStub,
        QBadge: true,
        QTooltip: true,
      },
    },
  })
  return { wrapper, controller, queryState, runQueryChange, applyAdvancedQuery }
}

describe('QuerySchemeControls', () => {
  it('composes scheme UI while forwarding stable page actions', () => {
    const { wrapper, controller } = createHarness()
    const scheme = {
      id: 1,
      name: '我的方案',
      type: QuerySchemeType.PERSONAL,
      is_default: false,
      status: QuerySchemeValidationStatus.VALID,
    }

    wrapper.findComponent(QuerySchemeSelector).vm.$emit('select', scheme)
    wrapper.findComponent(QuerySchemeSelector).vm.$emit('manage')

    expect(controller.selectScheme).toHaveBeenCalledWith(scheme)
    expect(controller.openManager).toHaveBeenCalledOnce()
    expect(wrapper.find('[data-testid="quick-search"]').exists()).toBe(true)
  })

  it('owns only advanced/save presentation and delegates query mutations', async () => {
    const { wrapper, controller, queryState, runQueryChange, applyAdvancedQuery } = createHarness()

    await wrapper.find('[data-icon="tune"]').trigger('click')
    expect(queryState.beginAdvancedEdit).toHaveBeenCalledOnce()

    wrapper.findComponent(AdvancedQuery).vm.$emit('search')
    expect(runQueryChange).toHaveBeenCalledOnce()
    expect(applyAdvancedQuery).toHaveBeenCalledWith(queryState.draftAdvanced.value)

    await wrapper.find('[data-icon="bookmark_add"]').trigger('click')
    expect(controller.showSaveDialog.value).toBe(true)
  })

  it('does not mount the advanced editor when the page has no advanced fields', () => {
    const { wrapper } = createHarness({ advancedEnabled: false })

    expect(wrapper.find('[data-icon="tune"]').exists()).toBe(false)
    expect(wrapper.findComponent(AdvancedQuery).exists()).toBe(false)
  })
})
