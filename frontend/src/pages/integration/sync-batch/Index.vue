<template>
  <base-content class="q-pa-sm">
    <q-table v-model:pagination="pagination" class="fit sticky-header-table" color="primary" :dense="$q.screen.lt.md" separator="cell" flat bordered :rows="rows" :columns="columns" row-key="id" :loading="loading">
      <template #top><div class="row q-gutter-xs full-width items-center"><q-input v-model="query.quick_query!.keyword" dense outlined debounce="300" placeholder="搜索批次编号或任务" @keyup.enter="resetAndFetch"><template #append><q-icon name="search" /></template></q-input><q-select v-model="query.status" dense outlined emit-value map-options clearable :options="statusOptions" label="状态" style="min-width:140px" @update:model-value="resetAndFetch" /><q-select v-model="query.trigger_type" dense outlined emit-value map-options clearable :options="triggerOptions" label="触发类型" style="min-width:140px" @update:model-value="resetAndFetch" /><q-btn color="primary" label="搜索" @click="resetAndFetch" /><q-space /></div></template>
      <template #body-cell-batch_no="props"><q-td :props="props"><q-btn v-if="canDetail" flat dense no-caps color="primary" :label="props.row.batch_no" @click="openDetail(props.row.id)" /><span v-else class="text-mono">{{ props.row.batch_no }}</span><div class="text-caption text-grey-7">{{ props.row.task_name }} · v{{ props.row.task_version }}</div></q-td></template>
      <template #body-cell-status="props"><q-td :props="props"><q-chip dense square outline :color="statusFor(props.row).color" :label="statusFor(props.row).label" /></q-td></template>
      <template #body-cell-window="props"><q-td :props="props">{{ props.row.window_start || '-' }}<div class="text-caption text-grey-7">至 {{ props.row.window_end || '-' }}</div></q-td></template>
      <template #body-cell-progress="props"><q-td :props="props">{{ props.row.current_slice_no }} / {{ props.row.planned_slice_count }}<div class="text-caption text-grey-7">Execution {{ props.row.execution_count }}</div></q-td></template>
      <template #body-cell-result="props"><q-td :props="props">技术 {{ props.row.technical_success_count }} / {{ props.row.technical_failed_count }}<div class="text-caption text-grey-7">业务 {{ props.row.business_success_count }} / {{ props.row.business_failed_count }}</div></q-td></template>
      <template #bottom><q-space /><table-pagination v-model:page="query.page" v-model:page-size="query.num" :total="total" /></template>
    </q-table>
    <form-dialog-shell v-model="showDetail" title="同步批次详情" :subtitle="detail?.batch_no || '正在读取批次'" icon="view_timeline" readonly :loading="detailLoading" width="min(960px, calc(100vw - 48px))"><div v-if="detail" class="row q-col-gutter-lg"><div v-for="item in detailItems" :key="item.label" class="col-12 col-md-4"><div class="text-caption text-grey-7">{{ item.label }}</div><div class="text-body1">{{ item.value }}</div></div></div></form-dialog-shell>
  </base-content>
</template>
<script setup lang="ts">
defineOptions({ name: 'integration_sync_batch' })
import { computed, onMounted, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import BaseContent from 'src/components/BaseContent/BaseContent.vue'
import TablePagination from 'src/components/Table/TablePagination.vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { type SyncBatchDetail, type SyncBatchListItem, type SyncBatchQuery, useIntegrationApi } from 'src/api/services/integration'
import { useUserStore } from 'src/stores/user'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
const $q = useQuasar(); const api = useIntegrationApi(); const userStore = useUserStore(); const { loading } = storeToRefs(useLoadingStore()); const rows = ref<SyncBatchListItem[]>([]); const total = ref(0); const initialized = ref(false); const showDetail = ref(false); const detailLoading = ref(false); const detail = ref<SyncBatchDetail | null>(null)
const canDetail = computed(() => userStore.buttons.includes('integration_sync_batch_detail'))
const statusMeta = { created: { label: '待运行', color: 'grey-7' }, running: { label: '运行中', color: 'primary' }, succeeded: { label: '成功', color: 'positive' }, failed: { label: '失败', color: 'negative' } }
const statusFor = (row: SyncBatchListItem) => statusMeta[row.status]
const statusOptions = Object.entries(statusMeta).map(([value, item]) => ({ label: item.label, value })); const triggerOptions = [{ label: '手工', value: 'manual' }, { label: '定时', value: 'scheduled' }]
const columns: QTableProps['columns'] = [{ name: 'batch_no', label: '批次', field: 'batch_no', align: 'left', sortable: true }, { name: 'trigger_type', label: '触发类型', field: 'trigger_type', align: 'center', sortable: true }, { name: 'status', label: '状态', field: 'status', align: 'center', sortable: true }, { name: 'window', label: '时间窗口', field: 'window_start', align: 'left', sortable: true }, { name: 'checkpoint_before', label: 'Checkpoint', field: 'checkpoint_before', align: 'left' }, { name: 'progress', label: '切片进度', field: 'current_slice_no', align: 'center' }, { name: 'result', label: '成功 / 失败', field: 'technical_success_count', align: 'center' }, { name: 'started_at', label: '开始时间', field: 'started_at', align: 'left', sortable: true }, { name: 'completed_at', label: '结束时间', field: 'completed_at', align: 'left', sortable: true }]
const query = ref<SyncBatchQuery>({ page: 1, num: 15, order: { field: '', is_asc: false }, quick_query: { keyword: '' }, expressions: [{ rules: [{ field: '', value: null }], nested: [] }] }); const pagination = ref({ page: 1, rowsPerPage: 0, sortBy: '', descending: true })
const fetchData = async () => { const result = await api.querySyncBatches(query.value); rows.value = result.data || []; total.value = result.total || 0 }; const resetAndFetch = () => { if (query.value.page !== 1) query.value.page = 1; else void fetchData() }
const openDetail = async (id: number) => { if (!canDetail.value) return; showDetail.value = true; detailLoading.value = true; try { detail.value = (await api.getSyncBatch(id)).data || null } finally { detailLoading.value = false } }
const detailItems = computed(() => detail.value ? [{ label: '任务版本', value: `${detail.value.task_code} · v${detail.value.task_version}` }, { label: '接口版本', value: `${detail.value.interface_code} · v${detail.value.interface_version}` }, { label: 'Consumer', value: `${detail.value.consumer_code} · v${detail.value.consumer_version}` }, { label: 'Checkpoint 模式', value: detail.value.checkpoint_mode }, { label: '切片进度', value: `${detail.value.current_slice_no} / ${detail.value.planned_slice_count}` }, { label: '结果摘要', value: detail.value.result_summary || detail.value.reason_code || '-' }] : [])
onMounted(async () => { await fetchData(); initialized.value = true }); watch(() => [query.value.page, query.value.num] as const, () => { if (initialized.value) void fetchData() }); watch(() => [pagination.value.sortBy, pagination.value.descending] as const, ([field, descending]) => { if (!initialized.value) return; query.value.order = { field: field || '', is_asc: field ? !descending : false }; resetAndFetch() })
</script>
