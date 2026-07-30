<template>
  <div class="report-v2-designer-page">
    <q-card v-if="loading" flat bordered class="state-card">
      <q-skeleton type="text" width="260px" />
      <q-skeleton type="text" width="420px" />
      <q-skeleton type="rect" height="280px" />
    </q-card>

    <q-card v-else-if="loadError" flat bordered class="state-card">
      <q-icon name="error_outline" color="negative" size="40px" />
      <div class="text-subtitle1 text-weight-medium">报表加载失败</div>
      <div class="text-body2 text-grey-7">{{ loadError }}</div>
      <q-btn outline color="primary" icon="arrow_back" label="返回工作台" @click="goWorkbench" />
    </q-card>

    <template v-else>
      <div class="designer-toolbar">
        <div class="toolbar-left">
          <q-btn flat dense icon="arrow_back" label="返回" @click="goWorkbench" />
          <div class="toolbar-title">
            <div class="text-subtitle1 text-weight-bold">{{ report?.report_name || '未命名报表' }}</div>
            <div class="text-caption text-grey-7">
              {{ report?.report_code || '-' }}
            </div>
          </div>
          <q-chip dense square :color="statusMeta.color" text-color="white" :label="statusMeta.label" />
          <q-chip
            v-if="report?.published_version_no"
            dense
            square
            color="primary"
            text-color="white"
            :label="`V${report.published_version_no}`"
          />
        </div>
        <div class="toolbar-actions">
          <q-chip v-if="dirty" dense square color="amber-1" text-color="orange-10" label="内存中有未保存变更" />
          <q-btn flat dense color="primary" icon="tune" label="参数" @click="openParameterDialog()" />
          <q-btn
            outline
            color="primary"
            icon="visibility"
            label="保存并预览"
            :loading="previewLoading"
            :disable="!report"
            @click="previewDesign"
          />
          <q-btn flat dense color="primary" icon="fact_check" label="校验" @click="validateCurrentDesign" />
          <q-btn
            outline
            color="primary"
            icon="save"
            label="保存草稿"
            :loading="saving"
            @click="saveDraft"
          />
          <q-btn
            color="primary"
            icon="publish"
            label="保存并发布"
            unelevated
            :loading="publishing"
            :disable="!report || saving || previewLoading"
            @click="publishVersion"
          />
          <q-btn flat dense color="grey-8" icon="history" label="版本" @click="notifyDesignerPlaceholder('版本列表')" />
        </div>
      </div>

      <div class="designer-shell">
        <aside class="designer-panel">
          <q-card flat bordered class="panel-card">
            <q-card-section>
              <div class="panel-title row items-center justify-between no-wrap">
                <span>数据来源</span>
                <q-btn dense flat color="primary" icon="add" label="新增" @click="openDatasetDialog()" />
              </div>
              <q-banner v-if="datasets.length > 1" rounded dense class="bg-orange-1 text-orange-10 q-mb-sm">
                第一阶段仅支持单结果数据集，当前仅使用第一个数据集。
              </q-banner>
              <q-banner v-if="!datasets.length" rounded dense class="bg-grey-2 text-grey-8">
                暂无数据来源，请先新增现有表或 SQL 数据集。
              </q-banner>
              <q-list v-else dense>
                <q-item
                  v-for="dataset in datasets"
                  :key="dataset.id"
                  class="resource-row dataset-resource-row"
                  clickable
                  :active="selectedDatasetId === dataset.id"
                  active-class="active-area-nav"
                  @click="selectedDatasetId = dataset.id"
                >
                  <q-item-section avatar class="resource-avatar">
                    <q-icon :name="dataset.type === 'sql' ? 'data_object' : 'storage'" color="primary" size="18px" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label class="resource-title">
                      {{ dataset.name || dataset.source_code || '数据集' }}
                      <q-badge v-if="dataset.primary" color="primary" class="dataset-primary-badge" label="主" />
                    </q-item-label>
                    <q-item-label caption class="resource-caption">
                      {{ dataset.type === 'sql' ? 'SQL 数据集' : dataset.source_code || '-' }}
                      · {{ dataset.fields?.length || 0 }} 字段
                    </q-item-label>
                  </q-item-section>
                  <q-item-section side class="resource-side">
                    <div class="field-actions">
                      <q-btn dense flat round size="sm" color="primary" icon="edit" @click.stop="openDatasetDialog(dataset.id)">
                        <q-tooltip>编辑数据集</q-tooltip>
                      </q-btn>
                      <q-btn dense flat round size="sm" color="negative" icon="delete" @click.stop="removeDataset(dataset.id)">
                        <q-tooltip>删除数据集</q-tooltip>
                      </q-btn>
                    </div>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>
          </q-card>

          <q-card flat bordered class="panel-card resource-fields-card">
            <q-card-section>
              <div class="panel-title row items-center justify-between no-wrap">
                <span>字段列表</span>
                <q-chip dense square color="grey-2" text-color="grey-8" :label="`${joinedFieldKeys.size}/${selectedDatasetFields.length}`" />
              </div>
              <div class="text-caption text-grey-7 q-mb-sm">
                当前数据集：{{ selectedResourceDataset?.name || '未选择' }}
              </div>
              <q-input
                v-model="fieldKeyword"
                dense
                outlined
                clearable
                class="q-mb-sm"
                placeholder="搜索字段"
              >
                <template #prepend>
                  <q-icon name="search" />
                </template>
              </q-input>
              <q-banner v-if="!selectedDatasetFields.length" rounded dense class="bg-grey-2 text-grey-8">
                暂无字段，无法配置表头和明细区。
              </q-banner>
              <q-list v-else dense separator>
                <q-item v-for="field in filteredPrimaryFields" :key="field.code">
                  <q-item-section avatar>
                    <q-icon :name="fieldIcon(field)" color="primary" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>{{ field.name || field.code }}</q-item-label>
                    <q-item-label caption>{{ field.code }} · {{ field.type || '-' }}</q-item-label>
                  </q-item-section>
                  <q-item-section side>
                    <div class="field-actions">
                      <q-btn
                        dense
                        flat
                        no-caps
                        color="primary"
                        :disable="isFieldJoined(field, selectedResourceDataset)"
                        :label="isFieldJoined(field, selectedResourceDataset) ? '已加入' : '加入'"
                        @click="addFieldToReport(field, selectedResourceDataset)"
                      />
                      <q-btn
                        dense
                        flat
                        round
                        color="primary"
                        icon="link"
                        :disable="!canBindFieldToSelection"
                        @click="bindFieldToSelection(field, selectedResourceDataset)"
                      >
                        <q-tooltip>绑定到当前单元格</q-tooltip>
                      </q-btn>
                    </div>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>
          </q-card>

          <q-card flat bordered class="panel-card">
            <q-card-section>
              <div class="panel-title row items-center justify-between no-wrap">
                <span>查询参数</span>
                <q-btn dense flat color="primary" icon="add" label="新增" @click="openParameterDialog()" />
              </div>
              <q-banner v-if="!parameters.length" rounded dense class="bg-grey-2 text-grey-8">
                暂无查询参数，查询条件展示区保留为空。
              </q-banner>
              <q-list v-else dense separator>
                <q-item
                  v-for="parameter in parameters"
                  :key="parameter.id"
                  clickable
                  :active="selectedParameterId === parameter.id"
                  active-class="active-area-nav"
                  @click="selectedParameterId = parameter.id"
                >
                  <q-item-section avatar>
                    <q-icon name="filter_alt" color="primary" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>{{ parameter.label || parameter.field }}</q-item-label>
                    <q-item-label caption>
                      {{ parameterDatasetLabel(parameter.dataset_id) }}
                      · {{ parameter.field }} · {{ parameter.type }} · {{ parameter.operator }}
                    </q-item-label>
                  </q-item-section>
                  <q-item-section side>
                    <div class="field-actions">
                      <q-btn dense flat round color="primary" icon="edit" @click.stop="openParameterDialog(parameter.id)">
                        <q-tooltip>编辑参数</q-tooltip>
                      </q-btn>
                      <q-btn dense flat round color="negative" icon="delete" @click.stop="removeParameter(parameter.id)">
                        <q-tooltip>删除参数</q-tooltip>
                      </q-btn>
                    </div>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>
          </q-card>

          <q-card flat bordered class="panel-card">
            <q-card-section>
              <div class="panel-title">区域 / 数据带</div>
              <q-list dense>
                <q-item
                  v-for="area in navigationAreas"
                  :key="area.id"
                  clickable
                  :active="selectedAreaId === area.id"
                  active-class="active-area-nav"
                  @click="selectArea(area.id)"
                >
                  <q-item-section avatar>
                    <q-icon :name="areaIcon(area.type)" />
                  </q-item-section>
                  <q-item-section>
                    <q-item-label>
                      {{ area.title }}
                      <q-chip v-if="area.visible === false" dense square color="grey-3" text-color="grey-8" label="隐藏" />
                    </q-item-label>
                    <q-item-label caption>{{ visibleAreaItems(area).length }}/{{ area.items.length }} 个元素</q-item-label>
                  </q-item-section>
                </q-item>
              </q-list>
            </q-card-section>
          </q-card>

          <q-card flat bordered class="panel-card">
            <q-card-section>
              <div class="panel-title">组件</div>
              <div class="component-palette">
                <q-chip dense square outline icon="notes" label="静态文本" />
                <q-chip dense square outline icon="input" label="字段" />
                <q-chip dense square outline icon="functions" label="汇总" />
                <q-chip dense square outline icon="schedule" label="运行时变量" />
              </div>
            </q-card-section>
          </q-card>
        </aside>

        <main class="designer-canvas">
          <div class="canvas-header">
            <div>
              <div class="text-subtitle1 text-weight-medium">Sheet 画布</div>
              <div class="text-caption text-grey-7">
                选择单元格后可在右侧绑定字段或调整显示属性。
              </div>
            </div>
            <div class="canvas-tools">
              <q-chip
                dense
                square
                color="blue-1"
                text-color="primary"
                :label="report?.layout_config?.designer_mode === 'layout' ? '已有 layout_areas' : '默认初始化 layout_areas'"
              />
              <q-chip dense square color="grey-2" text-color="grey-8" :label="`${sheetColumns.length} 列`" />
            </div>
          </div>

          <div class="canvas-formatbar">
            <q-btn flat round dense icon="undo" @click="notifyDesignerPlaceholder('撤销')" />
            <q-btn flat round dense icon="redo" @click="notifyDesignerPlaceholder('重做')" />
            <q-separator vertical />
            <q-btn flat round dense icon="format_bold" @click="toggleSelectedBold" />
            <q-btn flat round dense icon="format_align_left" @click="setSelectedItemAlign('left')" />
            <q-btn flat round dense icon="format_align_center" @click="setSelectedItemAlign('center')" />
            <q-btn flat round dense icon="format_align_right" @click="setSelectedItemAlign('right')" />
            <q-separator vertical />
            <q-btn flat dense icon="cell_merge" label="合并选区" @click="notifyDesignerPlaceholder('合并选区')" />
            <q-btn flat dense icon="call_split" label="取消合并" @click="notifyDesignerPlaceholder('取消合并')" />
            <q-btn flat dense icon="delete_sweep" label="清除选区" @click="clearCurrentSelection" />
            <q-separator vertical />
            <q-btn flat dense icon="table_rows" label="新增行" @click="notifyDesignerPlaceholder('新增行')" />
            <q-btn flat dense icon="view_column" label="新增列" @click="notifyDesignerPlaceholder('新增列')" />
            <q-space />
            <q-select
              v-model="canvasZoom"
              dense
              outlined
              options-dense
              emit-value
              map-options
              class="zoom-select"
              :options="zoomOptions"
            />
          </div>

          <q-banner
            v-if="!datasets.length || !primaryFields.length"
            rounded
            class="bg-orange-1 text-orange-10"
          >
            当前报表缺少可用的单结果数据集或字段，版式区域仅显示结构占位。
          </q-banner>

          <div class="sheet-scroll">
            <div class="sheet-grid" :style="sheetGridStyle">
              <div class="sheet-corner"></div>
              <div
                v-for="column in sheetColumns"
                :key="column.key"
                class="sheet-column-head"
              >
                {{ column.label }}
              </div>

              <template v-for="row in designerSheetRows" :key="row.areaId">
                <div
                  class="sheet-row-head"
                  :class="{ 'is-active-row': selectedAreaId === row.areaId && !selectedItem }"
                  @click="selectAreaOnly(row.areaId)"
                >
                  {{ row.index }}
                </div>
                <button
                  v-for="cell in row.cells"
                  :id="cell.anchor ? areaDomId(cell.areaId) : undefined"
                  :key="cell.id"
                  type="button"
                  class="sheet-cell"
                  :class="[
                    `sheet-cell-${cell.tone}`,
                    {
                      'is-active-cell': isSheetCellSelected(cell),
                      'is-area-cell': !cell.itemId,
                      'is-empty-cell': cell.empty,
                    },
                  ]"
                  :style="sheetCellStyle(cell)"
                  @click="selectSheetCell(cell)"
                >
                  <span v-if="cell.bandLabel" class="cell-band-label">{{ cell.bandLabel }}</span>
                  <span class="cell-text">{{ cell.text }}</span>
                  <span v-if="cell.subtext" class="cell-subtext">{{ cell.subtext }}</span>
                  <span v-if="cell.itemId && !cell.empty" class="cell-actions">
                    <q-icon
                      v-if="cell.canMoveUp"
                      name="keyboard_arrow_up"
                      size="16px"
                      @click.stop="moveSheetCell(cell, 'up')"
                    />
                    <q-icon
                      v-if="cell.canMoveDown"
                      name="keyboard_arrow_down"
                      size="16px"
                      @click.stop="moveSheetCell(cell, 'down')"
                    />
                    <q-icon
                      v-if="cell.canRemove"
                      name="close"
                      size="16px"
                      class="text-negative"
                      @click.stop="removeSheetCell(cell)"
                    />
                  </span>
                </button>
              </template>
            </div>
          </div>
        </main>

        <aside class="designer-inspector">
          <q-tabs
            v-model="inspectorTab"
            dense
            class="inspector-tabs"
            active-color="primary"
            indicator-color="primary"
            align="justify"
          >
            <q-tab name="cell" label="单元格" />
            <q-tab name="data" label="数据" />
            <q-tab name="report" label="报表" />
          </q-tabs>

          <q-tab-panels v-model="inspectorTab" animated class="inspector-panels">
            <q-tab-panel name="cell" class="inspector-panel">
              <div class="panel-title">{{ selectedItem ? '单元格属性' : '区域属性' }}</div>
              <div class="property-form">
                <q-input dense outlined readonly label="当前单元格" :model-value="currentCellLabel" />

                <template v-if="selectedItem">
                  <q-input
                    dense
                    outlined
                    label="标题 / 别名"
                    :model-value="selectedItem.label || ''"
                    @update:model-value="setSelectedItemLabel"
                  />
                  <q-input
                    dense
                    outlined
                    type="number"
                    label="宽度"
                    :model-value="selectedItem.width || 120"
                    @update:model-value="setSelectedItemWidth"
                  />
                  <q-input
                    dense
                    outlined
                    type="number"
                    label="顺序"
                    :model-value="selectedItem.order || 1"
                    @update:model-value="setSelectedItemOrder"
                  />
                  <q-select
                    dense
                    outlined
                    emit-value
                    map-options
                    label="对齐"
                    :options="alignOptions"
                    :model-value="selectedItem.align || 'left'"
                    @update:model-value="setSelectedItemAlign"
                  />
                  <q-select
                    dense
                    outlined
                    emit-value
                    map-options
                    label="格式"
                    :options="formatOptions"
                    :model-value="selectedItem.format || 'text'"
                    @update:model-value="setSelectedItemFormat"
                  />
                  <q-select
                    v-if="selectedItemArea?.type === 'summary'"
                    dense
                    outlined
                    emit-value
                    map-options
                    label="汇总方式"
                    :options="aggregateOptions"
                    :model-value="selectedItem.aggregate || 'sum'"
                    @update:model-value="setSelectedItemAggregate"
                  />
                  <q-toggle
                    :model-value="selectedItem.visible !== false"
                    label="显示该字段"
                    @update:model-value="setSelectedItemVisible"
                  />
                  <q-btn
                    v-if="selectedItemArea && isFieldOperationItem(selectedItemArea, selectedItem)"
                    outline
                    color="negative"
                    icon="close"
                    label="移除该项"
                    @click="removeItem(selectedItemArea, selectedItem)"
                  />
                </template>

                <template v-else>
                  <q-input
                    dense
                    outlined
                    label="区域标题"
                    :model-value="selectedArea?.title || ''"
                    @update:model-value="setSelectedAreaTitle"
                  />
                  <q-toggle
                    :model-value="selectedArea?.visible !== false"
                    label="显示该区域"
                    @update:model-value="setSelectedAreaVisible"
                  />
                  <q-input dense outlined readonly label="区域类型" :model-value="selectedArea ? areaTypeLabel(selectedArea.type) : '-'" />
                  <q-input dense outlined readonly label="元素数量" :model-value="String(selectedArea?.items.length || 0)" />
                </template>
              </div>
            </q-tab-panel>

            <q-tab-panel name="data" class="inspector-panel">
              <div class="panel-title">数据绑定</div>
              <div class="property-form">
                <q-input dense outlined readonly label="绑定类型" :model-value="selectedItem?.type || selectedArea?.type || '-'" />
                <q-input dense outlined readonly label="数据集" :model-value="selectedItem?.dataset_id || primaryDataset?.id || '-'" />
                <q-input dense outlined readonly label="数据字段" :model-value="selectedItem?.field || selectedItem?.parameter_id || '-'" />
                <q-input dense outlined readonly label="公式 / 表达式" model-value="第一阶段暂不支持复杂公式" />

                <q-separator />
                <div class="panel-title q-mb-none">字段绑定摘要</div>
                <q-banner v-if="!detailBindings.length" rounded dense class="bg-grey-2 text-grey-8">
                  暂无明细字段绑定。
                </q-banner>
                <q-list v-else dense separator class="binding-list">
                  <q-item v-for="item in detailBindings" :key="item.id" clickable @click="selectItem('detail', item.id)">
                    <q-item-section>
                      <q-item-label>{{ item.label || item.field }}</q-item-label>
                      <q-item-label caption>{{ item.field }} · {{ item.format || 'text' }}</q-item-label>
                    </q-item-section>
                    <q-item-section side>
                      <q-chip dense square color="grey-2" text-color="grey-8" :label="`${item.order || 0}`" />
                    </q-item-section>
                  </q-item>
                </q-list>

                <div class="summary-field-picker">
                  <div class="text-caption text-grey-7 q-mb-sm">添加汇总字段</div>
                  <q-banner v-if="!numericFields.length" rounded dense class="bg-grey-2 text-grey-8">
                    当前数据集暂无可汇总的数值字段。
                  </q-banner>
                  <div v-else class="summary-field-list">
                    <q-btn
                      v-for="field in numericFields"
                      :key="field.code"
                      dense
                      outline
                      no-caps
                      color="primary"
                      :disable="isSummaryFieldJoined(field)"
                      :label="isSummaryFieldJoined(field) ? `${field.name || field.code} 已汇总` : `添加 ${field.name || field.code}`"
                      @click="addSummaryItem(field)"
                    />
                  </div>
                </div>
              </div>
            </q-tab-panel>

            <q-tab-panel name="report" class="inspector-panel">
              <div class="panel-title">报表属性</div>
              <div class="property-form">
                <q-input dense outlined readonly label="报表 ID" :model-value="String(report?.id || '-')" />
                <q-input dense outlined readonly label="报表编码" :model-value="report?.report_code || '-'" />
                <q-input dense outlined readonly label="设计模式" model-value="Sheet / 版式设计" />
                <q-input dense outlined readonly label="运行分页" :model-value="String(report?.layout_config?.runtime_page_size || 20)" />
              </div>

              <q-separator class="q-my-md" />
              <div class="panel-title">发布检查摘要</div>
              <div v-for="check in publishChecks" :key="check.label" class="check-item" :class="{ ok: check.ok, optional: check.optional }">
                <q-icon :name="check.ok ? 'check_circle' : check.optional ? 'info' : 'error_outline'" />
                <span>{{ check.label }}</span>
              </div>
            </q-tab-panel>
          </q-tab-panels>
        </aside>
      </div>

      <div class="designer-statusbar">
        <span>当前单元格：{{ currentCellLabel }}</span>
        <span>数据集：{{ datasets.length }}</span>
        <span>已绑定字段：{{ detailBindings.length }}</span>
        <span>报表 ID：{{ report?.id || '-' }}</span>
        <span>状态：{{ dirty ? '有未保存修改' : '已同步' }}</span>
      </div>

      <q-dialog v-model="previewDialogVisible">
        <q-card class="preview-dialog">
          <q-card-section class="dialog-head">
            <div>
              <div class="dialog-title">设计预览</div>
              <div class="dialog-caption">
                已先保存草稿，再使用发布前草稿配置调用后端 design-preview。
              </div>
            </div>
            <q-btn flat round dense icon="close" v-close-popup />
          </q-card-section>
          <q-card-section>
            <div class="preview-meta">
              <q-chip dense square color="primary" text-color="white">
                {{ previewData.total }} 行
              </q-chip>
              <q-chip dense square outline color="primary">
                {{ previewSheet.cells.length }} 个单元格
              </q-chip>
              <q-chip
                v-if="previewData.meta?.dataset_id"
                dense
                square
                outline
                color="primary"
              >
                主数据集 {{ previewData.meta.dataset_id }}
              </q-chip>
            </div>
            <report-sheet-preview
              :sheet="previewSheet"
              :datasets="datasets"
              :preview-data="previewData"
              :loading="previewLoading"
              :report-kind="report?.report_kind || 'detail'"
            />
          </q-card-section>
        </q-card>
      </q-dialog>

      <report-dataset-dialog
        v-model="datasetDialogVisible"
        :editing-dataset="editingDataset"
        :draft="datasetDraft"
        :dataset-type-options="reportDatasetTypeOptions"
        :data-sources="dataSources"
        :preview-fields="datasetDraftPreviewFields"
        :sql-fields-loading="sqlFieldsLoading"
        @update:type="handleDraftTypeChange"
        @update:name="datasetDraft.name = $event"
        @update:source-code="handleDraftSourceChange"
        @update:sql="handleDatasetDraftSqlChange"
        @update:fields-text="datasetDraft.fieldsText = $event"
        @infer-sql-fields="inferSqlDatasetFields"
        @confirm="confirmDataset"
      />

      <report-parameter-dialog
        v-model="parameterDialogVisible"
        :editing="!!editingParameterId"
        :draft="parameterDraft"
        :dataset-options="datasetOptions"
        :field-options="parameterFieldOptions"
        :type-options="reportParameterTypeOptions"
        :operator-options="reportParameterOperatorOptions"
        @update:label="parameterDraft.label = $event"
        @update:dataset-id="handleParameterDatasetChange"
        @update:field="handleParameterFieldChange"
        @update:type="parameterDraft.type = $event"
        @update:operator="parameterDraft.operator = $event"
        @update:placeholder="parameterDraft.placeholder = $event"
        @update:default-value="parameterDraft.default_value = $event"
        @confirm="confirmParameter"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'report_v2_designer' })

