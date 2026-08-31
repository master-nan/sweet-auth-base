<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      flat
      bordered
      separator="cell"
      row-key="id"
      :rows="rows"
      :columns="columns"
      :loading="loading"
      hide-pagination
    >
      <template #top>
        <div class="full-width">
          <q-tabs
            v-model="activeType"
            dense
            align="left"
            active-color="primary"
            indicator-color="primary"
            @update:model-value="reloadFirstPage"
          >
            <q-tab :name="QuerySchemeType.PERSONAL" :label="t('ui.myPlan')" />
            <q-tab :name="QuerySchemeType.PUBLIC" :label="t('ui.publicSchemes')" />
            <q-tab :name="QuerySchemeType.ROLE" :label="t('ui.roleProgram')" />
            <q-tab :name="QuerySchemeType.PAGE_DEFAULT" :label="t('ui.pageDefault')" />
          </q-tabs>
          <q-separator class="q-mb-sm" />
          <standard-table-toolbar :refreshing="loading" @refresh="fetchData">
            <template #quick-search>
              <q-input
                :model-value="nameFilter"
                dense
                outlined
                clearable
                debounce="300"
                :label="t('ui.schemeName')"
                @update:model-value="updateNameFilter"
                @keyup.enter="reloadFirstPage"
              />
              <q-select
                v-model="scopeFilter"
                dense
                outlined
                clearable
                emit-value
                map-options
                :options="scopeOptions"
                :label="t('ui.page')"
                style="min-width: 190px"
                @update:model-value="reloadFirstPage"
              />
              <q-btn color="primary" :label="t('ui.query')" @click="reloadFirstPage" />
            </template>
            <template #right-actions>
              <q-btn
                v-if="canManageShared && activeType !== QuerySchemeType.PERSONAL"
                color="primary"
                icon="add"
                :label="t('ui.newScheme')"
                @click="openCreate"
              />
            </template>
          </standard-table-toolbar>
        </div>
      </template>

      <template #body-cell-name="props">
        <q-td :props="props">
          <div class="query-scheme-name text-weight-medium">
            {{ props.row.name }}
            <q-tooltip>{{ props.row.name }}</q-tooltip>
          </div>
          <div class="query-scheme-scope text-caption text-grey-7">
            {{ scopeLabel(props.row.scope_label) }}
          </div>
        </q-td>
      </template>
      <template #body-cell-type="props"
        ><q-td :props="props"><status-chip color="primary" :label="typeLabel(props.row)" /></q-td
      ></template>
      <template #body-cell-is_default="props"
        ><q-td :props="props"
          ><q-icon v-if="props.row.is_default" name="star" color="amber-7" size="sm"
            ><q-tooltip>{{ t('ui.defaultScheme') }}</q-tooltip></q-icon
          ><span v-else>-</span></q-td
        ></template
      >
      <template #body-cell-status="props"
        ><q-td :props="props"
          ><status-chip
            :label="statusLabel(props.row.status)"
            :color="statusColor(props.row.status)" /></q-td
      ></template>
      <template #body-cell-enabled="props"
        ><q-td :props="props"
          ><status-chip
            :label="props.row.enabled ? t('ui.activatedStatus') : t('ui.deactivatedStatus')"
            :color="props.row.enabled ? 'positive' : 'grey'" /></q-td
      ></template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="no-wrap">
          <div class="query-scheme-row-actions">
            <span
              v-for="action in inlineSchemeActions(props.row)"
              :key="action.key"
              class="query-scheme-row-action"
            >
              <q-btn
                flat
                round
                dense
                size="sm"
                :icon="action.icon"
                :color="action.color"
                :aria-label="action.label"
                :disable="action.disabled"
                @click="action.handler"
              />
              <q-tooltip>{{ action.tooltip }}</q-tooltip>
            </span>
            <q-btn
              v-if="overflowSchemeActions(props.row).length"
              class="query-scheme-row-more"
              flat
              round
              dense
              size="sm"
              icon="more_horiz"
              color="primary"
              :aria-label="t('ui.moreProgramOperations')"
            >
              <q-tooltip>{{ t('ui.moreOperations') }}</q-tooltip>
              <q-menu auto-close>
                <q-list dense style="min-width: 180px">
                  <template
                    v-for="(action, index) in overflowSchemeActions(props.row)"
                    :key="action.key"
                  >
                    <q-separator v-if="action.destructive && index > 0" />
                    <q-item
                      clickable
                      :class="action.destructive ? 'text-negative' : ''"
                      @click="action.handler"
                    >
                      <q-item-section avatar>
                        <q-icon :name="action.icon" :color="action.color" />
                      </q-item-section>
                      <q-item-section>{{ action.label }}</q-item-section>
                    </q-item>
                  </template>
                </q-list>
              </q-menu>
            </q-btn>
          </div>
        </q-td>
      </template>
      <template #no-data>
        <div class="full-width column flex-center q-gutter-sm q-pa-xl text-grey-7">
          <q-icon :name="error ? 'cloud_off' : 'inbox'" color="grey-5" size="48px" />
          <div>{{ emptyMessage }}</div>
          <q-btn
            v-if="error"
            outline
            color="primary"
            icon="refresh"
            :label="t('ui.retry')"
            @click="fetchData"
          />
        </div>
      </template>
      <template #bottom
        ><q-space /><table-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="total"
      /></template>
    </q-table>

    <query-scheme-detail-drawer
      v-model="showDetail"
      :scheme-id="selectedId"
      :editable="!!selected && canEdit(selected)"
      @edit="editDetail"
      @copy="copyScheme"
    />
    <query-scheme-edit-dialog
      v-model="showEdit"
      :scheme-type="activeType"
      :detail="editValue"
      :scope-options="scopeOptions"
      @saved="fetchData"
    />
  </base-content>
