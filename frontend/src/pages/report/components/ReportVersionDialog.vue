<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @hide="clearState"
    @update:model-value="handleDialogValue"
  >
    <q-card class="version-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">{{ t('ui.release') }}</div>
          <div class="dialog-caption">
            {{ t('ui.viewTheReleaseSnapshotOfTheCurrentReportAndOnly') }}
          </div>
        </div>
      </q-card-section>

      <q-card-section class="dialog-body">
        <q-table
          flat
          bordered
          dense
          row-key="id"
          separator="cell"
          class="version-table"
          :rows="versions"
          :columns="columns"
          :loading="loading"
          :pagination="{ rowsPerPage: 0 }"
          hide-pagination
        >
          <template #body-cell-version_no="slotProps">
            <q-td :props="slotProps">
              <strong>V{{ slotProps.row.version_no }}</strong>
            </q-td>
          </template>

          <template #body-cell-status="slotProps">
            <q-td :props="slotProps">
              <q-chip dense square :color="statusColor(slotProps.row.status)" text-color="white">
                {{ statusLabel(slotProps.row.status) }}
              </q-chip>
            </q-td>
          </template>

          <template #body-cell-publisher="slotProps">
            <q-td :props="slotProps">
              {{ publisherName(slotProps.row) }}
            </q-td>
          </template>

          <template #body-cell-change_log="slotProps">
            <q-td :props="slotProps">
              <span class="change-log">{{ slotProps.row.change_log || '-' }}</span>
            </q-td>
          </template>

          <template #body-cell-current="slotProps">
            <q-td :props="slotProps">
              <q-chip
                v-if="isCurrentVersion(slotProps.row)"
                dense
                square
                color="primary"
                text-color="white"
                icon="verified"
              >
                {{ t('ui.currentVersion') }}
              </q-chip>
              <span v-else class="text-grey-6">-</span>
            </q-td>
          </template>

          <template #no-data>
            <div class="empty-state">
              <q-icon
                :name="errorMessage ? 'error_outline' : 'history'"
                size="36px"
                :color="errorMessage ? 'negative' : 'grey-6'"
              />
              <span>{{ emptyText }}</span>
            </div>
          </template>
        </q-table>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat :label="t('ui.close')" @click="handleDialogValue(false)" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useReportApi, type ReportVersion } from 'src/api/services/report'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  modelValue: boolean
  reportId?: number | undefined
  currentVersionId?: number | undefined
  currentVersionNo?: number | undefined
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const $q = useQuasar()
const reportApi = useReportApi()

const versions = ref<ReportVersion[]>([])
const loading = ref(false)
const errorMessage = ref('')
let requestSeq = 0

const columns = computed<QTableProps['columns']>(() => [
  {
    name: 'version_no',
    field: 'version_no',
    get label() {
      return t('ui.versionNumber')
    },
    align: 'left',
  },
  {
    name: 'status',
    field: 'status',
    get label() {
      return t('ui.status')
    },
    align: 'left',
  },
  {
    name: 'published_at',
    field: (row: ReportVersion) => row.published_at || '-',
    get label() {
      return t('ui.releaseTime')
    },
    align: 'left',
  },
  {
    name: 'publisher',
    field: publisherName,
    get label() {
      return t('ui.publisher')
    },
    align: 'left',
  },
  {
    name: 'change_log',
    field: 'change_log',
    get label() {
      return t('ui.issuanceOfNotes')
    },
    align: 'left',
  },
  {
    name: 'current',
    field: (row: ReportVersion) => isCurrentVersion(row),
    get label() {
      return t('ui.currentVersion')
    },
    align: 'left',
  },
])

const emptyText = computed(() => {
  if (errorMessage.value) return errorMessage.value
  if (!props.reportId) return t('ui.pleaseSelectTheReportAndSeeTheVersion')
  return t('ui.noReleaseVersionAvailable')
})

watch(
  () => props.modelValue,
  (visible) => {
    if (visible) {
      void loadVersions()
      return
    }
    clearState()
  },
  { immediate: true },
)

watch(
  () => props.reportId,
  () => {
    if (props.modelValue) void loadVersions()
  },
)

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
  if (!value) clearState()
}

async function loadVersions() {
  requestSeq += 1
  const currentRequest = requestSeq
  versions.value = []
  errorMessage.value = ''

  if (!props.reportId) {
    loading.value = false
    return
  }

  loading.value = true
  try {
    const items = await reportApi.queryReportVersions(props.reportId)
    if (currentRequest !== requestSeq || !props.modelValue) return
    versions.value = items
  } catch (error) {
    if (currentRequest !== requestSeq || !props.modelValue) return
    const message =
      error instanceof Error && error.message ? error.message : t('ui.failedToLoadVersionList')
    errorMessage.value = message
    $q.notify({
      type: 'negative',
      position: 'top-right',
      message,
    })
  } finally {
    if (currentRequest === requestSeq) {
      loading.value = false
    }
  }
}

function clearState() {
  requestSeq += 1
  versions.value = []
  errorMessage.value = ''
  loading.value = false
}

function publisherName(version: ReportVersion) {
  return version.published_name || version.published_by_name || version.published_by || '-'
}

function isCurrentVersion(version: ReportVersion) {
  if (version.is_current) return true
  if (props.currentVersionId !== undefined && version.id === props.currentVersionId) return true
  return props.currentVersionNo !== undefined && version.version_no === props.currentVersionNo
}

function statusLabel(status: string) {
  const labels: Record<string, string> = {
    get draft() {
      return t('ui.draft')
    },
    get published() {
      return t('ui.published')
    },
    get archived() {
      return t('ui.archived')
    },
    get disabled() {
      return t('ui.deactivatedStatus')
    },
  }
  return labels[status] || status || '-'
}

function statusColor(status: string) {
  const colors: Record<string, string> = {
    draft: 'grey-7',
    published: 'positive',
    archived: 'grey-6',
    disabled: 'warning',
  }
  return colors[status] || 'grey-7'
}
</script>

<style scoped lang="scss">
.version-dialog {
  width: min(980px, 94vw);
  max-height: min(760px, 88vh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dialog-head {
  flex: 0 0 auto;
  padding: 24px 28px;
  border-bottom: 1px solid #e7ecf6;
}

.dialog-title {
  font-size: 20px;
  font-weight: 900;
}

.dialog-caption {
  color: #71809a;
}

.dialog-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  padding: 20px 28px;
}

.version-dialog :deep(.q-card__actions) {
  flex: 0 0 auto;
  padding: 18px 28px;
  border-top: 1px solid #e7ecf6;
  background: #fff;
}

.version-table {
  max-height: 520px;
}

.change-log {
  display: inline-block;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.empty-state {
  width: 100%;
  min-height: 180px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #71809a;
}
</style>
