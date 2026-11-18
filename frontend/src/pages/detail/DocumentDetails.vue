<template>
  <section v-if="tabs.length" ref="host" class="document-details" :class="{ 'is-readonly': readonly }" :aria-label="t('document.details')">
    <header class="document-heading">
      <h3><q-icon name="table_rows" />{{ t('document.details') }}</h3>
      <q-btn
        v-if="ready && !readonly && activeGroup?.create"
        type="button" outline dense no-caps color="primary" icon="add"
        :label="t('document.newRow')" @click="addLine(activeGroup)"
      />
    </header>
    <q-banner v-if="failure" dense class="text-negative">
      {{ failure }}
      <q-btn flat dense icon="refresh" :aria-label="t('ui.refresh')" @click="reload++" />
    </q-banner>
    <q-linear-progress v-else-if="!ready" indeterminate />
    <template v-else>
      <q-tabs
        v-model="active"
        class="document-tabs"
        dense
        inline-label
        no-caps
        align="left"
        active-color="primary"
        indicator-color="primary"
        mobile-arrows
      >
        <q-tab
          v-for="group in groups"
          :key="group.relation.id"
          :name="group.relation.id"
          :label="group.relation.document_tab"
          :aria-label="group.relation.document_tab"
        ><q-badge class="tab-count">{{ group.lines.length }}</q-badge></q-tab>
      </q-tabs>
      <section
        v-for="group in sections"
        v-show="active === group.relation.id"
        :key="group.relation.id"
        class="document-section"
        :data-relation-id="group.relation.id"
        data-document-section
        @document-invalid="active = group.relation.id"
      >
        <q-markup-table flat square dense separator="horizontal" class="document-scroll" :aria-label="group.relation.document_tab">
            <thead>
              <tr>
                <th scope="col" class="line-number">#</th>
                <th v-for="field in group.columns" :key="field.field_code" scope="col" :class="{ numeric: isNumeric(field) }">
                  <span v-if="!field.is_null && group.scenes.some(s => fieldAccess(field, s) === 'editable')" aria-hidden="true" class="text-negative">*</span>
                  {{ field.field_name }}
                </th>
                <th
                  v-if="!readonly && (group.create || group.remove)"
                  scope="col"
                  class="line-action"
                >
                  {{ t('ui.actions') }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!group.lines.length">
                <td :colspan="group.columns.length + 1 + Number(!readonly && (group.create || group.remove))" class="document-empty">
                  {{ t('document.empty') }}
                </td>
              </tr>
              <template v-for="(line, index) in group.lines" :key="line.key">
                <tr v-if="readonly">
                  <td class="line-number">{{ index + 1 }}</td>
                  <td v-for="field in group.columns" :key="field.field_code" :class="{ numeric: isNumeric(field) }">
                    {{ display(group, field, line.data) }}
                  </td>
                </tr>
                <dynamic-form-dialog
                  v-else
                  :inline-row="line.data"
                  :fields="group.columns"
                  :inline-scene="line.data.id ? persistedScene(group) : 'insert'"
                  :table-code="group.code"
                  :menu-id="group.menuId"
                  :model-value="true"
                  :readonly="line.data.id ? !group.update : !group.create"
                >
                  <template #line-number
                    ><td class="line-number">{{ index + 1 }}</td></template
                  >
                  <template #line-value="{ field }">{{
                    display(group, field, line.data)
                  }}</template>
                  <template #line-actions
                    ><td v-if="group.create || group.remove" class="line-action">
                      <q-btn
                        v-if="line.data.id ? group.remove : group.create"
                        type="button"
                        flat
                        round
                        dense
                        icon="delete_outline"
                        color="negative"
                        :aria-label="t('document.delRow')"
                        @click="group.lines.splice(index, 1)"
                        ><q-tooltip>{{ t('document.delRow') }}</q-tooltip></q-btn
                      >
                    </td></template
                  >
                </dynamic-form-dialog>
              </template>
            </tbody>
          <tfoot v-if="active === group.relation.id && subtotal" class="document-totals">
            <tr>
              <th scope="row" class="line-number">{{ t('document.lines', { count: group.lines.length }) }}</th>
              <td v-for="field in group.columns" :key="field.field_code" class="numeric">
                <span v-if="field.field_code === subtotal.qtyColumn">{{ t('document.qtyTotal') }} <b>{{ subtotal.quantity }}</b></span>
                <span v-if="field.field_code === subtotal.amountColumn">{{ t('document.amountTotal') }} <b>{{ subtotal.amount }}</b></span>
              </td>
              <td v-if="!readonly && (group.create || group.remove)" class="line-action" />
            </tr>
          </tfoot>
        </q-markup-table>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGeneralizationApi } from '@/api/services/generalization'