</template>

<script setup lang="ts">
defineOptions({ name: 'query_scheme_manager' })
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import StandardTableToolbar from 'src/components/Table/StandardTableToolbar.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import StatusChip from 'src/components/Display/StatusChip.vue'
import QuerySchemeDetailDrawer from './QuerySchemeDetailDrawer.vue'
import QuerySchemeEditDialog from './QuerySchemeEditDialog.vue'
import { useQuerySchemeApi } from 'src/api/services/query-scheme'
import { collectQueryScopes } from 'src/composables/query-scope'
import { QUERY_SCHEME_NAVIGATION_STATE_KEY } from 'src/composables/query-scheme-page'
import { useConfirmDialog } from 'src/composables/confirm-dialog'
import { useUserStore } from 'src/stores/user'
import { hasGrantedActionCapability } from 'src/utils/menu-button'
import type { TableColumn } from 'src/types/global'
import {
  QUERY_SCHEME_TYPE_LABELS,
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeDetail,
  type QuerySchemeListItem,
  type QuerySchemeType as SchemeType,
  type QuerySchemeValidationStatus as ValidationStatus,
} from 'src/modules/query-scheme/types'
import { notifyQuerySchemeDeleted } from 'src/modules/query-scheme/events'
import {
  buildQuerySchemeCopyName,
  isValidQuerySchemeName,
  normalizeQuerySchemeName,
  truncateQuerySchemeName,
} from 'src/modules/query-scheme/name'

const $q = useQuasar()
const api = useQuerySchemeApi()
const router = useRouter()
const { t } = useI18n()
const userStore = useUserStore()
const { confirmDanger, confirmAction } = useConfirmDialog($q)
const canManageShared = computed(() =>
  hasGrantedActionCapability(userStore.menus, 'query_scheme_shared_manage'),
)
const activeType = ref<SchemeType>(QuerySchemeType.PERSONAL)
const nameFilter = ref('')
const scopeFilter = ref<string | null>(null)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<QuerySchemeListItem[]>([])
const loading = ref(false)
const error = ref('')
const showDetail = ref(false)
const showEdit = ref(false)
const selected = ref<QuerySchemeListItem | null>(null)
const selectedId = computed(() => selected.value?.id || 0)
const editValue = ref<QuerySchemeDetail | null>(null)
const scopeLabel = (label: string) => t(label)
const scopeOptions = computed(() =>
  collectQueryScopes(userStore.menus).map((scope) => ({
    label: scopeLabel(scope.scope_label),
    value: scope.scope_code,
  })),
)
const scopeMenus = computed(() => collectQueryScopes(userStore.menus))
const columns: TableColumn<QuerySchemeListItem>[] = [
  {
    name: 'name',
    field: 'name',
    get label() {
      return t('ui.schemeName')
    },
    align: 'left',
  },
  {
    name: 'type',
    field: 'type',
    get label() {
      return t('ui.type')
    },
    align: 'center',
  },
  {
    name: 'is_default',
    field: 'is_default',
    get label() {
      return t('ui.default')
    },
    align: 'center',
  },
  {
    name: 'status',
    field: 'status',
    get label() {
      return t('ui.verifyStatus')
    },
    align: 'center',
  },
  {
    name: 'enabled',
    field: 'enabled',
    get label() {
      return t('ui.enableStatus')
    },
    align: 'center',
  },
  {
    name: 'creator',
    field: 'creator_display_name',
    get label() {
      return t('ui.createdBy')
    },
    align: 'left',
  },
  {
    name: 'updated_at',
    field: 'updated_at',
    get label() {
      return t('ui.updatedAt')
    },
    align: 'left',
  },
  {
    name: 'actions',
    field: 'actions',
    get label() {
      return t('ui.actions')
    },
    align: 'center',
  },
]

