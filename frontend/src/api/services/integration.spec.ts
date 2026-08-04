import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.hoisted(() => vi.fn())
const getMock = vi.hoisted(() => vi.fn())
const putMock = vi.hoisted(() => vi.fn())

vi.mock('boot/axios', () => ({
  instance: {
    post: postMock,
    get: getMock,
    put: putMock,
  },
}))

import { useIntegrationApi } from 'src/api/services/integration'

describe('external system API', () => {
  beforeEach(() => {
    postMock.mockReset()
    getMock.mockReset()
    putMock.mockReset()
    postMock.mockResolvedValue({ data: { success: true, data: [], total: 0 } })
    getMock.mockResolvedValue({ data: { success: true, data: {} } })
    putMock.mockResolvedValue({ data: { success: true, data: {} } })
  })

  it('uses the reviewed query, detail, create and update endpoints', async () => {
    const api = useIntegrationApi()
    const query = {
      page: 1,
      num: 15,
      quick_query: { keyword: 'ERP' },
      expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
    }
    const createRequest = {
      system_code: 'demo_erp',
      name: 'Demo ERP',
      system_type: 'erp' as const,
      base_url: 'https://erp.example.com',
      owner_identifier: 'owner-1',
      owner_name: '实施负责人',
    }

    await api.queryExternalSystems(query)
    await api.getExternalSystem(12)
    await api.createExternalSystem(createRequest)
    await api.updateExternalSystem(12, { name: 'ERP', revision: 3 })

    expect(postMock).toHaveBeenNthCalledWith(
      1,
      '/admin/integration/external-system/query',
      query,
    )
    expect(getMock).toHaveBeenCalledWith('/admin/integration/external-system/12')
    expect(postMock).toHaveBeenNthCalledWith(
      2,
      '/admin/integration/external-system',
      createRequest,
    )
    expect(putMock).toHaveBeenNthCalledWith(
      1,
      '/admin/integration/external-system/12',
      { name: 'ERP', revision: 3 },
    )
  })

  it('sends the optimistic revision when enabling and disabling', async () => {
    const api = useIntegrationApi()

    await api.enableExternalSystem(21, 4)
    await api.disableExternalSystem(21, 5)

    expect(putMock).toHaveBeenNthCalledWith(
      1,
      '/admin/integration/external-system/21/enable',
      { revision: 4 },
    )
    expect(putMock).toHaveBeenNthCalledWith(
      2,
      '/admin/integration/external-system/21/disable',
      { revision: 5 },
    )
  })

  it('uses the versioned interface definition endpoints', async () => {
    const api = useIntegrationApi()
    const query = { page: 1, num: 15, quick_query: { keyword: '订单' }, expressions: [] }
    const createRequest = {
      external_system_id: 12,
      interface_code: 'order_query',
      name: '订单查询',
      protocol: 'https' as const,
      http_method: 'GET' as const,
      relative_path: '/api/orders',
      timeout_seconds: 30,
      response_limit: 10485760,
    }

    await api.queryInterfaceDefinitions(query)
    await api.getInterfaceDefinition(31)
    await api.createInterfaceDefinition(createRequest)
    await api.updateInterfaceDefinition(31, { name: '订单明细查询', revision: 2 })
    await api.createInterfaceDefinitionVersion(31, 3)
    await api.enableInterfaceDefinition(32, 1)
    await api.disableInterfaceDefinition(32, 2)

    expect(postMock).toHaveBeenCalledWith('/admin/integration/interface-definition/query', query)
    expect(getMock).toHaveBeenCalledWith('/admin/integration/interface-definition/31')
    expect(postMock).toHaveBeenCalledWith('/admin/integration/interface-definition', createRequest)
    expect(putMock).toHaveBeenCalledWith('/admin/integration/interface-definition/31', { name: '订单明细查询', revision: 2 })
    expect(postMock).toHaveBeenCalledWith('/admin/integration/interface-definition/31/versions', { revision: 3 })
    expect(putMock).toHaveBeenCalledWith('/admin/integration/interface-definition/32/enable', { revision: 1 })
    expect(putMock).toHaveBeenCalledWith('/admin/integration/interface-definition/32/disable', { revision: 2 })
  })
})
