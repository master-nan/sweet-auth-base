<template>
  <div class="workbench-page">
    <div class="workbench-toolbar">
      <div class="toolbar-copy">
        <div class="section-title">报表设计管理工作台</div>
        <div class="section-desc">
          统一管理报表定义、草稿、发布版本、菜单挂载和执行日志。业务用户运行入口由左侧菜单指向通用报表运行页。
        </div>
      </div>
      <div class="toolbar-filters">
        <q-input v-model="keyword" dense outlined clearable placeholder="搜索报表名称、编码或负责人">
          <template #prepend>
            <q-icon name="search" />
          </template>
        </q-input>
        <q-select v-model="statusFilter" dense outlined emit-value map-options label="状态" :options="statusOptions" />
        <q-select v-model="categoryFilter" dense outlined label="分类" :options="categories" />
        <q-btn color="primary" unelevated icon="add" label="新建报表" />
      </div>
    </div>

    <div class="stats-grid">
      <div v-for="stat in reportStats" :key="stat.label" class="stat-card">
        <q-icon :name="stat.icon" color="primary" size="26px" />
        <div>
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
          <div class="stat-caption">{{ stat.caption }}</div>
        </div>
      </div>
    </div>

    <div class="workbench-layout">
      <aside class="sidebar-card">
        <div class="sidebar-title">管理范围</div>
        <div v-for="item in navItems" :key="item.label" class="nav-item" :class="{ active: item.active }">
          <q-icon :name="item.icon" size="18px" />
          <span>{{ item.label }}</span>
          <q-chip dense square color="grey-2" text-color="grey-8" :label="item.count" />
        </div>
        <q-separator spaced />
        <div class="sidebar-title">执行审计</div>
        <div class="nav-item">
          <q-icon name="receipt_long" size="18px" />
          <span>执行日志</span>
          <q-chip dense square color="blue-1" text-color="primary" label="入口" />
        </div>
      </aside>

      <main class="content-card">
        <q-table
          flat
          bordered
          row-key="id"
          :rows="filteredReports"
          :columns="columns"
          :pagination="{ rowsPerPage: 8 }"
        >
          <template #body-cell-name="props">
            <q-td :props="props">
              <div class="report-name">{{ props.row.name }}</div>
              <div class="report-desc">{{ props.row.description }}</div>
            </q-td>
          </template>

          <template #body-cell-status="props">
            <q-td :props="props">
              <q-chip
                dense
                square
                :color="getStatusMeta(props.row.status).color"
                :text-color="getStatusMeta(props.row.status).textColor"
                :label="getStatusMeta(props.row.status).label"
              />
            </q-td>
          </template>

          <template #body-cell-version="props">
            <q-td :props="props">
              <div class="version-cell">
                <strong>{{ props.row.version ? `V${props.row.version}` : '-' }}</strong>
                <span>{{ props.row.versions }} 个版本</span>
              </div>
            </q-td>
          </template>

          <template #body-cell-menuPublished="props">
            <q-td :props="props">
              <q-chip
                dense
                square
                :color="props.row.menuPublished ? 'green-1' : 'grey-2'"
                :text-color="props.row.menuPublished ? 'positive' : 'grey-7'"
                :icon="props.row.menuPublished ? 'account_tree' : 'link_off'"
                :label="props.row.menuPublished ? '已挂菜单' : '未挂菜单'"
              />
            </q-td>
          </template>

          <template #body-cell-menuName="props">
            <q-td :props="props">
              <span v-if="props.row.menuName">{{ props.row.menuName }}</span>
              <span v-else class="empty-text">-</span>
            </q-td>
          </template>

          <template #body-cell-menuPath="props">
            <q-td :props="props">
              <code v-if="props.row.menuPath" class="path-text">{{ props.row.menuPath }}</code>
              <span v-else class="empty-text">-</span>
            </q-td>
          </template>

          <template #body-cell-actions="props">
            <q-td :props="props">
              <div class="action-row">
                <q-btn dense flat color="primary" icon="design_services" label="设计" />
                <q-btn dense flat color="primary" icon="publish" label="发布" :disable="props.row.status === 'disabled'" />
                <q-btn
                  dense
                  flat
                  color="primary"
                  icon="account_tree"
                  label="发布到菜单"
                  :disable="props.row.status !== 'published'"
                />
                <q-btn dense flat color="grey-8" icon="history" label="版本" />
                <q-btn
                  dense
                  flat
                  color="warning"
                  icon="block"
                  label="停用"
                  :disable="props.row.status !== 'published'"
                />
                <q-btn dense flat color="negative" icon="delete_outline" label="删除" />
              </div>
            </q-td>
          </template>
        </q-table>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QTableProps } from 'quasar'
