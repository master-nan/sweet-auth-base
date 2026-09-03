<template>
  <header class="designer-topbar">
    <div class="brand-block">
      <q-btn flat round dense icon="arrow_back" @click="$emit('back')">
        <q-tooltip>{{ t('ui.returnToReportManagement') }}</q-tooltip>
      </q-btn>
      <div>
        <div class="designer-title">{{ t('ui.reportDesigner') }}</div>
        <div class="designer-subtitle">
          {{ reportName || t('ui.unnamedReport') }} ·
          {{ primarySourceCode || t('ui.noMasterChartBound') }}
        </div>
      </div>
    </div>

    <div class="meta-editor">
      <q-input
        :model-value="reportName"
        dense
        outlined
        :label="t('ui.reportName')"
        @update:model-value="$emit('update:reportName', String($event || ''))"
      />
      <q-input
        :model-value="reportCode"
        dense
        outlined
        :label="t('ui.reportEncoding')"
        @update:model-value="$emit('update:reportCode', String($event || ''))"
      />
      <div class="status-strip">
        <q-chip dense square :color="statusColor" text-color="white">
          {{ statusLabel }}
        </q-chip>
        <q-chip v-if="publishedVersionNo" dense square outline color="primary">
          {{ t('ui.onlineVersionV') }}{{ publishedVersionNo }}
        </q-chip>
      </div>
    </div>

    <div class="topbar-actions">
      <q-btn
        outline
        color="primary"
        icon="tune"
        :label="t('ui.parameters')"
        @click="$emit('addParameter')"
      />
      <q-btn
        outline
        color="primary"
        icon="preview"
        :label="t('ui.saveAndPreview')"
        :disable="previewDisabled"
        :loading="previewing"
        @click="$emit('preview')"
      />
      <q-btn
        outline
        color="primary"
        icon="rule"
        :label="t('ui.verify')"
        @click="$emit('validate')"
      />
      <q-btn
        unelevated
        color="primary"
        icon="save"
        :label="t('ui.saveDraft')"
        :loading="saving"
        @click="$emit('saveDraft')"
      />
      <q-btn
        unelevated
        color="primary"
        icon="publish"
        :label="t('ui.saveAndPublish')"
        :disable="publishDisabled"
        :loading="publishing"
        @click="$emit('publish')"
      />
      <q-btn
        outline
        color="primary"
        icon="history"
        :label="t('ui.version')"
        :disable="versionDisabled"
        @click="$emit('versions')"
      />
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed } from 'vue'
import type { ReportStatus } from '@/api/services/report'

const { t } = useI18n({ useScope: 'global' })

const props = defineProps<{
  reportName: string
  reportCode: string
  primarySourceCode: string | undefined
  reportStatus: ReportStatus
  publishedVersionNo?: number | undefined
  saving: boolean
  previewing?: boolean
  publishing?: boolean
  previewDisabled?: boolean
  publishDisabled?: boolean
  versionDisabled?: boolean
}>()

defineEmits<{
  'update:reportName': [value: string]
  'update:reportCode': [value: string]
  back: []
  addParameter: []
  preview: []
  validate: []
  saveDraft: []
  publish: []
  versions: []
}>()

const statusLabel = computed(() => {
  const labels: Record<ReportStatus, string> = {
    get draft() {
      return t('ui.draft')
    },
    get published() {
      return t('ui.published')
    },
    get disabled() {
      return t('ui.deactivatedStatus')
    },
  }
  return labels[props.reportStatus] || t('ui.draft')
})

const statusColor = computed(() => {
  const colors: Record<ReportStatus, string> = {
    draft: 'grey-7',
    published: 'positive',
    disabled: 'warning',
  }
  return colors[props.reportStatus] || 'grey-7'
})
</script>

<style scoped lang="scss">
.designer-topbar {
  min-width: 0;
  display: grid;
  grid-template-columns: 290px minmax(360px, 1fr) auto;
  align-items: center;
  gap: 14px;
  padding: 10px 18px;
  border-bottom: 1px solid #dfe5f2;
  background: #fff;
}

.brand-block,
.topbar-actions {
  display: flex;
  align-items: center;
}

.brand-block {
  min-width: 0;
  gap: 10px;
}

.designer-title {
  font-size: 20px;
  font-weight: 900;
}

.designer-subtitle {
  margin-top: 3px;
  color: #71809a;
  font-size: 12px;
}

.meta-editor {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) minmax(180px, 0.9fr) auto;
  align-items: center;
  gap: 8px;
}

.status-strip {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.topbar-actions {
  gap: 8px;
}

@media (max-width: 1360px) {
  .designer-topbar {
    grid-template-columns: 240px minmax(300px, 1fr) auto;
  }

  .topbar-actions :deep(.q-btn__content .block) {
    display: none;
  }
}
</style>
