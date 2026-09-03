<template>
  <div class="query-preview">
    <div v-if="!treeLines.length" class="query-preview__empty">
      {{ t('ui.noAdvancedQueryConditionSet') }}
    </div>
    <div
      v-else
      class="query-preview-tree"
      role="tree"
      :aria-label="t('ui.levelOfQueryConditionals')"
    >
      <div
        v-for="line in treeLines"
        :key="line.key"
        :class="['query-preview-tree__line', `query-preview-tree__line--${line.kind}`]"
        role="treeitem"
        :aria-level="line.depth + 1"
      >
        <span class="query-preview-tree__branch" aria-hidden="true">{{ line.branch }}</span>
        <q-badge
          v-if="line.logic"
          outline
          :color="line.logic === 'OR' ? 'warning' : 'primary'"
          :label="line.logic"
          class="query-preview-tree__logic"
        />
        <span class="query-preview-tree__text">{{ line.text }}</span>
      </div>
    </div>
    <div v-if="payload.quick_query.keyword || payload.order.field" class="query-preview-meta">
      <div v-if="payload.quick_query.keyword" class="query-preview-meta__item">
        <q-badge outline color="secondary" :label="t('ui.keyword')" />
        <span>{{ t('ui.containsOpeningQuote') }}{{ payload.quick_query.keyword }}”</span>
      </div>
      <div v-if="payload.order.field" class="query-preview-meta__item">
        <q-badge outline color="secondary" :label="t('ui.sort')" />
        <span
          >{{ fieldLabel(payload.order.field) }}
          {{ payload.order.is_asc ? t('ui.raise') : t('ui.descending') }}</span
        >
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { primitiveText } from '@/utils/primitive-text'

import { computed, ref, watch } from 'vue'
import { ExpressionLogic, ExpressionTypeMap } from '@/types/enum'
import { useDictStore } from '@/stores/dict'
import type { ExpressionGroup, QueryRule } from '@/types/global'
import type { TableField } from '@/api/services/sys-table'
import { queryRuntimeRelationOptions } from '@/api/services/runtime-relation'
import {
  QUERY_SCHEME_BINDING_LABELS,
  type QuerySchemeBinding,
  type QuerySchemePayloadV1,
} from '@/modules/query-scheme/types'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  payload: QuerySchemePayloadV1
  fields?: TableField[]
  menuId?: number
}>()

const dictStore = useDictStore()
const fieldMap = computed(
  () => new Map((props.fields || []).map((field) => [field.field_code, field])),
)
const bindingMap = computed(
  () => new Map((props.payload.bindings || []).map((binding) => [binding.pointer, binding])),
)
const relationLabels = ref<Record<string, Record<string, string>>>({})
const fieldLabel = (code: string) => fieldMap.value.get(code)?.field_name || t('ui.expiredField')

const bindingLabel = (binding: QuerySchemeBinding) => {
  const base = QUERY_SCHEME_BINDING_LABELS[binding.kind]
  const offset =
    binding.params?.day_offset ?? binding.params?.week_offset ?? binding.params?.month_offset
  return offset ? t('ui.diversion', { base: base, offset: offset }) : base
}

const singleValueLabel = (rule: QueryRule, value: unknown, pointer: string) => {
  const binding = bindingMap.value.get(pointer)
  if (binding) return bindingLabel(binding)
  const field = fieldMap.value.get(rule.field)
  if (field?.dict_code)
    return dictStore.getDictLabel(field.dict_code, value) || primitiveText(value, '-')
  if (field?.relation)
    return (
      relationLabels.value[rule.field]?.[primitiveText(value)] || t('ui.associationValueUnsolved')
    )
  if (value === null || value === undefined || value === '') return t('ui.noNeedToFill')
  return primitiveText(value, '-')
}