import { categories, reportList, reportStats, type PrototypeReport } from '../../mock'

const keyword = ref('')
const statusFilter = ref('all')
const categoryFilter = ref('全部分类')

const navItems = [
  { label: '全部定义', icon: 'description', count: '42', active: true },
  { label: '草稿', icon: 'edit_note', count: '9' },
  { label: '已发布', icon: 'verified', count: '31' },
  { label: '已停用', icon: 'block', count: '2' },
  { label: '已挂菜单', icon: 'account_tree', count: '18' },
  { label: '版本管理', icon: 'history', count: '96' },
]

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'published' },
  { label: '已停用', value: 'disabled' },
]

const statusMap: Record<PrototypeReport['status'], { label: string; color: string; textColor: string }> = {
  draft: { label: '草稿', color: 'orange-1', textColor: 'warning' },
  published: { label: '已发布', color: 'green-1', textColor: 'positive' },
  disabled: { label: '已停用', color: 'grey-3', textColor: 'grey-8' },
}

const getStatusMeta = (status: PrototypeReport['status']) => statusMap[status]

const columns: QTableProps['columns'] = [
  { name: 'name', label: '报表定义', field: 'name', align: 'left' },
  { name: 'code', label: '编码', field: 'code', align: 'left' },
  { name: 'type', label: '类型', field: 'type', align: 'left' },
  { name: 'category', label: '分类', field: 'category', align: 'left' },
  { name: 'status', label: '状态', field: 'status', align: 'center' },
  { name: 'version', label: '当前版本', field: 'version', align: 'center' },
  { name: 'menuPublished', label: '是否发布到菜单', field: 'menuPublished', align: 'center' },
  { name: 'menuName', label: '菜单名称', field: 'menuName', align: 'left' },
  { name: 'menuPath', label: '运行路径', field: 'menuPath', align: 'left' },
  { name: 'updatedAt', label: '更新时间', field: 'updatedAt', align: 'left' },
  { name: 'owner', label: '负责人', field: 'owner', align: 'left' },
  { name: 'actions', label: '操作', field: 'actions', align: 'left' },
]

const filteredReports = computed(() => {
  const text = keyword.value.trim().toLowerCase()
  return reportList.filter((report) => {
    const matchKeyword =
      !text ||
      report.name.toLowerCase().includes(text) ||
      report.code.toLowerCase().includes(text) ||
      report.owner.toLowerCase().includes(text)
    const matchStatus = statusFilter.value === 'all' || report.status === statusFilter.value
    const matchCategory = categoryFilter.value === '全部分类' || report.category === categoryFilter.value
    return matchKeyword && matchStatus && matchCategory
  })
})
</script>

<style scoped lang="scss">
.workbench-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.workbench-toolbar,
.stat-card,
.sidebar-card,
.content-card {
  border: 1px solid #dfe7f3;
  border-radius: 8px;
  background: #fff;
}

.workbench-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
}

.toolbar-copy {
  min-width: 320px;
  max-width: 560px;
}

.toolbar-filters {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) 150px 160px auto;
  align-items: center;
  gap: 10px;
  min-width: 720px;
}

.section-title {
  color: #172033;
  font-size: 22px;
  font-weight: 700;
}

.section-desc {
  margin-top: 4px;
  color: #667085;
  font-size: 13px;
  line-height: 1.6;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
}

.stat-card {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 14px;
}

.stat-value {
  color: #172033;
  font-size: 24px;
  font-weight: 700;
  line-height: 1;
}

.stat-label {
  margin-top: 6px;
  color: #2f3b52;
  font-weight: 600;
}

.stat-caption {
  margin-top: 4px;
  color: #7a8699;
  font-size: 12px;
  line-height: 1.5;
}

.workbench-layout {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 14px;
}

.sidebar-card {
  padding: 14px;
}

.sidebar-title {
  margin-bottom: 8px;
  color: #475467;
  font-size: 12px;
  font-weight: 700;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 0 8px;
  border-radius: 6px;
  color: #4d5b70;
  font-size: 13px;

  span {
    flex: 1;
  }

  &.active {
    background: #eef4ff;
    color: #1d4ed8;
    font-weight: 700;
  }
}

.content-card {
  padding: 14px;
}

.report-name {
  color: #172033;
  font-weight: 700;
}

.report-desc {
  max-width: 320px;
  margin-top: 3px;
  color: #7a8699;
  font-size: 12px;
  line-height: 1.45;
}

.version-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  align-items: center;

  span {
    color: #7a8699;
    font-size: 12px;
  }
}

.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 4px;
  min-width: 360px;
}

.empty-text {
  color: #98a2b3;
}

.path-text {
  color: #475467;
  font-size: 12px;
  white-space: nowrap;
}
</style>
