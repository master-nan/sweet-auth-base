<template>
  <div class="wizard-page">
    <div class="wizard-intro">
      <div>
        <div class="section-title">新建报表向导</div>
        <div class="section-desc">
          先判断业务目标，再进入版式设计或高级版式设计。普通数据表页面应通过低代码页面发布创建。
        </div>
      </div>
      <q-chip square color="blue-1" text-color="primary" icon="route">不直接进入空白设计器</q-chip>
    </div>

    <q-stepper v-model="step" flat bordered animated color="primary" class="wizard-stepper">
      <q-step :name="1" title="选择报表类型" icon="category" :done="step > 1">
        <div class="step-desc">报表模块聚焦版式输出、对账单、汇总报表和固定格式运行页，不重复低代码普通列表页。</div>
        <div class="report-type-grid">
          <div
            v-for="type in reportTypes"
            :key="type.title"
            class="type-card"
            :class="{ selected: selectedType === type.value, disabled: type.disabled }"
            @click="!type.disabled && (selectedType = type.value)"
          >
            <div class="type-icon">
              <q-icon :name="type.icon" size="28px" />
            </div>
            <div class="type-title">
              {{ type.title }}
              <q-chip v-if="type.recommended" dense square color="green-1" text-color="positive" label="推荐" />
            </div>
            <div class="type-desc">{{ type.desc }}</div>
            <q-chip v-if="type.badge" dense square color="grey-2" text-color="grey-8" :label="type.badge" />
          </div>
        </div>
      </q-step>

      <q-step :name="2" title="基本信息" icon="article" :done="step > 2">
        <div class="step-desc">报表名称、编码、分类和描述会用于设计管理、菜单挂载和运行页展示。</div>
        <div class="form-grid">
          <q-input v-model="form.name" dense outlined label="报表名称" />
          <q-input v-model="form.code" dense outlined label="报表编码" />
          <q-select v-model="form.category" dense outlined label="分类" :options="categoryOptions" />
          <q-input v-model="form.owner" dense outlined label="负责人" />
          <q-input
            v-model="form.description"
            class="span-2"
            dense
            outlined
            autogrow
            type="textarea"
            label="描述"
          />
        </div>
      </q-step>

      <q-step :name="3" title="数据来源" icon="storage" :done="step > 3">
        <div class="step-desc">
          第一阶段仍复用系统表元数据和受控 SQL 能力，查询控件尽量从 sys_table_field、字典和字段类型推导。
        </div>
        <div class="source-grid">
          <div
            v-for="source in sourceOptions"
            :key="source.title"
            class="source-card"
            :class="{ selected: selectedSource === source.value, disabled: source.disabled }"
            @click="!source.disabled && (selectedSource = source.value)"
          >
            <q-icon :name="source.icon" size="24px" />
            <div>
              <div class="source-title">{{ source.title }}</div>
              <div class="source-desc">{{ source.desc }}</div>
            </div>
            <q-chip v-if="source.badge" dense square color="orange-1" text-color="warning" :label="source.badge" />
          </div>
        </div>
      </q-step>

      <q-step :name="4" title="版式和参数" icon="tune" :done="step > 4">
        <div class="step-desc">
          选择会出现在版式区域中的字段，并确认查询参数控件来源。这里不创建普通 CRUD 页面。
        </div>
        <div class="wizard-grid">
          <section class="wizard-panel">
            <div class="panel-title">字段绑定候选</div>
            <q-table
              flat
              bordered
              dense
              row-key="id"
              :rows="fieldRows"
              :columns="fieldColumns"
              :pagination="{ rowsPerPage: 0 }"
              hide-pagination
            >
              <template #body-cell-selected="props">
                <q-td :props="props">
                  <q-checkbox v-model="props.row.selected" dense />
                </q-td>
              </template>
              <template #body-cell-title="props">
                <q-td :props="props">
                  <q-input v-model="props.row.title" dense outlined />
                </q-td>
              </template>
              <template #body-cell-aggregate="props">
                <q-td :props="props">
                  <q-chip v-if="props.row.aggregate" dense square color="blue-1" text-color="primary" :label="props.row.aggregate" />
                  <span v-else class="muted">-</span>
                </q-td>
              </template>
            </q-table>
          </section>
          <section class="wizard-panel">
            <div class="panel-title">查询参数控件来源</div>
            <div class="parameter-list">
              <div v-for="item in parameters" :key="item.id" class="parameter-item">
                <q-icon name="filter_alt" color="primary" />
                <div>
                  <div class="parameter-label">{{ item.label }}</div>
                  <div class="parameter-meta">
                    {{ item.field }} {{ item.operator }} · {{ item.control }}
                  </div>
                  <div class="parameter-source">{{ item.sourceMeta }}</div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </q-step>

      <q-step :name="5" title="预览并保存" icon="preview">
        <div class="step-desc">保存草稿后可设计时预览。发布生成版本快照，之后可以挂载到左侧菜单。</div>
        <div class="summary-grid">
          <div class="summary-card">
            <div class="summary-value">{{ selectedFieldCount }}</div>
            <div class="summary-label">字段绑定</div>
          </div>
          <div class="summary-card">
            <div class="summary-value">{{ parameters.length }}</div>
            <div class="summary-label">查询参数</div>
          </div>
          <div class="summary-card">
            <div class="summary-value">系统表</div>
            <div class="summary-label">元数据来源</div>
          </div>
        </div>
        <div class="preview-panel">
          <div class="panel-title">预览摘要</div>
          <div class="preview-row">
            <span>报表类型</span>
            <strong>版式报表</strong>
          </div>
          <div class="preview-row">
            <span>运行方式</span>
            <strong>菜单绑定 report_id / report_code，通用运行页读取发布版本</strong>
          </div>
          <div class="preview-row">
            <span>控件来源</span>
            <strong>report parameters + sys_table_field + dict</strong>
          </div>
          <div class="preview-row">
            <span>导出策略</span>
            <strong>后端受控导出，默认限制 5000 行</strong>
          </div>
        </div>
        <div class="final-actions">
          <q-btn outline color="primary" icon="save" label="保存草稿" />
          <q-btn outline color="primary" icon="visibility" label="保存并预览" />
          <q-btn color="primary" unelevated icon="publish" label="发布并生成版本快照" />
          <q-btn outline color="primary" icon="account_tree" label="发布到菜单" />
        </div>
      </q-step>

      <template #navigation>
        <q-stepper-navigation>
          <q-btn v-if="step > 1" flat color="primary" label="上一步" @click="step -= 1" />
          <q-btn
            v-if="step < 5"
            color="primary"
            unelevated
            label="下一步"
            class="q-ml-sm"
            @click="step += 1"
          />
        </q-stepper-navigation>
      </template>
    </q-stepper>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { categories, fields, parameters } from '../../mock'

