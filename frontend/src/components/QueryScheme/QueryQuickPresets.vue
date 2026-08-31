<template>
  <div v-if="presets.length" class="row items-center q-gutter-xs">
    <q-btn
      v-for="preset in presets"
      :key="preset.code"
      flat
      dense
      no-caps
      :label="preset.label"
      @click="$emit('apply', preset.payload)"
    />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import { ExpressionLogic, ExpressionType } from 'src/types/enum'
import {
  QuerySchemeBindingKind,
  type QuerySchemePayloadV1,
  type QueryScopeConfig,
  type QueryScopeQuickPreset,
} from 'src/modules/query-scheme/types'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{ config: QueryScopeConfig | null }>()
defineEmits<{ apply: [payload: QuerySchemePayloadV1] }>()

const datePreset = (
  code: string,
  label: string,
  start: QuerySchemeBindingKind,
  end?: QuerySchemeBindingKind,
  offset?: number,
): QueryScopeQuickPreset => {
  const range = !!end
  const value = range ? [null, null] : null
  const params =
    start === QuerySchemeBindingKind.TODAY
      ? { day_offset: offset || 0 }
      : start === QuerySchemeBindingKind.START_OF_MONTH
        ? { month_offset: offset || 0 }
        : { week_offset: offset || 0 }
  const pointer = '/expressions/0/rules/0/value'
  return {
    code,
    label,
    payload: {
      expressions: [
        {
          logic: ExpressionLogic.AND,
          rules: [
            {
              field: props.config?.quick_date_field || '',
              expression_type: range ? ExpressionType.BETWEEN : ExpressionType.EQ,
              value,
            },
          ],
          nested: [],
        },
      ],
      quick_query: { keyword: '' },
      order: { field: '', is_asc: false },
      bindings: range
        ? [
            { pointer: `${pointer}/0`, kind: start, params },
            { pointer: `${pointer}/1`, kind: end!, params },
          ]
        : [{ pointer, kind: start, params }],
    },
  }
}

const presets = computed(() => {
  const result = [...(props.config?.quick_presets || [])]
  if (!props.config?.quick_date_field) return result
  return [
    datePreset('today', t('ui.today'), QuerySchemeBindingKind.TODAY),
    datePreset('yesterday', t('ui.yesterday'), QuerySchemeBindingKind.TODAY, undefined, -1),
    datePreset(
      'this_week',
      t('ui.thisWeek'),
      QuerySchemeBindingKind.START_OF_WEEK,
      QuerySchemeBindingKind.END_OF_WEEK,
    ),
    datePreset(
      'this_month',
      t('ui.thisMonth'),
      QuerySchemeBindingKind.START_OF_MONTH,
      QuerySchemeBindingKind.END_OF_MONTH,
    ),
    datePreset(
      'last_month',
      t('ui.lastMonth'),
      QuerySchemeBindingKind.START_OF_MONTH,
      QuerySchemeBindingKind.END_OF_MONTH,
      -1,
    ),
    ...result,
  ]
})
</script>
