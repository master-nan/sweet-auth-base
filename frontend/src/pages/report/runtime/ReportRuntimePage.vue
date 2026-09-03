<template>
  <div class="report-runtime-page">
    <q-card v-if="reportLoading" flat bordered class="runtime-state-card">
      <q-skeleton type="text" width="260px" />
      <q-skeleton type="rect" height="180px" />
    </q-card>

    <q-card v-else-if="loadError" flat bordered class="runtime-state-card">
      <q-icon name="error_outline" color="negative" size="40px" />
      <div class="text-subtitle1 text-weight-medium">{{ t('ui.failedToLoadReport') }}</div>
      <div class="text-body2 text-grey-7">{{ loadError }}</div>
      <q-btn outline color="primary" icon="arrow_back" :label="t('ui.back')" @click="goBack" />
    </q-card>

    <q-card v-else flat bordered class="runtime-card">
      <report-runtime-view :report="report" :menu-id="routeMenuId" />
    </q-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useReportApi, type Report } from '@/api/services/report'
import ReportRuntimeView from '../components/ReportRuntimeView.vue'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const reportApi = useReportApi()

const report = ref<Report | null>(null)
const reportLoading = ref(false)
const loadError = ref('')

const reportId = computed(() =>
  firstNumber(route.meta.reportId, route.params.id, route.query.report_id),
)
const routeMenuId = computed(() => firstNumber(route.meta.menuId, route.query.menu_id) || 0)

onMounted(() => {
  void loadReport()
})

watch([reportId, routeMenuId], () => {
  void loadReport()
})

async function loadReport() {
  const id = reportId.value
  if (!id) {
    report.value = null
    loadError.value = t('ui.missingReportId')
    return
  }
  reportLoading.value = true
  loadError.value = ''
  try {
    report.value = await reportApi.queryReportById(id, routeMenuId.value).then((res) => res.data)
  } catch (error) {
    report.value = null
    loadError.value =
      error instanceof Error && error.message ? error.message : t('ui.failedToLoadReportDetails')
    $q.notify({ type: 'negative', message: loadError.value })
  } finally {
    reportLoading.value = false
  }
}

function goBack() {
  router.back()
}

function firstNumber(...values: unknown[]): number | undefined {
  for (const value of values) {
    const normalized = Array.isArray(value) ? value[0] : value
    if (normalized === '' || normalized === null || normalized === undefined) continue
    const numberValue = Number(normalized)
    if (Number.isFinite(numberValue) && numberValue > 0) return numberValue
  }
  return undefined
}
</script>

<style scoped lang="scss">
.report-runtime-page {
  height: 100%;
  min-height: 0;
  padding: 12px;
}

.runtime-card {
  display: flex;
  height: 100%;
  min-height: 0;
  border-radius: 8px;
}

.runtime-state-card {
  display: grid;
  min-height: 240px;
  place-items: center;
  align-content: center;
  gap: 12px;
  padding: 24px;
  border-radius: 8px;
}
</style>
