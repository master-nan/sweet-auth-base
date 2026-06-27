<template>
  <q-table
    :rows="flatRows"
    :columns="columns"
    row-key="id"
    :loading="loading"
    :dark="dark"
    :dense="dense"
    :flat="flat"
    :bordered="bordered"
    :square="square"
    :separator="separator"
    :pagination="{ rowsPerPage: 0 }"
    hide-pagination
    color="primary"
    v-bind="$attrs"
  >
    <template v-slot:top>
      <slot name="top"></slot>
    </template>
    <template v-slot:body="props">
      <q-tr
        :props="props"
        @click="$emit('node-selected', props.row)"
        :class="['cursor-pointer', { 'tree-table-row--active': isSelectedRow(props.row) }]"
      >
        <!-- 遍历所有列 -->
        <q-td v-for="(col, index) in columns" :key="col.name" :props="props">
          <!-- 第一列特殊处理，添加树结构的缩进和展开/折叠按钮 -->
          <template v-if="index === 0">
            <div class="flex items-center">
              <div :style="getIndentation(props.row)" class="items-center q-gutter-xs">
                <!-- 展开/折叠按钮 -->
                <q-btn
                  v-if="props.row.children && props.row.children.length > 0"
                  flat
                  dense
                  :icon="expandedRows[props.row.id] ? 'expand_more' : 'chevron_right'"
                  @click.stop="toggleExpand(props.row)"
                />
                <!-- 内容部分：使用插槽或默认内容 -->
                <slot
                  v-if="$slots['body-cell-' + col.name] != null"
                  :name="'body-cell-' + col.name"
                  :props="props"
                  :row="props.row"
                />
                <span v-else>{{ getCellValue(props.row, col) }}</span>
              </div>
            </div>
          </template>
          <!-- 其他列正常处理 -->
          <template v-else>
            <slot
              v-if="$slots['body-cell-' + col.name] != null"
              :name="'body-cell-' + col.name"
              :props="props"
              :row="props.row"
              :value="getCellValue(props.row, col)"
            >
            </slot>
            <span v-else>{{ getCellValue(props.row, col) }}</span>
          </template>
        </q-td>
      </q-tr>
    </template>
  </q-table>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { TableColumn, TreeTableRow } from 'src/types/global'
defineOptions({ name: 'TreeTable' })
// 定义树形表格行的基本接口

const props = withDefaults(
  defineProps<{
    data: TreeTableRow[]
    columns: TableColumn[]
    loading?: boolean
    dense?: boolean
    dark?: boolean
    flat?: boolean
    bordered?: boolean
    square?: boolean
    separator?: 'horizontal' | 'vertical' | 'cell' | 'none'
    selectedRowId?: number | string | null
    indent?: boolean
  }>(),
  {
    loading: false,
    dense: false,
    dark: false,
    flat: false,
    bordered: false,
    square: false,
    separator: 'horizontal',
    selectedRowId: null,
    indent: true,
  },
)

const expandedRows = ref<Record<number, boolean>>({})

const toggleExpand = (row: TreeTableRow) => {
  expandedRows.value[row.id] = !expandedRows.value[row.id]
}

const expandAncestorsForSelected = (
  nodes: TreeTableRow[],
  selectedId: number | string,
  ancestors: number[] = [],
) => {
  for (const node of nodes || []) {
    if (String(node.id) === String(selectedId)) {
      ancestors.forEach((id) => {
        expandedRows.value[id] = true
      })
      return true
    }

    if (expandAncestorsForSelected(node.children || [], selectedId, [...ancestors, node.id])) {
      return true
    }
  }
  return false
}

const getIndentation = (row: TreeTableRow) => {
  if (!props.indent) return undefined
  return `padding-left: ${(row.level || 0) * 24}px`
}

const getCellValue = (row: TreeTableRow, col: TableColumn) => {
  let val: any
  if (typeof col.field === 'function') {
    val = col.field(row)
  } else {
    val = row[col.field]
  }
  // 应用列的 format 函数
  if (col.format) {
    return col.format(val, row)
  }
  return val
}

const isSelectedRow = (row: TreeTableRow) => {
  if (props.selectedRowId === null || props.selectedRowId === undefined) return false
  return String(row.id) === String(props.selectedRowId)
}

const flatRows = computed(() => {
  const flatten = (nodes: TreeTableRow[], level = 0): TreeTableRow[] => {
    if (!nodes) return []
    return nodes.flatMap((node) => {
      const flattened = [
        {
          ...node,
          level,
        } as TreeTableRow,
      ]
      if (expandedRows.value[node.id] && node.children) {
        flattened.push(...flatten(node.children, level + 1))
      }
      return flattened
    })
  }
  return flatten(props.data)
})

watch(
  () => [props.data, props.selectedRowId] as const,
  ([data, selectedId]) => {
    if (selectedId === null || selectedId === undefined) return
    expandAncestorsForSelected(data, selectedId)
  },
  { immediate: true, deep: true },
)

// 添加节点选择事件
defineEmits(['node-selected'])
</script>

<style scoped lang="scss">
.tree-table-row--active > td {
  background: #f3f1ff !important;
  color: #172033;
  font-weight: 700;
}

</style>
