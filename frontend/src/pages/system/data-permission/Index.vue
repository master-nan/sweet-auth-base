<template>
  <base-content
    class="q-pa-sm data-permission-page"
    :class="{ 'data-permission-page--dark': isDarkMode }"
  >
    <div class="data-permission-shell">
      <section class="data-permission-hero">
        <q-avatar
          class="data-permission-hero__icon"
          color="primary"
          text-color="white"
          :icon="pageIcon"
          rounded
        />
        <div class="data-permission-hero__content">
          <div class="data-permission-hero__title">{{ t('ui.dataPermissions') }}</div>
          <div class="data-permission-hero__subtitle">
            {{
              t('ui.maintainResourcesAttributionStrategyAndDelegationOfAuthorityNotImplementing')
            }}
          </div>
        </div>
        <q-btn
          outline
          color="primary"
          icon="refresh"
          :label="t('ui.refresh')"
          :loading="activeLoading || preflightLoading"
          @click="refreshActiveTab"
        />
      </section>

      <div class="data-permission-layout">
        <aside
          class="data-permission-sidebar"
          :aria-label="t('ui.dataPermissionConfigurationModule')"
        >
          <button
            v-for="section in permissionSections"
            :key="section.name"
            type="button"
            class="data-permission-nav-item"
            :class="{ 'data-permission-nav-item--active': activeTab === section.name }"
            :data-tab="section.name"
            :aria-selected="activeTab === section.name"
            @click="activeTab = section.name"
          >
            <span class="data-permission-nav-item__icon">
              <q-icon :name="section.icon" />
            </span>
            <span class="data-permission-nav-item__body">
              <span class="data-permission-nav-item__title">{{ section.label }}</span>
              <span class="data-permission-nav-item__caption">{{ section.caption }}</span>
            </span>
          </button>
        </aside>

        <main class="data-permission-main">
          <header class="data-permission-section-head">
            <div>
              <div class="data-permission-section-head__title">{{ currentSection.label }}</div>
              <div class="data-permission-section-head__caption">{{ currentSection.caption }}</div>
            </div>
            <q-icon :name="currentSection.icon" />
          </header>

          <section
            v-for="tab in listTabs"
            v-show="activeTab === tab"
            :key="tab"
            class="data-permission-panel"
            :data-panel="tab"
          >
            <q-table
              class="fit sticky-header-table data-permission-table"
              color="primary"
              :dense="isCompactTable"
              :rows="rowsByTab[tab]"
              :columns="columnsByTab[tab]"
              row-key="id"
              flat
              bordered
              separator="cell"
              :dark="isDarkMode"
              :loading="loadingByTab[tab]"
              :pagination="{ rowsPerPage: 0 }"
              hide-pagination
              @row-click="(_, row) => openDetail(tab, row)"
            >
              <template #top>
                <div class="row q-gutter-xs full-width">
                  <div class="col-grow row q-gutter-xs">
                    <q-input
                      v-model="queries[tab].quick_query!.keyword"
                      dense
                      outlined
                      :dark="isDarkMode"
                      debounce="300"
                      :placeholder="t('ui.searchKeywords')"
                      @keyup.enter="searchTab(tab)"
                    >
                      <template #append>
                        <q-icon name="search" />
                      </template>
                    </q-input>
                    <q-btn
                      color="primary"
                      :label="t('ui.search')"
                      :disable="loadingByTab[tab]"
                      @click="searchTab(tab)"
                    />
                    <q-btn
                      outline
                      icon="tune"
                      color="primary"
                      class="q-ml-xs"
                      :aria-label="
                        filterCountForTab(tab) > 0
                          ? t('ui.advancedQueryEnabled', { count: filterCountForTab(tab) })
                          : t('ui.advancedQuery')
                      "
                      @click="openAdvancedQuery(tab)"
                    >
                      <q-badge v-if="filterCountForTab(tab) > 0" floating color="red">
                        {{ filterCountForTab(tab) }}
                      </q-badge>
                      <q-tooltip>
                        {{
                          filterCountForTab(tab) > 0
                            ? t('ui.advancedQueryEnabled', { count: filterCountForTab(tab) })
                            : t('ui.advancedQuery')
                        }}
                      </q-tooltip>
                    </q-btn>
                  </div>
                  <q-space />
                  <div class="row q-gutter-xs">
                    <q-btn
                      v-for="button in topButtonsForTab(tab)"
                      :key="button.id"
                      v-bind="menuButtonDisplayProps(button)"
                      :color="button.color || 'primary'"
                      :disable="loadingByTab[tab]"
                      @click="handleButtonClick(button)"
                    />
                  </div>
                </div>
              </template>

              <template #body-cell-permission_enabled="props">
                <q-td :props="props">
                  <q-badge :color="props.value ? 'positive' : 'grey-6'" outline>
                    {{ props.value ? t('ui.activatedStatus') : t('ui.notEnabled') }}
                  </q-badge>
                </q-td>
              </template>

              <template #body-cell-state="props">
                <q-td :props="props">
                  <q-badge :color="props.value ? 'positive' : 'grey-6'" outline>
                    {{ props.value ? t('ui.enabled') : t('ui.disabled') }}
                  </q-badge>
                </q-td>
              </template>

              <template #body-cell-actions="props">
                <q-td :props="props" class="q-gutter-xs">
                  <q-btn
                    v-for="button in lineButtonsForTab(tab)"
                    :key="button.id"
                    v-bind="lineButtonDisplayProps(button, props.row)"
                    flat
                    dense
                    size="sm"
                    @click.stop="handleButtonClick(button, props.row)"
                  >
                    <q-tooltip>{{ lineButtonLabel(button, props.row) }}</q-tooltip>
                  </q-btn>
                </q-td>
              </template>

              <template #bottom>
                <q-space />
                <table-pagination
                  :page="queries[tab].page"
                  :page-size="queries[tab].num"
                  :total="totals[tab]"
                  @update:page="setTabPage(tab, $event)"
                  @update:page-size="setTabPageSize(tab, $event)"
                />
              </template>
            </q-table>
          </section>

          <section
            v-show="activeTab === 'preflight'"
            class="data-permission-panel data-permission-preflight"
            data-panel="preflight"
          >
            <div class="row q-col-gutter-md items-end">
              <q-select
                v-model="preflightType"
                class="col-12 col-md-3"
                outlined
                dense
                :dark="isDarkMode"
                emit-value
                map-options
                :label="t('ui.checkObject')"
                :options="preflightTypeOptions"
                @update:model-value="resetPreflightTarget"
              />
              <q-select
                v-model="preflightId"
                class="col-12 col-md"
                outlined
                dense
                :dark="isDarkMode"
                emit-value
                map-options
                use-input
                input-debounce="200"
                :label="t('ui.selectConfiguration')"
                :options="preflightTargetOptions"
                @filter="filterPreflightOptions"
              />
              <div class="col-auto">
                <q-btn
                  v-for="button in preflightTopButtons"
                  :key="button.id"
                  v-bind="menuButtonDisplayProps(button)"
                  :color="button.color || 'primary'"
                  :disable="!preflightId || preflightLoading"
                  :loading="preflightLoading"
                  @click="handleButtonClick(button)"
                />
              </div>
            </div>

            <q-separator class="q-my-md" />

            <q-banner
              v-if="preflightResult"
              rounded
              :dark="isDarkMode"
              :class="preflightResult.valid ? 'bg-green-1 text-positive' : 'bg-red-1 text-negative'"
            >
              <template #avatar>
                <q-icon :name="preflightResult.valid ? 'task_alt' : 'error_outline'" />
              </template>
              {{
                preflightResult.valid
                  ? t('ui.configureCheckPassed')
                  : t('ui.configurationCheckFailed')
              }}
            </q-banner>

            <q-table
              class="col q-mt-md sticky-header-table data-permission-table"
              :rows="preflightResult?.errors || []"
              :columns="preflightColumns"
              row-key="code"
              flat
              bordered
              separator="cell"
              :dark="isDarkMode"
              hide-pagination
              :pagination="{ rowsPerPage: 0 }"
              :no-data-label="t('ui.selectObjectsAndPerformConfigurationChecks')"
            />
          </section>
        </main>
      </div>
    </div>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:query-model="tempAdvancedQuery"
      :fields="advancedQueryFields"
      :enable-nested="false"
      @search="applyAdvancedQuery"
    />

    <data-permission-config-dialog
      v-model="showConfigDialog"
      :kind="configDialogKind"
      :edit-data="currentEditData"
      @saved="handleSaved"
    />

    <data-permission-detail-dialog v-model="showDetailDialog" :kind="detailKind" :id="detailId" />
  </base-content>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'system_data_permission' })

