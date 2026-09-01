import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ExpressionLogic, ExpressionType } from 'src/types/enum'

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
    for (const icon of [
      'design_services',
      'content_copy',
      'publish',
      'history',
      'pause_circle',
      'delete',
    ]) {
      expect(wrapper.find(`[data-icon="${icon}"]`).exists()).toBe(true)
    }
  })

  it('keeps the category summary and queries uncategorized reports by empty value', async () => {
    const uncategorized = {
      id: 2,
      report_name: '未分类报表',
      report_code: 'uncategorized_report',
      report_kind: 'detail',
      category: '',
      status: 'draft',
    }
    const categorized = {
      id: 3,
      report_name: '审计报表',
      report_code: 'audit_report',
      report_kind: 'detail',
      category: '系统审计',
      status: 'published',
    }
    reportApi.queryReports
      .mockResolvedValueOnce({ data: [uncategorized, categorized], total: 2 })
      .mockResolvedValueOnce({ data: [uncategorized, categorized], total: 2 })
      .mockResolvedValueOnce({ data: [uncategorized], total: 1 })
      .mockResolvedValueOnce({ data: [uncategorized, categorized], total: 2 })

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.findAll('.category-item')).toHaveLength(3)
    await wrapper.findAll('.category-item')[1]!.trigger('click')
    await flushPromises()

    expect(wrapper.findAll('.category-item')).toHaveLength(3)
    const listQuery = reportApi.queryReports.mock.calls[2]![0]
    expect(listQuery.filters).not.toHaveProperty('category')
    expect(listQuery.expressions[0].logic).toBe(ExpressionLogic.OR)
    expect(listQuery.expressions[0].rules).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          field: 'category',
          expression_type: ExpressionType.EQ,
          value: '',
        }),
        expect.objectContaining({ field: 'category', expression_type: ExpressionType.IS_NULL }),
      ]),
    )
    expect(reportApi.queryReports.mock.calls[3]![0].filters).not.toHaveProperty('category')
  })
})