const valueLabel = (rule: QueryRule, pointer: string) => {
  if (Array.isArray(rule.value)) {
    return rule.value
      .map((value, index) => singleValueLabel(rule, value, `${pointer}/${index}`))
      .join(t('ui.to'))
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

type PreviewTreeNode = {
  key: string
  kind: 'group' | 'rule'
  logic?: 'AND' | 'OR'
  text: string
  children: PreviewTreeNode[]
}

type PreviewTreeLine = {
  key: string
  kind: PreviewTreeNode['kind']
  logic: PreviewTreeNode['logic'] | undefined
  text: string
  branch: string
  depth: number
}

const groupLogic = (group: ExpressionGroup): 'AND' | 'OR' =>
  (group.logic ?? ExpressionLogic.AND) === ExpressionLogic.OR ? 'OR' : 'AND'

const buildGroupNode = (group: ExpressionGroup, path: string): PreviewTreeNode | null => {
  const children: PreviewTreeNode[] = group.rules.map((rule, index) => ({
    key: `${path}/rules/${index}`,
    kind: 'rule',
    get text() {
      return t('ui.queryRuleSummary', {
        value1: fieldLabel(rule.field),
        value2: ExpressionTypeMap[rule.expression_type!] || t('ui.match'),
        value3: valueLabel(rule, `${path}/rules/${index}/value`),
      })
    },
    children: [],
  }))
  ;(group.nested || []).forEach((nested, index) => {
    const nestedNode = buildGroupNode(nested, `${path}/nested/${index}`)
    if (nestedNode) children.push(nestedNode)
  })
  if (!children.length) return null
  const logic = groupLogic(group)
  return {
    key: path,
    kind: 'group',
    logic,
    get text() {
      return logic === 'AND' ? t('ui.allConditionsMet') : t('ui.eitherConditionIsMet')
    },
    children,
  }
}

const treeRoots = computed<PreviewTreeNode[]>(() => {
  const groups = (props.payload.expressions || [])
    .map((group, index) => buildGroupNode(group, `/expressions/${index}`))
    .filter((group): group is PreviewTreeNode => !!group)
  if (groups.length <= 1) return groups
  return [
    {
      key: '/expressions',
      kind: 'group',
      logic: 'AND',
      get text() {
        return t('ui.meetAllExpressionGroups')
      },
      children: groups,
    },
  ]
})

const flattenTree = (
  nodes: PreviewTreeNode[],
  prefix = '',
  depth = 0,
  roots = true,
): PreviewTreeLine[] =>
  nodes.flatMap((node, index) => {
    const isLast = index === nodes.length - 1
    const branch = roots ? '' : `${prefix}${isLast ? '└─ ' : '├─ '}`
    const childPrefix = roots ? '' : `${prefix}${isLast ? '   ' : '│  '}`
    return [
      {
        key: node.key,
        kind: node.kind,
        logic: node.logic,
        text: node.text,
        branch,
        depth,
      },
      ...flattenTree(node.children, childPrefix, depth + 1, false),
    ]
  })

const treeLines = computed(() => flattenTree(treeRoots.value))
</script>

<style scoped lang="scss">
.query-preview {
  color: var(--app-text-strong);
}

.query-preview__empty {
  padding: 6px 8px;
  color: var(--app-text-muted);
}

.query-preview-tree {
  padding: 6px 10px;
  border: 1px solid var(--app-border);
  border-left: 3px solid $primary;
  border-radius: 6px;
  background: var(--app-surface);
}

.query-preview-tree__line {
  display: flex;
  min-height: 28px;
  align-items: center;
  line-height: 1.4;
}

.query-preview-tree__line--group {
  font-weight: 600;
}

.query-preview-tree__branch {
  flex: 0 0 auto;
  color: var(--app-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  white-space: pre;
}

.query-preview-tree__logic {
  min-width: 38px;
  margin-right: 8px;
  justify-content: center;
}

.query-preview-tree__text {
  min-width: 0;
  overflow-wrap: anywhere;
}

.query-preview-tree__line--rule .query-preview-tree__text {
  font-weight: 400;
}

.query-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 18px;
  margin-top: 8px;
  padding: 0 10px;
}

.query-preview-meta__item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 24px;
}
</style>