const step = ref(1)
const selectedType = ref('layout_report')
const selectedSource = ref('system_table')

const form = reactive({
  name: '供应商月度对账单',
  code: 'supplier_monthly_statement',
  category: '供应链',
  owner: '陈佳',
  description: '按账期输出供应商对账单，包含查询条件展示、明细、汇总和页脚。',
})

const categoryOptions = categories.filter((item) => item !== '全部分类')

const fieldRows = ref(fields.map((field) => ({ ...field })))

const reportTypes = [
  {
    title: '版式报表',
    value: 'layout_report',
    icon: 'dashboard_customize',
    desc: '用于对账单、费用统计、汇总报表和固定格式输出，是 V2 默认推荐入口。',
    recommended: true,
  },
  {
    title: 'SQL / 聚合报表',
    value: 'sql_report',
    icon: 'terminal',
    desc: '用于复杂统计和跨表分析，运行和导出需要管理员权限与 SQL 安全校验。',
    badge: '管理员可用',
  },
  {
    title: '高级 Sheet 报表',
    value: 'advanced_sheet',
    icon: 'grid_on',
    desc: '用于复杂表头、合并单元格、分组汇总和自定义布局。',
  },
  {
    title: '普通数据表页面',
    value: 'crud_page',
    icon: 'table_view',
    desc: '不在报表模块创建，请到数据管理 / 低代码页面发布中创建。',
    badge: '转到低代码页面',
    disabled: true,
  },
]

