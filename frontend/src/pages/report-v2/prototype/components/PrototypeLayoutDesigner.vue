<template>
  <div class="layout-designer-page">
    <div class="designer-topbar">
      <div class="topbar-left">
        <q-btn flat dense icon="arrow_back" label="返回" />
        <div>
          <div class="designer-title">供应商月度对账单</div>
          <div class="designer-meta">版式报表 · 可发布到左侧菜单 · 通用运行页渲染</div>
        </div>
        <q-chip dense square color="orange-1" text-color="warning" label="草稿" />
        <q-chip dense square color="grey-2" text-color="grey-8" label="线上版本 V3" />
      </div>
      <div class="topbar-actions">
        <q-btn outline color="primary" icon="save" label="保存草稿" />
        <q-btn outline color="primary" icon="visibility" label="保存并预览" />
        <q-btn color="primary" unelevated icon="publish" label="发布" />
        <q-btn outline color="primary" icon="account_tree" label="发布到菜单" />
        <q-btn flat color="grey-8" icon="history" label="版本" />
      </div>
    </div>

    <div class="designer-layout">
      <aside class="steps-card">
        <div class="steps-title">设计步骤</div>
        <div v-for="step in steps" :key="step.label" class="step-item" :class="{ active: step.active }">
          <q-icon :name="step.icon" size="18px" />
          <div>
            <div>{{ step.label }}</div>
            <span>{{ step.desc }}</span>
          </div>
        </div>
      </aside>

      <main class="designer-main">
        <section class="main-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">版式区域结构</div>
              <div class="panel-desc">围绕标题、查询条件展示、表头、明细、分组、汇总和页脚组织报表，而不是生成普通列表页。</div>
            </div>
            <q-btn color="primary" unelevated icon="auto_fix_high" label="根据元数据生成版式草稿" />
          </div>

          <div class="layout-canvas">
            <div class="layout-region title-region">
              <div class="region-label">标题区</div>
              <strong>供应商月度对账单</strong>
            </div>
            <div class="layout-region condition-region">
              <div class="region-label">查询条件展示区</div>
              <span>账期：{{ startDate }} 至 {{ endDate }} ｜ 供应商：全部 ｜ 状态：全部</span>
            </div>
            <div class="layout-region header-region">
              <div class="region-label">表头区</div>
              <div class="layout-row header-row">
                <span>供应商</span>
                <span>账期</span>
                <span>应付金额</span>
                <span>已付金额</span>
                <span>状态</span>
              </div>
            </div>
            <div class="layout-region detail-region">
              <div class="region-label">明细区</div>
              <div class="layout-row detail-row">
                <span v-text="'{{ supplier_name }}'" />
                <span v-text="'{{ period }}'" />
                <span v-text="'{{ payable_amount }}'" />
                <span v-text="'{{ paid_amount }}'" />
                <span v-text="'{{ status | dict }}'" />
              </div>
              <div class="detail-hint">明细区会按运行结果自动展开，应用菜单权限和数据权限。</div>
            </div>
            <div class="layout-region group-region">
              <div class="region-label">分组区</div>
              <span>按供应商 / 账期分组展示，复杂一对多关系第一阶段建议通过 SQL、视图或预聚合结果解决。</span>
            </div>
            <div class="layout-region summary-region">
              <div class="region-label">汇总区</div>
              <span>合计应付：SUM(payable_amount) ｜ 合计已付：SUM(paid_amount)</span>
            </div>
            <div class="layout-region footer-region">
              <div class="region-label">页脚区</div>
              <span>制表人：系统生成 ｜ 运行版本：V3 ｜ 导出受后端限制</span>
            </div>
          </div>
        </section>

        <section class="main-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">字段绑定与聚合</div>
              <div class="panel-desc">绑定字段来自系统表元数据或受控 SQL 数据集，字段控件和字典优先复用低代码能力。</div>
            </div>
          </div>
          <q-table
            flat
            bordered
            dense
            row-key="id"
            :rows="bindingRows"
            :columns="bindingColumns"
            :pagination="{ rowsPerPage: 0 }"
            hide-pagination
          >
            <template #body-cell-aggregate="props">
              <q-td :props="props">
                <q-chip v-if="props.row.aggregate" dense square color="blue-1" text-color="primary" :label="props.row.aggregate" />
                <span v-else class="muted">-</span>
              </q-td>
            </template>
          </q-table>
        </section>

        <section class="main-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">查询参数控件</div>
              <div class="panel-desc">参数定义来自 report parameters，控件类型从 sys_table_field 和 dict 推导。</div>
            </div>
          </div>
          <div class="parameter-grid">
            <div v-for="item in parameters" :key="item.id" class="parameter-card">
              <q-icon name="filter_alt" color="primary" />
              <div>
                <div class="parameter-label">{{ item.label }}</div>
                <div class="parameter-meta">{{ item.control }} · {{ item.field }} {{ item.operator }}</div>
                <div class="parameter-source">{{ item.dictCode || item.sourceMeta }}</div>
              </div>
            </div>
          </div>
        </section>

        <section class="main-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">静态运行预览</div>
              <div class="panel-desc">真实实现中会先保存草稿，再调用设计时预览接口读取数据库中的草稿配置。</div>
            </div>
            <div class="preview-actions">
              <q-btn outline color="primary" icon="visibility" label="保存并预览" />
              <q-btn color="primary" unelevated icon="publish" label="发布" />
            </div>
          </div>
          <q-table
            flat
            bordered
            dense
            row-key="order_no"
            :rows="runtimeRows"
            :columns="previewColumns"
            :pagination="{ rowsPerPage: 4 }"
          />
        </section>
      </main>

      <aside class="assist-card">
        <section class="assist-section">
          <div class="assist-title">当前步骤说明</div>
          <p>先定义版式区域，再把字段和聚合绑定到区域。这里关注固定格式输出，不重复低代码普通表格页面。</p>
        </section>

        <section class="assist-section">
          <div class="assist-title">数据来源摘要</div>
          <div class="summary-item">
            <span>来源</span>
            <strong>系统表 supplier_statement</strong>
          </div>
          <div class="summary-item">
            <span>字段元数据</span>
            <strong>sys_table_field</strong>
          </div>
          <div class="summary-item">
            <span>字典控件</span>
            <strong>dict status</strong>
          </div>
        </section>

        <section class="assist-section">
          <div class="assist-title">设计统计</div>
          <div class="summary-item">
            <span>版式区域</span>
            <strong>{{ regions.length }}</strong>
          </div>
          <div class="summary-item">
            <span>字段绑定</span>
            <strong>{{ bindingRows.length }}</strong>
          </div>
          <div class="summary-item">
            <span>聚合字段</span>
            <strong>{{ aggregateCount }}</strong>
          </div>
          <div class="summary-item">
            <span>查询参数</span>
            <strong>{{ parameters.length }}</strong>
          </div>
        </section>

        <section class="assist-section">
          <div class="assist-title">发布检查摘要</div>
          <div class="check-item ok">
            <q-icon name="check_circle" />
            <span>已配置发布版本快照</span>
          </div>
          <div class="check-item ok">
            <q-icon name="check_circle" />
            <span>可挂载到左侧菜单</span>
          </div>
          <div class="check-item ok">
            <q-icon name="check_circle" />
            <span>运行页根据 report_id 自动渲染</span>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { parameters, runtimeRows } from '../../mock'

