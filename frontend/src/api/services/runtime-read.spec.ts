import { beforeEach, describe, expect, it, vi } from 'vitest'

const getMock = vi.hoisted(() => vi.fn())

vi.mock('boot/axios', () => ({
  instance: { get: getMock },
}))

import { useDictApi } from './sys-dict'
import { useTableApi } from './sys-table'

describe('authenticated runtime read APIs', () => {
  beforeEach(() => {
    getMock.mockReset()
    getMock.mockResolvedValue({ data: { success: true, data: {} } })
  })

  it('uses the dictionary runtime route instead of the administration route', async () => {
    await useDictApi().queryRuntimeDictByCode('status')

    expect(getMock).toHaveBeenCalledWith('/admin/runtime/dict/status', expect.any(Object))
  })

  it('uses the metadata runtime route instead of the administration route', async () => {
    await useTableApi().queryRuntimeTableByCode('orders')

    expect(getMock).toHaveBeenCalledWith('/admin/runtime/table/orders', expect.any(Object))
  })
})