import { useTableApi, type TableField, type TableRelation } from '@/api/services/sys-table'
import DynamicFormDialog from '@/components/FormDialog/DynamicFormDialog.vue'
import { useUserStore } from '@/stores/user'
import { useDictStore } from '@/stores/dict'
import { findMenuByTableCode } from '@/utils/menu-context'
import { SysTableFieldType } from '@/types/enum'
import { hasButtonActionCapability } from '@/utils/menu-button'
import { defaultValueForField } from '@/utils/field-metadata'
import { fieldAccess, type FieldScene } from '@/utils/field-layout'
import { buildColumnFormat, hydrateRelationLookups, type LookupMap } from '@/utils/column-format'
import {
  documentRelations,
  documentTotals,
  draftLines,
  lineChanges,
  type LineDraft,
} from './document-lines'

const props = defineProps<{
  relations: TableRelation[]
  tableCode: string
  masterId: number
  menuId: number
  readonly?: boolean
}>()
const { t } = useI18n()
const user = useUserStore(),
  dict = useDictStore(),
  api = useGeneralizationApi(),
  meta = useTableApi()
const tabs = computed(() =>
  documentRelations(props.relations).flatMap((relation) => {
    const code = relation.related_table_code || ''
    const menu = findMenuByTableCode(user.menus, code)
    const can = (action: string) => hasButtonActionCapability(menu?.menu_buttons || [], action)
    if (
      !menu ||
      !(props.masterId || props.readonly ? can('detail') || can('update') : can('create'))
    )
      return []
    return [
      {
        relation,
        code,
        menuId: menu.id,
        create: can('create'),
        update: can('update'),
        remove: can('delete'),
      },
    ]
  }),
)
type Group = (typeof tabs.value)[number] &
  LineDraft & { fields: TableField[]; lookups: Record<string, LookupMap> }
const groups = ref<Group[]>([]),
  active = ref(0),
  ready = ref(false),
  failure = ref(''),
  reload = ref(0)
let nextKey = 0
const host = ref<HTMLElement>()
const persistedScene = (group: { update: boolean }): 'detail' | 'update' =>
  props.readonly || !group.update ? 'detail' : 'update'