const startDate = ref('2026-06-01')
const endDate = ref('2026-06-30')

const steps = [
  { label: '基本信息', desc: '名称、编码、分类', icon: 'article' },
  { label: '数据来源', desc: '系统表或受控 SQL', icon: 'storage' },
  { label: '区域设计', desc: '标题、明细、汇总', icon: 'view_quilt', active: true },
  { label: '字段绑定', desc: '字段、字典、聚合', icon: 'link' },
  { label: '查询参数', desc: '元数据控件', icon: 'filter_alt' },
  { label: '预览发布', desc: '版本快照和菜单', icon: 'rocket_launch' },
]

const regions = ['标题区', '查询条件展示区', '表头区', '明细区', '分组区', '汇总区', '页脚区']

const bindingRows = [
  { id: 1, region: '标题区', item: '报表标题', binding: '静态文本', sourceType: 'static', aggregate: '', format: '文本' },
  { id: 2, region: '查询条件展示区', item: '账期范围', binding: 'period_range', sourceType: 'parameter', aggregate: '', format: '日期范围' },
  { id: 3, region: '明细区', item: '供应商名称', binding: 'supplier_name', sourceType: 'field', aggregate: '', format: '文本' },
  { id: 4, region: '明细区', item: '应付金额', binding: 'payable_amount', sourceType: 'field', aggregate: '', format: '金额' },
  { id: 5, region: '明细区', item: '状态', binding: 'status', sourceType: 'dict field', aggregate: '', format: '字典标签' },
  { id: 6, region: '汇总区', item: '合计应付', binding: 'payable_amount', sourceType: 'field', aggregate: 'SUM', format: '金额' },
]

