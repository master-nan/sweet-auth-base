<template>
  <div class="runtime-page">
    <div class="runtime-header">
      <div>
        <div class="menu-title">当前菜单：{{ report.menuName }}</div>
        <div class="runtime-title">{{ report.name }}</div>
        <div class="runtime-meta">
          <q-chip dense square color="blue-1" text-color="primary" label="当前版本 V3" />
          <span>{{ report.category }}</span>
          <span>{{ report.source }}</span>
          <span>{{ report.menuPath }}</span>
        </div>
      </div>
      <div class="runtime-actions">
        <q-btn outline color="primary" icon="refresh" label="刷新" />
        <q-btn color="primary" unelevated icon="file_download" label="导出 CSV" />
      </div>
    </div>

    <div class="binding-banner">
      <q-icon name="account_tree" color="primary" size="22px" />
      <div>
        <strong>此页面由菜单绑定报表定义自动渲染</strong>
        <span>菜单只保存 report_id / report_code，通用运行页读取已发布版本并调用 /admin/report/:id/run。</span>
      </div>
    </div>

    <div class="runtime-layout">
      <main class="runtime-main">
        <section class="filter-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">查询参数</div>
              <div class="panel-desc">参数控件来自 report parameters + sys_table_field 元数据 + dict，不在报表模块重复造控件体系。</div>
            </div>
            <div class="filter-actions">
              <q-btn outline color="primary" icon="restart_alt" label="重置" />
              <q-btn color="primary" unelevated icon="search" label="查询" />
            </div>
          </div>
          <div class="filter-grid">
            <q-input v-model="filters.startDate" dense outlined label="开始日期">
              <template #prepend>
                <q-icon name="event" />
              </template>
              <template #hint>sys_table_field.created_at · date</template>
            </q-input>
            <q-input v-model="filters.endDate" dense outlined label="结束日期">
              <template #prepend>
                <q-icon name="event" />
              </template>
              <template #hint>sys_table_field.created_at · date</template>
            </q-input>
            <q-input v-model="filters.customerName" dense outlined label="客户名称">
              <template #hint>sys_table_field.customer_name · varchar</template>
            </q-input>
            <q-select v-model="filters.status" dense outlined label="状态" :options="statusOptions">
              <template #hint>dict: order_status</template>
            </q-select>
          </div>
          <div class="parameter-source-list">
            <div v-for="item in parameters" :key="item.id" class="parameter-source-item">
              <q-icon name="input" color="primary" />
              <span>{{ item.label }}</span>
              <q-chip dense square color="grey-2" text-color="grey-8" :label="item.control" />
              <small>{{ item.dictCode || item.sourceMeta }}</small>
            </div>
          </div>
        </section>

        <section class="result-panel">
          <div class="panel-head">
            <div>
              <div class="panel-title">查询结果</div>
              <div class="panel-desc">共 128 条，当前展示第 1 页。实际结果由发布版本的版式配置渲染。</div>
            </div>
            <q-chip square color="green-1" text-color="positive" icon="verified">数据权限已应用</q-chip>
          </div>
          <q-table
            flat
            bordered
            row-key="order_no"
            :rows="runtimeRows"
            :columns="columns"
            :pagination="{ rowsPerPage: 5 }"
          />
          <div class="pagination-row">
            <span>总数：128</span>
            <q-pagination v-model="page" :max="13" max-pages="6" direction-links boundary-links color="primary" />
          </div>
        </section>
      </main>

      <aside class="runtime-summary">
        <section class="summary-section">
          <div class="summary-title">菜单绑定</div>
          <div class="summary-item">
            <span>menu_name</span>
            <strong>{{ report.menuName }}</strong>
          </div>
          <div class="summary-item">
            <span>report_id</span>
            <strong>{{ report.id }}</strong>
          </div>
          <div class="summary-item">
            <span>report_code</span>
            <strong>{{ report.code }}</strong>
          </div>
          <div class="summary-item">
            <span>运行组件</span>
            <strong>ReportRuntimePage</strong>
          </div>
        </section>

        <section class="summary-section">
          <div class="summary-title">运行上下文</div>
          <div class="summary-item">
            <span>本次运行版本</span>
            <strong>V3</strong>
          </div>
          <div class="summary-item">
            <span>查询耗时</span>
            <strong>128 ms</strong>
          </div>
          <div class="summary-item">
            <span>导出限制</span>
            <strong>最多 10000 行</strong>
          </div>
          <div class="summary-item">
            <span>运行来源</span>
            <strong>发布版本快照</strong>
          </div>
        </section>

        <section class="summary-section">
          <div class="summary-title">版本隔离提示</div>
          <div class="hint-box">
            这里运行的是 V3 发布版本。设计人员继续修改草稿时，不会影响业务用户从菜单进入后看到的当前结果。
          </div>
        </section>

        <section class="summary-section">
          <div class="summary-title">数据权限提示</div>
          <div class="hint-list">
            <div>
              <q-icon name="check_circle" color="positive" />
              <span>menu_id 参与按钮权限和数据权限判断</span>
            </div>
            <div>
              <q-icon name="check_circle" color="positive" />
              <span>permission_table_code 继续复用低代码表权限</span>
            </div>
            <div>
              <q-icon name="check_circle" color="positive" />
              <span>字段字典和日期控件复用低代码元数据</span>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { parameters, reportList, runtimeRows, type PrototypeReport } from '../../mock'

