<template>
  <q-page class="report-v2-prototype">
    <div class="prototype-shell">
      <div class="prototype-header">
        <div>
          <div class="eyebrow">Report V2 Prototype</div>
          <h1>报表模块 V2 UI 原型</h1>
          <p>
            用静态 mock 验证报表设计管理工作台、发布到菜单、通用运行页和版式报表设计流程。
          </p>
        </div>
        <q-chip square color="blue-1" text-color="primary" icon="science">
          静态原型，不接真实接口
        </q-chip>
      </div>

      <q-btn-toggle
        v-model="activeView"
        class="prototype-tabs"
        unelevated
        spread
        no-caps
        toggle-color="primary"
        color="white"
        text-color="grey-8"
        :options="viewOptions"
      />

      <div class="prototype-body">
        <PrototypeWorkbench v-if="activeView === 'workbench'" />
        <PrototypeWizard v-else-if="activeView === 'wizard'" />
        <PrototypeLayoutDesigner v-else-if="activeView === 'layout'" />
        <PrototypeAdvancedDesigner v-else-if="activeView === 'advanced'" />
        <PrototypeRuntime v-else />
      </div>
    </div>
  </q-page>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import PrototypeAdvancedDesigner from './components/PrototypeAdvancedDesigner.vue'
import PrototypeLayoutDesigner from './components/PrototypeLayoutDesigner.vue'
import PrototypeRuntime from './components/PrototypeRuntime.vue'
import PrototypeWorkbench from './components/PrototypeWorkbench.vue'
import PrototypeWizard from './components/PrototypeWizard.vue'

const activeView = ref('workbench')

const viewOptions = [
  { label: '报表设计管理工作台', value: 'workbench' },
  { label: '新建报表向导', value: 'wizard' },
  { label: '版式报表设计器', value: 'layout' },
  { label: '高级版式设计器', value: 'advanced' },
  { label: '通用报表运行页', value: 'runtime' },
]
</script>

<style scoped lang="scss">
.report-v2-prototype {
  min-height: 100%;
  background: #f5f7fb;
  color: #172033;
}

.prototype-shell {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
  min-width: 1180px;
  max-width: 1680px;
  margin: 0 auto;
  padding: 20px;
}

.prototype-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px;
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;

  h1 {
    margin: 4px 0 8px;
    font-size: 26px;
    font-weight: 700;
    line-height: 1.25;
  }

  p {
    max-width: 760px;
    margin: 0;
    color: #5f6b7a;
    font-size: 14px;
    line-height: 1.7;
  }
}

.eyebrow {
  color: #2f6fed;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

.prototype-tabs {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  overflow: hidden;
  background: #fff;
}

.prototype-body {
  min-height: 680px;
}
</style>