const statusLabel = (status: ValidationStatus) =>
  status === QuerySchemeValidationStatus.VALID
    ? t('ui.available')
    : status === QuerySchemeValidationStatus.DEGRADED
      ? t('ui.needsAttention')
      : t('ui.unavailable')
const statusColor = (status: ValidationStatus) =>
  status === QuerySchemeValidationStatus.VALID
    ? 'positive'
    : status === QuerySchemeValidationStatus.DEGRADED
      ? 'warning'
      : 'negative'
const canEdit = (row: QuerySchemeListItem) =>
  row.type === QuerySchemeType.PERSONAL || canManageShared.value
const typeLabel = (row: QuerySchemeListItem) => QUERY_SCHEME_TYPE_LABELS[row.type]
const canUseScheme = (row: QuerySchemeListItem) =>
  row.enabled && row.status === QuerySchemeValidationStatus.VALID
const useSchemeTooltip = (row: QuerySchemeListItem) => {
  if (!row.enabled) return t('ui.programDisabledNotAvailable')
  if (row.status === QuerySchemeValidationStatus.DEGRADED)
    return t('ui.theProgrammeNeedsToBeRepairedBeforeItCanBe')
  if (row.status === QuerySchemeValidationStatus.INVALID)
    return t('ui.programValidationFailedNotAvailable')
  return t('ui.enterThePageAndUseTheScheme')
}
type SchemeRowAction = {
  key: string
  label: string
  tooltip: string
  icon: string
  color: string
  disabled?: boolean
  destructive?: boolean
  handler: () => void
}
const schemeRowActions = (row: QuerySchemeListItem): SchemeRowAction[] => {
  const actions: SchemeRowAction[] = [
    {
      key: 'use',
      get label() {
        return t('ui.useProgram')
      },
      tooltip: useSchemeTooltip(row),
      icon: 'open_in_new',
      color: 'primary',
      disabled: !canUseScheme(row),
      handler: () => useScheme(row),
    },
  ]
  if (canEdit(row)) {
    actions.push({
      key: 'edit',
      get label() {
        return t('ui.editScheme')
      },
      get tooltip() {
        return t('ui.editScheme')
      },
      icon: 'edit',
      color: 'primary',
      handler: () => void openEdit(row),
    })
  }
  actions.push({
    key: 'detail',
    get label() {
      return t('ui.viewSchemeDetails')
    },
    get tooltip() {
      return t('ui.viewSchemeDetails')
    },
    icon: 'visibility',
    color: 'primary',
    handler: () => openDetail(row),
  })
  if (row.type === QuerySchemeType.PERSONAL) {
    actions.push({
      key: 'default',
      get label() {
        return row.is_default ? t('ui.cancelDefault') : t('ui.setAsDefault')
      },
      get tooltip() {
        return row.is_default ? t('ui.removeDefaultScheme') : t('ui.setAsDefaultScheme')
      },
      icon: row.is_default ? 'star_border' : 'star',
      color: 'amber-8',
      handler: () => setPersonalDefault(row),
    })
  } else {
    actions.push({
      key: 'copy',
      get label() {
        return t('ui.copyToMySchemes')
      },
      get tooltip() {
        return t('ui.copyToMySchemes')
      },
      icon: 'content_copy',
      color: 'primary',
      handler: () => copyScheme(row),
    })
    if (canManageShared.value) {
      actions.push({
        key: 'toggle',
        get label() {
          return row.enabled ? t('ui.disableScheme') : t('ui.enableScheme')
        },
        get tooltip() {
          return row.enabled ? t('ui.disableScheme') : t('ui.enableScheme')
        },
        icon: row.enabled ? 'toggle_off' : 'toggle_on',
        color: row.enabled ? 'warning' : 'positive',
        handler: () => toggleEnabled(row),
      })
    }
  }
  if (canEdit(row)) {
    actions.push({
      key: 'delete',
      get label() {
        return t('ui.deleteScheme')
      },
      get tooltip() {
        return t('ui.deleteScheme')
      },
      icon: 'delete',
      color: 'negative',
      destructive: true,
      handler: () => deleteScheme(row),
    })
  }
  return actions
}
const inlineSchemeActions = (row: QuerySchemeListItem) => schemeRowActions(row).slice(0, 3)
const overflowSchemeActions = (row: QuerySchemeListItem) => schemeRowActions(row).slice(3)
const updateNameFilter = (value: string | number | null) => {
  nameFilter.value = truncateQuerySchemeName(String(value ?? ''))
}
const hasFilters = computed(() => !!nameFilter.value.trim() || !!scopeFilter.value)
const emptyMessage = computed(() => {
  if (error.value) return t('ui.failedToLoadQuerySchemeTryAgain')
  if (hasFilters.value) return t('ui.noSchemeMeetingCurrentQueryConditions')
  return t('ui.currentClassificationHasNoQueryOption')
})

