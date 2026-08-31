<template>
  <div class="organization-readonly-tree">
    <q-linear-progress v-if="loading" indeterminate color="primary" />

    <q-tree
      v-if="nodes.length"
      v-model:selected="selectedKey"
      v-model:expanded="expandedKeys"
      :nodes="nodes"
      :dark="Dark.isActive"
      node-key="id"
      label-key="name"
      children-key="children"
      selected-color="primary"
      no-selection-unset
      no-transition
      class="organization-readonly-tree__content q-pa-sm"
    >
      <template #default-header="{ node }">
        <div
          class="organization-readonly-tree__node row items-center no-wrap full-width q-py-xs q-pr-sm"
          :class="{ 'text-grey-6': node.muted }"
        >
          <q-icon :name="node.icon || 'corporate_fare'" size="20px" class="text-grey-6 q-mr-sm" />
          <div class="organization-readonly-tree__identity col">
            <div class="organization-readonly-tree__name text-body2 text-weight-bold ellipsis">
              {{ node.name }}
            </div>
            <div
              class="organization-readonly-tree__meta row items-center no-wrap text-caption text-grey-7"
            >
              <span class="ellipsis">{{ node.code }}</span>
              <span v-if="node.typeLabel" class="q-mx-xs">/</span>
              <span v-if="node.typeLabel" class="ellipsis">{{ node.typeLabel }}</span>
            </div>
          </div>
          <q-chip
            v-if="node.statusLabel"
            dense
            square
            outline
            :color="node.statusColor || 'grey-6'"
            class="q-ml-sm"
          >
            {{ node.statusLabel }}
          </q-chip>
        </div>
      </template>
    </q-tree>

    <div v-else-if="!loading" class="absolute-full column flex-center q-gutter-sm text-grey-6">
      <q-icon name="account_tree" size="42px" />
      <span>{{ displayEmptyText }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import { Dark } from 'quasar'
import type { OrganizationReadOnlyTreeNode } from './organization-read-only-tree'

const { t } = useI18n({ useScope: 'global' })

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
    emptyText: '',
    expandAll: false,
  },
)

const displayEmptyText = computed(() => props.emptyText || t('ui.noOrganizationData'))

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
}

.organization-readonly-tree__content {
  min-width: 340px;
}

.organization-readonly-tree__identity {
  min-width: 0;
}
</style>
