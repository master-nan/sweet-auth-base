import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { buildOrganizationDetailRoute } from './organization-detail-route'

const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    {
      path: '/admin/detail/:source/:table_code/:id',
      name: 'record_detail',
      component: { template: '<div />' },
    },
  ],
})

describe('Organization detail route', () => {
  it('places the record id in the URL and keeps different records as different routes', () => {
    const first = router.resolve(
      buildOrganizationDetailRoute('org_sync_batch', 41, 'BATCH-41'),
    )
    const second = router.resolve(
      buildOrganizationDetailRoute('org_sync_batch', 42, 'BATCH-42'),
    )

    expect(first.path).toBe('/admin/detail/organization/org_sync_batch/41')
    expect(second.path).toBe('/admin/detail/organization/org_sync_batch/42')
    expect(first.fullPath).not.toBe(second.fullPath)
  })
})