import { computed, onMounted, reactive, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useRoute } from 'vue-router'
import cloneDeep from 'lodash/cloneDeep'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DataPermissionConfigDialog from './components/DataPermissionConfigDialog.vue'
import DataPermissionDetailDialog from './components/DataPermissionDetailDialog.vue'
import {
  type DataGrant,
  type DataOwnership,
  type DataPermissionConfigQuery,
  type DataPermissionDimension,
  type DataPolicy,
  type DataPolicyRule,
  type DataResource,
  type ValidationResult,
  useDataPermissionConfigApi,
} from 'src/api/services/data-permission-config'
import type { MenuButton } from 'src/api/services/sys-menu'
import type { TableField } from 'src/api/services/sys-table'
import type { Query } from 'src/types/global'
import { SysTableFieldInputType, SysTableFieldType } from 'src/types/enum'
import { usePageButtons } from 'src/composables/page-buttons'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { countEffectiveQueryRules } from 'src/utils/query-state'

const { t } = useI18n({ useScope: 'global' })

type ListTabName = 'resources' | 'ownerships' | 'policies' | 'grants'
type ActiveTabName = ListTabName | 'preflight'
type ConfigDialogKind = 'resource' | 'ownership' | 'policy' | 'grant'
type ConfigRow = DataResource | DataOwnership | DataPolicy | DataGrant
type PreflightType = 'resource' | 'policy' | 'grant'
interface LineButtonPresentation {
  label: string
  icon: string | undefined
  color: string
  disable?: boolean
}

