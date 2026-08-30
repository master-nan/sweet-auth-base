<template>
  <div class="table-pagination row items-center no-wrap">
    <span class="table-pagination__total">共 {{ formattedTotal }} 条</span>
    <q-separator vertical class="table-pagination__separator" />

    <div class="table-pagination__navigation row items-center no-wrap">
      <q-btn
        flat
        round
        dense
        icon="first_page"
        aria-label="首页"
        :disable="isFirstPage"
        @click="goToPage(1)"
      >
        <q-tooltip>首页</q-tooltip>
      </q-btn>
      <q-btn
        flat
        round
        dense
        icon="chevron_left"
        aria-label="上一页"
        :disable="isFirstPage"
        @click="goToPage(currentPage - 1)"
      >
        <q-tooltip>上一页</q-tooltip>
      </q-btn>

      <q-input
        v-model="pageDraft"
        outlined
        dense
        inputmode="numeric"
        aria-label="当前页"
        class="table-pagination__page-input"
        input-class="text-center text-weight-medium"
        @keyup.enter="commitPage"
        @blur="commitPage"
      >
        <template #append>
          <span class="table-pagination__page-total">/ {{ formattedTotalPages }} 页</span>
        </template>
      </q-input>

      <q-btn
        flat
        round
        dense
        icon="chevron_right"
        aria-label="下一页"
        :disable="isLastPage"
        @click="goToPage(currentPage + 1)"
      >
        <q-tooltip>下一页</q-tooltip>
      </q-btn>
      <q-btn
        flat
        round
        dense
        icon="last_page"
        aria-label="末页"
        :disable="isLastPage"
        @click="goToPage(totalPages)"
      >
        <q-tooltip>末页</q-tooltip>
      </q-btn>
    </div>

    <q-separator vertical class="table-pagination__separator" />
    <q-select
      v-model="currentPageSize"
      outlined
      dense
      options-dense
      :options="resolvedPageSizeOptions"
      aria-label="每页条数"
      class="table-pagination__page-size"
      @update:model-value="onPageSizeChange"
    >
      <template #append>
        <q-item-label class="table-pagination__page-size-suffix">/ 页</q-item-label>
      </template>
      <template #option="scope">
        <q-item v-bind="scope.itemProps">
          <q-item-section>
            <q-item-label>{{ scope.opt }}</q-item-label>
          </q-item-section>
        </q-item>
      </template>
    </q-select>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'

const props = defineProps<{
  page: number
  pageSize: number
  total: number
  pageSizeOptions?: number[]
}>()

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
}>()

const resolvedPageSizeOptions = computed(() => props.pageSizeOptions || [20, 30, 50, 100, 200, 500])
const currentPage = ref(Math.max(1, props.page))
const currentPageSize = ref(props.pageSize)
const pageDraft = ref(String(currentPage.value))

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / currentPageSize.value)))
const formattedTotal = computed(() => Math.max(0, props.total).toLocaleString('zh-CN'))
const formattedTotalPages = computed(() => totalPages.value.toLocaleString('zh-CN'))
const isFirstPage = computed(() => currentPage.value <= 1)
const isLastPage = computed(() => currentPage.value >= totalPages.value)

const goToPage = (page: number) => {
  const nextPage = Math.min(Math.max(page, 1), totalPages.value)
  const pageChanged = nextPage !== currentPage.value
  currentPage.value = nextPage
  pageDraft.value = String(nextPage)
  if (pageChanged) {
    emit('update:page', nextPage)
  }
}

const commitPage = () => {
  const parsedPage = Number.parseInt(pageDraft.value, 10)
  goToPage(Number.isFinite(parsedPage) ? parsedPage : currentPage.value)
}

const onPageSizeChange = (pageSize: number) => {
  currentPage.value = 1
  pageDraft.value = '1'
  emit('update:pageSize', pageSize)
  emit('update:page', 1)
}

watch(
  () => [props.page, props.pageSize, props.total] as const,
  ([newPage, newPageSize]) => {
    currentPageSize.value = newPageSize
    currentPage.value = Math.min(Math.max(newPage, 1), totalPages.value)
    pageDraft.value = String(currentPage.value)
  },
)
</script>

<style scoped>
.table-pagination {
  max-width: 100%;
  min-height: 40px;
  gap: 6px;
}

.table-pagination__total {
  flex: 0 0 auto;
  color: var(--app-text-muted);
  font-size: 13px;
  white-space: nowrap;
}

.table-pagination__separator {
  align-self: center;
  flex: 0 0 1px;
  height: 24px;
  min-height: 24px;
  margin: 0 2px;
}

.table-pagination__navigation {
  flex: 0 0 auto;
  gap: 2px;
}

.table-pagination__page-input {
  flex: 0 0 150px;
  width: 150px;
  margin: 0 2px;
}

.table-pagination__page-input :deep(.q-field__control),
.table-pagination__page-size :deep(.q-field__control) {
  height: 36px;
  min-height: 36px;
}

.table-pagination__page-input :deep(.q-field__native) {
  min-width: 34px;
  padding-right: 4px;
}

.table-pagination__page-input :deep(.q-field__append) {
  padding-left: 2px;
}

.table-pagination__page-total,
.table-pagination__page-size-suffix {
  color: var(--app-text-muted);
  font-size: 13px;
  font-weight: 400;
  white-space: nowrap;
}

.table-pagination__page-size {
  flex: 0 0 104px;
  width: 104px;
}

.table-pagination__page-size :deep(.q-field__native) {
  min-width: 28px;
}

@media (max-width: 720px) {
  .table-pagination {
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .table-pagination__total {
    margin-right: auto;
  }

  .table-pagination__separator {
    display: none;
  }
}
</style>
