<template>
  <header class="designer-topbar">
    <div class="brand-block">
      <q-btn flat round dense icon="arrow_back" @click="$emit('back')">
        <q-tooltip>返回报表管理</q-tooltip>
      </q-btn>
      <div>
        <div class="designer-title">报表设计器</div>
        <div class="designer-subtitle">
          {{ reportName || '未命名报表' }} · {{ primarySourceCode || '未绑定主表' }}
        </div>
      </div>
    </div>

    <div class="meta-editor">
      <q-input
        :model-value="reportName"
        dense
        outlined
        label="报表名称"
        @update:model-value="$emit('update:reportName', String($event || ''))"
      />
      <q-input
        :model-value="reportCode"
        dense
        outlined
        label="报表编码"
        @update:model-value="$emit('update:reportCode', String($event || ''))"
      />
      <q-select
        :model-value="reportKind"
        dense
        outlined
        emit-value
        map-options
        label="模板模式"
        :options="kindOptions"
        @update:model-value="updateReportKind"
      >
        <q-tooltip>
          明细模板按数据逐行展开，汇总模板优先使用汇总行。
        </q-tooltip>
      </q-select>
    </div>

    <div class="topbar-actions">
      <q-btn outline color="primary" icon="tune" label="参数" @click="$emit('addParameter')" />
      <q-btn outline color="primary" icon="preview" label="预览" @click="$emit('preview')" />
      <q-btn outline color="primary" icon="rule" label="校验" @click="$emit('validate')" />
      <q-btn
        unelevated
        color="primary"
        icon="save"
        label="保存草稿"
        :loading="saving"
        @click="$emit('saveDraft')"
      />
      <q-btn
        unelevated
        color="primary"
        icon="publish"
        label="发布"
        :loading="saving"
        @click="$emit('publish')"
      />
    </div>
  </header>
</template>

<script setup lang="ts">
import type { ReportKind } from 'src/api/services/report'

defineProps<{
  reportName: string
  reportCode: string
  reportKind: ReportKind
  primarySourceCode: string | undefined
  saving: boolean
  kindOptions: Array<{ label: string; value: ReportKind; disable?: boolean }>
}>()

const emit = defineEmits<{
  'update:reportName': [value: string]
  'update:reportCode': [value: string]
  'update:reportKind': [value: ReportKind]
  back: []
  addParameter: []
  preview: []
  validate: []
  saveDraft: []
  publish: []
}>()

function updateReportKind(value: unknown) {
  if (value === 'detail' || value === 'summary') {
    emit('update:reportKind', value)
  }
}
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
  grid-template-columns: minmax(160px, 1fr) minmax(150px, 0.8fr) 140px;
  gap: 8px;
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