const api = useDataPermissionConfigApi()
const $q = useQuasar()
const route = useRoute()
const isDarkMode = computed(() => Boolean($q?.dark?.isActive))
const isCompactTable = computed(() => Boolean($q?.screen?.lt?.md))
const pageIcon = computed(() => {
  const icon = route.meta.icon
  return typeof icon === 'string' && icon.trim() ? icon : 'rule'
})
const { confirmAction } = useConfirmDialog($q)
const { top_buttons: topButtons, line_buttons: lineButtons } =
  usePageButtons('system_data_permission')

const permissionSections: Array<{
  name: ActiveTabName
  label: string
  caption: string
  icon: string
}> = [
  {
    name: 'resources',
    get label() {
      return t('ui.dataResource')
    },
    get caption() {
      return t('ui.maintainingOperationalResourcesThatRequireDataAccess')
    },
    icon: 'dataset',
  },
  {
    name: 'ownerships',
    get label() {
      return t('ui.ownershipDefinition')
    },
    get caption() {
      return t('ui.defineTheDimensionsOfTheOperationalDataToDetermineAttribution')
    },
    icon: 'account_tree',
  },
  {
    name: 'policies',
    get label() {
      return t('ui.permissionPolicy')
    },
    get caption() {
      return t('ui.clusterAttributionScopeSourcesAndRelationshipRules')
    },
    icon: 'policy',
  },
  {
    name: 'grants',
    get label() {
      return t('ui.permissionGrant')
    },
    get caption() {
      return t('ui.authorizePermissionPolicyToRoleOrUser')
    },
    icon: 'verified_user',
  },
  {
    name: 'preflight',
    get label() {
      return t('ui.configureCheck')
    },
    get caption() {
      return t('ui.checkResourceStrategyAndAuthorizationConfigurationBeforeUsing')
    },
    icon: 'fact_check',
  },
]
const listTabs: ListTabName[] = ['resources', 'ownerships', 'policies', 'grants']
const activeTab = ref<ActiveTabName>('resources')
const currentSection = computed(() =>
  permissionSections.find((section) => section.name === activeTab.value)!,
)
const loadingByTab = reactive<Record<ListTabName, boolean>>({
  resources: false,
  ownerships: false,
  policies: false,
  grants: false,
})
const loadedTabs = reactive<Record<ListTabName, boolean>>({
  resources: false,
  ownerships: false,
  policies: false,
  grants: false,
})
const activeLoading = computed(() =>
  activeTab.value === 'preflight' ? false : loadingByTab[activeTab.value],
)
const showAdvancedQuery = ref(false)
const showConfigDialog = ref(false)
const configDialogKind = ref<ConfigDialogKind>('resource')
const currentEditData = ref<ConfigRow | null>(null)
const showDetailDialog = ref(false)
const detailKind = ref<ConfigDialogKind>('resource')
const detailId = ref(0)

const resources = ref<DataResource[]>([])
const ownerships = ref<DataOwnership[]>([])
const policies = ref<DataPolicy[]>([])
const grants = ref<DataGrant[]>([])
const dimensions = ref<DataPermissionDimension[]>([])
const policyRules = ref<DataPolicyRule[]>([])
const resourceLookup = ref<DataResource[]>([])
const policyLookup = ref<DataPolicy[]>([])
const preflightGrants = ref<DataGrant[]>([])
const preflightGrantsLoaded = ref(false)

const totals = ref<Record<ListTabName, number>>({
  resources: 0,
  ownerships: 0,
  policies: 0,
  grants: 0,
})

const newQuery = (): DataPermissionConfigQuery => ({
  page: 1,
  num: 20,
  order: { field: '', is_asc: false },
  expressions: [{ rules: [{ field: '', value: null }], nested: [] }],
  quick_query: { keyword: '' },
})
const queries = ref<Record<ListTabName, DataPermissionConfigQuery>>({
  resources: newQuery(),
  ownerships: newQuery(),
  policies: newQuery(),
  grants: newQuery(),
})
const advancedQueryTab = ref<ListTabName>('resources')
const tempAdvancedQuery = ref<Query>(cloneDeep(queries.value.resources))

const resourceTypeLabels: Record<string, string> = {
  get low_code_table() {
    return t('ui.lowCodeDataTable')
  },
  get business_service() {
    return t('ui.businessService')
  },
  get report() {
    return t('ui.report')
  },
}
const bindingTypeLabels: Record<string, string> = {
  get metadata_field() {
    return t('ui.metadataField')
  },
  get registered_field() {
    return t('ui.registeredField')
  },
}
const operationLabels: Record<string, string> = {
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
}

const resourceColumns: QTableProps['columns'] = [
  {
    name: 'resource_code',
    get label() {
      return t('ui.resourceCode')
    },
    field: 'resource_code',
    align: 'left',
  },
  {
    name: 'name',
    get label() {
      return t('ui.resourceName')
    },
    field: 'name',
    align: 'left',
  },
  {
    name: 'resource_type',
    get label() {
      return t('ui.resourceType')
    },
    field: (row) => resourceTypeLabels[row.resource_type] || row.resource_type,
    align: 'left',
  },
  {
    name: 'permission_enabled',
    get label() {
      return t('ui.dataPermissions')
    },
    field: 'permission_enabled',
    align: 'center',
  },
  {
    name: 'state',
    get label() {
      return t('ui.status')
    },
    field: 'state',
    align: 'center',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'actions',
    align: 'center',
  },
]

