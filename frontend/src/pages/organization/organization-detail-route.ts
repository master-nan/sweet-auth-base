import type { RouteLocationRaw } from 'vue-router'

export const buildOrganizationDetailRoute = (
  tableCode: string,
  recordId: number,
  itemLabel: string,
): RouteLocationRaw => ({
  name: 'record_detail',
  params: {
    source: 'organization',
    table_code: tableCode,
    id: recordId,
  },
  query: {
    item_label: itemLabel,
  },
})
