<template>
  <div class="advanced-page">
    <div class="advanced-topbar">
      <div class="topbar-left">
        <q-btn flat dense icon="arrow_back" label="返回" />
        <div>
          <div class="designer-title">订单结算明细版式报表</div>
          <div class="designer-meta">高级版式报表 · 区域化设计 · 用于复杂表头和固定格式输出</div>
        </div>
        <q-chip dense square color="green-1" text-color="positive" label="已发布" />
        <q-chip dense square color="blue-1" text-color="primary" label="当前版本 V4" />
      </div>
      <div class="topbar-actions">
        <q-btn outline color="primary" icon="save" label="保存草稿" />
        <q-btn outline color="primary" icon="visibility" label="保存并预览" />
        <q-btn color="primary" unelevated icon="publish" label="发布" />
        <q-btn outline color="primary" icon="account_tree" label="发布到菜单" />
        <q-btn flat color="grey-8" icon="history" label="版本" />
        <q-btn outline color="warning" icon="swap_horiz" label="切回版式设计">
          <q-tooltip>高级布局切回版式设计需要确认是否覆盖手工区域</q-tooltip>
        </q-btn>
      </div>
    </div>

    <div class="advanced-layout">
      <aside class="resource-panel">
        <section v-for="group in resourceGroups" :key="group.title" class="resource-section">
          <div class="resource-title">{{ group.title }}</div>
          <div v-for="item in group.items" :key="item.label" class="resource-item">
            <q-icon :name="item.icon" size="17px" />
            <span>{{ item.label }}</span>
            <q-chip v-if="item.badge" dense square color="grey-2" text-color="grey-8" :label="item.badge" />
          </div>
        </section>
      </aside>

      <main class="sheet-panel">
        <div class="sheet-head">
          <div>
            <div class="panel-title">高级版式画布</div>
            <div class="panel-desc">
              用区域组织复杂报表。默认给出标题、参数摘要、表头、分组、明细、汇总和页脚，避免用户面对空白画布。
            </div>
          </div>
          <q-chip square color="orange-1" text-color="warning" icon="psychology">
            高级能力，不作为默认入口
          </q-chip>
        </div>

        <div class="sheet-canvas">
          <div class="corner-cell"></div>
          <div v-for="column in sheetColumns" :key="column" class="column-head">{{ column }}</div>
          <template v-for="row in sheetRows" :key="row.index">
            <div class="row-head">{{ row.index }}</div>
            <div
              v-for="cell in row.cells"
              :key="`${row.index}-${cell.text}`"
              class="sheet-cell"
              :class="cell.tone"
              :style="{ gridColumn: `span ${cell.span || 1}` }"
            >
              <small v-if="cell.region">{{ cell.region }}</small>
              <span>{{ cell.text }}</span>
            </div>
          </template>
        </div>
      </main>

      <aside class="inspector-panel">
        <div class="inspector-title">当前区域与单元格属性</div>
        <div class="property-grid">
          <div class="property-item">
            <span>当前区域</span>
            <strong>明细区</strong>
          </div>
          <div class="property-item">
            <span>当前单元格</span>
            <strong>D5</strong>
          </div>
          <div class="property-item">
            <span>绑定类型</span>
            <strong>明细字段</strong>
          </div>
          <div class="property-item">
            <span>数据集</span>
            <strong>order_settlement</strong>
          </div>
          <div class="property-item">
            <span>绑定字段</span>
            <strong>settlement_amount</strong>
          </div>
          <div class="property-item">
            <span>分组字段</span>
            <strong>customer_name</strong>
          </div>
          <div class="property-item">
            <span>格式</span>
            <strong>金额</strong>
          </div>
          <div class="property-item">
            <span>样式</span>
            <strong>右对齐 / 加粗</strong>
          </div>
          <div class="property-item">
            <span>合并区域</span>
            <strong>D5:D6</strong>
          </div>
          <div class="property-item">
            <span>对齐方式</span>
            <strong>右对齐</strong>
          </div>
        </div>
      </aside>
    </div>

    <div class="advanced-footer">
      <span>当前单元格：D5</span>
      <span>当前区域：明细区</span>
      <span>数据集数量：2</span>
      <span>已绑定字段：12</span>
      <span>区域：标题 / 查询条件 / 表头 / 分组 / 明细 / 汇总 / 页脚</span>
    </div>
  </div>
</template>

<script setup lang="ts">
interface SheetCell {
  text: string
  span?: number
  tone?: string
  region?: string
}

interface SheetRow {
  index: number
  cells: SheetCell[]
}

const sheetColumns = ['A', 'B', 'C', 'D', 'E', 'F', 'G']

const resourceGroups = [
  {
    title: '数据集',
    items: [
      { label: 'order_settlement', icon: 'table_view', badge: '主数据集' },
      { label: 'customer_profile', icon: 'dataset_linked' },
    ],
  },
  {
    title: '字段',
    items: [
      { label: '客户名称', icon: 'text_fields' },
      { label: '订单编号', icon: 'tag' },
      { label: '结算金额', icon: 'payments' },
      { label: '签收状态', icon: 'fact_check' },
    ],
  },
  {
    title: '参数',
    items: [
      { label: '结算日期范围', icon: 'date_range' },
      { label: '客户', icon: 'business' },
      { label: '状态字典', icon: 'list_alt' },
    ],
  },
  {
    title: '区域',
    items: [
      { label: '标题区', icon: 'title' },
      { label: '查询条件展示区', icon: 'filter_alt' },
      { label: '表头区', icon: 'view_headline' },
      { label: '分组区', icon: 'account_tree' },
      { label: '明细区', icon: 'format_list_bulleted' },
      { label: '汇总区', icon: 'functions' },
      { label: '页脚区', icon: 'vertical_align_bottom' },
    ],
  },
  {
    title: '组件',
    items: [
      { label: '静态文本', icon: 'notes' },
      { label: '字段', icon: 'input' },
      { label: '汇总', icon: 'calculate' },
      { label: '公式', icon: 'data_object' },
    ],
  },
]

