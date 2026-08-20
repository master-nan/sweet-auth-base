<template>
  <div class="row items-center q-gutter-xs">
    <query-scheme-selector
      :schemes="runtime.schemes.value"
      :current-label="runtime.currentLabel.value"
      :loading="runtime.loading.value"
      :dirty="dirty"
      :load-error="runtime.error.value"
      @select="selectScheme"
      @restore-current="restoreCurrent"
      @reset-default="resetDefault"
      @retry="runtime.loadAvailable"
      @manage="openManager"
    />
    <q-separator vertical inset />
    <query-quick-presets :config="runtime.scope.config.value" @apply="applyPreset" />
    <slot name="quick-search" />
    <q-btn
      v-if="advancedEnabled"
      outline
      icon="tune"
      color="primary"
      :aria-label="filterCountLabel"
      @click="openAdvancedQuery"
    >
      <q-badge v-if="showFilterCount && activeFilterCount" floating color="red">{{
        activeFilterCount
      }}</q-badge>
      <q-tooltip>{{ filterCountLabel }}</q-tooltip>
    </q-btn>
    <q-btn
      outline
      color="primary"
      icon="bookmark_add"
      label="保存方案"
      @click="showSaveDialog = true"
    />
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
    :title="advancedTitle || '高级查询'"
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
  }>(),
  {
    advancedTitle: '高级查询',
    advancedEnabled: true,
    enableNested: true,
    showFilterCount: true,
  },
)

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
    ? `高级查询，已启用 ${activeFilterCount.value} 个条件`
    : '高级查询',
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
