export interface OrganizationDetailField {
  key: string
  label: string
  value: string
  kind?: 'text' | 'code' | 'status'
  color?: string
  wide?: boolean
}

export interface OrganizationDetailGroup {
  key: string
  title: string
  icon?: string
  fields: OrganizationDetailField[]
}
