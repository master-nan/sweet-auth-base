<template>
  <form-dialog-shell
    v-model="visible"
    :title="title"
    :subtitle="subtitle"
    :icon="icon"
    readonly
    :show-preview="false"
  >
    <div class="q-pa-lg">
      <q-inner-loading :showing="loading">
        <q-spinner color="primary" size="40px" />
      </q-inner-loading>

      <template v-if="detail">
        <q-list bordered separator>
          <template v-if="kind === 'resource'">
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.resourceCode') }}</q-item-label>
                <q-item-label>{{ resourceDetail.resource_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.resourceName') }}</q-item-label>
                <q-item-label>{{ resourceDetail.name }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.resourceType') }}</q-item-label>
                <q-item-label>{{ resourceTypeLabel(resourceDetail.resource_type) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.dataPermissions') }}</q-item-label>
                <q-item-label>
                  <q-badge
                    :color="resourceDetail.permission_enabled ? 'positive' : 'grey-6'"
                    outline
                  >
                    {{
                      resourceDetail.permission_enabled ? t('ui.activatedStatus') : t('ui.notEnabled')
                    }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.supportingOperations') }}</q-item-label>
                <q-item-label>
                  <q-chip
                    v-for="operation in resourceOperations"
                    :key="operation.id"
                    dense
                    square
                    color="primary"
                    text-color="white"
                  >
                    {{ operationLabel(operation.operation) }}
                  </q-chip>
                  <span v-if="resourceOperations.length === 0">-</span>
                </q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.ownershipDefinition') }}</q-item-label>
                <q-item-label>{{ resourceOwnerships.length }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.associationPolicy') }}</q-item-label>
                <q-item-label>{{ resourcePolicyCount }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.numberOfAuthorizations') }}</q-item-label>
                <q-item-label>{{ resourceGrants.length }}</q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else-if="kind === 'ownership'">
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.ownershipCode') }}</q-item-label>
                <q-item-label>{{ ownershipDetail.ownership_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.dataResource') }}</q-item-label>
                <q-item-label>{{ ownershipDetail.resource?.name || '-' }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.dataDimension') }}</q-item-label>
                <q-item-label>{{ ownershipDetail.dimension?.name || '-' }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.bindingType') }}</q-item-label>
                <q-item-label>{{ bindingTypeLabel(ownershipDetail.binding_type) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.valueType') }}</q-item-label>
                <q-item-label>{{ valueTypeLabel(ownershipDetail.value_type) }}</q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else-if="kind === 'policy'">
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.policyCode') }}</q-item-label>
                <q-item-label>{{ policyDetail.policy_code }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.policyNameLabel') }}</q-item-label>
                <q-item-label>{{ policyDetail.name }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.numberOfRules') }}</q-item-label>
                <q-item-label>{{ policyDetail.rules?.length || 0 }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.status') }}</q-item-label>
                <q-item-label>
                  <q-badge :color="policyDetail.state ? 'positive' : 'grey-6'" outline>
                    {{ policyDetail.state ? t('ui.enabled') : t('ui.disabled') }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
          </template>

          <template v-else>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.authorisationSubject') }}</q-item-label>
                <q-item-label>{{ grantSubjectLabel }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.dataResource') }}</q-item-label>
                <q-item-label>{{
                  grantDetail.resource?.name || t('ui.dataResourcesNotAvailable')
                }}</q-item-label>
              </q-item-section>
            </q-item>
            <q-item>
              <q-item-section>
                <q-item-label caption>{{ t('ui.resourceAction') }}</q-item-label>
                <q-item-label>{{ operationLabel(grantDetail.operation) }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.permissionPolicy') }}</q-item-label>
                <q-item-label>{{
                  grantDetail.policy?.name || t('ui.permissionPolicyNotAvailable')
                }}</q-item-label>
              </q-item-section>
              <q-item-section>
                <q-item-label caption>{{ t('ui.status') }}</q-item-label>
                <q-item-label>
                  <q-badge :color="grantDetail.state ? 'positive' : 'grey-6'" outline>
                    {{ grantDetail.state ? t('ui.enabled') : t('ui.disabled') }}
                  </q-badge>
                </q-item-label>
              </q-item-section>
            </q-item>
          </template>
        </q-list>

        <div v-if="kind === 'policy'" class="text-subtitle1 text-weight-medium q-mt-lg q-mb-sm">
          {{ t('ui.policyRules') }}
        </div>
        <q-table
          v-if="kind === 'policy'"
          :rows="policyDetail.rules || []"
          :columns="ruleColumns"
          row-key="id"
          flat
          bordered
          separator="cell"
          hide-pagination
          :pagination="{ rowsPerPage: 0 }"
          :no-data-label="t('ui.thereAreNoStrategicRules')"
        />
      </template>
    </div>
  </form-dialog-shell>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import type { QTableProps } from 'quasar'
import FormDialogShell from '@/components/FormDialog/FormDialogShell.vue'
import {
  type DataGrant,
  type DataOwnership,
  type DataPolicy,
  type DataResource,
  type DataResourceOperationItem,
  useDataPermissionConfigApi,
} from '@/api/services/data-permission-config'
import type { Query } from '@/types/global'

const { t } = useI18n({ useScope: 'global' })

type DetailKind = 'resource' | 'ownership' | 'policy' | 'grant'

const props = defineProps<{
  modelValue: boolean
  kind: DetailKind
  id: number
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
}>()

const api = useDataPermissionConfigApi()
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
const loading = ref(false)
const detail = ref<DataResource | DataOwnership | DataPolicy | DataGrant | null>(null)
const resourceOperations = ref<DataResourceOperationItem[]>([])
const resourceOwnerships = ref<DataOwnership[]>([])
const resourceGrants = ref<DataGrant[]>([])

const resourceDetail = computed(() => detail.value as DataResource)
const ownershipDetail = computed(() => detail.value as DataOwnership)
const policyDetail = computed(() => detail.value as DataPolicy)
const grantDetail = computed(() => detail.value as DataGrant)
const grantSubjectLabel = computed(() => {
  const subject = grantDetail.value.subject
  if (!subject)
    return grantDetail.value.subject_type === 'role'
      ? t('ui.roleUnavailable')
      : t('ui.userUnavailable')
  return subject.code ? `${subject.name} · ${subject.code}` : subject.name
})
const resourcePolicyCount = computed(
  () => new Set(resourceGrants.value.map((grant) => grant.policy_id)).size,
)

const title = computed(
  () =>
    ({
      get resource() {
        return t('ui.dataResourcesDetails')
      },
      get ownership() {
        return t('ui.detailsOfAttributionDefinition')
      },
      get policy() {
        return t('ui.permissionPolicyDetails')
      },
      get grant() {
        return t('ui.detailsOfAuthority')
      },
    })[props.kind],
)
const subtitle = computed(() => {
  if (!detail.value) return t('ui.readingConfiguration')
  if (props.kind === 'resource') return resourceDetail.value.resource_code
  if (props.kind === 'ownership') return ownershipDetail.value.ownership_code
  if (props.kind === 'policy') return policyDetail.value.policy_code
  return grantSubjectLabel.value
})
const icon = computed(
  () =>
    ({
      resource: 'dataset',
      ownership: 'account_tree',
      policy: 'policy',
      grant: 'verified_user',
    })[props.kind],
)

const ruleColumns: QTableProps['columns'] = [
  {
    name: 'sequence',
    get label() {
      return t('ui.order')
    },
    field: 'sequence',
    align: 'left',
  },
  {
    name: 'ownership_code',
    get label() {
      return t('ui.ownershipCode')
    },
    field: 'ownership_code',
    align: 'left',
  },
  {
    name: 'dimension',
    get label() {
      return t('ui.dataDimension')
    },
    field: (row) => row.dimension?.name || t('ui.dataDimensionUnavailable'),
    align: 'left',
  },
  {
    name: 'scope_source',
    get label() {
      return t('ui.scopeSource')
    },
    field: (row) => scopeSourceLabel(row.scope_source),
    align: 'left',
  },
  {
    name: 'relation',
    get label() {
      return t('ui.relationLabel')
    },
    field: (row) => relationLabel(row.relation),
    align: 'left',
  },
  {
    name: 'operator',
    get label() {
      return t('ui.operator')
    },
    field: (row) => operatorLabel(row.operator),
    align: 'left',
  },
]

const baseQuery = (resourceId: number): Query & { resource_id: number } => ({
  page: 1,
  num: 500,
  order: { field: '', is_asc: true },
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword: '' },
  resource_id: resourceId,
})

const loadDetail = async () => {
  if (!props.id) return
  loading.value = true
  try {
    resourceOperations.value = []
    resourceOwnerships.value = []
    resourceGrants.value = []
    if (props.kind === 'resource') {
      const [resource, operations, ownerships, grants] = await Promise.all([
        api.getResource(props.id),
        api.listResourceOperations(props.id),
        api.listResourceOwnerships(props.id),
        api.queryGrants(baseQuery(props.id)),
      ])
      detail.value = resource.data
      resourceOperations.value = operations.data || []
      resourceOwnerships.value = ownerships.data || []
      resourceGrants.value = grants.data || []
    } else if (props.kind === 'ownership') {
      detail.value = (await api.getOwnership(props.id)).data
    } else if (props.kind === 'policy') {
      detail.value = (await api.getPolicy(props.id)).data
    } else {
      detail.value = (await api.getGrant(props.id)).data
    }
  } finally {
    loading.value = false
  }
}

const resourceTypeLabel = (value: string) =>
  ({
    get low_code_table() {
      return t('ui.lowCodeDataTable')
    },
    get business_service() {
      return t('ui.businessService')
    },
    get report() {
      return t('ui.report')
    },
  })[value] || value
const operationLabel = (value: string) =>
  ({
    get query() {
      return t('ui.query')
    },
    get detail() {
      return t('ui.details')
    },
    get create() {
      return t('ui.create')
    },
    get update() {
      return t('ui.modify')
    },
    get delete() {
      return t('ui.delete')
    },
    get export() {
      return t('ui.export')
    },
    get run() {
      return t('ui.run')
    },
  })[value] || value
const bindingTypeLabel = (value: string) =>
  ({
    get metadata_field() {
      return t('ui.metadataField')
    },
    get registered_field() {
      return t('ui.registeredField')
    },
  })[value] || value
const valueTypeLabel = (value: string) =>
  ({
    get bigint() {
      return t('ui.numericId')
    },
    get string() {
      return t('ui.stringCode')
    },
  })[value] || value
const scopeSourceLabel = (value: string) =>
  ({
    get effective_legal_entities() {
      return t('ui.currentValidLegalEntity')
    },
    get effective_org_units() {
      return t('ui.currentValidOrganization')
    },
    get current_employee() {
      return t('ui.currentEmployee')
    },
    get specified_values() {
      return t('ui.specifiedValue')
    },
  })[value] || value
const relationLabel = (value: string) =>
  ({
    get exact() {
      return t('ui.exactMatch')
    },
    get self_and_descendants() {
      return t('ui.currentLevelAndBelow')
    },
  })[value] || value
const operatorLabel = (value: string) =>
  ({
    get eq() {
      return t('ui.equals')
    },
    get in() {
      return t('ui.containedIn')
    },
  })[value] || value

watch(
  () => [props.modelValue, props.kind, props.id],
  ([open]) => {
    if (open) void loadDetail()
  },
)
</script>