const ownershipColumns: QTableProps['columns'] = [
  {
    name: 'ownership_code',
    get label() {
      return t('ui.ownershipCode')
    },
    field: 'ownership_code',
    align: 'left',
  },
  {
    name: 'resource',
    get label() {
      return t('ui.dataResource')
    },
    field: (row) => resourceLabel(row.resource_id),
    align: 'left',
  },
  {
    name: 'dimension',
    get label() {
      return t('ui.dataDimension')
    },
    field: (row) => dimensionLabel(row.dimension_id),
    align: 'left',
  },
  {
    name: 'binding_type',
    get label() {
      return t('ui.bindingType')
    },
    field: (row) => bindingTypeLabels[row.binding_type] || row.binding_type,
    align: 'left',
  },
  {
    name: 'state',
    get label() {
      return t('ui.status')
    },
    field: 'state',
    align: 'center',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'actions',
    align: 'center',
  },
]

const policyColumns: QTableProps['columns'] = [
  {
    name: 'policy_code',
    get label() {
      return t('ui.policyCode')
    },
    field: 'policy_code',
    align: 'left',
  },
  {
    name: 'name',
    get label() {
      return t('ui.policyNameLabel')
    },
    field: 'name',
    align: 'left',
  },
  {
    name: 'rule_count',
    get label() {
      return t('ui.numberOfRules')
    },
    field: (row) => policyRules.value.filter((rule) => rule.policy_id === row.id).length,
    align: 'center',
  },
  {
    name: 'state',
    get label() {
      return t('ui.status')
    },
    field: 'state',
    align: 'center',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'actions',
    align: 'center',
  },
]

const grantColumns: QTableProps['columns'] = [
  {
    name: 'subject',
    get label() {
      return t('ui.authorisationSubject')
    },
    field: (row) =>
      row.subject?.name ||
      (row.subject_type === 'role'
        ? t('ui.roleDisabledOrNotAvailable')
        : t('ui.userDisabledOrNotAvailable')),
    align: 'left',
  },
  {
    name: 'resource',
    get label() {
      return t('ui.dataResource')
    },
    field: (row) => resourceLabel(row.resource_id),
    align: 'left',
  },
  {
    name: 'operation',
    get label() {
      return t('ui.resourceAction')
    },
    field: (row) => operationLabels[row.operation] || row.operation,
    align: 'left',
  },
  {
    name: 'policy',
    get label() {
      return t('ui.permissionPolicy')
    },
    field: (row) => policyLabel(row.policy_id),
    align: 'left',
  },
  {
    name: 'state',
    get label() {
      return t('ui.status')
    },
    field: 'state',
    align: 'center',
  },
  {
    name: 'actions',
    get label() {
      return t('ui.actions')
    },
    field: 'actions',
    align: 'center',
  },
]

const preflightColumns: QTableProps['columns'] = [
  {
    name: 'code',
    get label() {
      return t('ui.errorEncoding')
    },
    field: 'code',
    align: 'left',
  },
  {
    name: 'message',
    get label() {
      return t('ui.descriptionLabel')
    },
    field: 'message',
    align: 'left',
  },
  {
    name: 'object_type',
    get label() {
      return t('ui.objectType')
    },
    field: 'object_type',
    align: 'left',
  },
  {
    name: 'object_id',
    get label() {
      return t('ui.objectId')
    },
    field: 'object_id',
    align: 'left',
  },
]

const rowsByTab = computed<Record<ListTabName, ConfigRow[]>>(() => ({
  resources: resources.value,
  ownerships: ownerships.value,
  policies: policies.value,
  grants: grants.value,
}))
const columnsByTab: Record<ListTabName, QTableProps['columns']> = {
  resources: resourceColumns,
  ownerships: ownershipColumns,
  policies: policyColumns,
  grants: grantColumns,
}

const topActionByTab: Record<ActiveTabName, string[]> = {
  resources: ['create_resource'],
  ownerships: ['create_ownership'],
  policies: ['create_policy'],
  grants: ['create_grant'],
  preflight: ['preflight_resource', 'preflight_policy', 'preflight_grant'],
}
const lineActionByTab: Record<ListTabName, string[]> = {
  resources: ['update_resource', 'configure_operations', 'toggle_permission'],
  ownerships: ['update_ownership'],
  policies: ['update_policy', 'configure_rules', 'toggle_policy'],
  grants: ['toggle_grant'],
}
const preflightTopButtons = computed(() =>
  topButtons.value.filter((button) => {
    if (!topActionByTab.preflight.includes(button.event_action)) return false
    return button.event_action === `preflight_${preflightType.value}`
  }),
)
const topButtonsForTab = (tab: ListTabName) =>
  topButtons.value.filter((button) => topActionByTab[tab].includes(button.event_action))
const lineButtonsForTab = (tab: ListTabName) =>
  lineButtons.value.filter((button) => lineActionByTab[tab].includes(button.event_action))
const filterCountForTab = (tab: ListTabName) => countEffectiveQueryRules(queries.value[tab])

