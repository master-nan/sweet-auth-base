<template>
  <q-btn-dropdown
    class="query-scheme-selector"
    flat
    dense
    color="primary"
    icon="bookmark_border"
    :label="currentLabel"
    :title="currentLabel"
    :loading="loading"
    :disable="disabled"
    content-class="query-scheme-selector-menu"
    no-caps
  >
    <q-list dense class="query-scheme-selector-list">
      <template v-for="group in groupedSchemes" :key="group.type">
        <q-item-label header class="query-scheme-selector-group-label">
          {{ group.label }}
        </q-item-label>
        <q-item
          v-for="scheme in group.items"
          :key="scheme.id"
          class="query-scheme-selector-item"
          :active="source?.id === scheme.id"
          active-class="query-scheme-selector-item--active"
          clickable
          v-close-popup
          @click="requestSelect(scheme)"
        >
          <q-item-section>
            <q-item-label>{{ scheme.name }}</q-item-label>
          </q-item-section>
          <q-item-section side class="row items-center no-wrap q-gutter-xs">
            <q-icon v-if="scheme.is_default" name="star" color="amber-7" size="16px">
              <q-tooltip>默认方案</q-tooltip>
            </q-icon>
            <q-icon
              v-if="scheme.status !== QuerySchemeValidationStatus.VALID"
              name="warning_amber"
              size="17px"
              :color="
                scheme.status === QuerySchemeValidationStatus.INVALID ? 'negative' : 'warning'
              "
            >
              <q-tooltip>{{
                scheme.status === QuerySchemeValidationStatus.INVALID
                  ? '方案不可用'
                  : '方案需要修复'
              }}</q-tooltip>
            </q-icon>
          </q-item-section>
        </q-item>
      </template>
      <q-item
        v-if="loadError && !loading"
        class="query-scheme-selector-item"
        clickable
        @click="$emit('retry')"
      >
        <q-item-section avatar
          ><q-icon name="error_outline" color="negative" size="20px"
        /></q-item-section>
        <q-item-section>
          <q-item-label class="text-negative">查询方案加载失败</q-item-label>
          <q-item-label caption>点击重试</q-item-label>
        </q-item-section>
      </q-item>
      <q-item v-else-if="!schemes.length && !loading" class="query-scheme-selector-empty">
        <q-item-section class="text-grey-7">
          暂无已保存方案，可保存当前查询条件以便下次使用
        </q-item-section>
      </q-item>
      <q-separator />
      <q-item
        class="query-scheme-selector-action"
        clickable
        v-close-popup
        @click="$emit('save-current')"
      >
        <q-item-section avatar
          ><q-icon name="bookmark_add" color="primary" size="20px"
        /></q-item-section>
        <q-item-section>{{ saveActionLabel }}</q-item-section>
      </q-item>
      <q-item
        v-if="dirty"
        class="query-scheme-selector-action"
        clickable
        v-close-popup
        @click="$emit('restore-current')"
      >
        <q-item-section avatar><q-icon name="undo" color="primary" size="20px" /></q-item-section>
        <q-item-section>撤销当前方案修改</q-item-section>
      </q-item>
      <q-item
        class="query-scheme-selector-action"
        clickable
        v-close-popup
        @click="$emit('reset-default')"
      >
        <q-item-section avatar
          ><q-icon name="restart_alt" color="primary" size="20px"
        /></q-item-section>
        <q-item-section>恢复默认查询</q-item-section>
      </q-item>
      <q-item class="query-scheme-selector-action" clickable v-close-popup @click="$emit('manage')">
        <q-item-section avatar
          ><q-icon name="settings" color="primary" size="20px"
        /></q-item-section>
        <q-item-section>管理查询方案</q-item-section>
      </q-item>
    </q-list>
  </q-btn-dropdown>
  <q-dialog v-model="confirmVisible">
    <q-card style="width: 420px; max-width: 100%">
      <q-card-section class="text-h6">切换查询方案</q-card-section>
      <q-card-section>当前方案有未保存修改，切换后这些修改将丢失。</q-card-section>
      <q-card-actions align="right">
        <q-btn v-close-popup flat label="取消" />
        <q-btn color="primary" label="继续切换" @click="confirmSelect" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  QUERY_SCHEME_TYPE_LABELS,
  QuerySchemeType,
  QuerySchemeValidationStatus,
  type QuerySchemeSource,
  type QuerySchemeSummary,
} from 'src/modules/query-scheme/types'

const props = withDefaults(
  defineProps<{
    schemes: QuerySchemeSummary[]
    currentLabel?: string
    loading?: boolean
    disabled?: boolean
    dirty?: boolean
    loadError?: string
    source?: QuerySchemeSource | null
  }>(),
  {
    currentLabel: '查询方案',
    loading: false,
    disabled: false,
    dirty: false,
    loadError: '',
  },
)

const emit = defineEmits<{
  select: [scheme: QuerySchemeSummary]
  manage: []
  'restore-current': []
  'reset-default': []
  'save-current': []
  retry: []
}>()

const confirmVisible = ref(false)
const pendingSelection = ref<QuerySchemeSummary | null>(null)
const requestSelect = (scheme: QuerySchemeSummary) => {
  if (!props.dirty) {
    emit('select', scheme)
    return
  }
  pendingSelection.value = scheme
  confirmVisible.value = true
}
const confirmSelect = () => {
  if (pendingSelection.value) emit('select', pendingSelection.value)
  pendingSelection.value = null
  confirmVisible.value = false
}

const order = [
  QuerySchemeType.PERSONAL,
  QuerySchemeType.PUBLIC,
  QuerySchemeType.ROLE,
  QuerySchemeType.PAGE_DEFAULT,
]
const groupedSchemes = computed(() =>
  order
    .map((type) => ({
      type,
      label: QUERY_SCHEME_TYPE_LABELS[type],
      items: props.schemes.filter((scheme) => scheme.type === type),
    }))
    .filter((group) => group.items.length > 0),
)
const saveActionLabel = computed(() => {
  if (props.source?.type === QuerySchemeType.PERSONAL && props.dirty) {
    return '保存当前方案修改'
  }
  if (props.source) return '另存为我的方案'
  return '保存当前查询为方案'
})
</script>

<style scoped>
.query-scheme-selector {
  min-width: 0;
  max-width: 240px;
  overflow: hidden;
}

.query-scheme-selector :deep(.q-btn__content) {
  min-width: 0;
  flex-wrap: nowrap;
}

.query-scheme-selector :deep(.q-btn__content .block) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(.query-scheme-selector-menu) {
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface);
  box-shadow: 0 8px 24px rgba(31, 42, 68, 0.14);
}

:global(.query-scheme-selector-list) {
  width: 300px;
  max-width: calc(100vw - 24px);
  padding: 6px 0;
}

:global(.query-scheme-selector-list .q-item__section--avatar) {
  min-width: 34px;
}

:global(.query-scheme-selector-group-label) {
  min-height: 0;
  padding: 8px 14px 4px;
  color: var(--app-text-muted);
  font-size: 12px;
  line-height: 18px;
}

:global(.query-scheme-selector-item),
:global(.query-scheme-selector-action) {
  min-height: 38px;
  padding: 6px 14px;
  color: var(--app-text-strong);
}

:global(.query-scheme-selector-item--active) {
  color: var(--q-primary);
  background: var(--app-primary-soft);
}

:global(.query-scheme-selector-empty) {
  min-height: 54px;
  padding: 8px 14px;
  color: var(--app-text-muted);
  line-height: 20px;
}
</style>
