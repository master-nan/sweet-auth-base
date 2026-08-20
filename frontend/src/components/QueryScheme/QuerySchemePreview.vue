<template>
  <div class="column q-gutter-sm">
    <div v-if="!lines.length" class="text-grey-7">未设置高级查询条件</div>
    <div v-for="line in lines" :key="line.key" class="row items-start no-wrap q-gutter-sm">
      <q-badge outline color="primary" :label="line.logic" />
      <span :style="{ paddingLeft: `${line.depth * 12}px` }">{{ line.text }}</span>
    </div>
    <div v-if="payload.quick_query.keyword" class="row q-gutter-sm">
      <q-badge outline color="secondary" label="快捷" />
      <span>关键词包含“{{ payload.quick_query.keyword }}”</span>
    </div>
    <div v-if="payload.order.field" class="row q-gutter-sm">
      <q-badge outline color="secondary" label="排序" />
      <span>{{ fieldLabel(payload.order.field) }} {{ payload.order.is_asc ? '升序' : '降序' }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ExpressionLogic, ExpressionLogicMap, ExpressionTypeMap } from 'src/types/enum'
import { useDictStore } from 'src/stores/dict'
import type { ExpressionGroup, QueryRule } from 'src/types/global'
import type { TableField } from 'src/api/services/sys-table'
import { queryRuntimeRelationOptions } from 'src/api/services/runtime-relation'
import {
  QUERY_SCHEME_BINDING_LABELS,
  type QuerySchemeBinding,
  type QuerySchemePayloadV1,
} from 'src/modules/query-scheme/types'

const props = defineProps<{
  payload: QuerySchemePayloadV1
  fields?: TableField[]
  menuId?: number
}>()

const dictStore = useDictStore()
const fieldMap = computed(() => new Map((props.fields || []).map((field) => [field.field_code, field])))
const bindingMap = computed(() => new Map((props.payload.bindings || []).map((binding) => [binding.pointer, binding])))
const relationLabels = ref<Record<string, Record<string, string>>>({})
const fieldLabel = (code: string) => fieldMap.value.get(code)?.field_name || '已失效字段'

const bindingLabel = (binding: QuerySchemeBinding) => {
  const base = QUERY_SCHEME_BINDING_LABELS[binding.kind]
  const offset = binding.params?.day_offset ?? binding.params?.week_offset ?? binding.params?.month_offset
  return offset ? `${base}（偏移 ${offset}）` : base
}

const singleValueLabel = (rule: QueryRule, value: unknown, pointer: string) => {
  const binding = bindingMap.value.get(pointer)
  if (binding) return bindingLabel(binding)
  const field = fieldMap.value.get(rule.field)
  if (field?.dict_code) return dictStore.getDictLabel(field.dict_code, value) || String(value ?? '-')
  if (field?.relation) return relationLabels.value[rule.field]?.[String(value)] || '关联值未解析'
  if (value === null || value === undefined || value === '') return '无需填写'
  return String(value)
}

const valueLabel = (rule: QueryRule, pointer: string) => {
  if (Array.isArray(rule.value)) {
    return rule.value
      .map((value, index) => singleValueLabel(rule, value, `${pointer}/${index}`))
      .join(' 至 ')
  }
  return singleValueLabel(rule, rule.value, pointer)
}

let relationLoadSequence = 0
watch(
  () => [props.payload, props.fields, props.menuId] as const,
  async () => {
    const sequence = ++relationLoadSequence
    const valuesByField = new Map<string, Set<string>>()
    const collect = (group: ExpressionGroup) => {
      group.rules.forEach((rule) => {
        const field = fieldMap.value.get(rule.field)
        if (!field?.relation || !field.id) return
        const values = Array.isArray(rule.value) ? rule.value : [rule.value]
        const selected = valuesByField.get(rule.field) || new Set<string>()
        values.forEach((value) => {
          if (value !== null && value !== undefined && value !== '') selected.add(String(value))
        })
        valuesByField.set(rule.field, selected)
      })
      group.nested?.forEach(collect)
    }
    ;(props.payload.expressions || []).forEach(collect)
    if (!props.menuId || valuesByField.size === 0) {
      relationLabels.value = {}
      return
    }
    const next: Record<string, Record<string, string>> = {}
    await Promise.all(
      Array.from(valuesByField.entries()).map(async ([fieldCode, values]) => {
        const field = fieldMap.value.get(fieldCode)
        if (!field?.id || values.size === 0) return
        try {
          const result = await queryRuntimeRelationOptions(field.id, {
            menu_id: props.menuId!,
            selected_values: Array.from(values),
            page: 1,
            num: Math.max(values.size, 20),
          })
          next[fieldCode] = Object.fromEntries(result.items.map((item) => [item.value, item.label]))
        } catch {
          next[fieldCode] = {}
        }
      }),
    )
    if (sequence === relationLoadSequence) relationLabels.value = next
  },
  { deep: true, immediate: true },
)

type PreviewLine = { key: string; logic: string; text: string; depth: number }
const collectLines = (group: ExpressionGroup, path: string, depth: number): PreviewLine[] => {
  const logic = ExpressionLogicMap[group.logic ?? ExpressionLogic.AND]
  const rules = group.rules.map((rule, index) => ({
    key: `${path}/rules/${index}`,
    logic: index === 0 ? logic : ExpressionLogicMap[group.logic ?? ExpressionLogic.AND],
    depth,
    text: `${fieldLabel(rule.field)} ${ExpressionTypeMap[rule.expression_type!] || '匹配'} ${valueLabel(rule, `${path}/rules/${index}/value`)}`,
  }))
  return rules.concat(
    (group.nested || []).flatMap((nested, index) =>
      collectLines(nested, `${path}/nested/${index}`, depth + 1),
    ),
  )
}

const lines = computed(() =>
  (props.payload.expressions || []).flatMap((group, index) =>
    collectLines(group, `/expressions/${index}`, 0),
  ),
)
</script>