const fetchData = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.list({
      page: page.value,
      num: pageSize.value,
      scheme_type: activeType.value,
      ...(nameFilter.value.trim() ? { name: nameFilter.value.trim() } : {}),
      ...(scopeFilter.value ? { scope_code: scopeFilter.value } : {}),
    })
    rows.value = response.data || []
    total.value = response.total || 0
  } catch {
    rows.value = []
    total.value = 0
    error.value = t('ui.failedToLoadQueryScheme')
  } finally {
    loading.value = false
  }
}
const reloadFirstPage = () => {
  if (page.value !== 1) page.value = 1
  else void fetchData()
}
const openDetail = (row: QuerySchemeListItem) => {
  selected.value = row
  showDetail.value = true
}
const useScheme = (row: QuerySchemeListItem) => {
  if (!canUseScheme(row)) return
  const target = scopeMenus.value.find((scope) => scope.scope_code === row.scope_code)
  if (!target) {
    $q.notify({
      type: 'warning',
      get message() {
        return t('ui.currentAccountCannotAccessThePageOfTheProgram')
      },
    })
    return
  }
  void router.push({
    name: target.route_name,
    state: { [QUERY_SCHEME_NAVIGATION_STATE_KEY]: String(row.id) },
  })
}
const openCreate = () => {
  editValue.value = null
  showEdit.value = true
}
const openEdit = async (row: QuerySchemeListItem) => {
  editValue.value = (await api.detail(row.id)).data || null
  if (editValue.value) showEdit.value = true
}
const editDetail = (detail: QuerySchemeDetail) => {
  editValue.value = detail
  showDetail.value = false
  showEdit.value = true
}
const copyScheme = (row: QuerySchemeListItem | QuerySchemeDetail) => {
  $q.dialog({
    get title() {
      return t('ui.copyToMySchemes')
    },
    get message() {
      return t('ui.pleaseEnterANewProjectName')
    },
    prompt: {
      model: buildQuerySchemeCopyName(row.name),
      type: 'text',
      get hint() {
        return t('ui.maximum64Characters')
      },
      isValid: isValidQuerySchemeName,
    },
    cancel: true,
    persistent: true,
  }).onOk((name: string) => {
    if (!isValidQuerySchemeName(name)) return
    void api.copyToPersonal(row.id, row.scope_code, normalizeQuerySchemeName(name)).then(fetchData)
  })
}
const setPersonalDefault = (row: QuerySchemeListItem) =>
  confirmAction({
    get title() {
      return row.is_default ? t('ui.removeDefaultScheme') : t('ui.setDefaultScheme')
    },
    get message() {
      return t('ui.confirmingAsAPersonalDefaultScheme', {
        value1: row.is_default ? t('ui.cancel') : t('ui.set'),
        value2: row.name,
      })
    },
  }).onOk(() => {
    void api.setPersonalDefault(row.id, !row.is_default, row.revision).then(fetchData)
  })
const toggleEnabled = (row: QuerySchemeListItem) =>
  confirmAction({
    get title() {
      return row.enabled ? t('ui.disableScheme') : t('ui.enableScheme')
    },
    get message() {
      return t('ui.confirmNamedAction', {
        value1: row.enabled ? t('ui.disabled') : t('ui.enabled'),
        value2: row.name,
      })
    },
  }).onOk(() => {
    void api.setSharedEnabled(row.id, !row.enabled, row.revision).then(fetchData)
  })
const deleteScheme = (row: QuerySchemeListItem) =>
  confirmDanger({
    get title() {
      return t('ui.deleteQueryScheme')
    },
    get message() {
      return t('ui.couldNotResumeAfterDeletingToConfirmContinuation', { value1: row.name })
    },
  }).onOk(() => {
    const request =
      row.type === QuerySchemeType.PERSONAL
        ? api.deletePersonal(row.id, row.revision)
        : api.deleteShared(row.id, row.revision)
    void request.then(() => {
      notifyQuerySchemeDeleted(row.id)
      return fetchData()
    })
  })

watch(
  () => [page.value, pageSize.value],
  () => void fetchData(),
)
onMounted(fetchData)
</script>

<style scoped>
.query-scheme-name,
.query-scheme-scope {
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.query-scheme-row-action {
  display: inline-flex;
  align-items: center;
}

.query-scheme-row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  vertical-align: middle;
}

.query-scheme-row-more {
  align-self: center;
}
</style>
