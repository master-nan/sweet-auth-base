<template>
  <div class="organization-readonly-tree">
    <q-linear-progress v-if="loading" indeterminate color="primary" />

    <q-tree
      v-if="nodes.length"
      v-model:selected="selectedKey"
      v-model:expanded="expandedKeys"
      :nodes="nodes"
      node-key="id"
      label-key="name"
      children-key="children"
      selected-color="primary"
      no-selection-unset
      no-transition
      class="organization-readonly-tree__content"
    >
      <template #default-header="{ node }">
        <div
          class="organization-readonly-tree__node"
          :class="{ 'organization-readonly-tree__node--muted': node.muted }"
        >
          <span class="organization-readonly-tree__icon">
            <q-icon :name="node.icon || 'corporate_fare'" size="19px" />
          </span>
          <div class="organization-readonly-tree__identity">
            <div class="organization-readonly-tree__name">{{ node.name }}</div>
            <div class="organization-readonly-tree__meta">
              <span class="organization-readonly-tree__code">{{ node.code }}</span>
              <span v-if="node.typeLabel" class="organization-readonly-tree__separator">/</span>
              <span v-if="node.typeLabel">{{ node.typeLabel }}</span>
            </div>
          </div>
          <q-chip
            v-if="node.statusLabel"
            dense
            square
            outline
            :color="node.statusColor || 'grey-6'"
            class="organization-readonly-tree__status"
          >
            {{ node.statusLabel }}
          </q-chip>
        </div>
      </template>
    </q-tree>

    <div v-else-if="!loading" class="organization-readonly-tree__empty">
      <q-icon name="account_tree" size="42px" />
      <span>{{ emptyText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { OrganizationReadOnlyTreeNode } from './organization-read-only-tree'

const props = withDefaults(
  defineProps<{
    nodes: OrganizationReadOnlyTreeNode[]
    selectedId?: number | null
    loading?: boolean
    emptyText?: string
    expandAll?: boolean
  }>(),
  {
    selectedId: null,
    loading: false,
    emptyText: '暂无组织数据',
    expandAll: false,
  },
)

const emit = defineEmits<{
  select: [id: number]
}>()

const expandedKeys = ref<number[]>([])

const selectedKey = computed<number | null>({
  get: () => props.selectedId,
  set: (id) => {
    if (typeof id === 'number' && id > 0 && id !== props.selectedId) emit('select', id)
  },
})

watch(
  () => [props.nodes, props.expandAll] as const,
  ([nodes, expandAll]) => {
    const branchKeys = collectBranchKeys(nodes)
    if (expandAll) {
      expandedKeys.value = branchKeys
      return
    }

    const available = new Set(branchKeys)
    const retained = expandedKeys.value.filter((id) => available.has(id))
    expandedKeys.value = retained.length ? retained : rootBranchKeys(nodes)
  },
  { immediate: true, deep: true },
)

function collectBranchKeys(nodes: OrganizationReadOnlyTreeNode[]): number[] {
  return nodes.flatMap((node) => [
    ...(node.children?.length ? [node.id] : []),
    ...collectBranchKeys(node.children || []),
  ])
}

function rootBranchKeys(nodes: OrganizationReadOnlyTreeNode[]): number[] {
  return nodes.filter((node) => node.children?.length).map((node) => node.id)
}
</script>

<style scoped lang="scss">
.organization-readonly-tree {
  position: relative;
  min-height: 240px;
  height: 100%;
  overflow: auto;
  background: #fff;
}

.organization-readonly-tree__content {
  min-width: 340px;
  padding: 10px 12px 18px;
}

.organization-readonly-tree__node {
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) max-content;
  align-items: center;
  gap: 9px;
  padding: 6px 8px 6px 2px;
}

.organization-readonly-tree__node--muted {
  opacity: 0.68;
}

.organization-readonly-tree__icon {
  width: 24px;
  height: 28px;
  display: grid;
  place-items: center;
  color: #667085;
}

.organization-readonly-tree__identity {
  min-width: 0;
}

.organization-readonly-tree__name {
  overflow: hidden;
  color: #1b2940;
  font-size: 15px;
  font-weight: 650;
  line-height: 21px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organization-readonly-tree__meta {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  color: #7a879b;
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
}

.organization-readonly-tree__meta span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.organization-readonly-tree__code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 11px;
}

.organization-readonly-tree__separator {
  color: #b1bac8;
}

.organization-readonly-tree__status {
  margin: 0;
  background: transparent;
  font-size: 11px;
}

.organization-readonly-tree__empty {
  min-height: 240px;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #8792a6;
}

:deep(.q-tree__node-header) {
  min-height: 54px;
  flex-wrap: nowrap;
  padding: 0 4px;
  border-radius: 6px;
  transition:
    background-color 120ms ease,
    box-shadow 120ms ease;
}

:deep(.q-tree__node-header-content) {
  min-width: 0;
  flex: 1 1 auto;
}

:deep(.q-tree__arrow) {
  flex: 0 0 24px;
  margin-right: 2px;
  color: #7b8798;
}

:deep(.q-tree__children) {
  color: #c4cbd5;
}

:deep(.q-tree__node-header:hover) {
  background: #f7f9fc;
}

:deep(.q-tree__node--selected > .q-tree__node-header) {
  background: #eef5ff;
  box-shadow: inset 3px 0 0 var(--q-primary);
}

:deep(.q-tree__node--selected > .q-tree__node-header .organization-readonly-tree__icon) {
  color: var(--q-primary);
}

:global(.body--dark .organization-readonly-tree) {
  background: var(--app-dark-surface);
}

:global(.body--dark .organization-readonly-tree__name) {
  color: var(--app-dark-heading);
}

:global(.body--dark .organization-readonly-tree__meta),
:global(.body--dark .organization-readonly-tree__empty),
:global(.body--dark .organization-readonly-tree__icon),
:global(.body--dark .organization-readonly-tree .q-tree__arrow) {
  color: var(--app-dark-muted);
}

:global(.body--dark .organization-readonly-tree__separator) {
  color: var(--app-dark-border-strong);
}

:global(.body--dark .organization-readonly-tree .q-tree__children) {
  color: var(--app-dark-border-strong);
}

:global(.body--dark .organization-readonly-tree .q-tree__node-header:hover) {
  background: var(--app-dark-surface-soft);
}

:global(
  .body--dark
    .organization-readonly-tree
    .q-tree__node--selected
    > .q-tree__node-header
) {
  background: var(--app-dark-primary-soft);
}

:global(
  .body--dark
    .organization-readonly-tree
    .q-tree__node--selected
    > .q-tree__node-header
    .organization-readonly-tree__icon
) {
  color: #a9a1ff;
}
</style>