const bindingColumns: QTableProps['columns'] = [
  { name: 'region', label: '区域', field: 'region', align: 'left' },
  { name: 'item', label: '元素', field: 'item', align: 'left' },
  { name: 'binding', label: '绑定字段', field: 'binding', align: 'left' },
  { name: 'sourceType', label: '来源类型', field: 'sourceType', align: 'center' },
  { name: 'aggregate', label: '聚合', field: 'aggregate', align: 'center' },
  { name: 'format', label: '格式', field: 'format', align: 'center' },
]

const previewColumns: QTableProps['columns'] = [
  { name: 'customer_name', label: '客户名称', field: 'customer_name', align: 'left' },
  { name: 'order_no', label: '订单编号', field: 'order_no', align: 'left' },
  { name: 'amount', label: '订单金额', field: 'amount', align: 'right' },
  { name: 'created_at', label: '创建时间', field: 'created_at', align: 'center' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
]

const aggregateCount = computed(() => bindingRows.filter((row) => row.aggregate).length)
</script>

<style scoped lang="scss">
.layout-designer-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.designer-topbar,
.steps-card,
.main-panel,
.assist-card {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
}

.designer-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
}

.topbar-left,
.topbar-actions,
.panel-head,
.preview-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.topbar-left {
  flex: 1;
}

.topbar-actions {
  flex-wrap: wrap;
  justify-content: flex-end;
}

.designer-title {
  color: #172033;
  font-size: 18px;
  font-weight: 700;
}

.designer-meta {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
}

.designer-layout {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) 300px;
  gap: 12px;
}

.steps-card,
.assist-card,
.main-panel {
  padding: 14px;
}

.steps-title,
.panel-title,
.assist-title {
  color: #172033;
  font-weight: 700;
}

.steps-title {
  margin-bottom: 10px;
}

.step-item {
  display: flex;
  gap: 10px;
  padding: 10px;
  border-radius: 8px;
  color: #4d5b70;

  span {
    color: #7a8699;
    font-size: 12px;
  }

  &.active {
    background: #eef4ff;
    color: #1d4ed8;
    font-weight: 700;
  }
}

.designer-main {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel-head {
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-desc {
  color: #667085;
  font-size: 12px;
  line-height: 1.6;
}

.layout-canvas {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #f8fafc;
}

.layout-region {
  position: relative;
  padding: 16px 14px 12px;
  border: 1px solid #d7dee9;
  border-radius: 6px;
  background: #fff;
}

.region-label {
  position: absolute;
  top: -10px;
  left: 10px;
  padding: 0 6px;
  background: #fff;
  color: #2f6fed;
  font-size: 12px;
  font-weight: 700;
}

.title-region {
  text-align: center;
  font-size: 20px;
}

.condition-region,
.footer-region {
  color: #5f6b7a;
  font-size: 13px;
}

.layout-row {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr 1fr 0.8fr;
  gap: 6px;

  span {
    padding: 8px;
    border-radius: 4px;
    background: #f1f5f9;
  }
}

.header-row span {
  background: #e8f1ff;
  color: #1d4ed8;
  font-weight: 700;
  text-align: center;
}

.detail-row span {
  color: #475467;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.detail-hint {
  margin-top: 8px;
  color: #7a8699;
  font-size: 12px;
}

.summary-region {
  background: #fffaf0;
}

.group-region {
  background: #f0f9ff;
  color: #0369a1;
}

.parameter-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.parameter-card {
  display: flex;
  gap: 10px;
  padding: 10px;
  border: 1px solid #edf2f7;
  border-radius: 8px;
  background: #f8fafc;
}

.parameter-label {
  color: #172033;
  font-weight: 700;
}

.parameter-meta,
.parameter-source,
.muted {
  color: #7a8699;
  font-size: 12px;
}

.assist-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.assist-section {
  padding-bottom: 12px;
  border-bottom: 1px solid #edf2f7;

  &:last-child {
    border-bottom: 0;
    padding-bottom: 0;
  }

  p {
    margin: 8px 0 0;
    color: #667085;
    font-size: 13px;
    line-height: 1.65;
  }
}

.summary-item,
.check-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 0;
  color: #667085;
  font-size: 13px;

  strong {
    color: #172033;
  }
}

.check-item {
  justify-content: flex-start;

  &.ok {
    color: #16a34a;
  }
}
</style>
