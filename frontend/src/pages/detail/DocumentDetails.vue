<template>
  <section v-if="tabs.length" class="document-details">
    <q-banner v-if="failure" dense class="text-negative">
      {{ failure }}
      <q-btn flat dense icon="refresh" :aria-label="t('ui.refresh')" @click="reload++" />
    </q-banner>
    <q-linear-progress v-else-if="!ready" indeterminate />
    <template v-else>
      <q-tabs
        v-model="active"
        dense
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
        />
      </q-tabs>
      <section
        v-for="group in groups"
        v-show="active === group.relation.id"
        :key="group.relation.id"
        class="document-section"
        data-document-section
        @document-invalid="active = group.relation.id"
      >
        <div class="document-scroll">
          <table>
            <thead>
              <tr>
                <th scope="col" class="line-number">#</th>
                <th v-for="field in group.fields" :key="field.field_code" scope="col">
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
                <td :colspan="group.fields.length + 2" class="document-empty">
                  {{ t('document.empty') }}
                </td>
              </tr>
              <template v-for="(line, index) in group.lines" :key="line.key">
                <tr v-if="readonly">
                  <td class="line-number">{{ index + 1 }}</td>
                  <td v-for="field in group.fields" :key="field.field_code">
                    {{ display(group, field, line.data) }}
                  </td>
                </tr>
                <dynamic-form-dialog
                  v-else
                  :inline-row="line.data"
                  :fields="group.fields"
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
          </table>
        </div>
        <div class="lines-footer">
          <q-btn
            v-if="!readonly && group.create"
            type="button"
            flat
            dense
            no-caps
            color="primary"
            icon="add"
            :label="t('document.newRow')"
            @click="addLine(group)"
          />
          <span>{{ t('document.lines', { count: group.lines.length }) }}</span>
        </div>
      </section>
      <div v-if="hasQuantity || hasAmount" class="document-totals">
        <strong>{{ t('document.total') }}</strong>
        <span v-if="hasQuantity"
          >{{ t('document.qtyTotal') }} <b>{{ totals.quantity }}</b></span
        >
        <span v-if="hasAmount"
          >{{ t('document.amountTotal') }} <b>{{ totals.amount }}</b></span
        >
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useGeneralizationApi } from '@/api/services/generalization'
import { useTableApi, type TableField, type TableRelation } from '@/api/services/sys-table'
import DynamicFormDialog from '@/components/FormDialog/DynamicFormDialog.vue'
import { useUserStore } from '@/stores/user'
import { useDictStore } from '@/stores/dict'
import { findMenuByTableCode } from '@/utils/menu-context'
import { hasButtonActionCapability } from '@/utils/menu-button'
import { defaultValueForField } from '@/utils/field-metadata'
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
watch(
  [tabs, () => props.tableCode, () => props.masterId, () => props.menuId, reload],
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
            id ? api.getDocumentRows(code, id, item.relation.id, menuId) : null,
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
                (props.readonly
                  ? (f.detail_visible ?? f.is_list_show)
                  : f.is_insert_show || f.is_update_show || f.is_list_show),
            )
            .sort((a, b) => a.sequence - b.sequence)
          const rows = records?.data || []
          const dicts = [...new Set(fields.map((f) => f.dict_code).filter(Boolean))]
          if (dicts.length) await dict.loadDicts(dicts)
          const lookups = await hydrateRelationLookups(fields, rows, {}, item.menuId)
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

const hasQuantity = computed(() => groups.value.some((g) => g.relation.quantity_field))
const hasAmount = computed(() =>
  groups.value.some((g) => g.relation.price_field || g.relation.amount_field),
)
const totals = computed(() => documentTotals(groups.value))
function addLine(group: Group) {
  group.lines.push({
    key: 'new-' + ++nextKey,
    data: Object.fromEntries(
      group.fields
        .filter((f) => f.is_insert_show)
        .map((f) => [f.field_code, defaultValueForField(f)]),
    ),
  })
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
        g.fields.filter((f) => f.is_insert_show).map((f) => f.field_code),
        g.fields.filter((f) => f.is_update_show).map((f) => f.field_code),
      ),
    )
    .filter((c) => c.added.length || c.updated.length || c.deleted.length)
}
defineExpose({ changes })
</script>

<style scoped lang="scss">
.document-details {
  min-width: 0;
  border-block: 1px solid var(--app-border-color, #dce1e7);
}
.document-details .q-tabs {
  background: var(--app-surface-muted, #f6f7f9);
}
.document-scroll {
  overflow-x: auto;
}
.document-scroll table {
  border-collapse: collapse;
  min-width: 100%;
  font-size: 13px;
}
.document-scroll :is(th, td) {
  text-align: start;
  padding: 6px 10px;
  border-bottom: 1px solid #dde2e8;
}
.document-scroll th {
  white-space: nowrap;
  font-weight: 600;
}
.document-scroll td:not(.line-number):not(.line-action) {
  min-width: 144px;
}
.document-scroll .line-number,
.document-scroll .line-action {
  width: 42px;
  white-space: nowrap;
  text-align: center;
  color: var(--app-text-secondary, #586578);
}
.document-scroll .document-empty {
  text-align: center;
  padding: 18px;
  color: var(--app-text-secondary, #586578);
}
.document-scroll :deep(.q-field__bottom:empty) {
  display: none;
}
.lines-footer,
.document-totals {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  padding: 8px 12px;
}
.lines-footer {
  justify-content: space-between;
}
.document-totals {
  justify-content: flex-end;
  border-top: 1px solid #dde2e8;
  font-variant-numeric: tabular-nums;
}
.document-totals strong {
  margin-right: auto;
}
.document-totals b {
  margin-left: 6px;
}
</style>
