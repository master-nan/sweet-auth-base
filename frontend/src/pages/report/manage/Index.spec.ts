import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const permissionCodes = vi.hoisted(() => [] as string[])
const reportApi = vi.hoisted(() => ({
  queryReports: vi.fn(),
  queryDataSources: vi.fn(),
}))

vi.mock('quasar', () => ({
  useQuasar: () => ({
    notify: vi.fn(),
    dialog: vi.fn(),
    screen: { lt: { md: false } },
  }),
}))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))
vi.mock('src/api/services/report', () => ({
  defaultReportSheet: () => ({}),
  useReportApi: () => reportApi,
}))
vi.mock('src/stores/loading', () => ({ useLoadingStore: () => ({ loading: ref(false) }) }))
vi.mock('pinia', () => ({ storeToRefs: (store: { loading: unknown }) => store }))
vi.mock('src/composables/page-buttons', () => ({
  usePageButtons: () => ({
    hasGrantedCapability: (code: string) => permissionCodes.includes(code),
  }),
}))
vi.mock('../composables/useReportExport', () => ({
  useReportExport: () => ({ exportingReportId: ref(null), exportReportRow: vi.fn() }),
}))
vi.mock('components/BaseContent/BaseContent.vue', () => ({
  default: { template: '<div><slot /></div>' },
}))
vi.mock('components/Table/TablePagination.vue', () => ({
  default: { template: '<div />' },
}))
vi.mock('../components/ReportRuntimeDialog.vue', () => ({
  default: { template: '<div />' },
}))
vi.mock('../components/ReportVersionDialog.vue', () => ({
  default: { template: '<div />' },
}))

import ReportManagePage from './Index.vue'

const actionButtonStub = {
  props: ['icon', 'label'],
  template: '<button :data-icon="icon">{{ label }}</button>',
}

const mountPage = () =>
  mount(ReportManagePage, {
    global: {
      stubs: {
        QInput: true,
        QIcon: true,
        QBadge: true,
        QSeparator: true,
        QSelect: true,
        QChip: true,
        QTd: { template: '<div><slot /></div>' },
        QTooltip: { template: '<span><slot /></span>' },
        QSpace: true,
        QBtn: actionButtonStub,
        QTable: {
          props: ['rows'],
          template:
            '<div><slot v-if="rows.length" name="body-cell-actions" :row="rows[0]" /></div>',
        },
      },
    },
  })

describe('report management capabilities', () => {
  beforeEach(() => {
    permissionCodes.splice(0)
    reportApi.queryReports.mockReset()
    reportApi.queryDataSources.mockReset()
    reportApi.queryReports.mockResolvedValue({
      data: [
        {
          id: 1,
          report_name: '订单明细',
          report_code: 'order_detail',
          report_kind: 'detail',
          status: 'published',
        },
      ],
      total: 1,
    })
    reportApi.queryDataSources.mockResolvedValue({ data: [], total: 0 })
  })

  it('renders runtime actions but no mutation actions for a read-only user', async () => {
    permissionCodes.push('report_manage_run', 'report_manage_export')
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-icon="add"]').exists()).toBe(false)
    expect(wrapper.find('[data-icon="play_arrow"]').exists()).toBe(true)
    expect(wrapper.find('[data-icon="download"]').exists()).toBe(true)
    for (const icon of ['design_services', 'content_copy', 'publish', 'pause_circle', 'delete']) {
      expect(wrapper.find(`[data-icon="${icon}"]`).exists()).toBe(false)
    }
  })

  it('renders management actions only when their capabilities are granted', async () => {
    permissionCodes.push(
      'report_manage_create',
      'report_manage_design',
      'report_manage_copy',
      'report_manage_status',
      'report_manage_delete',
      'report_manage_publish',
      'report_manage_versions',
    )
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('[data-icon="add"]').exists()).toBe(true)
    for (const icon of ['design_services', 'content_copy', 'publish', 'history', 'pause_circle', 'delete']) {
      expect(wrapper.find(`[data-icon="${icon}"]`).exists()).toBe(true)
    }
  })
})
