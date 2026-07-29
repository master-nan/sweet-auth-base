import { describe, expect, it } from 'vitest'
import type { Menu } from 'src/api/services/sys-menu'
import { SysDetailOpenMode } from 'src/types/enum'
import { resolveOrganizationDetailMode } from './organization-detail-mode'

const menu = (detailOpenMode?: SysDetailOpenMode): Menu =>
  ({
    id: 1,
    name: 'organization_employee',
    detail_open_mode: detailOpenMode,
    children: [],
  }) as unknown as Menu

describe('resolveOrganizationDetailMode', () => {
  it('uses the configured dialog or page mode', () => {
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
  })

  it('uses the page default when metadata is automatic', () => {
    expect(
      resolveOrganizationDetailMode(
        [menu(SysDetailOpenMode.AUTO)],
        'organization_employee',
        'page',
      ),
    ).toBe('page')
  })
})
