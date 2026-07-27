export interface OrganizationReadOnlyTreeNode {
  id: number
  code: string
  name: string
  icon?: string
  typeLabel?: string
  statusLabel?: string
  statusColor?: string
  muted?: boolean
  children?: OrganizationReadOnlyTreeNode[]
}
