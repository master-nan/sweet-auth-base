<template>
  <div class="report-data-table">
    <q-table
      flat
      bordered
      dense
      hide-pagination
      :row-key="props.rowKey"
      :rows="props.rows"
      :columns="props.columns"
      :loading="props.loading"
      :pagination="{ rowsPerPage: 0 }"
    >
      <template #body-cell-status="props">
        <slot name="body-cell-status" v-bind="props">
          <q-td :props="props">{{ props.value }}</q-td>
        </slot>
      </template>

      <template #body-cell-menu="props">
        <slot name="body-cell-menu" v-bind="props">
          <q-td :props="props">{{ props.value }}</q-td>
        </slot>
      </template>

      <template #body-cell-menuName="props">
        <slot name="body-cell-menuName" v-bind="props">
          <q-td :props="props">{{ props.value }}</q-td>
        </slot>
      </template>

      <template #body-cell-actions="props">
        <slot name="body-cell-actions" v-bind="props">
          <q-td :props="props" />
        </slot>
      </template>

      <template #no-data>
        <div class="empty-state">
          <q-icon name="inbox" size="28px" />
          <span>{{ emptyText }}</span>
        </div>
      </template>
    </q-table>

    <div v-if="showPagination" class="table-pagination">
      <table-pagination
        :page="props.page"
        :page-size="props.pageSize"
        :total="props.total"
        @update:page="$emit('update:page', $event)"
        @update:page-size="$emit('update:pageSize', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { QTableProps } from 'quasar'
import TablePagination from 'components/Table/TablePagination.vue'

const props = withDefaults(defineProps<{
  rows: unknown[]
  columns: QTableProps['columns']
  rowKey?: string
  loading?: boolean
  page?: number
  pageSize?: number
  total?: number
  showPagination?: boolean
  emptyText?: string
}>(), {
  rowKey: 'id',
  loading: false,
  page: 1,
  pageSize: 20,
  total: 0,
  showPagination: true,
  emptyText: '暂无数据',
})

defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()
</script>

<style scoped lang="scss">
.report-data-table {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.table-pagination {
  display: flex;
  justify-content: flex-end;
}

.empty-state {
  width: 100%;
  min-height: 120px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 8px;
  color: #98a2b3;
}
</style>
