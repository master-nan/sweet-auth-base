<template>
  <div
    class="query-scheme-controls row items-center no-wrap q-gutter-xs"
    :class="`query-scheme-controls--${layout}`"
  >
    <query-scheme-selector
      :schemes="runtime.schemes.value"
      :current-label="runtime.currentLabel.value"
      :source="schemeSource"
      :loading="runtime.loading.value"
      :dirty="dirty"
      :load-error="runtime.error.value"
      @select="selectScheme"
      @restore-current="restoreCurrent"
      @reset-default="resetDefault"
      @retry="runtime.loadAvailable"
      @save-current="showSaveDialog = true"
      @manage="openManager"
    />
    <query-quick-presets :config="runtime.scope.config.value" @apply="applyPreset" />
    <slot name="quick-search" />
    <q-btn
      v-if="advancedEnabled"
      outline
      icon="tune"
      color="primary"
      class="query-scheme-controls__advanced"
      :aria-label="filterCountLabel"
      @click="openAdvancedQuery"
    >
      <q-badge v-if="showFilterCount && activeFilterCount" floating color="red">{{
        activeFilterCount
      }}</q-badge>
      <q-tooltip>{{ filterCountLabel }}</q-tooltip>
    </q-btn>
  </div>

  <advanced-query
    v-if="advancedEnabled"
    v-model="showAdvancedQuery"
    v-model:query-model="draftAdvanced"
    v-model:bindings="bindings"
    :fields="fields"
    :source-name="schemeSource?.name || ''"
    :dirty="dirty"
    :enable-nested="enableNested !== false"
    :title="displayAdvancedTitle"
    @search="applyAdvancedQuery"
  />

  <query-scheme-save-dialog
    v-model="showSaveDialog"
    :source="schemeSource"
    :loading="saving"
    @save="savePersonal"
  />
</template>

<script setup lang="ts" generic="TQuery extends Query">
import { useI18n } from 'vue-i18n'

import { computed, ref } from 'vue'
import type { TableField } from 'src/api/services/sys-table'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import QueryQuickPresets from 'src/components/QueryScheme/QueryQuickPresets.vue'
import QuerySchemeSaveDialog from 'src/components/QueryScheme/QuerySchemeSaveDialog.vue'
import QuerySchemeSelector from 'src/components/QueryScheme/QuerySchemeSelector.vue'
import type { QuerySchemePageController } from 'src/composables/query-scheme-page'
import type { TableQueryState } from 'src/composables/table-query-state'
import type { Query } from 'src/types/global'
import { countEffectiveQueryRules } from 'src/utils/query-state'

const { t } = useI18n({ useScope: 'global' })

type ControlsController<TQuery extends Query> = Pick<
  QuerySchemePageController<TQuery>,
  | 'runtime'
  | 'showSaveDialog'
  | 'saving'
  | 'runQueryChange'
  | 'selectScheme'
  | 'applyPreset'
  | 'restoreCurrent'
  | 'resetDefault'
  | 'openManager'
  | 'savePersonal'
>

type ControlsQueryState<TQuery extends Query> = Pick<
  TableQueryState<TQuery>,
  | 'draftAdvanced'
  | 'appliedAdvanced'
  | 'bindings'
  | 'schemeSource'
  | 'dirty'
  | 'beginAdvancedEdit'
  | 'applyAdvancedQuery'
>

const props = withDefaults(
  defineProps<{
    controller: ControlsController<TQuery>
    queryState: ControlsQueryState<TQuery>
    fields: TableField[]
    advancedTitle?: string
    advancedEnabled?: boolean
    enableNested?: boolean
    showFilterCount?: boolean
    layout?: 'standard' | 'compact'
  }>(),
  {
    advancedTitle: '',
    advancedEnabled: true,
    enableNested: true,
    showFilterCount: true,
    layout: 'standard',
  },
)

const displayAdvancedTitle = computed(() => props.advancedTitle || t('ui.advancedQuery'))

const showAdvancedQuery = ref(false)
const {
  runtime,
  showSaveDialog,
  saving,
  runQueryChange,
  selectScheme,
  applyPreset,
  restoreCurrent,
  resetDefault,
  openManager,
  savePersonal,
} = props.controller
const {
  draftAdvanced,
  appliedAdvanced,
  bindings,
  schemeSource,
  dirty,
  beginAdvancedEdit,
  applyAdvancedQuery: applyDraftAdvanced,
} = props.queryState
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvanced.value))
const filterCountLabel = computed(() =>
  props.showFilterCount && activeFilterCount.value
    ? t('ui.advancedQueryEnabled', { count: activeFilterCount.value })
    : t('ui.advancedQuery'),
)

const openAdvancedQuery = () => {
  beginAdvancedEdit()
  showAdvancedQuery.value = true
}

const applyAdvancedQuery = () => {
  runQueryChange(() => applyDraftAdvanced(draftAdvanced.value))
  showAdvancedQuery.value = false
}
</script>

<style scoped>
.query-scheme-controls {
  min-width: 0;
  padding-right: 4px;
}

.query-scheme-controls :deep(.query-scheme-selector) {
  max-width: 156px;
}

.query-scheme-controls :deep(.q-input) {
  width: 180px;
}

.query-scheme-controls--compact :deep(.query-scheme-selector) {
  max-width: 132px;
}

.query-scheme-controls--compact :deep(.q-input) {
  width: 150px;
}
</style>