const report: PrototypeReport = reportList[0]!
const page = ref(1)
const filters = reactive({
  startDate: '2026-07-01',
  endDate: '2026-07-04',
  customerName: '',
  status: '全部状态',
})

const statusOptions = ['全部状态', '待发货', '已付款', '已完成', '对账中']

const columns: QTableProps['columns'] = [
  { name: 'customer_name', label: '客户名称', field: 'customer_name', align: 'left' },
  { name: 'order_no', label: '订单编号', field: 'order_no', align: 'left' },
  { name: 'amount', label: '订单金额', field: 'amount', align: 'right' },
  { name: 'created_at', label: '创建时间', field: 'created_at', align: 'center' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
]
</script>

<style scoped lang="scss">
.runtime-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.runtime-header,
.binding-banner,
.filter-panel,
.result-panel,
.runtime-summary {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
}

.runtime-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px;
}

.menu-title {
  margin-bottom: 4px;
  color: #2f6fed;
  font-size: 13px;
  font-weight: 700;
}

.runtime-title {
  color: #172033;
  font-size: 22px;
  font-weight: 700;
}

.runtime-meta,
.runtime-actions,
.panel-head,
.filter-actions,
.pagination-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.runtime-meta {
  flex-wrap: wrap;
  margin-top: 6px;
  color: #6b778c;
  font-size: 12px;
}

.binding-banner {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 12px 14px;
  background: #f8fbff;

  strong {
    display: block;
    color: #172033;
  }

  span {
    display: block;
    margin-top: 3px;
    color: #667085;
    font-size: 13px;
  }
}

.runtime-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 12px;
}

.runtime-main {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.filter-panel,
.result-panel,
.runtime-summary {
  padding: 14px;
}

.panel-head {
  justify-content: space-between;
  margin-bottom: 12px;
}

.panel-title,
.summary-title {
  color: #1f2a3d;
  font-weight: 700;
}

.panel-desc {
  color: #6b778c;
  font-size: 12px;
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.parameter-source-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 16px;
}

.parameter-source-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid #edf2f7;
  border-radius: 8px;
  background: #f8fafc;

  span {
    color: #172033;
    font-weight: 700;
  }

  small {
    margin-left: auto;
    color: #7a8699;
  }
}

.pagination-row {
  justify-content: space-between;
  margin-top: 12px;
  color: #4d5b70;
  font-size: 13px;
}

.runtime-summary {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.summary-section {
  padding-bottom: 14px;
  border-bottom: 1px solid #edf2f7;

  &:last-child {
    border-bottom: 0;
    padding-bottom: 0;
  }
}

.summary-title {
  margin-bottom: 10px;
}

.summary-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0;
  color: #667085;
  font-size: 13px;

  strong {
    color: #172033;
  }
}

.hint-box {
  padding: 10px;
  border-radius: 8px;
  background: #eef4ff;
  color: #1d4ed8;
  font-size: 13px;
  line-height: 1.6;
}

.hint-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: #4d5b70;
  font-size: 13px;

  div {
    display: flex;
    align-items: center;
    gap: 8px;
  }
}
</style>