const lineButtonPresentation = (button: MenuButton, row: ConfigRow): LineButtonPresentation => {
  if (button.event_action === 'toggle_permission') {
    const enabled = Boolean((row as DataResource).permission_enabled)
    return {
      get label() {
        return enabled ? t('ui.disableDataPermission') : t('ui.enableDataPermission')
      },
      icon: enabled ? 'pause_circle' : 'play_circle',
      color: enabled ? 'warning' : 'positive',
    }
  }
  if (button.event_action === 'toggle_policy') {
    const enabled = row.state !== false
    return {
      get label() {
        return enabled ? t('ui.disablePermissionPolicy') : t('ui.enablePermissionPolicy')
      },
      icon: enabled ? 'pause_circle' : 'play_circle',
      color: enabled ? 'warning' : 'positive',
    }
  }
  if (button.event_action === 'toggle_grant') {
    const enabled = row.state !== false
    return {
      get label() {
        return enabled ? t('ui.disablePermissionGrant') : t('ui.enablePermissionGrant')
      },
      icon: enabled ? 'pause_circle' : 'play_circle',
      color: enabled ? 'warning' : 'positive',
    }
  }
  if (button.event_action === 'update_ownership') {
    const enabled = row.state !== false
    return {
      get label() {
        return enabled ? t('ui.disableOwnershipDefinition') : t('ui.ownershipDefinitionDisabled')
      },
      icon: enabled ? 'pause_circle' : 'block',
      color: enabled ? 'warning' : 'grey-6',
      disable: !enabled,
    }
  }
  return {
    label: button.name,
    icon: button.icon || undefined,
    color: button.color || 'primary',
    disable: false,
  }
}

const lineButtonDisplayProps = (button: MenuButton, row: ConfigRow) => {
  const presentation = lineButtonPresentation(button, row)
  return {
    ...menuButtonDisplayProps(button, {
      label: presentation.label,
      icon: presentation.icon,
    }),
    color: presentation.color,
    disable: presentation.disable || false,
  }
}

const lineButtonLabel = (button: MenuButton, row: ConfigRow) =>
  lineButtonPresentation(button, row).label

const field = (
  code: string,
  name: string,
  type: SysTableFieldType = SysTableFieldType.VARCHAR,
  inputType: SysTableFieldInputType = SysTableFieldInputType.INPUT,
): Partial<TableField> => ({
  id: 0,
  field_code: code,
  field_name: name,
  field_type: type,
  input_type: inputType,
  state: true,
})
const advancedFields: Record<ListTabName, Partial<TableField>[]> = {
  resources: [
    field('resource_code', t('ui.resourceCode')),
    field('name', t('ui.resourceName')),
    field('resource_type', t('ui.resourceType')),
    field('permission_enabled', t('ui.dataPermissions'), SysTableFieldType.BOOLEAN),
    field('state', t('ui.status'), SysTableFieldType.BOOLEAN),
  ],
  ownerships: [
    field('ownership_code', t('ui.ownershipCode')),
    field(
      'resource_id',
      t('ui.resourceId'),
      SysTableFieldType.BIGINT,
      SysTableFieldInputType.INPUT_NUMBER,
    ),
    field(
      'dimension_id',
      t('ui.dimensionId'),
      SysTableFieldType.BIGINT,
      SysTableFieldInputType.INPUT_NUMBER,
    ),
    field('binding_type', t('ui.bindingType')),
    field('state', t('ui.status'), SysTableFieldType.BOOLEAN),
  ],
  policies: [
    field('policy_code', t('ui.policyCode')),
    field('name', t('ui.policyNameLabel')),
    field('policy_type', t('ui.policyType')),
    field('state', t('ui.status'), SysTableFieldType.BOOLEAN),
  ],
  grants: [
    field('subject_type', t('ui.subjectType')),
    field(
      'subject_id',
      t('ui.subjectId'),
      SysTableFieldType.BIGINT,
      SysTableFieldInputType.INPUT_NUMBER,
    ),
    field(
      'resource_id',
      t('ui.resourceId'),
      SysTableFieldType.BIGINT,
      SysTableFieldInputType.INPUT_NUMBER,
    ),
    field('operation', t('ui.resourceAction')),
    field(
      'policy_id',
      t('ui.policyId'),
      SysTableFieldType.BIGINT,
      SysTableFieldInputType.INPUT_NUMBER,
    ),
    field('state', t('ui.status'), SysTableFieldType.BOOLEAN),
  ],
}
const advancedQueryFields = computed(() => advancedFields[advancedQueryTab.value])

const preflightType = ref<PreflightType>('resource')
const preflightId = ref<number | null>(null)
const preflightLoading = ref(false)
const preflightResult = ref<ValidationResult | null>(null)
const preflightTypeOptions = [
  {
    get label() {
      return t('ui.dataResource')
    },
    value: 'resource',
  },
  {
    get label() {
      return t('ui.permissionPolicy')
    },
    value: 'policy',
  },
  {
    get label() {
      return t('ui.permissionGrant')
    },
    value: 'grant',
  },
]
const preflightTargetOptions = ref<Array<{ label: string; value: number }>>([])

