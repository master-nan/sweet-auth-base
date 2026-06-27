import { SysMasterDetailMode } from 'src/types/enum'

export const MasterDetailDisplayMode = {
  SUMMARY: SysMasterDetailMode.SUMMARY,
  TABLE: SysMasterDetailMode.TABLE,
  STACKED: SysMasterDetailMode.STACKED,
} as const

export type MasterDetailDisplayMode =
  (typeof MasterDetailDisplayMode)[keyof typeof MasterDetailDisplayMode]

export type MasterDetailModePreference = SysMasterDetailMode | MasterDetailDisplayMode | null

export interface MasterDetailLayoutConfig {
  mode?: MasterDetailModePreference
  masterWidth?: string
  masterHeight?: string
  minWidth?: string
  minHeight?: string
}

export const masterDetailDisplayModeOptions = [
  {
    label: '摘要主表',
    value: MasterDetailDisplayMode.SUMMARY,
    maxFields: 5,
  },
  {
    label: '主表表格',
    value: MasterDetailDisplayMode.TABLE,
    maxFields: 12,
  },
  {
    label: '上下主子表',
    value: MasterDetailDisplayMode.STACKED,
    maxFields: Infinity,
  },
]

export function resolveMasterDetailDisplayMode(
  visibleMasterFieldCount: number,
  preferredMode?: MasterDetailModePreference,
): MasterDetailDisplayMode {
  if (
    preferredMode &&
    preferredMode !== SysMasterDetailMode.AUTO &&
    masterDetailDisplayModeOptions.some((option) => option.value === preferredMode)
  ) {
    return preferredMode
  }

  if (visibleMasterFieldCount > 12) return MasterDetailDisplayMode.STACKED
  if (visibleMasterFieldCount > 5) return MasterDetailDisplayMode.TABLE
  return MasterDetailDisplayMode.SUMMARY
}