const sourceOptions = [
  {
    title: '系统表',
    value: 'system_table',
    icon: 'table_view',
    desc: '复用低代码平台 sys_table / sys_table_field 元数据和字典渲染。',
  },
  {
    title: 'SQL 数据集',
    value: 'sql_dataset',
    icon: 'terminal',
    desc: '适合管理员定义复杂统计查询，需通过 SQL 安全守卫。',
    badge: '管理员可用',
  },
  {
    title: '已保存数据集',
    value: 'saved_dataset',
    icon: 'dataset_linked',
    desc: '沉淀可复用的数据集资产。',
    badge: '后续阶段',
    disabled: true,
  },
  {
    title: '外部数据源',
    value: 'external_source',
    icon: 'hub',
    desc: '连接外部数据库或第三方系统。',
    badge: '后续阶段',
    disabled: true,
  },
]

const fieldColumns: QTableProps['columns'] = [
  { name: 'selected', label: '使用', field: 'selected', align: 'center' },
  { name: 'name', label: '字段名称', field: 'name', align: 'left' },
  { name: 'code', label: '字段编码', field: 'code', align: 'left' },
  { name: 'type', label: '类型', field: 'type', align: 'center' },
  { name: 'title', label: '显示标题', field: 'title', align: 'left' },
  { name: 'aggregate', label: '聚合', field: 'aggregate', align: 'center' },
]

const selectedFieldCount = computed(() => fieldRows.value.filter((field) => field.selected).length)
</script>

<style scoped lang="scss">
.wizard-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.wizard-intro,
.wizard-stepper,
.wizard-panel,
.summary-card,
.preview-panel {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
}

.wizard-intro {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
}

.section-title {
  color: #172033;
  font-size: 22px;
  font-weight: 700;
}

.section-desc,
.step-desc {
  color: #667085;
  font-size: 13px;
  line-height: 1.6;
}

.section-desc {
  margin-top: 4px;
}

.step-desc {
  margin-bottom: 14px;
}

.report-type-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.type-card,
.source-card {
  cursor: pointer;
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
  transition: border-color 0.18s ease, box-shadow 0.18s ease;

  &.selected {
    border-color: #2f6fed;
    box-shadow: 0 0 0 2px rgba(47, 111, 237, 0.08);
  }

  &.disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}

.type-card {
  min-height: 180px;
  padding: 16px;
}

.type-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  margin-bottom: 12px;
  border-radius: 8px;
  background: #eef4ff;
  color: #1d4ed8;
}

.type-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  color: #172033;
  font-size: 16px;
  font-weight: 700;
}

.type-desc,
.source-desc {
  color: #667085;
  font-size: 13px;
  line-height: 1.55;
}

.form-grid,
.source-grid,
.wizard-grid,
.summary-grid {
  display: grid;
  gap: 12px;
}

.form-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.span-2 {
  grid-column: span 2;
}

.source-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.source-card {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: flex-start;
  padding: 14px;
}

.source-title,
.panel-title {
  color: #172033;
  font-weight: 700;
}

.wizard-grid {
  grid-template-columns: minmax(0, 1.35fr) minmax(320px, 0.65fr);
}

.wizard-panel {
  padding: 14px;
}

.panel-title {
  margin-bottom: 12px;
}

.parameter-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.parameter-item {
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

.summary-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-bottom: 12px;
}

.summary-card {
  padding: 14px;
}

.summary-value {
  color: #172033;
  font-size: 22px;
  font-weight: 700;
}

.summary-label {
  margin-top: 4px;
  color: #667085;
  font-size: 13px;
}

.preview-panel {
  padding: 14px;
}

.preview-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0;
  border-bottom: 1px solid #edf2f7;
  color: #667085;

  &:last-child {
    border-bottom: 0;
  }

  strong {
    color: #172033;
  }
}

.final-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
}
</style>