import { computed, reactive, ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import {
  makeReportCellId,
  useReportApi,
  type Report,
  type ReportDataSource,
  type ReportDataset,
  type ReportDatasetType,
  type ReportField,
  type ReportLayoutConfig,
  type ReportLayoutArea,
  type ReportLayoutAreaAggregate,
  type ReportLayoutAreaItem,
  type ReportLayoutAreaItemAlign,
  type ReportLayoutAreaItemFormat,
  type ReportLayoutAreaType,
  type ReportParameter,
  type ReportParameterOperator,
  type ReportParameterType,
  type ReportPreviewRes,
  type ReportSaveReq,
  type ReportSheetCell,
  type ReportSheetConfig,
  type ReportStatus,
} from 'src/api/services/report'
import {
  reportDatasetTypeOptions,
  reportParameterOperatorOptions,
  reportParameterTypeOptions,
} from 'src/modules/report/options'
import { reportParameterDefaultsForField } from 'src/modules/report/schema'
import ReportDatasetDialog from 'src/pages/report/design/components/ReportDatasetDialog.vue'
import ReportParameterDialog from 'src/pages/report/design/components/ReportParameterDialog.vue'
import ReportSheetPreview from 'src/pages/report/components/ReportSheetPreview.vue'

type SelectedItemRef = {
  areaId: string
  itemId: string
}

type MoveDirection = 'up' | 'down'

interface DesignerSheetColumn {
  key: string
  label: string
  width: number
}

interface DesignerSheetCell {
  id: string
  areaId: string
  itemId?: string
  col: number
  colspan?: number
  text: string
  subtext?: string
  bandLabel?: string
  tone: ReportLayoutAreaType | 'empty'
  align?: ReportLayoutAreaItemAlign | 'center'
  empty?: boolean
  anchor?: boolean
  canMoveUp?: boolean
  canMoveDown?: boolean
  canRemove?: boolean
}

interface DesignerSheetRow {
  index: number
  areaId: string
  cells: DesignerSheetCell[]
}

const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const reportApi = useReportApi()

const report = ref<Report | null>(null)
const dataSources = ref<ReportDataSource[]>([])
const dataSourcesLoaded = ref(false)
const designerDatasets = ref<ReportDataset[]>([])
const designerParameters = ref<ReportParameter[]>([])
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const previewLoading = ref(false)
const previewDialogVisible = ref(false)
const previewData = ref<ReportPreviewRes>({ columns: [], rows: [], total: 0 })
const previewSheet = ref<ReportSheetConfig>(createEmptyPreviewSheet())
const loadError = ref('')
const layoutAreas = ref<ReportLayoutArea[]>([])
const selectedAreaId = ref('')
const selectedDatasetId = ref('')
const selectedParameterId = ref('')
const selectedItemRef = ref<SelectedItemRef | null>(null)
const dirty = ref(false)
const fieldKeyword = ref('')
const inspectorTab = ref<'cell' | 'data' | 'report'>('cell')
const canvasZoom = ref(1)
const datasetDialogVisible = ref(false)
const editingDatasetId = ref('')
const sqlFieldsLoading = ref(false)
const parameterDialogVisible = ref(false)
const editingParameterId = ref('')
const sqlDraftFields = ref<ReportField[]>([])

const datasetDraft = reactive<{
  type: ReportDatasetType
  name: string
  source_code: string
  sql: string
  fieldsText: string
}>({
  type: 'table',
  name: '',
  source_code: '',
  sql: '',
  fieldsText: '',
})

const parameterDraft = reactive<{
  id: string
  label: string
  dataset_id: string
  field: string
  type: ReportParameterType
  operator: ReportParameterOperator
  placeholder: string
  default_value: string
}>({
  id: '',
  label: '',
  dataset_id: '',
  field: '',
  type: 'text',
  operator: 'like',
  placeholder: '',
  default_value: '',
})

const alignOptions: Array<{ label: string; value: ReportLayoutAreaItemAlign }> = [
  { label: '左对齐', value: 'left' },
  { label: '居中', value: 'center' },
  { label: '右对齐', value: 'right' },
]

const formatOptions: Array<{ label: string; value: ReportLayoutAreaItemFormat }> = [
  { label: '文本', value: 'text' },
  { label: '数字', value: 'number' },
  { label: '金额', value: 'amount' },
  { label: '日期', value: 'date' },
  { label: '日期时间', value: 'datetime' },
  { label: '字典', value: 'dict' },
]

const aggregateOptions: Array<{ label: string; value: Exclude<ReportLayoutAreaAggregate, 'none'> }> = [
  { label: '求和', value: 'sum' },
  { label: '计数', value: 'count' },
]

const zoomOptions = [
  { label: '75%', value: 0.75 },
  { label: '90%', value: 0.9 },
  { label: '100%', value: 1 },
  { label: '125%', value: 1.25 },
]

const reportId = computed(() => toPositiveNumber(route.params.id))
const datasets = computed(() => designerDatasets.value)
const primaryDataset = computed(() => datasets.value.find((item) => item.primary) || datasets.value[0])
const selectedResourceDataset = computed(() =>
  datasets.value.find((item) => item.id === selectedDatasetId.value) || primaryDataset.value,
)
const parameters = computed(() => designerParameters.value)
const primaryFields = computed(() => primaryDataset.value?.fields || [])
const selectedDatasetFields = computed(() => selectedResourceDataset.value?.fields || [])
const filteredPrimaryFields = computed(() => {
  const keyword = fieldKeyword.value.trim().toLowerCase()
  if (!keyword) return selectedDatasetFields.value
  return selectedDatasetFields.value.filter((field) =>
    `${field.name || ''} ${field.code || ''} ${field.type || ''}`.toLowerCase().includes(keyword),
  )
})
const editingDataset = computed(() => datasets.value.find((item) => item.id === editingDatasetId.value))
const datasetDraftPreviewFields = computed(() => {
  if (datasetDraft.type === 'table') {
    return dataSources.value.find((item) => item.code === datasetDraft.source_code)?.fields || []
  }
  return sqlDraftFields.value.length ? sqlDraftFields.value : parseSqlDatasetFields(datasetDraft.fieldsText)
})
const datasetOptions = computed(() =>
  datasets.value.map((item) => ({
    label: `${item.name || item.id} (${item.type === 'sql' ? 'SQL' : item.source_code || item.id})`,
    value: item.id,
  })),
)
const parameterFieldOptions = computed(() => {
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  return (dataset?.fields || []).map((field) => ({
    label: `${field.name || field.code} (${field.code})`,
    value: field.code,
  }))
})
const navigationAreas = computed(() => layoutAreas.value)
const canvasLayoutAreas = computed(() => layoutAreas.value.filter((area) => area.visible !== false))
const selectedArea = computed(() =>
  layoutAreas.value.find((area) => area.id === selectedAreaId.value) || layoutAreas.value[0],
)
const selectedItemArea = computed(() => {
  if (!selectedItemRef.value) return null
  return layoutAreas.value.find((area) => area.id === selectedItemRef.value?.areaId) || null
})
const selectedItem = computed(() => {
  const target = selectedItemRef.value
  if (!target) return null
  return selectedItemArea.value?.items.find((item) => item.id === target.itemId) || null
})
const headerItems = computed(() => areaItems('header'))
const detailBindings = computed(() => areaItems('detail').filter((item) => item.type === 'field' && item.visible !== false))
const summaryItems = computed(() => areaItems('summary').filter((item) => item.type === 'summary'))
const joinedFieldKeys = computed(() => new Set(headerItems.value.map((item) => fieldJoinKey(item.dataset_id, item.field)).filter(Boolean)))
const summaryFieldKeys = computed(() => new Set(summaryItems.value.map((item) => fieldJoinKey(item.dataset_id, item.field)).filter(Boolean)))
const numericFields = computed(() => selectedDatasetFields.value.filter(isNumericField))
const statusMeta = computed(() => reportStatusMeta(report.value?.status || 'draft'))
const sheetColumns = computed<DesignerSheetColumn[]>(() => {
  const fieldItems = detailBindings.value.length
    ? detailBindings.value
    : headerItems.value.filter((item) => item.visible !== false && item.field)
  const columnCount = Math.max(6, fieldItems.length || 1)
  return Array.from({ length: columnCount }, (_, index) => {
    const item = fieldItems[index]
    return {
      key: `col_${index + 1}`,
      label: columnLetter(index),
      width: Math.max(112, Number(item?.width || 128)),
    }
  })
})
const sheetGridStyle = computed(() => ({
  gridTemplateColumns: `46px ${sheetColumns.value.map((column) => `${column.width}px`).join(' ')}`,
  transform: `scale(${canvasZoom.value})`,
  transformOrigin: '0 0',
}))
const designerSheetRows = computed<DesignerSheetRow[]>(() =>
  canvasLayoutAreas.value.map((area, index) => ({
    index: index + 1,
    areaId: area.id,
    cells: designerCellsForArea(area),
  })),
)
const canGenerateSheet = computed(() => {
  if (!report.value || !detailBindings.value.length) return false
  const sheet = layoutAreasToSheet(layoutAreas.value, report.value)
  return Boolean(sheet.cells.length && sheet.detail_rows?.length)
})
const currentCellLabel = computed(() => {
  const area = selectedItemArea.value || selectedArea.value
  if (!area) return 'A1'
  const rowIndex = Math.max(1, canvasLayoutAreas.value.findIndex((current) => current.id === area.id) + 1)
  if (!selectedItem.value) return `A${rowIndex}`
  const colIndex = selectedItemColumnIndex(area, selectedItem.value)
  return `${columnLetter(colIndex)}${rowIndex}`
})
const canBindFieldToSelection = computed(() => Boolean(selectedItem.value || selectedArea.value))
const hasUnsupportedItems = computed(() => hasUnsupportedLayoutItems())
const publishChecks = computed(() => [
  { label: '已配置数据来源', ok: datasets.value.length > 0 },
  { label: '仅使用单结果数据集', ok: datasets.value.length <= 1 },
  { label: '已读取可用字段', ok: primaryFields.value.length > 0 },
  { label: '至少一个可见明细字段', ok: detailBindings.value.length > 0 },
  { label: '存在标题区', ok: hasArea('title') },
  { label: '存在明细区', ok: hasArea('detail') },
  { label: '可生成 sheet/cell/binding', ok: canGenerateSheet.value },
  { label: '不存在不支持项', ok: !hasUnsupportedItems.value },
  { label: '汇总区为可选能力', ok: hasArea('summary'), optional: true },
])

watch(
  reportId,
  () => {
    void loadReport()
  },
  { immediate: true },
)

async function loadReport() {
  const id = reportId.value
  if (!id) {
    report.value = null
    designerDatasets.value = []
    designerParameters.value = []
    layoutAreas.value = []
    selectedAreaId.value = ''
    selectedDatasetId.value = ''
    selectedParameterId.value = ''
    selectedItemRef.value = null
    loadError.value = '缺少报表 ID'
    return
  }
  loading.value = true
  loadError.value = ''
  dirty.value = false
  try {
    await ensureDataSourcesLoaded()
    const res = await reportApi.queryReportById(id)
    const nextReport = withDesignerResourceState(res.data)
    report.value = nextReport
    designerDatasets.value = reportDatasets(nextReport)
    designerParameters.value = reportParameters(nextReport)
    layoutAreas.value = initializeLayoutAreas(nextReport)
    selectedAreaId.value = layoutAreas.value[0]?.id || ''
    selectedDatasetId.value = designerDatasets.value[0]?.id || ''
    selectedParameterId.value = ''
    selectedItemRef.value = null
  } catch (error) {
    report.value = null
    designerDatasets.value = []
    designerParameters.value = []
    layoutAreas.value = []
    selectedAreaId.value = ''
    selectedDatasetId.value = ''
    selectedParameterId.value = ''
    selectedItemRef.value = null
    loadError.value = error instanceof Error && error.message ? error.message : '报表详情加载失败'
    $q.notify({ type: 'negative', message: loadError.value })
  } finally {
    loading.value = false
  }
}

async function ensureDataSourcesLoaded() {
  if (dataSourcesLoaded.value) return
  try {
    const res = await reportApi.queryDataSources()
    dataSources.value = res.data || []
  } catch {
    dataSources.value = []
    $q.notify({ type: 'negative', message: '数据源加载失败' })
  } finally {
    dataSourcesLoaded.value = true
  }
}

function withDesignerResourceState(sourceReport: Report): Report {
  const currentLayout = sourceReport.layout_config
  const currentQuery = sourceReport.query_config
  const nextDatasets = enrichReportDatasets(reportDatasets(sourceReport))
  const nextParameters = reportParameters(sourceReport)
  return {
    ...sourceReport,
    layout_config: {
      ...(currentLayout || {}),
      version: currentLayout?.version || currentQuery?.version || 1,
      view: currentLayout?.view || 'sheet',
      datasets: nextDatasets,
      dataset_joins: currentLayout?.dataset_joins || currentQuery?.dataset_joins || [],
      parameters: nextParameters,
      sheet: currentLayout?.sheet || {
        rows: 20,
        cols: 8,
        cells: [],
      },
    },
    query_config: {
      ...(currentQuery || {}),
      version: currentQuery?.version || currentLayout?.version || 1,
      datasets: currentQuery?.datasets?.length ? currentQuery.datasets : nextDatasets,
      dataset_joins: currentQuery?.dataset_joins || currentLayout?.dataset_joins || [],
      fields: currentQuery?.fields?.length ? currentQuery.fields : nextDatasets[0]?.fields || [],
      parameters: currentQuery?.parameters?.length ? currentQuery.parameters : nextParameters,
    },
  }
}

function enrichReportDatasets(sourceDatasets: ReportDataset[]): ReportDataset[] {
  return sourceDatasets.map((dataset) => {
    if (dataset.type !== 'table' || !dataset.source_code) {
      return {
        ...dataset,
        fields: (dataset.fields || []).map((field) => ({ ...field })),
      }
    }
    const source = dataSources.value.find((item) => item.code === dataset.source_code)
    if (!source) {
      return {
        ...dataset,
        fields: (dataset.fields || []).map((field) => ({ ...field })),
      }
    }
    const existingByCode = new Map((dataset.fields || []).map((field) => [field.code, field]))
    const sourceCodes = new Set(source.fields.map((field) => field.code))
    return {
      ...dataset,
      name: dataset.name || source.name,
      fields: [
        ...source.fields.map((field) => ({ ...(existingByCode.get(field.code) || field) })),
        ...(dataset.fields || [])
          .filter((field) => !sourceCodes.has(field.code))
          .map((field) => ({ ...field })),
      ],
    }
  })
}

async function saveDraft() {
  await persistDraft({ notify: true })
}

async function previewDesign() {
  const sourceReport = report.value
  const id = Number(sourceReport?.id || reportId.value)
  if (!sourceReport || !id) {
    $q.notify({ type: 'warning', message: '缺少报表信息，无法设计预览' })
    return
  }

  const nextSheet = layoutAreasToSheet(layoutAreas.value, sourceReport)
  const errors = validateDesignPreview(nextSheet)
  if (errors.length) {
    $q.notify({ type: 'warning', message: errors[0] || '当前配置暂不能预览' })
    return
  }

  previewSheet.value = nextSheet
  previewData.value = { columns: [], rows: [], total: 0 }
  previewDialogVisible.value = true
  previewLoading.value = true
  const saved = await persistDraft({ sheet: nextSheet, notify: false })
  if (!saved) {
    previewLoading.value = false
    return
  }

  try {
    const res = await reportApi.designPreviewReport(id, {
      report_id: id,
      dataset_id: primaryDataset.value?.id,
      data_source_id: primaryDataset.value?.source_code || primaryDataset.value?.id,
      page: 1,
      num: Number(report.value?.layout_config?.runtime_page_size || 20),
    })
    previewData.value = res
    $q.notify({ type: 'positive', message: '设计预览已生成' })
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '设计预览失败'
    $q.notify({ type: 'negative', message })
  } finally {
    previewLoading.value = false
  }
}

async function publishVersion() {
  const sourceReport = report.value
  const id = Number(sourceReport?.id || reportId.value)
  if (!sourceReport || !id) {
    $q.notify({ type: 'warning', message: '缺少报表信息，无法发布版本' })
    return
  }

  const nextSheet = layoutAreasToSheet(layoutAreas.value, sourceReport)
  const errors = validatePublishVersion(nextSheet)
  if (errors.length) {
    $q.notify({ type: 'warning', message: errors[0] || '当前配置暂不能发布' })
    return
  }

  const changeLog = await confirmPublishReport(nextSheet)
  if (changeLog === null) return

  publishing.value = true
  try {
    const saved = await persistDraft({ sheet: nextSheet, notify: false })
    if (!saved) return
    const res = await reportApi.publishReport(id, changeLog ? { change_log: changeLog } : {})
    if (report.value) {
      report.value = {
        ...report.value,
        status: res.status || 'published',
        published_version_id: res.version_id,
        published_version_no: res.version_no,
      }
    }
    dirty.value = false
    $q.notify({ type: 'positive', message: `报表已发布为 V${res.version_no}` })
    await refreshCurrentReport(id)
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '发布版本失败'
    $q.notify({ type: 'negative', message })
  } finally {
    publishing.value = false
  }
}

async function persistDraft(options: { sheet?: ReportSheetConfig; notify?: boolean } = {}): Promise<boolean> {
  const sourceReport = report.value
  const id = Number(sourceReport?.id || reportId.value)
  if (!sourceReport || !id) {
    $q.notify({ type: 'warning', message: '缺少报表信息，无法保存草稿' })
    return false
  }

  saving.value = true
  try {
    const nextLayoutConfig = buildNextLayoutConfig(sourceReport, options.sheet)
    const req = buildSaveReq(sourceReport, nextLayoutConfig)
    await reportApi.updateReport(req)
    const nextReport: Report = {
      ...sourceReport,
      layout_config: nextLayoutConfig,
    }
    if (req.query_config) {
      nextReport.query_config = req.query_config
    }
    report.value = nextReport
    layoutAreas.value = normalizeLayoutAreas(nextLayoutConfig.layout_areas || [])
    dirty.value = false
    if (options.notify !== false) {
      $q.notify({ type: 'positive', message: '报表草稿已保存' })
    }
    return true
  } catch (error) {
    const message = error instanceof Error && error.message ? error.message : '报表草稿保存失败'
    $q.notify({ type: 'negative', message })
    return false
  } finally {
    saving.value = false
  }
}

function confirmPublishReport(sheet: ReportSheetConfig) {
  return new Promise<string | null>((resolve) => {
    $q.dialog({
      title: '发布版本',
      message: [
        `报表名称：${report.value?.report_name || report.value?.name || '未命名报表'}`,
        '当前设计模式：版式报表',
        `可见明细字段：${detailBindings.value.length} 个`,
        `汇总项：${areaItems('summary').filter((item) => item.visible !== false).length} 个`,
        `生成单元格：${sheet.cells.length} 个`,
        '发布后运行页将读取新的发布版本。',
        '如果报表已发布到菜单，菜单运行也会读取最新发布版本。',
      ].join('<br>'),
      html: true,
      prompt: {
        model: '',
        type: 'textarea',
        label: '发布说明（可选）',
      },
      cancel: true,
      persistent: true,
    })
      .onOk((value) => resolve(String(value || '').trim()))
      .onCancel(() => resolve(null))
      .onDismiss(() => resolve(null))
  })
}

async function refreshCurrentReport(id: number) {
  try {
    const res = await reportApi.queryReportById(id)
    const refreshedReport = withDesignerResourceState(res.data)
    const hasPersistedLayout = refreshedReport.layout_config?.designer_mode === 'layout' &&
      Boolean(refreshedReport.layout_config.layout_areas?.length)
    report.value = refreshedReport
    designerDatasets.value = reportDatasets(refreshedReport)
    designerParameters.value = reportParameters(refreshedReport)
    if (hasPersistedLayout) {
      layoutAreas.value = initializeLayoutAreas(refreshedReport)
      selectedAreaId.value = layoutAreas.value[0]?.id || ''
      selectedItemRef.value = null
    }
    selectedDatasetId.value = designerDatasets.value[0]?.id || ''
    selectedParameterId.value = ''
    dirty.value = false
  } catch {
    $q.notify({ type: 'warning', message: '发布成功，但报表详情刷新失败，请稍后手动刷新' })
  }
}

function buildNextLayoutConfig(sourceReport: Report, nextSheet?: ReportSheetConfig): ReportLayoutConfig {
  const current = sourceReport.layout_config
  return {
    ...(current || {}),
    version: current?.version || sourceReport.query_config?.version || 1,
    view: current?.view || 'sheet',
    title: current?.title || sourceReport.report_name || sourceReport.name || '',
    subtitle: current?.subtitle || sourceReport.description || '',
    kind: current?.kind || sourceReport.report_kind || 'detail',
    datasets: datasets.value.map((dataset) => ({
      ...dataset,
      fields: (dataset.fields || []).map((field) => ({ ...field })),
    })),
    dataset_joins: current?.dataset_joins || sourceReport.query_config?.dataset_joins || [],
    parameters: parameters.value.map((parameter) => ({ ...parameter })),
    sheet: nextSheet || current?.sheet || {
      rows: 20,
      cols: 8,
      cells: [],
    },
    runtime_display: current?.runtime_display || 'paged',
    runtime_page_size: Number(current?.runtime_page_size || 20),
    designer_mode: 'layout',
    layout_areas: normalizeLayoutAreas(layoutAreas.value),
  }
}

function buildSaveReq(sourceReport: Report, layoutConfig: ReportLayoutConfig): ReportSaveReq {
  const queryConfig = sourceReport.query_config || {
    version: layoutConfig.version,
    datasets: layoutConfig.datasets,
    dataset_joins: layoutConfig.dataset_joins || [],
    fields: primaryFields.value,
    parameters: layoutConfig.parameters,
  }
  const saveDatasets = layoutConfig.datasets?.length ? layoutConfig.datasets : queryConfig.datasets
  const saveParameters = layoutConfig.parameters?.length ? layoutConfig.parameters : queryConfig.parameters
  const saveDatasetJoins = queryConfig.dataset_joins || layoutConfig.dataset_joins || []
  const saveFields = primaryFields.value.length ? primaryFields.value : queryConfig.fields || []
  return {
    id: Number(sourceReport.id || reportId.value),
    report_name: sourceReport.report_name || sourceReport.name || '',
    report_code: sourceReport.report_code || sourceReport.code || '',
    report_kind: sourceReport.report_kind || layoutConfig.kind || 'detail',
    category: sourceReport.category || '',
    description: sourceReport.description || '',
    fields: saveFields,
    datasets: saveDatasets,
    dataset_joins: saveDatasetJoins,
    parameters: saveParameters,
    sheet: layoutConfig.sheet,
    status: sourceReport.status,
    query_config: {
      ...queryConfig,
      datasets: saveDatasets,
      dataset_joins: saveDatasetJoins,
      fields: saveFields,
      parameters: saveParameters,
    },
    layout_config: layoutConfig,
    ...(sourceReport.data_source_id || sourceReport.source_code || primaryDataset.value?.source_code || primaryDataset.value?.id
      ? { data_source_id: sourceReport.data_source_id || sourceReport.source_code || primaryDataset.value?.source_code || primaryDataset.value?.id }
      : {}),
    ...(sourceReport.permission_menu_id !== undefined ? { permission_menu_id: sourceReport.permission_menu_id } : {}),
    ...(sourceReport.permission_table_code !== undefined ? { permission_table_code: sourceReport.permission_table_code } : {}),
    ...(layoutConfig.runtime_display !== undefined ? { runtime_display: layoutConfig.runtime_display } : {}),
    ...(layoutConfig.runtime_page_size !== undefined ? { runtime_page_size: layoutConfig.runtime_page_size } : {}),
  }
}

async function openDatasetDialog(id = '') {
  await ensureDataSourcesLoaded()
  editingDatasetId.value = id
  const current = datasets.value.find((item) => item.id === id)
  if (current) {
    datasetDraft.type = current.type
    datasetDraft.source_code = current.source_code || ''
    datasetDraft.name = current.name || ''
    datasetDraft.sql = current.sql || ''
    datasetDraft.fieldsText = (current.fields || []).map((field) => field.code).join(',')
    sqlDraftFields.value = current.type === 'sql' ? [...(current.fields || [])] : []
    datasetDialogVisible.value = true
    return
  }
  datasetDraft.type = 'table'
  datasetDraft.source_code = ''
  datasetDraft.name = ''
  datasetDraft.sql = ''
  datasetDraft.fieldsText = ''
  sqlDraftFields.value = []
  datasetDialogVisible.value = true
}

function handleDraftTypeChange(value: ReportDatasetType) {
  datasetDraft.type = value
  if (datasetDraft.type === 'table') {
    sqlDraftFields.value = []
    handleDraftSourceChange(datasetDraft.source_code || '')
  } else {
    datasetDraft.name = 'SQL 数据集'
    datasetDraft.source_code = ''
    sqlDraftFields.value = parseSqlDatasetFields(datasetDraft.fieldsText)
  }
}

function handleDraftSourceChange(value: string) {
  const source = dataSources.value.find((item) => item.code === value)
  datasetDraft.source_code = value
  datasetDraft.name = source?.name || value
}

function handleDatasetDraftSqlChange(value: string) {
  datasetDraft.sql = value
  sqlDraftFields.value = []
  datasetDraft.fieldsText = ''
}

async function inferSqlDatasetFields() {
  const sql = datasetDraft.sql.trim()
  if (!sql) {
    $q.notify({ type: 'warning', message: '请先填写 SQL' })
    return false
  }
  sqlFieldsLoading.value = true
  try {
    const res = await reportApi.inferSqlFields(sql)
    sqlDraftFields.value = res.data || []
    datasetDraft.fieldsText = sqlDraftFields.value.map((field) => field.code).join(',')
    if (!sqlDraftFields.value.length) {
      $q.notify({ type: 'warning', message: 'SQL 未解析出字段' })
      return false
    }
    $q.notify({ type: 'positive', message: `已解析 ${sqlDraftFields.value.length} 个字段` })
    return true
  } catch (error) {
    sqlDraftFields.value = []
    $q.notify({ type: 'negative', message: sqlFieldInferErrorMessage(error) })
    return false
  } finally {
    sqlFieldsLoading.value = false
  }
}

async function confirmDataset() {
  const id = editingDatasetId.value || `dataset_${Date.now()}`
  if (datasetDraft.type === 'table') {
    const source = dataSources.value.find((item) => item.code === datasetDraft.source_code)
    if (!source) {
      $q.notify({ type: 'warning', message: '请选择来源表' })
      return
    }
    upsertDataset({
      id,
      name: datasetDraft.name || source.name,
      type: 'table',
      source_code: source.code,
      fields: source.fields,
      primary: editingDataset.value?.primary || datasets.value.length === 0,
    })
  } else {
    if (!datasetDraft.sql.trim()) {
      $q.notify({ type: 'warning', message: '请填写 SQL' })
      return
    }
    let fields = datasetDraftPreviewFields.value
    if (!fields.length) {
      const ok = await inferSqlDatasetFields()
      fields = datasetDraftPreviewFields.value
      if (!ok || !fields.length) {
        $q.notify({ type: 'warning', message: '请先解析 SQL 字段' })
        return
      }
    }
    upsertDataset({
      id,
      name: datasetDraft.name || 'SQL 数据集',
      type: 'sql',
      sql: datasetDraft.sql,
      fields,
      primary: editingDataset.value?.primary || datasets.value.length === 0,
    })
  }
  selectedDatasetId.value = id
  editingDatasetId.value = ''
  datasetDialogVisible.value = false
  dirty.value = true
}

function parseSqlDatasetFields(fieldsText: string): ReportField[] {
  return fieldsText
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((code) => ({ code, name: code, type: 'string', role: 'text' as const }))
}

function upsertDataset(dataset: ReportDataset) {
  const normalizedDataset = enrichReportDatasets([dataset])[0] || dataset
  const index = designerDatasets.value.findIndex((item) => item.id === dataset.id)
  if (index === -1) {
    designerDatasets.value = [...designerDatasets.value, normalizedDataset]
  } else {
    designerDatasets.value = designerDatasets.value.map((item) =>
      item.id === dataset.id ? normalizedDataset : item,
    )
  }
  if (!designerDatasets.value.some((item) => item.primary) && designerDatasets.value[0]) {
    designerDatasets.value[0].primary = true
  }
}

function removeDataset(id: string) {
  const removed = designerDatasets.value.find((item) => item.id === id)
  designerDatasets.value = designerDatasets.value.filter((item) => item.id !== id)
  designerParameters.value = designerParameters.value.filter((param) => param.dataset_id !== id)
  patchLayoutAreas((areas) => areas.map((area) => ({
    ...area,
    items: area.items.filter((item) => item.dataset_id !== id),
  })))
  if (removed?.primary && designerDatasets.value[0]) {
    designerDatasets.value[0].primary = true
  }
  selectedDatasetId.value = designerDatasets.value[0]?.id || ''
  selectedParameterId.value = ''
  dirty.value = true
}

function openParameterDialog(id = '') {
  const dataset = selectedResourceDataset.value || primaryDataset.value
  const field = dataset?.fields[0]
  if (!dataset || !field) {
    $q.notify({ type: 'warning', message: '请先添加数据集' })
    return
  }
  const current = parameters.value.find((item) => item.id === id)
  editingParameterId.value = id
  parameterDraft.id = current?.id || ''
  parameterDraft.label = current?.label || field.name || field.code
  parameterDraft.dataset_id = current?.dataset_id || dataset.id
  parameterDraft.field = current?.field || field.code
  const defaults = reportParameterDefaultsForField(field)
  parameterDraft.type = current?.type || defaults.type
  parameterDraft.operator = current?.operator || defaults.operator
  parameterDraft.placeholder = current?.placeholder || `请输入${field.name || field.code}`
  parameterDraft.default_value = stringifyParameterDefault(current?.default_value)
  parameterDialogVisible.value = true
}

function handleParameterDatasetChange(id: string) {
  parameterDraft.dataset_id = id
  const field = datasets.value.find((item) => item.id === id)?.fields[0]
  parameterDraft.field = field?.code || ''
  parameterDraft.default_value = ''
  if (field) {
    const defaults = reportParameterDefaultsForField(field)
    parameterDraft.label = field.name || field.code
    parameterDraft.placeholder = `请输入${field.name || field.code}`
    parameterDraft.type = defaults.type
    parameterDraft.operator = defaults.operator
  }
}

function handleParameterFieldChange(fieldCode: string) {
  parameterDraft.field = fieldCode
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  const field = dataset?.fields.find((item) => item.code === fieldCode)
  if (!field) return
  parameterDraft.label = field.name || field.code
  parameterDraft.placeholder = `请输入${field.name || field.code}`
  parameterDraft.default_value = ''
  const defaults = reportParameterDefaultsForField(field)
  parameterDraft.type = defaults.type
  parameterDraft.operator = defaults.operator
}

function confirmParameter() {
  if (!parameterDraft.label.trim() || !parameterDraft.dataset_id || !parameterDraft.field) {
    $q.notify({ type: 'warning', message: '请完整配置参数名称、数据集和字段' })
    return
  }
  const dataset = datasets.value.find((item) => item.id === parameterDraft.dataset_id)
  const field = dataset?.fields.find((item) => item.code === parameterDraft.field)
  const compatibilityError = parameterCompatibilityError(parameterDraft.type, parameterDraft.operator, field)
  if (compatibilityError) {
    $q.notify({ type: 'warning', message: compatibilityError })
    return
  }
  const id = editingParameterId.value || `param_${Date.now()}`
  const param: ReportParameter = {
    id,
    label: parameterDraft.label,
    dataset_id: parameterDraft.dataset_id,
    field: parameterDraft.field,
    type: parameterDraft.type,
    operator: parameterDraft.operator,
    placeholder: parameterDraft.placeholder,
    default_value: parseParameterDefault(parameterDraft.default_value, parameterDraft.type),
  }
  const index = designerParameters.value.findIndex((item) => item.id === id)
  if (index === -1) {
    designerParameters.value = [...designerParameters.value, param]
  } else {
    designerParameters.value = designerParameters.value.map((item) => item.id === id ? param : item)
  }
  syncParameterSummaryArea()
  selectedParameterId.value = param.id
  parameterDialogVisible.value = false
  editingParameterId.value = ''
  dirty.value = true
}

function removeParameter(id: string) {
  designerParameters.value = designerParameters.value.filter((item) => item.id !== id)
  if (selectedParameterId.value === id) selectedParameterId.value = ''
  syncParameterSummaryArea()
  dirty.value = true
}

function syncParameterSummaryArea() {
  patchArea('parameter_summary', (area) => ({
    ...area,
    items: parameters.value.map((parameter, index) => ({
      id: `parameter_${sanitizeId(parameter.id || parameter.field)}`,
      type: 'parameter',
      parameter_id: parameter.id,
      field: parameter.field,
      label: parameter.label || parameter.field,
      format: parameter.type === 'date' || parameter.type === 'date_range' ? 'date' : 'text',
      visible: true,
      order: index + 1,
      ...(parameter.dataset_id ? { dataset_id: parameter.dataset_id } : {}),
    })),
  }))
}

function stringifyParameterDefault(value: ReportParameter['default_value']) {
  if (Array.isArray(value)) return value.map((item) => String(item ?? '')).filter(Boolean).join(',')
  if (value === null || value === undefined) return ''
  return String(value)
}

function parseParameterDefault(value: string, type: ReportParameterType): ReportParameter['default_value'] {
  const text = value.trim()
  if (!text) return undefined
  if (type === 'date_range') {
    const parts = text.split(',').map((item) => item.trim()).filter(Boolean)
    return parts.length ? parts.slice(0, 2) : undefined
  }
  if (type === 'number') {
    const num = Number(text)
    return Number.isFinite(num) ? num : text
  }
  return text
}

function parameterCompatibilityError(
  type: ReportParameterType,
  operator: ReportParameterOperator,
  field?: ReportField,
) {
  if (!field) return '请选择有效字段'
  if (type === 'date_range' && operator !== 'between') return '日期范围参数必须使用区间匹配'
  if (operator === 'between' && type !== 'date_range') return '区间匹配请使用日期范围参数'
  if (operator === 'like' && isNumericField(field)) return '数字字段不能使用包含匹配'
  if (type === 'number' && operator === 'like') return '数字参数不能使用包含匹配'
  return ''
}

function parameterDatasetLabel(datasetId?: string) {
  return datasets.value.find((item) => item.id === datasetId)?.name || datasetId || '未绑定数据集'
}

function sqlFieldInferErrorMessage(error: unknown) {
  const status = (error as { response?: { status?: number } })?.response?.status
  if (status === 401 || status === 403) {
    return 'SQL 字段解析接口无权限，请给当前角色分配“SQL字段解析”接口权限'
  }
  return error instanceof Error && error.message ? error.message : 'SQL 字段解析失败，请检查 SQL 语句或后端接口'
}

function layoutAreasToSheet(areas: ReportLayoutArea[], sourceReport: Report): ReportSheetConfig {
  const normalizedAreas = normalizeLayoutAreas(areas)
  const dataset = primaryDataset.value || reportDatasets(sourceReport)[0]
  const detailItems = visibleItemsByType(normalizedAreas, 'detail')
    .filter((item) => item.type === 'field' && item.field)
  const headerMap = new Map(
    visibleItemsByType(normalizedAreas, 'header')
      .filter((item) => item.field)
      .map((item) => [item.field, item]),
  )
  const columns = detailItems.map((detailItem) => ({
    detail: detailItem,
    header: detailItem.field ? headerMap.get(detailItem.field) || detailItem : detailItem,
  }))
  const colCount = Math.max(1, columns.length)
  const cells: ReportSheetCell[] = []
  const detailRows: number[] = []
  const summaryRows: number[] = []
  let row = 1

  const titleArea = visibleAreaByType(normalizedAreas, 'title')
  if (titleArea) {
    const titleItem = visibleAreaItems(titleArea)[0]
    cells.push(makeSheetCell({
      row,
      col: 1,
      value: String(titleItem?.value || titleItem?.label || sourceReport.report_name || sourceReport.name || '未命名报表'),
      colspan: colCount,
      style: { bold: true, align: 'center', background: '#f8fbff' },
    }))
    row += 1
  }

  const parameterArea = visibleAreaByType(normalizedAreas, 'parameter_summary')
  if (parameterArea) {
    const labels = visibleAreaItems(parameterArea).map((item) => item.label || item.field || item.parameter_id).filter(Boolean)
    cells.push(makeSheetCell({
      row,
      col: 1,
      value: `查询条件：${labels.length ? labels.join(' / ') : '全部'}`,
      colspan: colCount,
      style: { align: 'left', background: '#f9fafb', color: '#667085' },
    }))
    row += 1
  }

  if (visibleAreaByType(normalizedAreas, 'header')) {
    columns.forEach(({ header }, index) => {
      cells.push(makeSheetCell({
        row,
        col: index + 1,
        value: header.label || header.field || `字段${index + 1}`,
        style: { bold: true, align: header.align || 'center', background: '#eef4ff' },
      }))
    })
    row += 1
  }

  if (visibleAreaByType(normalizedAreas, 'detail')) {
    detailRows.push(row)
    columns.forEach(({ detail }, index) => {
      cells.push(makeSheetCell({
        row,
        col: index + 1,
        value: detail.label || detail.field || '',
        binding: makeFieldBinding(detail, dataset),
        style: { align: detail.align || 'left', background: '#fff' },
      }))
    })
    row += 1
  }

  const summaryArea = visibleAreaByType(normalizedAreas, 'summary')
  const summaryItems = summaryArea
    ? visibleAreaItems(summaryArea).filter((item) => item.type === 'summary' && item.field)
    : []
  if (summaryItems.length) {
    summaryRows.push(row)
    if (colCount > 1) {
      cells.push(makeSheetCell({
        row,
        col: 1,
        value: '合计',
        style: { bold: true, align: 'right', background: '#fff7ed' },
      }))
    }
    const maxSummaryCells = colCount > 1 ? colCount - 1 : 1
    summaryItems.slice(0, maxSummaryCells).forEach((item, index) => {
      const col = colCount > 1 ? Math.min(index + 2, colCount) : 1
      cells.push(makeSheetCell({
        row,
        col,
        value: item.label || item.field || '',
        binding: makeSummaryBinding(item, dataset),
        style: { bold: true, align: item.align || 'right', background: '#fff7ed' },
      }))
    })
    row += 1
  }

  const footerArea = visibleAreaByType(normalizedAreas, 'footer')
  if (footerArea) {
    const footerItem = visibleAreaItems(footerArea)[0]
    cells.push(makeSheetCell({
      row,
      col: 1,
      value: String(footerItem?.value || footerItem?.label || '制表时间：{{now}}'),
      colspan: colCount,
      style: { align: 'right', background: '#f9fafb', color: '#667085' },
    }))
    row += 1
  }

  return {
    rows: Math.max(row - 1, 1),
    cols: colCount,
    scale: 0.85,
    detail_rows: detailRows,
    summary_rows: summaryRows,
    cells,
  }
}

function validateDesignPreview(sheet?: ReportSheetConfig) {
  const errors: string[] = []
  const dataset = primaryDataset.value
  if (!datasets.value.length || !dataset) errors.push('请先配置数据来源')
  const detailItems = areaItems('detail').filter((item) => item.visible !== false)
  const detailFields = detailItems.filter((item) => item.type === 'field' && item.field)
  if (!detailFields.length) errors.push('请至少配置一个可见明细字段')
  if (detailItems.some((item) => item.type === 'field' && !item.field)) {
    errors.push('明细区存在未绑定字段')
  }
  const summaryInvalid = areaItems('summary')
    .filter((item) => item.visible !== false)
    .some((item) => item.type === 'summary' && (!item.field || !item.aggregate || String(item.aggregate) === 'avg'))
  if (summaryInvalid) errors.push('汇总区仅支持 sum/count 且必须绑定字段')
  const primaryDatasetId = dataset?.id
  const boundItems = [...detailItems, ...areaItems('summary')]
    .filter((item) => item.visible !== false && item.field)
  if (primaryDatasetId && boundItems.some((item) => item.dataset_id && item.dataset_id !== primaryDatasetId)) {
    errors.push('第一阶段不支持跨数据集字段绑定')
  }
  if (sheet && !sheet.detail_rows?.length) errors.push('当前配置无法生成明细行')
  if (sheet && !sheet.cells.length) errors.push('当前配置无法生成 sheet 单元格')
  return errors
}

function validatePublishVersion(sheet: ReportSheetConfig) {
  const errors = validateDesignPreview(sheet)
  if (datasets.value.length > 1) {
    errors.push('第一阶段发布仅支持单结果数据集')
  }
  return errors
}

function hasUnsupportedLayoutItems() {
  const primaryDatasetId = primaryDataset.value?.id
  const items = [...areaItems('detail'), ...areaItems('summary')]
    .filter((item) => item.visible !== false && item.field)
  const hasCrossDataset = Boolean(primaryDatasetId && items.some((item) => item.dataset_id && item.dataset_id !== primaryDatasetId))
  const hasAvg = areaItems('summary')
    .filter((item) => item.visible !== false)
    .some((item) => String(item.aggregate || '') === 'avg')
  return hasCrossDataset || hasAvg
}

function visibleAreaByType(areas: ReportLayoutArea[], type: ReportLayoutAreaType) {
  return areas.find((area) => area.type === type && area.visible !== false)
}

function visibleItemsByType(areas: ReportLayoutArea[], type: ReportLayoutAreaType) {
  const area = visibleAreaByType(areas, type)
  return area ? visibleAreaItems(area) : []
}

function makeFieldBinding(item: ReportLayoutAreaItem, dataset?: ReportDataset): NonNullable<ReportSheetCell['binding']> {
  const binding: NonNullable<ReportSheetCell['binding']> = {
    type: 'field',
    field: item.field || '',
  }
  const datasetId = item.dataset_id || dataset?.id || ''
  if (datasetId) {
    binding.dataset_id = datasetId
  }
  return binding
}

function makeSummaryBinding(item: ReportLayoutAreaItem, dataset?: ReportDataset): NonNullable<ReportSheetCell['binding']> {
  const type = item.aggregate === 'count' ? 'count' : 'sum'
  const binding: NonNullable<ReportSheetCell['binding']> = {
    type,
    field: item.field || '',
  }
  const datasetId = item.dataset_id || dataset?.id || ''
  if (datasetId) {
    binding.dataset_id = datasetId
  }
  return binding
}

function makeSheetCell(options: {
  row: number
  col: number
  value: string
  binding?: ReportSheetCell['binding']
  style?: ReportSheetCell['style']
  colspan?: number
  rowspan?: number
}): ReportSheetCell {
  const cell: ReportSheetCell = {
    id: makeReportCellId(options.row, options.col),
    row: options.row,
    col: options.col,
    value: options.value,
  }
  if (options.binding) cell.binding = options.binding
  if (options.style) cell.style = options.style
  if (options.colspan && options.colspan > 1) cell.colspan = options.colspan
  if (options.rowspan && options.rowspan > 1) cell.rowspan = options.rowspan
  return cell
}

function createEmptyPreviewSheet(): ReportSheetConfig {
  return {
    rows: 1,
    cols: 1,
    scale: 0.85,
    detail_rows: [],
    summary_rows: [],
    cells: [],
  }
}

function initializeLayoutAreas(sourceReport: Report): ReportLayoutArea[] {
  const existing = sourceReport.layout_config?.layout_areas
  if (sourceReport.layout_config?.designer_mode === 'layout' && existing?.length) {
    return normalizeLayoutAreas(existing)
  }
  return createDefaultLayoutAreas(sourceReport)
}

function createDefaultLayoutAreas(sourceReport: Report): ReportLayoutArea[] {
  const dataset = reportDatasets(sourceReport)[0]
  const datasetId = dataset?.id || 'main'
  const fields = reportFields(sourceReport).slice(0, 8)
  const params = reportParameters(sourceReport)
  return normalizeLayoutAreas([
    {
      id: 'title',
      type: 'title',
      title: '标题区',
      visible: true,
      order: 1,
      items: [{
        id: 'title_text',
        type: 'static_text',
        value: sourceReport.report_name || sourceReport.name || '未命名报表',
        label: sourceReport.report_name || sourceReport.name || '未命名报表',
        align: 'center',
        format: 'text',
        visible: true,
        order: 1,
      }],
    },
    {
      id: 'parameter_summary',
      type: 'parameter_summary',
      title: '查询条件展示区',
      visible: true,
      order: 2,
      items: params.map((parameter, index) => ({
        id: `parameter_${sanitizeId(parameter.id || parameter.field)}`,
        type: 'parameter',
        parameter_id: parameter.id,
        field: parameter.field,
        label: parameter.label || parameter.field,
        format: parameter.type === 'date' || parameter.type === 'date_range' ? 'date' : 'text',
        visible: true,
        order: index + 1,
      })),
    },
    {
      id: 'header',
      type: 'header',
      title: '表头区',
      visible: true,
      order: 3,
      items: fields.map((field, index) => makeHeaderItem(field, datasetId, index + 1)),
    },
    {
      id: 'detail',
      type: 'detail',
      title: '明细区',
      visible: true,
      order: 4,
      items: fields.map((field, index) => makeDetailItem(field, datasetId, index + 1)),
    },
    {
      id: 'summary',
      type: 'summary',
      title: '汇总区',
      visible: true,
      order: 5,
      items: [],
    },
    {
      id: 'footer',
      type: 'footer',
      title: '页脚区',
      visible: true,
      order: 6,
      items: [{
        id: 'footer_runtime',
        type: 'runtime',
        label: '制表时间 / 制表人',
        value: '制表时间：运行时生成 ｜ 制表人：当前用户',
        align: 'right',
        format: 'text',
        visible: true,
        order: 1,
      }],
    },
  ])
}

function makeHeaderItem(field: ReportField, datasetId: string, order: number): ReportLayoutAreaItem {
  return {
    id: `header_${sanitizeId(field.code)}`,
    type: 'field',
    dataset_id: datasetId,
    field: field.code,
    label: field.name || field.code,
    width: defaultFieldWidth(field),
    align: defaultFieldAlign(field),
    format: formatForField(field),
    visible: true,
    order,
  }
}

function makeDetailItem(field: ReportField, datasetId: string, order: number): ReportLayoutAreaItem {
  return {
    id: `detail_${sanitizeId(field.code)}`,
    type: 'field',
    dataset_id: datasetId,
    field: field.code,
    label: field.name || field.code,
    width: defaultFieldWidth(field),
    align: defaultFieldAlign(field),
    format: formatForField(field),
    visible: true,
    order,
  }
}

function makeSummaryItem(field: ReportField, datasetId: string, order: number): ReportLayoutAreaItem {
  return {
    id: `summary_${sanitizeId(field.code)}`,
    type: 'summary',
    dataset_id: datasetId,
    field: field.code,
    label: `${field.name || field.code}合计`,
    width: defaultFieldWidth(field),
    align: 'right',
    format: formatForField(field),
    aggregate: 'sum',
    visible: true,
    order,
  }
}

function normalizeLayoutAreas(areas: ReportLayoutArea[]) {
  return [...areas]
    .map((area) => ({
      ...area,
      visible: area.visible !== false,
      items: normalizeItemOrders(area.items || []),
    }))
    .sort((a, b) => a.order - b.order)
}

function normalizeItemOrders(items: ReportLayoutAreaItem[]) {
  return orderedItems(items).map((item, index) => ({
    ...item,
    order: index + 1,
  }))
}

function orderedItems(items: ReportLayoutAreaItem[]) {
  return [...items].sort((a, b) => (a.order || 0) - (b.order || 0))
}

function reportDatasets(sourceReport: Report): ReportDataset[] {
  if (sourceReport.layout_config?.datasets?.length) return sourceReport.layout_config.datasets
  if (sourceReport.query_config?.datasets?.length) return sourceReport.query_config.datasets
  const sourceCode = sourceReport.source_code || sourceReport.data_source_id
  const source = dataSources.value.find((item) => item.code === sourceCode)
  if (!source) return []
  return [{
    id: 'main',
    name: source.name || '主数据集',
    type: 'table',
    source_code: source.code,
    fields: source.fields,
    primary: true,
  }]
}

function reportParameters(sourceReport: Report): ReportParameter[] {
  return sourceReport.layout_config?.parameters?.length
    ? sourceReport.layout_config.parameters
    : sourceReport.query_config?.parameters || []
}

function reportFields(sourceReport: Report): ReportField[] {
  const dataset = reportDatasets(sourceReport)[0]
  const fields = dataset?.fields?.length ? dataset.fields : sourceReport.query_config?.fields || []
  return fields.filter((field) => field.selected !== false)
}

function areaItems(type: ReportLayoutAreaType) {
  return orderedItems(layoutAreas.value.find((area) => area.type === type)?.items || [])
}

function hasArea(type: ReportLayoutAreaType) {
  return layoutAreas.value.some((area) => area.type === type)
}

function visibleAreaItems(area: ReportLayoutArea) {
  return orderedItems(area.items).filter((item) => item.visible !== false)
}

function designerCellsForArea(area: ReportLayoutArea): DesignerSheetCell[] {
  if (area.type === 'header' || area.type === 'detail') {
    return fieldBandCells(area)
  }
  if (area.type === 'summary') {
    return summaryBandCells(area)
  }
  return fullBandCell(area)
}

function fullBandCell(area: ReportLayoutArea): DesignerSheetCell[] {
  const item = visibleAreaItems(area)[0]
  const cell: DesignerSheetCell = {
    id: `${area.id}_full`,
    areaId: area.id,
    col: 1,
    colspan: sheetColumns.value.length,
    text: fullBandText(area, item),
    subtext: areaTypeLabel(area.type),
    bandLabel: sheetBandLabel(area.type),
    tone: area.type,
    align: item?.align || (area.type === 'title' ? 'center' : 'left'),
    anchor: true,
  }
  if (item?.id) {
    cell.itemId = item.id
  }
  return [cell]
}

function fieldBandCells(area: ReportLayoutArea): DesignerSheetCell[] {
  const items = visibleAreaItems(area).filter((item) => item.field)
  return sheetColumns.value.map((column, index) => {
    const item = items[index]
    if (!item) {
      return emptyDesignerCell(area, index + 1, column.label, index === 0)
    }
    return {
      id: `${area.id}_${item.id}`,
      areaId: area.id,
      itemId: item.id,
      col: index + 1,
      text: area.type === 'detail' ? `{{${item.field}}}` : (item.label || item.field || column.label),
      subtext: area.type === 'detail' ? (item.label || '') : (item.field || ''),
      bandLabel: index === 0 ? sheetBandLabel(area.type) : '',
      tone: area.type,
      align: item.align || defaultFieldAlignByFormat(item.format),
      anchor: index === 0,
      canMoveUp: canMoveItem(area, item, 'up'),
      canMoveDown: canMoveItem(area, item, 'down'),
      canRemove: true,
    }
  })
}

function summaryBandCells(area: ReportLayoutArea): DesignerSheetCell[] {
  const items = visibleAreaItems(area).filter((item) => item.type === 'summary' && item.field)
  if (!items.length) {
    return [{
      id: `${area.id}_empty`,
      areaId: area.id,
      col: 1,
      colspan: sheetColumns.value.length,
      text: '汇总行（可从右侧添加 sum / count）',
      subtext: areaTypeLabel(area.type),
      bandLabel: sheetBandLabel(area.type),
      tone: area.type,
      empty: true,
      anchor: true,
    }]
  }
  const cells: DesignerSheetCell[] = [{
    id: `${area.id}_label`,
    areaId: area.id,
    col: 1,
    text: '合计',
    subtext: areaTypeLabel(area.type),
    bandLabel: sheetBandLabel(area.type),
    tone: area.type,
    align: 'right',
    anchor: true,
  }]
  const maxItemCells = Math.max(1, sheetColumns.value.length - 1)
  items.slice(0, maxItemCells).forEach((item, index) => {
    cells.push({
      id: `${area.id}_${item.id}`,
      areaId: area.id,
      itemId: item.id,
      col: index + 2,
      text: `${item.aggregate || 'sum'}(${item.field})`,
      subtext: item.label || '',
      tone: area.type,
      align: item.align || 'right',
      canMoveUp: canMoveItem(area, item, 'up'),
      canMoveDown: canMoveItem(area, item, 'down'),
      canRemove: true,
    })
  })
  while (cells.reduce((sum, cell) => sum + (cell.colspan || 1), 0) < sheetColumns.value.length) {
    cells.push(emptyDesignerCell(area, cells.length + 1, '', false))
  }
  return cells
}

function emptyDesignerCell(area: ReportLayoutArea, col: number, label = '', anchor = false): DesignerSheetCell {
  return {
    id: `${area.id}_empty_${col}`,
    areaId: area.id,
    col,
    text: label,
    bandLabel: anchor ? sheetBandLabel(area.type) : '',
    tone: 'empty',
    empty: true,
    anchor,
  }
}

function fullBandText(area: ReportLayoutArea, item?: ReportLayoutAreaItem) {
  if (area.type === 'title') {
    return String(item?.value || item?.label || report.value?.report_name || report.value?.name || '未命名报表')
  }
  if (area.type === 'parameter_summary') {
    const labels = visibleAreaItems(area).map((current) => current.label || current.field || current.parameter_id).filter(Boolean)
    return labels.length ? `查询条件：${labels.join(' / ')}` : '查询条件：全部'
  }
  if (area.type === 'footer') {
    return String(item?.value || item?.label || '制表时间：运行时生成 ｜ 制表人：当前用户')
  }
  if (area.type === 'group') {
    return String(item?.value || item?.label || '分组区（后续阶段接入）')
  }
  return String(item?.value || item?.label || area.title)
}

function sheetBandLabel(type: ReportLayoutAreaType) {
  const map: Record<ReportLayoutAreaType, string> = {
    title: '标题行',
    parameter_summary: '参数行',
    header: '表头行',
    detail: '明细数据带',
    group: '分组行',
    summary: '汇总行',
    footer: '页脚行',
  }
  return map[type]
}

function defaultFieldAlignByFormat(format?: ReportLayoutAreaItemFormat): ReportLayoutAreaItemAlign {
  return format === 'number' || format === 'amount' ? 'right' : 'left'
}

function patchLayoutAreas(updater: (areas: ReportLayoutArea[]) => ReportLayoutArea[]) {
  layoutAreas.value = normalizeLayoutAreas(updater(layoutAreas.value))
  dirty.value = true
}

function patchArea(areaId: string, updater: (area: ReportLayoutArea) => ReportLayoutArea) {
  patchLayoutAreas((areas) => areas.map((area) => area.id === areaId ? updater({ ...area, items: [...area.items] }) : area))
}

function patchItem(areaId: string, itemId: string, patch: Partial<ReportLayoutAreaItem>) {
  patchArea(areaId, (area) => ({
    ...area,
    items: normalizeItemOrders(area.items.map((item) => item.id === itemId ? { ...item, ...patch } : item)),
  }))
}

function syncDetailFromHeader() {
  const datasetId = primaryDataset.value?.id || 'main'
  patchLayoutAreas((areas) => {
    const headerArea = areas.find((area) => area.type === 'header')
    const detailArea = areas.find((area) => area.type === 'detail')
    if (!headerArea || !detailArea) return areas

    const existingDetails = new Map(detailArea.items.map((item) => [item.field, item]))
    const detailItems = orderedItems(headerArea.items)
      .filter((item) => item.field)
      .map((headerItem, index) => {
        const existing = existingDetails.get(headerItem.field)
        return {
          ...(existing || {}),
          id: existing?.id || `detail_${sanitizeId(headerItem.field || `field_${index + 1}`)}`,
          type: 'field',
          dataset_id: headerItem.dataset_id || datasetId,
          field: headerItem.field,
          label: headerItem.label,
          width: headerItem.width,
          align: headerItem.align,
          format: headerItem.format,
          visible: headerItem.visible !== false,
          order: index + 1,
        } as ReportLayoutAreaItem
      })

    return areas.map((area) => area.id === detailArea.id ? { ...area, items: detailItems } : area)
  })
}

function addFieldToReport(field: ReportField, dataset = selectedResourceDataset.value) {
  if (!dataset || isFieldJoined(field, dataset)) return
  const datasetId = dataset.id || primaryDataset.value?.id || 'main'
  const order = headerItems.value.length + 1
  patchArea('header', (area) => ({
    ...area,
    visible: true,
    items: [...area.items, makeHeaderItem(field, datasetId, order)],
  }))
  syncDetailFromHeader()
  selectItem('header', `header_${sanitizeId(field.code)}`)
}

function bindFieldToSelection(field: ReportField, dataset = selectedResourceDataset.value) {
  if (!dataset) return
  const datasetId = dataset.id || primaryDataset.value?.id || 'main'
  const patch: Partial<ReportLayoutAreaItem> = {
    dataset_id: datasetId,
    field: field.code,
    label: field.name || field.code,
    width: defaultFieldWidth(field),
    align: defaultFieldAlign(field),
    format: formatForField(field),
    visible: true,
  }
  const area = selectedItemArea.value || selectedArea.value
  if (area?.type === 'summary') {
    patch.type = 'summary'
    patch.aggregate = 'sum'
  } else {
    patch.type = 'field'
  }

  if (selectedItem.value) {
    updateSelectedItem(patch)
    return
  }
  addFieldToReport(field, dataset)
}

function removeFieldFromReport(fieldCode?: string) {
  if (!fieldCode) return
  patchLayoutAreas((areas) => areas.map((area) => {
    if (area.type !== 'header' && area.type !== 'detail' && area.type !== 'summary') return area
    return {
      ...area,
      items: normalizeItemOrders(area.items.filter((item) => item.field !== fieldCode)),
    }
  }))
  selectedItemRef.value = null
}

function updateFieldPair(fieldCode: string, patch: Partial<ReportLayoutAreaItem>) {
  patchLayoutAreas((areas) => areas.map((area) => {
    if (area.type !== 'header' && area.type !== 'detail') return area
    return {
      ...area,
      items: normalizeItemOrders(area.items.map((item) => item.field === fieldCode ? { ...item, ...patch } : item)),
    }
  }))
}

function updateFieldOrder(fieldCode: string, order: number) {
  const header = areaItems('header')
  const currentIndex = header.findIndex((item) => item.field === fieldCode)
  if (currentIndex < 0) return
  const targetIndex = Math.max(0, Math.min(header.length - 1, order - 1))
  const nextItems = [...header]
  const [target] = nextItems.splice(currentIndex, 1)
  if (!target) return
  nextItems.splice(targetIndex, 0, target)
  patchArea('header', (area) => ({ ...area, items: normalizeItemOrders(nextItems) }))
  syncDetailFromHeader()
}

function addSummaryItem(field: ReportField) {
  const dataset = selectedResourceDataset.value || primaryDataset.value
  if (!dataset || isSummaryFieldJoined(field, dataset)) return
  const datasetId = dataset.id || 'main'
  const item = makeSummaryItem(field, datasetId, summaryItems.value.length + 1)
  patchArea('summary', (area) => ({
    ...area,
    visible: true,
    items: normalizeItemOrders([...area.items, item]),
  }))
  selectItem('summary', item.id)
}

function updateSelectedArea(patch: Partial<ReportLayoutArea>) {
  const area = selectedArea.value
  if (!area) return
  patchArea(area.id, (current) => ({ ...current, ...patch }))
}

function updateSelectedItem(patch: Partial<ReportLayoutAreaItem>) {
  const area = selectedItemArea.value
  const item = selectedItem.value
  if (!area || !item) return
  if ((area.type === 'header' || area.type === 'detail') && item.field) {
    updateFieldPair(item.field, patch)
    return
  }
  patchItem(area.id, item.id, patch)
}

function moveItem(area: ReportLayoutArea, item: ReportLayoutAreaItem, direction: MoveDirection) {
  if ((area.type === 'header' || area.type === 'detail') && item.field) {
    moveFieldItem(item.field, direction)
    return
  }
  moveAreaItem(area.id, item.id, direction)
}

function moveFieldItem(fieldCode: string, direction: MoveDirection) {
  const header = areaItems('header')
  const currentIndex = header.findIndex((item) => item.field === fieldCode)
  if (currentIndex < 0) return
  const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
  if (targetIndex < 0 || targetIndex >= header.length) return
  const nextItems = [...header]
  const [target] = nextItems.splice(currentIndex, 1)
  if (!target) return
  nextItems.splice(targetIndex, 0, target)
  patchArea('header', (area) => ({ ...area, items: normalizeItemOrders(nextItems) }))
  syncDetailFromHeader()
}

function moveAreaItem(areaId: string, itemId: string, direction: MoveDirection) {
  patchArea(areaId, (area) => {
    const items = orderedItems(area.items)
    const currentIndex = items.findIndex((item) => item.id === itemId)
    const targetIndex = direction === 'up' ? currentIndex - 1 : currentIndex + 1
    if (currentIndex < 0 || targetIndex < 0 || targetIndex >= items.length) return area
    const nextItems = [...items]
    const [target] = nextItems.splice(currentIndex, 1)
    if (!target) return area
    nextItems.splice(targetIndex, 0, target)
    return { ...area, items: normalizeItemOrders(nextItems) }
  })
}

function removeItem(area: ReportLayoutArea, item: ReportLayoutAreaItem) {
  if ((area.type === 'header' || area.type === 'detail') && item.field) {
    removeFieldFromReport(item.field)
    return
  }
  patchArea(area.id, (current) => ({
    ...current,
    items: normalizeItemOrders(current.items.filter((currentItem) => currentItem.id !== item.id)),
  }))
  selectedItemRef.value = null
}

function canMoveItem(area: ReportLayoutArea, item: ReportLayoutAreaItem, direction: MoveDirection) {
  const items = area.type === 'header' || area.type === 'detail'
    ? headerItems.value
    : orderedItems(area.items)
  const index = items.findIndex((current) => {
    if ((area.type === 'header' || area.type === 'detail') && item.field) {
      return current.field === item.field
    }
    return current.id === item.id
  })
  return direction === 'up' ? index > 0 : index >= 0 && index < items.length - 1
}

function fieldJoinKey(datasetId?: string, fieldCode?: string) {
  if (!fieldCode) return ''
  return `${datasetId || primaryDataset.value?.id || 'main'}:${fieldCode}`
}

function isFieldJoined(field: ReportField, dataset = selectedResourceDataset.value) {
  return joinedFieldKeys.value.has(fieldJoinKey(dataset?.id, field.code))
}

function isSummaryFieldJoined(field: ReportField, dataset = selectedResourceDataset.value) {
  return summaryFieldKeys.value.has(fieldJoinKey(dataset?.id, field.code))
}

function isFieldOperationItem(area: ReportLayoutArea, item: ReportLayoutAreaItem) {
  return (area.type === 'header' || area.type === 'detail' || area.type === 'summary') && Boolean(item.field)
}

function setSelectedAreaTitle(value: string | number | null) {
  updateSelectedArea({ title: String(value || '') })
}

function setSelectedAreaVisible(value: boolean) {
  updateSelectedArea({ visible: Boolean(value) })
}

function setSelectedItemLabel(value: string | number | null) {
  updateSelectedItem({ label: String(value || '') })
}

function setSelectedItemWidth(value: string | number | null) {
  updateSelectedItem({ width: Math.max(40, Number(value) || 120) })
}

function setSelectedItemOrder(value: string | number | null) {
  const nextOrder = Math.max(1, Number(value) || 1)
  const item = selectedItem.value
  const area = selectedItemArea.value
  if (!item || !area) return
  if ((area.type === 'header' || area.type === 'detail') && item.field) {
    updateFieldOrder(item.field, nextOrder)
    return
  }
  patchItem(area.id, item.id, { order: nextOrder })
}

function setSelectedItemAlign(value: ReportLayoutAreaItemAlign) {
  updateSelectedItem({ align: value })
}

function setSelectedItemFormat(value: ReportLayoutAreaItemFormat) {
  updateSelectedItem({ format: value })
}

function setSelectedItemAggregate(value: Exclude<ReportLayoutAreaAggregate, 'none'>) {
  updateSelectedItem({ aggregate: value })
}

function setSelectedItemVisible(value: boolean) {
  updateSelectedItem({ visible: Boolean(value) })
}

function selectArea(id: string) {
  selectAreaOnly(id)
  if (typeof document === 'undefined') return
  document.getElementById(areaDomId(id))?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function selectAreaOnly(id: string) {
  selectedAreaId.value = id
  selectedItemRef.value = null
}

function selectItem(areaId: string, itemId: string) {
  selectedAreaId.value = areaId
  selectedItemRef.value = { areaId, itemId }
}

function selectSheetCell(cell: DesignerSheetCell) {
  if (cell.itemId) {
    selectItem(cell.areaId, cell.itemId)
    inspectorTab.value = 'cell'
    return
  }
  selectAreaOnly(cell.areaId)
  inspectorTab.value = 'cell'
}

function isSelectedItem(areaId: string, itemId: string) {
  return selectedItemRef.value?.areaId === areaId && selectedItemRef.value.itemId === itemId
}

function isSheetCellSelected(cell: DesignerSheetCell) {
  if (cell.itemId) return isSelectedItem(cell.areaId, cell.itemId)
  return selectedAreaId.value === cell.areaId && !selectedItemRef.value
}

function moveSheetCell(cell: DesignerSheetCell, direction: MoveDirection) {
  if (!cell.itemId) return
  const area = layoutAreas.value.find((current) => current.id === cell.areaId)
  const item = area?.items.find((current) => current.id === cell.itemId)
  if (!area || !item) return
  moveItem(area, item, direction)
}

function removeSheetCell(cell: DesignerSheetCell) {
  if (!cell.itemId) return
  const area = layoutAreas.value.find((current) => current.id === cell.areaId)
  const item = area?.items.find((current) => current.id === cell.itemId)
  if (!area || !item) return
  removeItem(area, item)
}

function sheetCellStyle(cell: DesignerSheetCell) {
  return {
    gridColumn: `span ${cell.colspan || 1}`,
    textAlign: cell.align || 'left',
  }
}

function selectedItemColumnIndex(area: ReportLayoutArea, item: ReportLayoutAreaItem) {
  if (area.type === 'summary') {
    const index = visibleAreaItems(area).filter((current) => current.type === 'summary' && current.field)
      .findIndex((current) => current.id === item.id)
    return Math.max(0, index + 1)
  }
  if (area.type === 'header' || area.type === 'detail') {
    const index = visibleAreaItems(area).filter((current) => current.field)
      .findIndex((current) => current.id === item.id || current.field === item.field)
    return Math.max(0, index)
  }
  return 0
}

function validateCurrentDesign() {
  const sourceReport = report.value
  if (!sourceReport) {
    $q.notify({ type: 'warning', message: '缺少报表信息，无法校验' })
    return
  }
  const errors = validateDesignPreview(layoutAreasToSheet(layoutAreas.value, sourceReport))
  if (errors.length) {
    $q.notify({ type: 'warning', message: errors[0] || '当前配置存在问题' })
    inspectorTab.value = 'report'
    return
  }
  $q.notify({ type: 'positive', message: '当前版式配置校验通过' })
}

function clearCurrentSelection() {
  selectedItemRef.value = null
}

function toggleSelectedBold() {
  notifyDesignerPlaceholder('加粗样式')
}

function notifyDesignerPlaceholder(label: string) {
  $q.notify({ type: 'info', message: `${label}将在后续阶段接入` })
}

function isNumericField(field: ReportField) {
  const raw = `${field.type || ''} ${field.code || ''}`.toLowerCase()
  return field.role === 'metric' || /amount|money|price|total|sum|number|numeric|decimal|float|double|int/.test(raw)
}

function formatForField(field: ReportField): ReportLayoutAreaItemFormat {
  const raw = `${field.type || ''} ${field.code || ''}`.toLowerCase()
  if (/datetime|timestamp/.test(raw)) return 'datetime'
  if (/date|time/.test(raw) || field.role === 'time') return 'date'
  if (/amount|money|price|total|fee|cost/.test(raw)) return 'amount'
  if (isNumericField(field)) return 'number'
  if (/status|type|dict/.test(raw)) return 'dict'
  return 'text'
}

function defaultFieldAlign(field: ReportField): ReportLayoutAreaItemAlign {
  return isNumericField(field) ? 'right' : 'left'
}

function defaultFieldWidth(field: ReportField) {
  const raw = `${field.type || ''} ${field.code || ''}`.toLowerCase()
  if (/datetime|timestamp|created|updated/.test(raw)) return 160
  if (/amount|money|price|total|fee|cost/.test(raw)) return 130
  return 120
}

function fieldIcon(field: ReportField) {
  if (field.role === 'metric') return 'pin'
  if (field.role === 'time') return 'event'
  if (field.role === 'dimension') return 'category'
  return 'text_fields'
}

function areaIcon(type: ReportLayoutAreaType) {
  const map: Record<ReportLayoutAreaType, string> = {
    title: 'title',
    parameter_summary: 'filter_alt',
    header: 'view_column',
    detail: 'table_rows',
    group: 'account_tree',
    summary: 'functions',
    footer: 'notes',
  }
  return map[type]
}

function areaTypeLabel(type: ReportLayoutAreaType) {
  const map: Record<ReportLayoutAreaType, string> = {
    title: '报表标题、静态说明',
    parameter_summary: '运行参数摘要',
    header: '字段列标题',
    detail: '结果字段绑定',
    group: '分组展示，后续阶段接入',
    summary: '简单汇总',
    footer: '制表人、制表时间、备注',
  }
  return map[type]
}

function reportStatusMeta(status: ReportStatus) {
  if (status === 'published') return { label: '已发布', color: 'positive' }
  if (status === 'disabled') return { label: '已停用', color: 'negative' }
  return { label: '草稿', color: 'grey-7' }
}

function areaDomId(id: string) {
  return `layout-area-${id}`
}

function columnLetter(index: number) {
  let value = index + 1
  let label = ''
  while (value > 0) {
    const remainder = (value - 1) % 26
    label = String.fromCharCode(65 + remainder) + label
    value = Math.floor((value - 1) / 26)
  }
  return label
}

function goWorkbench() {
  router.push({ name: 'report_v2_workbench' })
}

function toPositiveNumber(value: unknown) {
  const raw = Array.isArray(value) ? value[0] : value
  const numeric = Number(raw)
  return Number.isFinite(numeric) && numeric > 0 ? numeric : 0
}

function sanitizeId(value: string) {
  return value.replace(/[^a-zA-Z0-9_-]/g, '_') || 'field'
}
</script>

<style scoped lang="scss">
.report-v2-designer-page {
  position: fixed;
  inset: 0;
  z-index: 2500;
  width: 100vw;
  height: 100vh;
  min-height: 0;
  background: #f6f7fb;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.state-card {
  flex: 1;
  margin: 20px;
  border-radius: 8px;
  padding: 24px;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
}

.designer-toolbar {
  flex: 0 0 56px;
  min-height: 56px;
  padding: 8px 14px;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.toolbar-left,
.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: nowrap;
}

.toolbar-left {
  min-width: 0;
  flex: 1 1 auto;
  overflow: hidden;
}

.toolbar-title {
  min-width: 180px;
  max-width: 360px;
  overflow: hidden;

  .text-subtitle1,
  .text-caption {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.toolbar-actions {
  flex: 0 0 auto;
  justify-content: flex-end;
}

.designer-shell {
  flex: 1;
  min-height: 0;
  padding: 8px 10px;
  display: grid;
  grid-template-columns: 282px minmax(0, 1fr) 304px;
  gap: 8px;
  overflow: hidden;
}

.designer-panel,
.designer-canvas,
.designer-inspector {
  min-height: 0;
  max-height: 100%;
}

.designer-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 2px;
}

.panel-card,
.designer-canvas {
  border-radius: 8px;
}

.panel-title {
  margin-bottom: 8px;
  color: #172033;
  font-weight: 700;
}

.resource-fields-card :deep(.q-list) {
  max-height: 280px;
  overflow-y: auto;
}

.resource-row {
  min-height: 52px;
  padding: 6px 8px;
  border-radius: 6px;

  :deep(.q-item__section--avatar) {
    min-width: 30px;
    padding-right: 8px;
  }

  :deep(.q-item__section--side) {
    padding-left: 6px;
  }
}

.dataset-resource-row {
  margin-bottom: 2px;
}

.resource-avatar {
  color: var(--q-primary);
}

.resource-title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: #1f2937;
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;

  span:first-child {
    min-width: 0;
  }
}

.resource-caption {
  color: #667085;
  font-size: 12px;
  line-height: 16px;
}

.dataset-primary-badge {
  min-height: 18px;
  padding: 2px 5px;
  border-radius: 4px;
  font-size: 11px;
  line-height: 1;
}

.resource-side {
  align-items: flex-end;
}

.component-palette {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.field-actions {
  display: flex;
  align-items: center;
  gap: 1px;

  :deep(.q-btn) {
    min-width: 26px;
    min-height: 26px;
  }
}

.active-area-nav {
  color: var(--q-primary);
  background: #eef4ff;
}

.designer-canvas {
  padding: 0;
  background: #fff;
  border: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.canvas-header {
  flex: 0 0 auto;
  padding: 10px 12px;
  border-bottom: 1px solid #edf1f7;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.canvas-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.canvas-formatbar {
  flex: 0 0 38px;
  padding: 4px 8px;
  border-bottom: 1px solid #edf1f7;
  background: #fbfcff;
  display: flex;
  align-items: center;
  gap: 4px;
}

.zoom-select {
  width: 92px;
}

.sheet-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  border: 1px solid #d8e1ef;
  border-radius: 8px;
  background:
    linear-gradient(90deg, rgba(248, 250, 252, 0.96) 0, rgba(248, 250, 252, 0.96) 46px, transparent 46px),
    #fff;
}

.sheet-grid {
  display: grid;
  grid-auto-rows: minmax(54px, auto);
  min-width: max-content;
  padding: 12px;
  transition: transform 0.12s ease;
}

.sheet-corner,
.sheet-column-head,
.sheet-row-head,
.sheet-cell {
  border-right: 1px solid #d8e1ef;
  border-bottom: 1px solid #d8e1ef;
}

.sheet-corner,
.sheet-column-head,
.sheet-row-head {
  background: #f3f6fb;
  color: #667085;
  font-size: 12px;
  font-weight: 700;
}

.sheet-corner {
  position: sticky;
  top: 0;
  left: 0;
  z-index: 4;
  border-top: 1px solid #d8e1ef;
  border-left: 1px solid #d8e1ef;
}

.sheet-column-head {
  position: sticky;
  top: 0;
  z-index: 3;
  height: 34px;
  border-top: 1px solid #d8e1ef;
  display: grid;
  place-items: center;
}

.sheet-row-head {
  position: sticky;
  left: 0;
  z-index: 2;
  border-left: 1px solid #d8e1ef;
  display: grid;
  place-items: center;
  cursor: pointer;
}

.sheet-row-head.is-active-row {
  color: var(--q-primary);
  background: #eaf1ff;
}

.sheet-cell {
  position: relative;
  min-height: 54px;
  padding: 7px 8px;
  background: #fff;
  color: #172033;
  appearance: none;
  border-top: 0;
  border-left: 0;
  cursor: pointer;
  display: grid;
  align-content: center;
  gap: 2px;
  transition: background 0.15s ease, box-shadow 0.15s ease;
}

.sheet-cell:hover {
  background: #f8fbff;
}

.sheet-cell.is-active-cell {
  z-index: 1;
  box-shadow: inset 0 0 0 2px var(--q-primary);
}

.sheet-cell.is-area-cell.is-active-cell {
  background: #edf4ff;
}

.sheet-cell-empty {
  background: #fbfcff;
  color: #b0b8c4;
}

.sheet-cell-title {
  background: #f8fbff;
  font-weight: 800;
  font-size: 16px;
}

.sheet-cell-parameter_summary,
.sheet-cell-footer {
  background: #f9fafb;
  color: #475467;
}

.sheet-cell-header {
  background: #eef4ff;
  font-weight: 700;
}

.sheet-cell-detail {
  background: #fff;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.sheet-cell-summary {
  background: #fff7ed;
  font-weight: 700;
}

.sheet-cell-group {
  background: #f0fdf4;
}

.cell-band-label {
  color: var(--q-primary);
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
}

.cell-text,
.cell-subtext {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cell-subtext {
  color: #667085;
  font-size: 11px;
}

.cell-actions {
  position: absolute;
  right: 4px;
  top: 3px;
  display: none;
  align-items: center;
  gap: 1px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.9);
}

.sheet-cell:hover .cell-actions,
.sheet-cell.is-active-cell .cell-actions {
  display: flex;
}

.property-form {
  display: grid;
  gap: 12px;
}

.designer-inspector {
  overflow: hidden;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  display: flex;
  flex-direction: column;
}

.inspector-tabs {
  flex: 0 0 auto;
  border-bottom: 1px solid #edf1f7;
}

.inspector-panels {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.inspector-panel {
  height: 100%;
  overflow-y: auto;
  padding: 12px;
}

.binding-list {
  max-height: 260px;
  overflow-y: auto;
}

.summary-field-picker {
  border-top: 1px solid #edf1f7;
  padding-top: 12px;
}

.summary-field-list {
  display: grid;
  gap: 8px;
}

.check-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: #c2410c;
}

.check-item.ok {
  color: #16803c;
}

.check-item.optional:not(.ok) {
  color: #667085;
}

.designer-statusbar {
  flex: 0 0 28px;
  padding: 0 12px;
  border-top: 1px solid #d8e1ef;
  background: #fff;
  color: #667085;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 18px;
  white-space: nowrap;
  overflow: hidden;
}

.preview-dialog {
  width: min(1180px, 92vw);
  max-width: 92vw;
}

.dialog-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.dialog-title {
  font-size: 16px;
  font-weight: 700;
}

.dialog-caption {
  margin-top: 2px;
  color: #667085;
  font-size: 12px;
}

.preview-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
</style>