const sheetRows: SheetRow[] = [
  {
    index: 1,
    cells: [{ text: '订单结算明细报表', span: 7, tone: 'title-cell', region: '标题区' }],
  },
  {
    index: 2,
    cells: [
      {
        text: '结算日期：2026-06-01 至 2026-06-30    客户：全部    状态：全部',
        span: 7,
        tone: 'condition-cell',
        region: '查询条件展示区',
      },
    ],
  },
  {
    index: 3,
    cells: [
      { text: '客户', region: '表头区' },
      { text: '订单编号' },
      { text: '结算日期' },
      { text: '结算金额' },
      { text: '优惠金额' },
      { text: '应收金额' },
      { text: '状态' },
    ],
  },
  {
    index: 4,
    cells: [{ text: '客户分组：{{ customer_name }}', span: 7, tone: 'group-cell', region: '分组区' }],
  },
  {
    index: 5,
    cells: [
      { text: '{{ customer_name }}', tone: 'field-cell', region: '明细区' },
      { text: '{{ order_no }}', tone: 'field-cell' },
      { text: '{{ settlement_date }}', tone: 'field-cell' },
      { text: '{{ settlement_amount }}', tone: 'field-cell selected-cell' },
      { text: '{{ discount_amount }}', tone: 'field-cell' },
      { text: '{{ receivable_amount }}', tone: 'field-cell' },
      { text: '{{ status | dict }}', tone: 'field-cell' },
    ],
  },
  {
    index: 6,
    cells: [{ text: '明细行按结果集自动扩展', span: 7, tone: 'detail-cell', region: '明细区' }],
  },
  {
    index: 7,
    cells: [
      { text: '本组小计', span: 3, tone: 'summary-cell', region: '汇总区' },
      { text: 'SUM(结算)', tone: 'summary-cell' },
      { text: 'SUM(优惠)', tone: 'summary-cell' },
      { text: 'SUM(应收)', tone: 'summary-cell' },
      { text: '-', tone: 'summary-cell' },
    ],
  },
  {
    index: 8,
    cells: [{ text: '制表人：系统自动生成    发布版本：V4    数据权限：按组织范围过滤', span: 7, tone: 'footer-cell', region: '页脚区' }],
  },
]
</script>

<style scoped lang="scss">
.advanced-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.advanced-topbar,
.resource-panel,
.sheet-panel,
.inspector-panel,
.advanced-footer {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
}

.advanced-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
}

.topbar-left,
.topbar-actions {
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
  color: #667085;
  font-size: 12px;
}

.advanced-layout {
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr) 300px;
  gap: 12px;
}

.resource-panel,
.sheet-panel,
.inspector-panel {
  padding: 14px;
}

.resource-section {
  margin-bottom: 16px;

  &:last-child {
    margin-bottom: 0;
  }
}

.resource-title,
.panel-title,
.inspector-title {
  color: #172033;
  font-weight: 700;
}

.resource-title {
  margin-bottom: 8px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 8px;
  border-radius: 6px;
  color: #4d5b70;
  font-size: 13px;

  span {
    flex: 1;
  }

  &:hover {
    background: #f1f5f9;
  }
}

.sheet-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.panel-desc {
  color: #667085;
  font-size: 12px;
  line-height: 1.6;
}

.sheet-canvas {
  display: grid;
  grid-template-columns: 42px repeat(7, minmax(90px, 1fr));
  gap: 1px;
  padding: 12px;
  border: 1px solid #d7dee9;
  border-radius: 8px;
  background: #cbd5e1;
  overflow: auto;
}

.corner-cell,
.column-head,
.row-head,
.sheet-cell {
  min-height: 40px;
  background: #fff;
}

.column-head,
.row-head {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.sheet-cell {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
  padding: 8px;
  color: #334155;
  font-size: 13px;
  line-height: 1.35;

  small {
    color: #2f6fed;
    font-size: 11px;
    font-weight: 700;
  }
}

.title-cell {
  align-items: center;
  min-height: 58px;
  color: #172033;
  font-size: 20px;
  font-weight: 700;
}

.condition-cell {
  background: #f8fafc;
}

.group-cell {
  background: #eef4ff;
  color: #1d4ed8;
  font-weight: 700;
}

.field-cell {
  background: #f8fafc;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.selected-cell {
  outline: 2px solid #2f6fed;
  outline-offset: -2px;
}

.detail-cell {
  align-items: center;
  background: #f8fafc;
  color: #7a8699;
  font-style: italic;
}

.summary-cell {
  background: #fff7ed;
  font-weight: 700;
}

.footer-cell {
  background: #f8fafc;
  color: #667085;
}

.property-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.property-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid #edf2f7;
  border-radius: 6px;
  background: #f8fafc;
  color: #667085;
  font-size: 13px;

  strong {
    color: #172033;
  }
}

.advanced-footer {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  padding: 10px 14px;
  color: #4d5b70;
  font-size: 13px;
}
</style>
