import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.hoisted(() => vi.fn())
const getMock = vi.hoisted(() => vi.fn())

vi.mock('@/boot/axios', () => ({
  instance: {
    post: postMock,
    get: getMock,
  },
}))

import {
  getEmployeeAssignmentSummary,
  getEmployeeDetail,
  getLegalEntityDetail,
  getLegalEntityTree,
  getOrgUnitDetail,
  getPositionDetail,
  getStructureOrgTree,
  getSyncBatchDetail,
  getSyncBatchError,
  getSyncRecordDetail,
  getSyncRecordError,
  organizationOptionsEndpoints,
  queryAssignments,
  queryEmployees,
  queryOrganizationOptions,
  queryPositions,
  queryStructureOptions,
  queryStructures,
  querySyncBatches,
  querySyncRecords,
  type OrganizationSelectorType,
} from '@/api/services/org'

describe('organization options API', () => {
  beforeEach(() => {
    postMock.mockReset()
    getMock.mockReset()
    postMock.mockResolvedValue({
      data: {
        success: true,
        data: [
          {
            value: 12,
            label: 'ORG-12 - 华东中心',
            code: 'ORG-12',
            name: '华东中心',
            disabled: false,
          },
        ],
        total: 1,
      },
    })
  })

  it.each(Object.entries(organizationOptionsEndpoints))(
    'routes %s through its options endpoint',
    async (selectorType, endpoint) => {
      const request = {
        page: 1,
        num: 50,
        keyword: '华东',
        selected_ids: [12],
        only_effective: true,
        include_history: false,
      }

      const result = await queryOrganizationOptions(
        selectorType as OrganizationSelectorType,
        request,
      )

      expect(postMock).toHaveBeenCalledWith(endpoint, request, {
        headers: {
          'X-Skip-Global-Loading': 'true',
        },
      })
      expect(result).toEqual({
        items: [
          {
            value: 12,
            label: 'ORG-12 - 华东中心',
            code: 'ORG-12',
            name: '华东中心',
            disabled: false,
          },
        ],
        total: 1,
      })
    },
  )

  it('drops options whose value is not a positive internal ID', async () => {
    postMock.mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          {
            value: 0,
            label: 'invalid',
            code: '',
            name: '',
            disabled: false,
          },
          {
            value: 18,
            label: 'EMP-18 - 张三',
            code: 'EMP-18',
            name: '张三',
            disabled: true,
          },
        ],
        total: 2,
      },
    })

    const result = await queryOrganizationOptions('employee', {
      page: 1,
      num: 50,
    })

    expect(result.items).toEqual([
      {
        value: 18,
        label: 'EMP-18 - 张三',
        code: 'EMP-18',
        name: '张三',
        disabled: true,
      },
    ])
  })
})

describe('organization read-only center API', () => {
  beforeEach(() => {
    postMock.mockReset()
    getMock.mockReset()
  })

  it('loads the legal-entity tree and keeps legal_entity_id as row identity', async () => {
    postMock.mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          {
            legal_entity_id: 10,
            value: 10,
            label: 'LE-10 - 集团',
            code: 'LE-10',
            name: '集团',
            short_name: '集团',
            entity_type: 'group',
            status: 'enabled',
            disabled: false,
            children: [
              {
                legal_entity_id: 11,
                value: 11,
                label: 'LE-11 - 子公司',
                code: 'LE-11',
                name: '子公司',
                short_name: '子公司',
                entity_type: 'legal_company',
                status: 'enabled',
                disabled: false,
              },
            ],
          },
        ],
      },
    })

    const result = await getLegalEntityTree({ only_effective: true })

    expect(postMock).toHaveBeenCalledWith(
      '/admin/org/legal-entity/tree',
      { only_effective: true },
      expect.any(Object),
    )
    expect(result[0]?.id).toBe(10)
    expect(result[0]?.children?.[0]?.id).toBe(11)
  })

  it('loads legal-entity and organization-unit details from reviewed endpoints', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { id: 10, code: 'LE-10', name: '集团' },
        },
      })
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: { id: 20, code: 'OU-20', name: '财务中心' },
        },
      })

    await getLegalEntityDetail(10, { only_effective: true })
    await getOrgUnitDetail(20, { only_effective: true })

    expect(getMock).toHaveBeenNthCalledWith(
      1,
      '/admin/org/legal-entity/10',
      expect.objectContaining({ params: { only_effective: true } }),
    )
    expect(getMock).toHaveBeenNthCalledWith(
      2,
      '/admin/org/unit/20',
      expect.objectContaining({ params: { only_effective: true } }),
    )
  })

  it('loads structure options from backend names and preserves structure-node occurrence identity', async () => {
    postMock
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: [
            {
              value: 30,
              label: 'GROUP - 集团组织视图',
              code: 'MGMT',
              name: '集团组织视图',
              disabled: false,
            },
          ],
          total: 1,
        },
      })
      .mockResolvedValueOnce({
        data: {
          success: true,
          data: [
            {
              structure_node_id: 301,
              structure_id: 30,
              org_unit_id: 40,
              code: 'OU-40',
              name: '共享中心',
              unit_type: 'center',
              status: 'enabled',
              node_status: 'enabled',
              level: 1,
              sort: 1,
              disabled: false,
            },
            {
              structure_node_id: 302,
              structure_id: 30,
              org_unit_id: 40,
              code: 'OU-40',
              name: '共享中心',
              unit_type: 'center',
              status: 'enabled',
              node_status: 'enabled',
              level: 1,
              sort: 2,
              disabled: false,
            },
          ],
        },
      })

    const structures = await queryStructureOptions({
      page: 1,
      num: 100,
      only_effective: true,
    })
    const tree = await getStructureOrgTree({
      structure_id: structures.items[0]!.value,
      only_effective: true,
    })

    expect(structures.items[0]).toEqual(expect.objectContaining({ value: 30, code: 'MGMT' }))
    expect(tree.map((node) => node.id)).toEqual([301, 302])
    expect(tree.map((node) => node.org_unit_id)).toEqual([40, 40])
  })

  it('loads structure definitions with their backend default marker', async () => {
    postMock.mockResolvedValueOnce({
      data: {
        success: true,
        data: [
          {
            id: 30,
            code: 'GROUP',
            name: '集团组织视图',
            structure_type: 'management',
            status: 'enabled',
            is_default: true,
          },
        ],
        total: 1,
      },
    })

    const result = await queryStructures({
      page: 1,
      num: 100,
      only_effective: true,
    })

    expect(postMock).toHaveBeenCalledWith(
      '/admin/org/structure/query',
      {
        page: 1,
        num: 100,
        only_effective: true,
      },
      expect.any(Object),
    )
    expect(result).toEqual({
      items: [
        expect.objectContaining({
          id: 30,
          name: '集团组织视图',
          is_default: true,
        }),
      ],
      total: 1,
    })
  })
})

