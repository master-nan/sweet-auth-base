import { computed, ref } from 'vue'

export function useReportQuery(defaultCategory = '') {
  const keyword = ref('')
  const status = ref('')
  const category = ref(defaultCategory)
  const page = ref(1)
  const pageSize = ref(20)

  const activeFilters = computed(() => ({
    keyword: keyword.value.trim(),
    status: status.value,
    category: category.value,
    page: page.value,
    pageSize: pageSize.value,
  }))

  function resetQuery() {
    keyword.value = ''
    status.value = ''
    category.value = defaultCategory
    page.value = 1
    pageSize.value = 20
  }

  return {
    keyword,
    status,
    category,
    page,
    pageSize,
    activeFilters,
    resetQuery,
  }
}
