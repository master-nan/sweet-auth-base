<template>
  <div :class="pageClass" :style="gridStyle">
    <q-card flat bordered class="master-detail-card master-card">
      <q-card-section class="master-detail-head">
        <div class="master-detail-title-wrap">
          <slot name="master-title">
            <div class="master-detail-title">{{ masterTitle }}</div>
            <div v-if="masterSubtitle" class="master-detail-subtitle">{{ masterSubtitle }}</div>
          </slot>
        </div>
        <q-space />
        <div class="master-detail-actions">
          <slot name="master-actions" />
        </div>
      </q-card-section>

      <slot name="master-toolbar" />

      <q-card-section class="master-detail-content master-content">
        <slot name="master-content" />
      </q-card-section>

      <slot name="master-footer" />
    </q-card>

    <q-card flat bordered class="master-detail-card detail-card">
      <div class="detail-context">
        <slot name="detail-context">
          <div class="master-detail-title-wrap">
            <div class="master-detail-title">{{ detailTitle }}</div>
            <div v-if="detailSubtitle" class="master-detail-subtitle">{{ detailSubtitle }}</div>
          </div>
        </slot>
      </div>

      <slot name="detail-toolbar" />

      <q-card-section class="master-detail-content detail-content">
        <slot name="detail-content" />
      </q-card-section>

      <slot name="detail-footer" />
    </q-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  MasterDetailDisplayMode,
  type MasterDetailDisplayMode as MasterDetailDisplayModeValue,
} from './types'

defineOptions({ name: 'MasterDetailPage' })

const props = withDefaults(
  defineProps<{
    mode?: MasterDetailDisplayModeValue
    masterTitle?: string
    masterSubtitle?: string
    detailTitle?: string
    detailSubtitle?: string
    masterWidth?: string
    masterHeight?: string
    minWidth?: string
    minHeight?: string
  }>(),
  {
    mode: MasterDetailDisplayMode.SUMMARY,
    masterTitle: '',
    masterSubtitle: '',
    detailTitle: '',
    detailSubtitle: '',
    masterWidth: '',
    masterHeight: '',
    minWidth: '',
    minHeight: 'calc(100vh - 132px)',
  },
)

const normalizedMode = computed(() => {
  const modes = Object.values(MasterDetailDisplayMode)
  return modes.includes(props.mode) ? props.mode : MasterDetailDisplayMode.SUMMARY
})

const resolvedMasterWidth = computed(() => {
  if (props.masterWidth) return props.masterWidth
  if (normalizedMode.value === MasterDetailDisplayMode.TABLE) return 'minmax(520px, 44%)'
  return '380px'
})

const resolvedMasterHeight = computed(() => {
  if (props.masterHeight) return props.masterHeight
  if (normalizedMode.value === MasterDetailDisplayMode.STACKED) return '42%'
  return 'auto'
})

const resolvedMinWidth = computed(() => {
  if (props.minWidth) return props.minWidth
  if (normalizedMode.value === MasterDetailDisplayMode.TABLE) return '1180px'
  if (normalizedMode.value === MasterDetailDisplayMode.STACKED) return '980px'
  return '960px'
})

const pageClass = computed(() => [
  'master-detail-page',
  `master-detail-page--${normalizedMode.value}`,
])

const gridStyle = computed(() => ({
  '--master-detail-master-width': resolvedMasterWidth.value,
  '--master-detail-master-height': resolvedMasterHeight.value,
  '--master-detail-min-width': resolvedMinWidth.value,
  '--master-detail-min-height': props.minHeight,
}))
</script>

<style scoped lang="scss">
.master-detail-page {
  width: 100%;
  min-width: var(--master-detail-min-width);
  height: var(--master-detail-min-height);
  min-height: var(--master-detail-min-height);
  display: grid;
  align-items: stretch;
  gap: 14px;
}

.master-detail-page--summary,
.master-detail-page--table {
  grid-template-columns: var(--master-detail-master-width) minmax(0, 1fr);
}

.master-detail-page--stacked {
  grid-template-rows: minmax(220px, var(--master-detail-master-height)) minmax(0, 1fr);
  grid-template-columns: minmax(0, 1fr);
}

.master-detail-card {
  min-height: 0;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-color: #e3e8f2;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 12px 32px rgba(15, 23, 42, 0.07);
}

.master-detail-head {
  min-height: 62px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid #e3e8f2;
  background: linear-gradient(180deg, #ffffff, #fbfcff);
}

.detail-context {
  min-height: 74px;
  padding: 14px;
  border-bottom: 1px solid #e3e8f2;
  background: #ffffff;
}

.master-detail-page--table .master-detail-head,
.master-detail-page--stacked .master-detail-head {
  min-height: 56px;
}

.master-detail-page--table .detail-context,
.master-detail-page--stacked .detail-context {
  min-height: 60px;
}

.master-detail-title-wrap {
  min-width: 0;
}

.master-detail-title {
  overflow: hidden;
  color: #172033;
  font-size: 17px;
  font-weight: 800;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.master-detail-subtitle {
  margin-top: 4px;
  overflow: hidden;
  color: #657189;
  font-size: 12px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.master-detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.master-detail-content {
  flex: 1;
  min-height: 0;
  padding: 0;
  overflow: hidden;
}

@media (max-width: 1023px) {
  .master-detail-page--summary,
  .master-detail-page--table {
    grid-template-columns: var(--master-detail-master-width) minmax(0, 1fr);
  }

  .master-detail-card {
    min-height: 0;
  }
}
</style>