describe('organization remaining read-only pages API', () => {
  beforeEach(() => {
    postMock.mockReset()
    getMock.mockReset()
  })

  it('uses the employee, assignment, and position read-only endpoints', async () => {
    postMock.mockResolvedValue({
      data: {
        success: true,
        data: [],
        total: 0,
      },
    })
    getMock.mockResolvedValue({
      data: {
        success: true,
        data: {},
      },
    })

    await queryEmployees({ page: 1, num: 20, expressions: [] })
    await getEmployeeDetail(21)
    await queryAssignments({
      page: 1,
      num: 20,
      expressions: [],
      employee_id: 21,
      time_scope: 'timeline',
    })
    await getEmployeeAssignmentSummary(21, '2026-07-29')
    await queryPositions({ page: 1, num: 20, expressions: [] })
    await getPositionDetail(31)

    expect(postMock).toHaveBeenNthCalledWith(1, '/admin/org/employee/query', {
      page: 1,
      num: 20,
      expressions: [],
    })
    expect(getMock).toHaveBeenCalledWith('/admin/org/employee/21')
    expect(postMock).toHaveBeenNthCalledWith(2, '/admin/org/assignment/query', {
      page: 1,
      num: 20,
      expressions: [],
      employee_id: 21,
      time_scope: 'timeline',
    })
    expect(getMock).toHaveBeenCalledWith('/admin/org/employee/21/assignments/summary', {
      params: { as_of_date: '2026-07-29' },
    })
    expect(postMock).toHaveBeenNthCalledWith(
      3,
      '/admin/org/position/query',
      {
        page: 1,
        num: 20,
        expressions: [],
      },
      { headers: { 'X-Skip-Global-Loading': 'true' } },
    )
    expect(getMock).toHaveBeenCalledWith('/admin/org/position/31')
  })

  it('keeps synchronization summaries separate from protected error details', async () => {
    postMock.mockResolvedValue({
      data: {
        success: true,
        data: [],
        total: 0,
      },
    })
    getMock.mockResolvedValue({
      data: {
        success: true,
        data: {},
      },
    })

    await querySyncBatches({ page: 1, num: 20, expressions: [], status: 'failed' })
    await getSyncBatchDetail(41)
    await getSyncBatchError(41)
    await querySyncRecords({
      page: 1,
      num: 20,
      expressions: [],
      object_type: 'employee',
      local_id: 21,
      status: 'failed',
    })
    await getSyncRecordDetail(51)
    await getSyncRecordError(51)

    expect(postMock).toHaveBeenNthCalledWith(1, '/admin/org/sync/batch/query', {
      page: 1,
      num: 20,
      expressions: [],
      status: 'failed',
    })
    expect(getMock).toHaveBeenCalledWith('/admin/org/sync/batch/41')
    expect(getMock).toHaveBeenCalledWith('/admin/org/sync/batch/41/error')
    expect(postMock).toHaveBeenNthCalledWith(2, '/admin/org/sync/record/query', {
      page: 1,
      num: 20,
      expressions: [],
      object_type: 'employee',
      local_id: 21,
      status: 'failed',
    })
    expect(getMock).toHaveBeenCalledWith('/admin/org/sync/record/51')
    expect(getMock).toHaveBeenCalledWith('/admin/org/sync/record/51/error')
  })
})
