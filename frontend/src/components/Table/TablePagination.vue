<template>
  <div class="row">
    <q-pagination
      direction-links
      icon-prev="arrow_left"
      icon-next="arrow_right"
      v-model="currentPage"
      boundary-numbers
      :max-pages="7"
      :max="totalPages"
      @update:model-value="onPageChange"
    />
    <q-separator vertical />
    <q-select
      v-model="currentPageSize"
      outlined
      :dense="true"
      :options-dense="true"
      :options="resolvedPageSizeOptions"
      class="q-ml-md"
      @update:model-value="onPageSizeChange"
    >
      <template #append>
        <q-item-label class="text-body2">/ 页</q-item-label>
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
import { ref, computed, watch } from 'vue'

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

const currentPage = ref(props.page)
const currentPageSize = ref(props.pageSize)

const totalPages = computed(() => {
  return Math.max(1, Math.ceil(props.total / currentPageSize.value))
})

const onPageChange = (page: number) => {
  emit('update:page', page)
}

const onPageSizeChange = (pageSize: number) => {
  currentPage.value = 1
  emit('update:pageSize', pageSize)
  emit('update:page', 1)
}

watch(
  () => [props.page, props.pageSize],
  ([newPage, newPageSize]) => {
    currentPage.value = newPage!
    currentPageSize.value = newPageSize!
  },
)
</script>