const resourceLabel = (id: number) => {
  const resource = resourceLookup.value.find((item) => item.id === id)
  return resource
    ? `${resource.resource_code} · ${resource.name}`
    : t('ui.dataResourcesNotAvailable')
}
const policyLabel = (id: number) => {
  const policy = policyLookup.value.find((item) => item.id === id)
  return policy ? `${policy.policy_code} · ${policy.name}` : t('ui.permissionPolicyNotAvailable')
}
const dimensionLabel = (id: number) => {
  const dimension = dimensions.value.find((item) => item.id === id)
  return dimension
    ? `${dimension.dimension_code} · ${dimension.name}`
    : t('ui.dataDimensionUnavailable')
}

const fetchTab = async (tab: ListTabName) => {
  loadingByTab[tab] = true
  try {
    const query = queries.value[tab]
    if (tab === 'resources') {
      const result = await api.queryResources(query)
      resources.value = result.data || []
      totals.value.resources = result.total || 0
    } else if (tab === 'ownerships') {
      const result = await api.queryOwnerships(query)
      ownerships.value = result.data || []
      totals.value.ownerships = result.total || 0
    } else if (tab === 'policies') {
      const [policyResult, ruleResult] = await Promise.all([
        api.queryPolicies(query),
        api.queryPolicyRules({ ...newQuery(), num: 500 }),
      ])
      policies.value = policyResult.data || []
      policyRules.value = ruleResult.data || []
      totals.value.policies = policyResult.total || 0
    } else {
      const result = await api.queryGrants(query)
      grants.value = result.data || []
      totals.value.grants = result.total || 0
    }
    loadedTabs[tab] = true
  } finally {
    loadingByTab[tab] = false
  }
}

const loadLookups = async () => {
  const lookupQuery = { ...newQuery(), num: 500 }
  const [resourceResult, policyResult, dimensionResult] = await Promise.all([
    api.queryResources(lookupQuery),
    api.queryPolicies(lookupQuery),
    api.queryDimensions(lookupQuery),
  ])
  resourceLookup.value = resourceResult.data || []
  policyLookup.value = policyResult.data || []
  dimensions.value = dimensionResult.data || []
}

const loadPreflightGrants = async (force = false) => {
  if (preflightGrantsLoaded.value && !force) return
  const result = await api.queryGrants({ ...newQuery(), num: 500 })
  preflightGrants.value = result.data || []
  preflightGrantsLoaded.value = true
}

const searchTab = (tab: ListTabName) => {
  queries.value[tab].expressions = cloneDeep(newQuery().expressions)
  queries.value[tab].page = 1
  void fetchTab(tab)
}
const refreshActiveTab = () => {
  if (activeTab.value === 'preflight') {
    void Promise.all([loadLookups(), loadPreflightGrants(true)]).then(resetPreflightTarget)
    return
  }
  void Promise.all([fetchTab(activeTab.value), loadLookups()])
}
const setTabPage = (tab: ListTabName, page: number) => {
  queries.value[tab].page = page
  void fetchTab(tab)
}
const setTabPageSize = (tab: ListTabName, pageSize: number) => {
  queries.value[tab].num = pageSize
  queries.value[tab].page = 1
  void fetchTab(tab)
}
const openAdvancedQuery = (tab: ListTabName) => {
  advancedQueryTab.value = tab
  tempAdvancedQuery.value = cloneDeep(queries.value[tab])
  showAdvancedQuery.value = true
}
const applyAdvancedQuery = () => {
  const tab = advancedQueryTab.value
  queries.value[tab].expressions = cloneDeep(tempAdvancedQuery.value.expressions)
  queries.value[tab].page = 1
  showAdvancedQuery.value = false
  void fetchTab(tab)
}

const openConfig = (kind: ConfigDialogKind, row: ConfigRow | null = null) => {
  configDialogKind.value = kind
  currentEditData.value = row
  showConfigDialog.value = true
}
const openDetail = (tab: ListTabName, row: ConfigRow) => {
  detailKind.value = tab.slice(0, -1) as ConfigDialogKind
  detailId.value = row.id
  showDetailDialog.value = true
}
const handleSaved = () => {
  const tabByKind: Record<ConfigDialogKind, ListTabName> = {
    resource: 'resources',
    ownership: 'ownerships',
    policy: 'policies',
    grant: 'grants',
  }
  const tab = tabByKind[configDialogKind.value]
  if (tab === 'grants') preflightGrantsLoaded.value = false
  void Promise.all([fetchTab(tab), loadLookups()])
}

