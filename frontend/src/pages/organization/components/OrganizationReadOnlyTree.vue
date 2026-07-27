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
          <q-icon :name="node.icon || 'apartment'" size="19px" color="primary" />
          <div class="organization-readonly-tree__identity">
            <div class="organization-readonly-tree__name">{{ node.name }}</div>
            <div class="organization-readonly-tree__meta">
              <span>{{ node.code }}</span>
              <span v-if="node.typeLabel">{{ node.typeLabel }}</span>
            </div>
          </div>
          <q-chip
            v-if="node.statusLabel"
            dense
            square
            :color="node.statusColor || 'grey-6'"
            text-color="white"
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
  min-width: 360px;
  padding: 8px 10px 16px;
}

.organization-readonly-tree__node {
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr) max-content;
  align-items: center;
  gap: 8px;
  padding: 5px 8px 5px 2px;
}

.organization-readonly-tree__node--muted {
  opacity: 0.68;
}

.organization-readonly-tree__identity {
  min-width: 0;
}

.organization-readonly-tree__name {
  overflow: hidden;
  color: #1d2738;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organization-readonly-tree__meta {
  min-width: 0;
  display: flex;
  gap: 8px;
  overflow: hidden;
  color: #718096;
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
}

.organization-readonly-tree__meta span {
  overflow: hidden;
  text-overflow: ellipsis;
}

.organization-readonly-tree__status {
  margin: 0;
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
  min-height: 48px;
  flex-wrap: nowrap;
  padding: 0 4px;
  border-radius: 6px;
}

:deep(.q-tree__node-header-content) {
  min-width: 0;
  flex: 1 1 auto;
}

:deep(.q-tree__arrow) {
  flex: 0 0 24px;
  margin-right: 2px;
}

:deep(.q-tree__node--selected > .q-tree__node-header) {
  background: #eef5ff;
}
</style>
