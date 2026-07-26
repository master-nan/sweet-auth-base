import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.hoisted(() => vi.fn())

vi.mock('boot/axios', () => ({
  instance: {
    post: postMock,
  },
}))

import {
  organizationOptionsEndpoints,
  queryOrganizationOptions,
  type OrganizationSelectorType,
} from 'src/api/services/org'

describe('organization options API', () => {
  beforeEach(() => {
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
