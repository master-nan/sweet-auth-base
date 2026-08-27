import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { Menu } from 'src/api/services/sys-menu'
import { SysDetailOpenMode } from 'src/types/enum'
import { resolveOrganizationDetailMode } from './organization-detail-mode'
import { buildOrganizationDetailRoute } from './organization-detail-route'
import { organizationSyncObjectLabel } from './organization-list-page'

const menu = (detailOpenMode?: SysDetailOpenMode): Menu =>
  ({
    id: 1,
    name: 'organization_employee',
    detail_open_mode: detailOpenMode,
    children: [],
  }) as unknown as Menu

describe('Organization detail navigation contract', () => {
  it('uses configured detail modes and falls back only for automatic metadata', () => {
    expect(
      resolveOrganizationDetailMode(
        [menu(SysDetailOpenMode.DIALOG)],
        'organization_employee',
        'page',
      ),
    ).toBe('dialog')
    expect(
      resolveOrganizationDetailMode(
        [menu(SysDetailOpenMode.PAGE)],
        'organization_employee',
        'dialog',
      ),
    ).toBe('page')
    expect(
      resolveOrganizationDetailMode(
        [menu(SysDetailOpenMode.AUTO)],
        'organization_employee',
        'page',
      ),
    ).toBe('page')
  })

  it('keeps record identity in distinct detail routes', () => {
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
    const first = router.resolve(buildOrganizationDetailRoute('org_sync_batch', 41, 'BATCH-41'))
    const second = router.resolve(buildOrganizationDetailRoute('org_sync_batch', 42, 'BATCH-42'))

    expect(first.path).toBe('/admin/detail/organization/org_sync_batch/41')
    expect(second.path).toBe('/admin/detail/organization/org_sync_batch/42')
    expect(first.fullPath).not.toBe(second.fullPath)
  })

  it('shows organization sync object codes as business labels', () => {
    expect(organizationSyncObjectLabel('management_company')).toBe('管理公司')
    expect(organizationSyncObjectLabel('management_unit')).toBe('管理组织')
    expect(organizationSyncObjectLabel('legal_unit')).toBe('法人组织')
    expect(organizationSyncObjectLabel('employee')).toBe('人员档案')
    expect(organizationSyncObjectLabel('unknown_kind')).toBe('unknown_kind')
  })
})
