export interface DetailFieldItem {
  label: string
  value?: string | number | boolean | null
  meta?: string
  chip?: boolean
  color?: string
  textColor?: string
  outline?: boolean
  fullWidth?: boolean
}

export interface DetailSection {
  key: string
  label: string
  caption?: string
  icon?: string
  count?: number
  items?: DetailFieldItem[]
}

export interface DetailSectionNavigationItem {
  key: string
  label: string
  caption?: string
  icon?: string
  count?: number
}
