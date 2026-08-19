<template>
  <sweet-select
    :model-value="modelValue"
    :options="options"
    :loading="loading"
    :disable="disabled"
    multiple
    use-chips
    use-input
    emit-value
    map-options
    input-debounce="300"
    option-value="value"
    option-label="label"
    :max-values="max"
    @update:model-value="emit('update:modelValue', $event)"
    @filter="filterRoles"
    @virtual-scroll="loadMoreRoles"
  >
    <template #no-option>
      <q-item>
        <q-item-section class="text-grey-7">
          {{ loadFailed ? '角色加载失败' : '暂无可选角色' }}
        </q-item-section>
      </q-item>
    </template>
  </sweet-select>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoleApi, type Role } from 'src/api/services/sys-role'
import SweetSelect from 'src/components/Select/SweetSelect.vue'

type RoleOption = { label: string; value: number; disabled?: boolean }
type FilterUpdate = (callback: () => void) => void
type FilterAbort = () => void

const pageSize = 20
const props = withDefaults(
  defineProps<{
    modelValue: number[]
    disabled?: boolean
    max?: number
  }>(),
  {
    disabled: false,
    max: 32,
  },
)
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()
const roleApi = useRoleApi()
const searchOptions = ref<RoleOption[]>([])
const selectedOptions = ref<RoleOption[]>([])
const loading = ref(false)
const loadFailed = ref(false)
const keyword = ref('')
const page = ref(1)
const total = ref(0)
let requestSequence = 0

const mergeOptions = (...groups: RoleOption[][]) => {
  const byID = new Map<number, RoleOption>()
  groups.flat().forEach((option) => byID.set(option.value, option))
  return [...byID.values()]
}
const options = computed(() => mergeOptions(selectedOptions.value, searchOptions.value))
const toOption = (role: Pick<Role, 'id' | 'name'>): RoleOption => ({
  label: role.name,
  value: role.id,
})

const queryPage = async (searchKeyword: string, targetPage: number) => {
  const response = await roleApi.queryRole({
    page: targetPage,
    num: pageSize,
    table_code: 'sys_role',
    expressions: [],
    quick_query: { keyword: searchKeyword },
    include_deleted: false,
  })
  return {
    items: (response.data || []).map(toOption),
    total: response.total || 0,
  }
}

const refresh = async (searchKeyword = '') => {
  const sequence = ++requestSequence
  loading.value = true
  loadFailed.value = false
  try {
    const normalizedKeyword = searchKeyword.trim()
    const result = await queryPage(normalizedKeyword, 1)
    if (sequence !== requestSequence) return
    keyword.value = normalizedKeyword
    page.value = 1
    total.value = result.total
    searchOptions.value = result.items
  } catch {
    if (sequence === requestSequence) loadFailed.value = true
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

const filterRoles = (value: string, update: FilterUpdate, abort: FilterAbort) => {
  const sequence = ++requestSequence
  loading.value = true
  loadFailed.value = false
  const normalizedKeyword = value.trim()
  void queryPage(normalizedKeyword, 1)
    .then((result) => {
      if (sequence !== requestSequence) {
        abort()
        return
      }
      update(() => {
        keyword.value = normalizedKeyword
        page.value = 1
        total.value = result.total
        searchOptions.value = result.items
      })
    })
    .catch(() => {
      if (sequence === requestSequence) loadFailed.value = true
      abort()
    })
    .finally(() => {
      if (sequence === requestSequence) loading.value = false
    })
}

const loadMoreRoles = (details: { to: number }) => {
  if (
    loading.value ||
    searchOptions.value.length >= total.value ||
    details.to < options.value.length - 1
  ) {
    return
  }
  const nextPage = page.value + 1
  loading.value = true
  void queryPage(keyword.value, nextPage)
    .then((result) => {
      page.value = nextPage
      total.value = result.total
      searchOptions.value = mergeOptions(searchOptions.value, result.items)
    })
    .catch(() => {
      loadFailed.value = true
    })
    .finally(() => {
      loading.value = false
    })
}

const hydrateSelectedRoles = async (ids: number[]) => {
  const loadedIDs = new Set(options.value.map((option) => option.value))
  const missingIDs = [...new Set(ids)].filter((id) => id > 0 && !loadedIDs.has(id))
  if (!missingIDs.length) return
  const results = await Promise.allSettled(missingIDs.map((id) => roleApi.queryRoleById(id)))
  const loaded = results.flatMap((result) =>
    result.status === 'fulfilled' && result.value.data ? [toOption(result.value.data)] : [],
  )
  selectedOptions.value = mergeOptions(selectedOptions.value, loaded)
}

watch(
  () => props.modelValue,
  (ids) => void hydrateSelectedRoles(ids),
  { immediate: true },
)
onMounted(() => void refresh())
</script>