const toggleResource = (row: DataResource) => {
  const enabled = !row.permission_enabled
  confirmAction({
    get title() {
      return enabled ? t('ui.enableDataPermission') : t('ui.disableDataPermission')
    },
    get message() {
      return enabled
        ? t('ui.beforeEnabledCompleteConfigurationChecksWillBePerformedToConfirmDataPermissions', {
            value1: row.name,
          })
        : t('ui.confirmDataPermissionToDisable', { value1: row.name })
    },
  }).onOk(() => {
    void (async () => {
      const result = await api.setResourcePermission(row.id, enabled)
      if (!result.data.valid) {
        $q.notify({
          type: 'negative',
          message: result.data.errors.map((item) => item.message).join('；'),
        })
        return
      }
      $q.notify({
        type: 'positive',
        get message() {
          return enabled ? t('ui.dataPermissionsEnabled') : t('ui.dataPermissionsDisabled')
        },
      })
      await fetchTab('resources')
    })()
  })
}
const toggleOwnership = (row: DataOwnership) => {
  if (!row.state) return
  confirmAction({
    get title() {
      return t('ui.disableOwnershipDefinition')
    },
    get message() {
      return t('ui.confirmThatIsDisabled', { value1: row.ownership_code })
    },
  }).onOk(() => {
    void (async () => {
      await api.disableOwnership(row.id)
      $q.notify({
        type: 'positive',
        get message() {
          return t('ui.ownershipDefinitionDisabled')
        },
      })
      await fetchTab('ownerships')
    })()
  })
}
const togglePolicy = (row: DataPolicy) => {
  const enabled = !row.state
  confirmAction({
    get title() {
      return enabled ? t('ui.enablePermissionPolicy') : t('ui.disablePermissionPolicy')
    },
    get message() {
      return t('ui.confirmNamedAction', {
        value1: enabled ? t('ui.enabled') : t('ui.disabled'),
        value2: row.name,
      })
    },
  }).onOk(() => {
    void (async () => {
      const result = await api.setPolicyState(row.id, enabled)
      if (!result.data.valid) {
        $q.notify({
          type: 'negative',
          message: result.data.errors.map((item) => item.message).join('；'),
        })
        return
      }
      $q.notify({
        type: 'positive',
        get message() {
          return t('ui.permissionPolicyActionResult', {
            value1: enabled ? t('ui.enabled') : t('ui.disabled'),
          })
        },
      })
      await fetchTab('policies')
    })()
  })
}
const toggleGrant = (row: DataGrant) => {
  const enabled = !row.state
  confirmAction({
    get title() {
      return enabled ? t('ui.enablePermissionGrant') : t('ui.disablePermissionGrant')
    },
    get message() {
      return t('ui.confirmSCurrentAuthorization', {
        value1: enabled ? t('ui.enabled') : t('ui.disabled'),
      })
    },
  }).onOk(() => {
    void (async () => {
      const result = await api.setGrantState(row.id, enabled)
      if (!result.data.valid) {
        $q.notify({
          type: 'negative',
          message: result.data.errors.map((item) => item.message).join('；'),
        })
        return
      }
      $q.notify({
        type: 'positive',
        get message() {
          return t('ui.permissionGrantActionResult', {
            value1: enabled ? t('ui.enabled') : t('ui.disabled'),
          })
        },
      })
      await fetchTab('grants')
    })()
  })
}

const actionHandlers: Record<string, (row?: ConfigRow) => void> = {
  create_resource: () => openConfig('resource'),
  update_resource: (row) => row && openConfig('resource', row),
  configure_operations: (row) => row && openConfig('resource', row),
  toggle_permission: (row) => row && void toggleResource(row as DataResource),
  create_ownership: () => openConfig('ownership'),
  update_ownership: (row) => row && void toggleOwnership(row as DataOwnership),
  create_policy: () => openConfig('policy'),
  update_policy: (row) => row && openConfig('policy', row),
  configure_rules: (row) => row && openConfig('policy', row),
  toggle_policy: (row) => row && void togglePolicy(row as DataPolicy),
  create_grant: () => openConfig('grant'),
  toggle_grant: (row) => row && void toggleGrant(row as DataGrant),
  preflight_resource: () => void runPreflight(),
  preflight_policy: () => void runPreflight(),
  preflight_grant: () => void runPreflight(),
}
const handleButtonClick = (button: MenuButton, row?: ConfigRow) => {
  actionHandlers[button.event_action]?.(row)
}

const targetOptions = (keyword = '') => {
  const normalized = keyword.toLowerCase()
  if (preflightType.value === 'resource') {
    return resourceLookup.value
      .filter((item) => `${item.resource_code} ${item.name}`.toLowerCase().includes(normalized))
      .map((item) => ({ label: `${item.resource_code} · ${item.name}`, value: item.id }))
  }
  if (preflightType.value === 'policy') {
    return policyLookup.value
      .filter((item) => `${item.policy_code} ${item.name}`.toLowerCase().includes(normalized))
      .map((item) => ({ label: `${item.policy_code} · ${item.name}`, value: item.id }))
  }
  return preflightGrants.value
    .filter((item) =>
      `${item.subject?.name || ''} ${item.subject?.code || ''} ${resourceLabel(item.resource_id)}`
        .toLowerCase()
        .includes(normalized),
    )
    .map((item) => ({
      get label() {
        return t('ui.subjectResourceSummary', {
          value1:
            item.subject?.name ||
            (item.subject_type === 'role' ? t('ui.roleUnavailable') : t('ui.userUnavailable')),
          value2: resourceLabel(item.resource_id),
        })
      },
      value: item.id,
    }))
}
const resetPreflightTarget = () => {
  preflightId.value = null
  preflightResult.value = null
  preflightTargetOptions.value = targetOptions()
}
const filterPreflightOptions = (value: string, update: (callback: () => void) => void) =>
  update(() => {
    preflightTargetOptions.value = targetOptions(value)
  })
const runPreflight = async () => {
  if (!preflightId.value) return
  preflightLoading.value = true
  try {
    const result = await api.preflight(preflightType.value, preflightId.value)
    preflightResult.value = result.data
  } finally {
    preflightLoading.value = false
  }
}