const sections = computed(() => groups.value.map(group => {
  const existing = persistedScene(group)
  const scenes = new Set<FieldScene>(group.lines.map(line => line.data.id ? existing : 'insert'))
  if (!scenes.size) scenes.add(!props.readonly && group.create ? 'insert' : existing)
  const modes = [...scenes]
  return { ...group, scenes: modes, columns: group.fields.filter(f => modes.some(s => fieldAccess(f, s) !== 'hidden')) }
}))
watch(
  [tabs, () => props.tableCode, () => props.masterId, () => props.menuId, reload, () => props.readonly],
  async ([items, code, id, menuId], _, onCleanup) => {
    let stale = false
    onCleanup(() => {
      stale = true
    })
    ready.value = false
    failure.value = ''
    groups.value = []
    try {
      const result = await Promise.all(
        items.map(async (item) => {
          const [schema, records] = await Promise.all([
            meta.queryRuntimeTableByCode(item.code),
            id ? api.getDocumentRows(code, id, item.relation.id, menuId, persistedScene(item)) : null,
          ])
          if (
            !schema.success ||
            !schema.data ||
            (records && (!records.success || !Array.isArray(records.data)))
          )
            throw new Error(records?.message || schema.message || t('document.failed'))
          const fields = schema.data.table_fields
            .filter(
              (f) =>
                f.field_code !== item.relation.foreign_key &&
                !f.system_managed &&
                !f.is_primary_key &&
                ((!props.readonly && item.create && fieldAccess(f, 'insert') !== 'hidden') ||
                  ((id || props.readonly) && fieldAccess(f, persistedScene(item)) !== 'hidden')),
            )
            .sort((a, b) => a.sequence - b.sequence)
          const rows = records?.data || []
          const dicts = [...new Set(fields.map((f) => f.dict_code).filter(Boolean))]
          if (dicts.length) await dict.loadDicts(dicts)
          const lookups = await hydrateRelationLookups(fields.filter(f => fieldAccess(f, persistedScene(item)) !== 'hidden'), rows, {}, item.menuId)
          return { ...item, fields, lookups, ...draftLines(rows) }
        }),
      )
      if (stale) return
      groups.value = result
      active.value = result[0]?.relation.id || 0
      ready.value = true
    } catch (err) {
      if (!stale) failure.value = err instanceof Error ? err.message : t('document.failed')
    }
  },
  { immediate: true },
)

const activeGroup = computed(() => sections.value.find(g => g.relation.id === active.value))
const subtotal = computed(() => {
  const group = activeGroup.value
  if (!group) return
  const visible = (code?: string) => group.columns.some(f =>
    f.field_code === code && group.scenes.every(s => fieldAccess(f, s) !== 'hidden'),
  )
  const { quantity_field: qty, amount_field: amount, price_field: price } = group.relation
  const qtyColumn = visible(qty) ? qty : undefined
  const amountColumn = amount ? (visible(amount) ? amount : undefined) : (qtyColumn && visible(price) ? price : undefined)
  const settings = {
    ...(qtyColumn && { quantity_field: qtyColumn }),
    ...(amountColumn && (amount ? { amount_field: amountColumn } : { price_field: amountColumn })),
  }
  return { ...documentTotals(settings, group.lines), qtyColumn, amountColumn }
})
const isNumeric = (field: TableField) =>
  [SysTableFieldType.INT, SysTableFieldType.SMALLINT, SysTableFieldType.BIGINT, SysTableFieldType.DECIMAL].includes(field.field_type)
function addLine(group: Group) {
  group.lines.push({
    key: 'new-' + ++nextKey,
    data: Object.fromEntries(
      group.fields
        .filter((f) => f.is_insert_show)
        .map((f) => [f.field_code, defaultValueForField(f)]),
    ),
  })
  void nextTick(() => host.value?.querySelector<HTMLElement>(
    '[data-relation-id="' + group.relation.id + '"] tbody tr:last-child td[data-field] :is(input:not([type=hidden]):not([readonly]):not([disabled]), textarea:not([readonly]), [role=combobox])',
  )?.focus())
}
function display(group: Group, field: TableField, data: Record<string, any>) {
  const format = buildColumnFormat(field, {
    getDictLabel: dict.getDictLabel,
    relationLookups: group.lookups,
  })
  const value = data[field.field_code]
  return format ? format(value, data) : (value ?? '')
}
function changes() {
  if (!ready.value || failure.value) throw new Error(failure.value || t('document.notReady'))
  return groups.value
    .map((g) =>
      lineChanges(
        g.relation.id,
        g,
        g.fields.filter((f) => g.create && fieldAccess(f, 'insert') === 'editable').map((f) => f.field_code),
        g.fields.filter((f) => g.update && fieldAccess(f, 'update') === 'editable').map((f) => f.field_code),
      ),
    )
    .filter((c) => c.added.length || c.updated.length || c.deleted.length)
}
defineExpose({ changes })
</script>

