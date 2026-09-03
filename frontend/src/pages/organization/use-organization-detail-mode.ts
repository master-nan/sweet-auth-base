import { computed, type ComputedRef } from 'vue'
import { useUserStore } from '@/stores/user'
import {
  resolveOrganizationDetailMode,
  type OrganizationDetailMode,
} from './organization-detail-mode'

export const useOrganizationDetailMode = (
  routeName: string,
  autoMode: OrganizationDetailMode,
): ComputedRef<OrganizationDetailMode> => {
  const userStore = useUserStore()
  return computed(() => resolveOrganizationDetailMode(userStore.menus, routeName, autoMode))
}