watch(activeTab, async (tab) => {
  if (tab === 'preflight') {
    await loadPreflightGrants()
    preflightTargetOptions.value = targetOptions()
    return
  }
  if (!loadedTabs[tab]) await fetchTab(tab)
})

onMounted(async () => {
  await Promise.all([loadLookups(), fetchTab('resources')])
})
</script>

<style scoped lang="scss">
.data-permission-page {
  --data-permission-border: var(--app-primary-border);
  --data-permission-ink: #1f2a44;
  --data-permission-muted: #6f7d95;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f7f9fc;
}

.data-permission-shell {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
}

.data-permission-hero {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 14px;
  min-height: 72px;
  padding: 12px 16px;
  border: 1px solid var(--data-permission-border);
  border-radius: 8px;
  background: linear-gradient(135deg, var(--app-primary-soft-strong), transparent 72%), #fff;
}

.data-permission-hero__icon {
  width: 48px;
  height: 48px;
  flex: 0 0 auto;
  border-radius: 8px;
  box-shadow: 0 10px 22px var(--app-primary-shadow);
}

.data-permission-hero__content {
  min-width: 0;
  flex: 1;
}

.data-permission-hero__title {
  color: var(--data-permission-ink);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.25;
}

.data-permission-hero__subtitle {
  margin-top: 2px;
  color: var(--data-permission-muted);
  font-size: 13px;
}

.data-permission-layout {
  display: grid;
  flex: 1 1 auto;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 10px;
  min-height: 0;
  margin-top: 10px;
  overflow: hidden;
}

.data-permission-sidebar,
.data-permission-main {
  border: 1px solid var(--data-permission-border);
  border-radius: 8px;
  background: #fff;
}

.data-permission-sidebar {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-height: 0;
  overflow: auto;
  padding: 8px;
}

.data-permission-nav-item {
  display: grid;
  width: 100%;
  min-height: 64px;
  grid-template-columns: 40px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 8px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--data-permission-ink);
  cursor: pointer;
  text-align: left;
  transition:
    border-color 0.18s ease,
    background 0.18s ease,
    box-shadow 0.18s ease;
}

.data-permission-nav-item:hover,
.data-permission-nav-item--active {
  border-color: var(--app-primary-border);
  background: var(--app-primary-soft);
}

.data-permission-nav-item--active {
  box-shadow: inset 3px 0 0 var(--q-primary);
}

.data-permission-nav-item__icon {
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  border-radius: 8px;
  background: var(--app-primary-soft-strong);
  color: var(--q-primary);
}

.data-permission-nav-item__icon .q-icon {
  font-size: 20px;
}

.data-permission-nav-item__body {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.data-permission-nav-item__title,
.data-permission-nav-item__caption {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.data-permission-nav-item__title {
  color: var(--data-permission-ink);
  font-size: 14px;
  font-weight: 700;
}

.data-permission-nav-item__caption {
  color: var(--data-permission-muted);
  font-size: 12px;
}

.data-permission-main {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.data-permission-section-head {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 70px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--data-permission-border);
}

.data-permission-section-head__title {
  color: var(--data-permission-ink);
  font-size: 18px;
  font-weight: 700;
}

.data-permission-section-head__caption {
  margin-top: 2px;
  color: var(--data-permission-muted);
  font-size: 13px;
}

.data-permission-section-head > .q-icon {
  color: var(--q-primary);
  font-size: 26px;
  opacity: 0.58;
}

.data-permission-panel {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.data-permission-table {
  height: 100%;
}

.data-permission-preflight {
  padding: 14px;
}

.data-permission-page--dark {
  --data-permission-border: var(--app-dark-border);
  --data-permission-ink: var(--app-dark-heading);
  --data-permission-muted: var(--app-dark-muted);
  background: var(--app-dark-page);
}

.data-permission-page--dark .data-permission-hero,
.data-permission-page--dark .data-permission-sidebar,
.data-permission-page--dark .data-permission-main {
  background: var(--app-dark-surface);
}

.data-permission-page--dark .data-permission-hero {
  background:
    linear-gradient(135deg, var(--app-primary-soft-strong), transparent 72%),
    var(--app-dark-surface);
}

.data-permission-page--dark .data-permission-nav-item:hover,
.data-permission-page--dark .data-permission-nav-item--active {
  background: var(--app-primary-soft-strong);
}

.data-permission-page--dark .data-permission-preflight {
  color: var(--app-dark-text);
}

@media (max-width: 960px) {
  .data-permission-layout {
    grid-template-columns: 1fr;
  }

  .data-permission-sidebar {
    display: grid;
    grid-template-columns: repeat(5, minmax(138px, 1fr));
    overflow-x: auto;
  }

  .data-permission-nav-item {
    min-height: 58px;
  }
}

@media (max-width: 640px) {
  .data-permission-hero {
    align-items: stretch;
    flex-wrap: wrap;
  }

  .data-permission-hero__content {
    align-self: center;
  }

  .data-permission-hero > .q-btn {
    width: 100%;
  }

  .data-permission-section-head {
    min-height: 64px;
  }
}
</style>