<style scoped lang="scss">
.document-details {
  min-width: 0;
  padding: 0 var(--document-gutter, 16px) 20px;
  background: var(--app-surface);
  border-top: 8px solid var(--app-surface-muted);
  box-shadow: inset 0 1px var(--app-border);
  color: var(--app-text-strong);
}
.document-heading {
  min-height: 52px;
  padding-block: 8px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.document-heading h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  line-height: 1.4;
  font-weight: 650;
}
.document-heading h3 .q-icon {
  color: var(--q-primary);
  font-size: 18px;
}
.document-heading .q-btn {
  padding-inline: 9px;
}
.document-tabs {
  min-width: 0;
  border-bottom: 1px solid var(--app-border);
  margin-bottom: 10px;
}
.document-tabs :deep(.q-tab__label) {
  font-size: 13px;
}
.tab-count {
  margin-inline-start: 6px;
  min-width: 18px;
  justify-content: center;
  font-variant-numeric: tabular-nums;
  background: var(--app-surface-muted);
  color: var(--app-text-muted);
}
.document-scroll {
  overflow-x: auto;
  border: 1px solid var(--app-border);
  border-radius: 4px;
  box-shadow: none;
}
.document-scroll :deep(table) {
  border-collapse: collapse;
  width: 100%;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}
.document-scroll :deep(:is(th, td)) {
  text-align: start;
  padding: 4px 10px;
  border-color: var(--app-border);
  vertical-align: middle;
}
.document-scroll thead th {
  white-space: nowrap;
  font-weight: 600;
  height: 36px;
  color: var(--app-text-muted);
  background: var(--app-surface-muted);
}
.document-scroll :deep(td:not(.line-number, .line-action, .document-empty)) {
  min-width: 112px;
  height: 40px;
  overflow-wrap: anywhere;
  white-space: normal;
}
.document-scroll :deep(:is(.line-number, .line-action)) {
  width: 54px;
  white-space: nowrap;
  text-align: center;
  color: var(--app-text-muted);
}
.document-scroll :deep(td.document-empty) {
  text-align: center;
  padding: 24px;
  color: var(--app-text-muted);
}
.document-scroll :deep(.numeric),
.document-scroll :deep(.number-input-field input) {
  text-align: end;
}
.document-scroll :deep(.line-readonly) { color: var(--app-text-strong); }
.document-scroll :deep(.q-field__control) { padding-inline: 4px; }
.document-scroll :deep(.q-field--with-bottom:not(.q-field--error):not(:has(.q-field__messages > div))) { padding-bottom: 0; }
.document-scroll :deep(.q-field--dense:not(.q-textarea) .q-field__control),
.document-scroll :deep(.q-field--dense:not(.q-textarea) .q-field__marginal) {
  min-height: 32px;
  height: 32px;
}
.document-scroll :deep(.q-field--outlined:not(.q-field--focused, .q-field--error) .q-field__control::before) {
  border-color: transparent;
}
.document-scroll :deep(.q-field--outlined:not(.q-field--focused, .q-field--error):hover .q-field__control::before) {
  border-color: var(--app-border);
}
.document-totals :is(td, th) {
  border-top: 1px solid var(--app-border);
  border-bottom: 0;
  background: var(--app-surface-muted);
  color: var(--app-text-muted);
  font-weight: 400;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.document-totals span { display: block; white-space: nowrap; }
.document-totals b {
  margin-inline-start: 4px;
  color: var(--app-text-strong);
}
@media (max-width: 600px) {
  .document-scroll :deep(td:not(.line-number, .line-action, .document-empty)) {
    min-width: 100px;
  }
}
@media (pointer: coarse) {
  .document-heading .q-btn, .document-scroll :deep(.line-action .q-btn),
  .document-tabs :deep(.q-tab) { min-width: 44px; min-height: 44px; }
}
</style>
