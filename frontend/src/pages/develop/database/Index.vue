<template>
  <base-content class="q-pa-sm">
    <q-table
      class="fit sticky-header-table"
      color="primary"
      selection="multiple"
      v-model:selected="selected"
      :dense="$q.screen.lt.md"
      separator="cell"
      flat
      bordered
      :rows="rows"
      :columns="columns"
      :visible-columns="visibleColumns"
      row-key="id"
      v-model:pagination="pagination"
      :loading="loading"
    >
      <template v-slot:top>
        <div class="row q-gutter-xs full-width">
          <div class="col-grow row q-gutter-xs">
            <q-input
              dense
              outlined
              debounce="300"
              v-model="query.quick_query!.keyword"
              placeholder="搜索关键词"
            >
              <template v-slot:append>
                <q-icon name="search" />
              </template>
            </q-input>
            <q-btn color="primary" label="搜索" :disable="loading" @click="handleBasicSearch" />
            <q-select
              v-model="visibleColumns"
              multiple
              outlined
              dense
              options-dense
              :display-value="compactSelectionDisplay(visibleColumns, columns, 2, '列')"
              emit-value
              map-options
              :options="columns"
              option-value="name"
              options-cover
            ></q-select>
            <q-btn
              outline
              icon="tune"
              color="primary"
              class="q-ml-xs"
              :aria-label="
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
              "
              @click="showAdvancedQuery = true"
            >
              <q-badge v-if="activeFilterCount > 0" floating color="red">{{
                activeFilterCount
              }}</q-badge>
              <q-tooltip>{{
                hasAppliedAdvancedFilters
                  ? `高级查询，已启用 ${activeFilterCount} 个条件`
                  : '高级查询'
              }}</q-tooltip>
            </q-btn>
          </div>

          <q-space />

          <div class="row q-gutter-xs">
            <q-btn
              v-for="btn in top_buttons"
              :key="btn.id"
              v-bind="menuButtonDisplayProps(btn)"
              :color="btn.color || 'primary'"
              :loading="loading"
              :disable="loading"
              @click="handleButtonClick(btn)"
            />
          </div>
        </div>
      </template>

      <template v-slot:body-cell-actions="props">
        <q-td :props="props" class="q-gutter-xs">
          <q-btn
            v-for="btn in line_buttons"
            :key="btn.id"
            flat
            v-bind="menuButtonDisplayProps(btn)"
            :color="btn.color || 'primary'"
            size="sm"
            :disable="loading || isButtonDisabled(btn, props.row)"
            @click.stop="handleButtonClick(btn, props.row)"
          >
            <q-tooltip>{{ btn.name }}</q-tooltip>
          </q-btn>
        </q-td>
      </template>

      <template v-slot:body-cell-publish_status="props">
        <q-td :props="props">
          <q-chip
            dense
            square
            :color="publishStatusColor(props.row)"
            text-color="white"
          >
            {{ props.row.publish_status }}
          </q-chip>
        </q-td>
      </template>

      <template v-slot:bottom>
        <q-space />
        <table-pagination v-model:page="query.page" v-model:pageSize="query.num" :total="total" />
      </template>
    </q-table>

    <advanced-query
      v-model="showAdvancedQuery"
      v-model:queryModel="tempAdvancedQuery"
      :fields="table_fields_advanced"
      @search="handleAdvancedSearch"
    />

    <dynamic-form-dialog
      v-model="showFormDialog"
      :edit-data="currentEditData"
      :title="currentEditData?.id ? '编辑数据表' : '新增数据表'"
      :fields="tableFields"
      :submit-btn-text="currentEditData?.id ? '保存' : '创建'"
      @submit="handleFormSubmit"
    />

    <q-dialog v-model="showFieldDialog" maximized>
      <q-card class="table-structure-workbench">
        <q-card-section class="structure-header">
          <div class="structure-title-area">
            <div class="structure-avatar">表</div>
            <div class="structure-title-copy">
              <div class="structure-title">表结构管理</div>
              <div class="structure-subtitle">
                <span>当前表：{{ currentTable?.table_name || '-' }}</span>
                <span class="structure-code">{{ currentTable?.table_code || '-' }}</span>
                <span>{{ tableKindLabel }}</span>
              </div>
            </div>
          </div>

          <div class="structure-actions">
            <template v-if="activeMetaTab === 'fields'">
              <q-btn
                outline
                color="primary"
                icon="sync"
                label="同步字段"
                :loading="loading"
                @click="confirmSyncCurrentTableFields"
              />
              <q-btn color="primary" icon="add" label="新增字段" @click="openAddFieldDialog" />
            </template>
            <template v-else-if="activeMetaTab === 'indexes'">
              <q-btn
                outline
                color="primary"
                icon="sync"
                label="同步索引"
                :loading="loading"
                @click="confirmSyncTableIndexes"
              />
              <q-btn color="primary" icon="add" label="新增索引" @click="openAddIndexDialog" />
            </template>
            <template v-else>
              <q-btn color="primary" icon="add" label="新增关联" @click="openAddRelationDialog" />
            </template>
            <q-btn flat round icon="close" v-close-popup>
              <q-tooltip>关闭</q-tooltip>
            </q-btn>
          </div>
        </q-card-section>

        <q-card-section class="structure-body">
          <detail-section-navigation
            :model-value="activeMetaTab"
            :items="structureNavigationItems"
            @update:model-value="setMetaTabValue"
          >
            <template #footer>
              <div class="structure-nav-meta">
              <q-chip dense square color="deep-purple-1" text-color="primary">
                {{ currentTable?.table_code || '-' }}
              </q-chip>
              <q-chip dense square color="grey-2" text-color="grey-8">
                {{ tableKindLabel }}
              </q-chip>
              </div>
            </template>
          </detail-section-navigation>

          <section class="structure-list-panel">
            <div class="structure-list-toolbar">
              <q-input
                v-model="structureKeyword"
                dense
                outlined
                debounce="200"
                class="structure-search"
                :placeholder="structureSearchPlaceholder"
              >
                <template v-slot:append>
                  <q-icon name="search" />
                </template>
              </q-input>
              <q-btn
                flat
                round
                color="primary"
                icon="refresh"
                :loading="loading"
                @click="refreshCurrentStructure"
              >
                <q-tooltip>刷新</q-tooltip>
              </q-btn>
            </div>

            <q-table
              v-if="activeMetaTab === 'fields'"
              flat
              class="structure-table"
              :rows="filteredFieldRows"
              :columns="structureFieldColumns"
              row-key="id"
              :pagination="{ rowsPerPage: 0 }"
              hide-bottom
              :loading="loading"
            >
              <template v-slot:body="props">
                <q-tr
                  :props="props"
                  class="structure-row"
                  :class="{ 'is-selected': selectedField?.id === props.row.id }"
                  @click="selectField(props.row)"
                >
                  <q-td key="sequence" :props="props">{{ props.row.sequence }}</q-td>
                  <q-td key="field" :props="props">
                    <div class="structure-primary-text">{{ props.row.field_name }}</div>
                    <div class="structure-secondary-text">{{ props.row.field_code }}</div>
                  </q-td>
                  <q-td key="type" :props="props">{{ fieldTypeLabel(props.row) }}</q-td>
                  <q-td key="input" :props="props">{{ inputTypeLabel(props.row) }}</q-td>
                  <q-td key="display" :props="props">
                    <div class="structure-tag-row">
                      <span
                        v-for="tag in fieldDisplayTags(props.row)"
                        :key="tag.label"
                        class="structure-tag"
                        :class="`is-${tag.tone}`"
                      >
                        {{ tag.label }}
                      </span>
                    </div>
                  </q-td>
                  <q-td key="search" :props="props">
                    <div class="structure-tag-row">
                      <span
                        v-for="tag in fieldSearchTags(props.row)"
                        :key="tag.label"
                        class="structure-tag"
                        :class="`is-${tag.tone}`"
                      >
                        {{ tag.label }}
                      </span>
                    </div>
                  </q-td>
                  <q-td key="constraints" :props="props">
                    <div class="structure-tag-row">
                      <span
                        v-for="tag in fieldConstraintTags(props.row)"
                        :key="tag.label"
                        class="structure-tag"
                        :class="`is-${tag.tone}`"
                      >
                        {{ tag.label }}
                      </span>
                    </div>
                  </q-td>
                  <q-td key="actions" :props="props" class="structure-row-actions">
                    <q-btn
                      flat
                      round
                      color="primary"
                      icon="arrow_upward"
                      size="sm"
                      @click.stop="moveField(props.row, -1)"
                    >
                      <q-tooltip>上移</q-tooltip>
                    </q-btn>
                    <q-btn
                      flat
                      round
                      color="primary"
                      icon="arrow_downward"
                      size="sm"
                      @click.stop="moveField(props.row, 1)"
                    >
                      <q-tooltip>下移</q-tooltip>
                    </q-btn>
                  </q-td>
                </q-tr>
              </template>
            </q-table>

            <q-table
              v-else-if="activeMetaTab === 'indexes'"
              flat
              class="structure-table"
              :rows="filteredIndexRows"
              :columns="structureIndexColumns"
              row-key="id"
              :pagination="{ rowsPerPage: 0 }"
              hide-bottom
              :loading="loading"
            >
              <template v-slot:body="props">
                <q-tr
                  :props="props"
                  class="structure-row"
                  :class="{ 'is-selected': selectedIndex?.id === props.row.id }"
                  @click="selectIndex(props.row)"
                >
                  <q-td key="name" :props="props">
                    <div class="structure-primary-text">{{ props.row.index_name }}</div>
                    <div class="structure-secondary-text">ID {{ props.row.id }}</div>
                  </q-td>
                  <q-td key="type" :props="props">
                    <span
                      class="structure-tag"
                      :class="props.row.is_unique ? 'is-warn' : 'is-blue'"
                    >
                      {{ props.row.is_unique ? '唯一索引' : '普通索引' }}
                    </span>
                  </q-td>
                  <q-td key="fields" :props="props">
                    <div class="structure-tag-row">
                      <span
                        v-for="field in props.row.index_fields || []"
                        :key="field.id"
                        class="structure-tag is-soft"
                      >
                        {{ field.field_name || field.field_code }}
                      </span>
                    </div>
                  </q-td>
                </q-tr>
              </template>
            </q-table>

            <q-table
              v-else
              flat
              class="structure-table"
              :rows="filteredRelationRows"
              :columns="structureRelationColumns"
              row-key="id"
              :pagination="{ rowsPerPage: 0 }"
              hide-bottom
              :loading="loading"
            >
              <template v-slot:body="props">
                <q-tr
                  :props="props"
                  class="structure-row"
                  :class="{ 'is-selected': selectedRelation?.id === props.row.id }"
                  @click="selectRelation(props.row)"
                >
                  <q-td key="relation" :props="props">
                    <div class="structure-primary-text">{{ relationTypeLabel(props.row) }}</div>
                    <div class="structure-secondary-text">
                      {{ relatedTableLabel(props.row.related_table_id) }}
                    </div>
                  </q-td>
                  <q-td key="mapping" :props="props">
                    <span class="structure-code">{{ props.row.reference_key }}</span>
                    <q-icon name="arrow_forward" class="q-mx-xs text-grey-6" />
                    <span class="structure-code">{{ props.row.foreign_key }}</span>
                  </q-td>
                  <q-td key="many" :props="props">
                    {{ props.row.many_table_code || '-' }}
                  </q-td>
                </q-tr>
              </template>
            </q-table>
          </section>

          <aside class="structure-detail-panel">
            <template v-if="activeMetaTab === 'fields'">
              <div v-if="selectedField" class="structure-detail">
                <div class="structure-detail-head">
                  <div>
                    <div class="structure-detail-title">{{ selectedField.field_name }}</div>
                    <div class="structure-subtitle">
                      <span class="structure-code">{{ selectedField.field_code }}</span>
                      <span>{{ fieldTypeLabel(selectedField) }}</span>
                    </div>
                  </div>
                  <span
                    class="structure-status"
                    :class="{ 'is-off': selectedField.state === false }"
                  >
                    {{ selectedField.state === false ? '停用' : '启用' }}
                  </span>
                </div>

                <div class="structure-detail-scroll">
                  <div class="structure-section">
                    <div class="structure-section-title">基础信息</div>
                    <div class="structure-info-grid">
                      <div>
                        <span>字段名称</span>
                        <strong>{{ selectedField.field_name || '-' }}</strong>
                      </div>
                      <div>
                        <span>字段编码</span>
                        <strong>{{ selectedField.field_code || '-' }}</strong>
                      </div>
                      <div>
                        <span>字段类型</span>
                        <strong>{{ fieldTypeLabel(selectedField) }}</strong>
                      </div>
                      <div>
                        <span>输入控件</span>
                        <strong>{{ inputTypeLabel(selectedField) }}</strong>
                      </div>
                      <div>
                        <span>默认值</span>
                        <strong>{{ selectedField.default_value || '-' }}</strong>
                      </div>
                      <div>
                        <span>排序</span>
                        <strong>{{ selectedField.sequence ?? '-' }}</strong>
                      </div>
                    </div>
                  </div>

                  <div class="structure-section">
                    <div class="structure-section-title">页面与查询</div>
                    <div class="structure-switch-grid">
                      <span
                        v-for="tag in fieldDisplayTags(selectedField)"
                        :key="`display-${tag.label}`"
                        :class="['structure-switch', `is-${tag.tone}`]"
                      >
                        {{ tag.label }}
                      </span>
                      <span
                        v-for="tag in fieldSearchTags(selectedField)"
                        :key="`search-${tag.label}`"
                        :class="['structure-switch', `is-${tag.tone}`]"
                      >
                        {{ tag.label }}
                      </span>
                    </div>
                  </div>

                  <div class="structure-section">
                    <div class="structure-section-title">约束与高级配置</div>
                    <div class="structure-tag-row q-mb-sm">
                      <span
                        v-for="tag in fieldConstraintTags(selectedField)"
                        :key="tag.label"
                        class="structure-tag"
                        :class="`is-${tag.tone}`"
                      >
                        {{ tag.label }}
                      </span>
                    </div>
                    <div class="structure-info-list">
                      <div>
                        <span>所用字典</span>
                        <strong>{{ selectedField.dict_code || '-' }}</strong>
                      </div>
                      <div>
                        <span>字段类别</span>
                        <strong>{{ fieldCategoryLabel(selectedField) }}</strong>
                      </div>
                      <div>
                        <span>计算表达式</span>
                        <strong>{{ selectedField.expression || '-' }}</strong>
                      </div>
                      <div>
                        <span>联动配置</span>
                        <strong>{{ selectedField.linkage_config || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="structure-detail-actions">
                  <q-btn
                    flat
                    color="negative"
                    label="删除字段"
                    @click="confirmDeleteField(selectedField)"
                  />
                  <q-btn
                    color="primary"
                    label="编辑字段"
                    @click="openEditFieldDialog(selectedField)"
                  />
                </div>
              </div>
              <div v-else class="structure-empty">暂无字段</div>
            </template>

            <template v-else-if="activeMetaTab === 'indexes'">
              <div v-if="selectedIndex" class="structure-detail">
                <div class="structure-detail-head">
                  <div>
                    <div class="structure-detail-title">{{ selectedIndex.index_name }}</div>
                    <div class="structure-subtitle">
                      <span>{{ selectedIndex.is_unique ? '唯一索引' : '普通索引' }}</span>
                    </div>
                  </div>
                  <span
                    class="structure-status"
                    :class="{ 'is-off': selectedIndex.state === false }"
                  >
                    {{ selectedIndex.state === false ? '停用' : '启用' }}
                  </span>
                </div>
                <div class="structure-detail-scroll">
                  <div class="structure-section">
                    <div class="structure-section-title">索引字段</div>
                    <div class="structure-index-fields">
                      <span
                        v-for="(field, index) in selectedIndex.index_fields || []"
                        :key="field.id"
                      >
                        {{ index + 1 }}. {{ field.field_name || field.field_code }}
                      </span>
                    </div>
                  </div>
                </div>
                <div class="structure-detail-actions">
                  <q-btn
                    flat
                    color="negative"
                    label="删除索引"
                    @click="confirmDeleteIndex(selectedIndex)"
                  />
                  <q-btn
                    color="primary"
                    label="编辑索引"
                    @click="openEditIndexDialog(selectedIndex)"
                  />
                </div>
              </div>
              <div v-else class="structure-empty">暂无索引</div>
            </template>

            <template v-else>
              <div v-if="selectedRelation" class="structure-detail">
                <div class="structure-detail-head">
                  <div>
                    <div class="structure-detail-title">
                      {{ relationTypeLabel(selectedRelation) }}
                    </div>
                    <div class="structure-subtitle">
                      <span>{{ relatedTableLabel(selectedRelation.related_table_id) }}</span>
                    </div>
                  </div>
                  <span
                    class="structure-status"
                    :class="{ 'is-off': selectedRelation.state === false }"
                  >
                    {{ selectedRelation.state === false ? '停用' : '启用' }}
                  </span>
                </div>
                <div class="structure-detail-scroll">
                  <div class="structure-section">
                    <div class="structure-section-title">字段映射</div>
                    <div class="structure-info-list">
                      <div>
                        <span>主表字段</span>
                        <strong>{{ selectedRelation.reference_key }}</strong>
                      </div>
                      <div>
                        <span>关联表字段</span>
                        <strong>{{ selectedRelation.foreign_key }}</strong>
                      </div>
                      <div>
                        <span>中间表</span>
                        <strong>{{ selectedRelation.many_table_code || '-' }}</strong>
                      </div>
                    </div>
                  </div>
                </div>
                <div class="structure-detail-actions">
                  <q-btn
                    flat
                    color="negative"
                    label="删除关联"
                    @click="confirmDeleteRelation(selectedRelation)"
                  />
                  <q-btn
                    color="primary"
                    label="编辑关联"
                    @click="openEditRelationDialog(selectedRelation)"
                  />
                </div>
              </div>
              <div v-else class="structure-empty">暂无关联关系</div>
            </template>
          </aside>
        </q-card-section>
      </q-card>
    </q-dialog>

    <dynamic-form-dialog
      v-model="showFieldFormDialog"
      :edit-data="currentEditField"
      :title="currentEditField?.id ? '编辑字段' : '新增字段'"
      :fields="fieldFormFields"
      :table-code="currentTable?.table_code || ''"
      :submit-btn-text="currentEditField?.id ? '保存' : '创建'"
      @submit="handleFieldFormSubmit"
    />

    <form-dialog-shell
      v-model="showIndexFormDialog"
      :title="currentEditIndex?.id ? '编辑索引' : '新增索引'"
      subtitle="配置索引名称、唯一性和参与字段"
      icon="toc"
      submit-text="保存"
      width="min(780px, calc(100vw - 48px))"
      @submit="handleIndexFormSubmit"
    >
      <div class="metadata-simple-form">
        <section class="metadata-simple-section">
          <div class="metadata-simple-section__head">
            <div>
              <div class="metadata-simple-section__title">索引信息</div>
              <div class="metadata-simple-section__desc">索引名称建议使用数据库可读的英文编码</div>
            </div>
          </div>
          <div class="metadata-form-grid">
            <q-input
              v-model="indexForm.index_name"
              label="索引名称"
              outlined
              dense
              maxlength="64"
            />
            <q-select
              v-model="indexForm.is_unique"
              :options="indexUniqueOptions"
              label="索引类型"
              emit-value
              map-options
              outlined
              dense
            />
            <q-select
              v-model="indexForm.field_ids"
              class="metadata-form-wide"
              :options="indexFieldOptions"
              label="索引字段"
              multiple
              emit-value
              map-options
              :display-value="
                compactSelectionDisplay(indexForm.field_ids, indexFieldOptions, 2, '索引字段')
              "
              outlined
              dense
            >
              <q-tooltip v-if="compactSelectionTooltip(indexForm.field_ids, indexFieldOptions)">
                {{ compactSelectionTooltip(indexForm.field_ids, indexFieldOptions) }}
              </q-tooltip>
            </q-select>
          </div>
        </section>
      </div>

      <template #footer-status>
        {{ indexFormStatusText }}
      </template>

      <template #preview>
        <div class="metadata-form-preview">
          <div class="metadata-form-preview__title">索引预览</div>
          <div
            v-for="item in indexPreviewItems"
            :key="item.label"
            class="metadata-form-preview__row"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </template>
    </form-dialog-shell>

    <form-dialog-shell
      v-model="showRelationFormDialog"
      :title="currentEditRelation?.id ? '编辑关联' : '新增关联'"
      subtitle="配置主表与关联表的字段映射"
      icon="device_hub"
      submit-text="保存"
      width="min(840px, calc(100vw - 48px))"
      @submit="handleRelationFormSubmit"
    >
      <div class="metadata-simple-form">
        <section class="metadata-simple-section">
          <div class="metadata-simple-section__head">
            <div>
              <div class="metadata-simple-section__title">关联关系</div>
              <div class="metadata-simple-section__desc">
                先选择关联表，再选择两端用于匹配的字段
              </div>
            </div>
          </div>
          <div class="metadata-form-grid">
            <q-select
              v-model="relationForm.related_table_id"
              class="metadata-form-wide"
              :options="tableOptions"
              label="关联表"
              emit-value
              map-options
              outlined
              dense
            />
            <q-select
              v-model="relationForm.reference_key"
              :options="currentTableFieldOptions"
              label="主表字段"
              emit-value
              map-options
              outlined
              dense
            />
            <q-select
              v-model="relationForm.foreign_key"
              :options="relatedTableFieldOptions"
              label="关联表字段"
              emit-value
              map-options
              outlined
              dense
            />
            <q-select
              v-model="relationForm.relation_type"
              :options="relationTypeOptions"
              label="关系类型"
              emit-value
              map-options
              outlined
              dense
            />
            <q-input
              v-model="relationForm.many_table_code"
              label="中间表（多对多）"
              outlined
              dense
              maxlength="64"
              :disable="relationForm.relation_type !== SysTableRelationType.MANY_TO_MANY"
            />
          </div>
        </section>
      </div>

      <template #footer-status>
        {{ relationFormStatusText }}
      </template>

      <template #preview>
        <div class="metadata-form-preview">
          <div class="metadata-form-preview__title">关联预览</div>
          <div
            v-for="item in relationPreviewItems"
            :key="item.label"
            class="metadata-form-preview__row"
          >
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
      </template>
    </form-dialog-shell>

    <q-dialog v-model="showInitDialog" persistent>
      <q-card style="min-width: 420px">
        <q-card-section class="row items-center">
          <div class="text-h6">初始化元数据</div>
        </q-card-section>
        <q-card-section>
          <q-input
            v-model="initTableCode"
            label="表名（table_code）"
            outlined
            dense
            :error="initTableCodeError"
            :error-message="initTableCodeError ? '表名不能为空' : ''"
          />
        </q-card-section>
        <q-card-actions align="right">
          <q-btn flat label="取消" v-close-popup :disable="initLoading" />
          <q-btn
            color="primary"
            label="确认"
            :loading="initLoading"
            :disable="initLoading"
            @click="confirmInitTableMeta"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </base-content>
</template>
<script setup lang="ts">
defineOptions({ name: 'develop_database' })
import BaseContent from 'components/BaseContent/BaseContent.vue'
import TablePagination from 'components/Table/TablePagination.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { type QTableProps, useQuasar } from 'quasar'
import { useI18n } from 'vue-i18n'
import {
  type Table,
  useTableApi,
  type TableField,
  type TableFieldCreateReq,
  type TableFieldUpdateReq,
  type TableIndex,
  type TableIndexCreateReq,
  type TableIndexUpdateReq,
  type TableRelation,
  type TableRelationCreateReq,
  type TableRelationUpdateReq,
} from 'src/api/services/sys-table'
import type { Query } from 'src/types/global'
import {
  SysTableFieldCategory,
  SysTableFieldCategoryMap,
  SysTableFieldInputType,
  SysTableFieldInputTypeMap,
  SysTableFieldType,
  SysTableFieldTypeMap,
  SysTableRelationType,
  SysTableRelationTypeMap,
  SysTableTypeMap,
} from 'src/types/enum'
import AdvancedQuery from 'src/components/Query/AdvancedQuery.vue'
import DetailSectionNavigation from 'src/components/Detail/DetailSectionNavigation.vue'
import type { DetailSectionNavigationItem } from 'src/components/Detail/types'
import DynamicFormDialog from 'src/components/FormDialog/DynamicFormDialog.vue'
import FormDialogShell from 'src/components/FormDialog/FormDialogShell.vue'
import { useLoadingStore } from 'src/stores/loading'
import { storeToRefs } from 'pinia'
import cloneDeep from 'lodash/cloneDeep'
import { useDictStore } from 'src/stores/dict'
import { buildTableColumns, buildRelationLookups } from 'src/utils/column-format'
import { usePageButtons } from 'src/composables/page-buttons'
import { useMenuApi, type Menu, type MenuButton } from 'src/api/services/sys-menu'
import { countEffectiveQueryRules, hasEffectiveQueryRules } from 'src/utils/query-state'
import { menuButtonDisplayProps } from 'src/utils/menu-button-display'
import { compactSelectionDisplay, compactSelectionTooltip } from 'src/utils/select-display'
import { useConfirmDialog } from 'src/composables/confirm-dialog'

const loadingStore = useLoadingStore()
const { loading } = storeToRefs(loadingStore)

const $q = useQuasar()
const { t } = useI18n()
const { confirmDanger } = useConfirmDialog($q)
const tableApi = useTableApi()
const menuApi = useMenuApi()
const dictStore = useDictStore()
type TableMenuBinding = {
  lowCode: boolean
  fixedMenuTitle?: string
}
type TableRow = Table & {
  is_published?: boolean
  is_low_code_publishable?: boolean
  publish_status?: string
  publish_block_reason?: string
}
type StructureTab = 'fields' | 'indexes' | 'relations'
type StructureTag = { label: string; tone: 'ok' | 'blue' | 'warn' | 'soft' | 'muted' }
const rows = ref<TableRow[]>([])
const tableMenuBindings = ref<Record<string, TableMenuBinding>>({})

const { line_buttons, top_buttons, has_line_buttons } = usePageButtons('develop_database')

const action_handlers: Record<string, (row?: any) => void | Promise<void>> = {
  init_meta: () => initTableMetaByCode(),
  create: () => openAddDialog(),
  publish: (row) => row && confirmPublishTable(row),
  unpublish: (row) => row && confirmUnpublishTable(row.table_code),
  sync_fields: (row) => row && confirmSyncTableFields(row.table_code),
  field_manager: (row) => row && openFieldManager(row),
  update: (row) => row && openEditDialog(row),
  delete: (row) => row && confirmDelete(row),
}

const handleButtonClick = async (btn: MenuButton, row?: any) => {
  if (isButtonDisabled(btn, row)) return
  const handler = action_handlers[btn.event_action]
  if (!handler) return
  try {
    await handler(row)
  } catch (error) {
    console.error(error)
  }
}

const isButtonDisabled = (btn: MenuButton, row?: TableRow) => {
  if (btn.is_disabled || loading.value) return true
  if (!row) return false
  if (btn.event_action === 'publish') {
    return !!row.is_published || row.is_low_code_publishable === false
  }
  if (btn.event_action === 'unpublish') return !row.is_published
  return false
}

const total = ref(0)
const selected = ref([])
const columns = ref<QTableProps['columns']>([])
const table_fields_advanced = ref<TableField[]>([])
const showAdvancedQuery = ref(false)
const visibleColumns = ref<string[]>([])
const tableFields = ref<TableField[]>([])
const showFormDialog = ref(false)
const currentEditData = ref<Table | null>(null)
const showFieldDialog = ref(false)
const currentTable = ref<Table | null>(null)
const activeMetaTab = ref<StructureTab>('fields')
const structureKeyword = ref('')
const fieldRows = ref<TableField[]>([])
const fieldFormFields = ref<TableField[]>([])
const showFieldFormDialog = ref(false)
const currentEditField = ref<TableField | null>(null)
const indexRows = ref<TableIndex[]>([])
const relationRows = ref<TableRelation[]>([])
const structureNavigationItems = computed<DetailSectionNavigationItem[]>(() => [
  {
    key: 'fields',
    label: '字段',
    caption: '列表、表单、查询能力',
    count: fieldRows.value.length,
  },
  {
    key: 'indexes',
    label: '索引',
    caption: '唯一索引、组合索引',
    count: indexRows.value.length,
  },
  {
    key: 'relations',
    label: '关联关系',
    caption: '主子表、联动数据源',
    count: relationRows.value.length,
  },
])
const relationFieldMeta = ref<TableField[]>([])
const showIndexFormDialog = ref(false)
const currentEditIndex = ref<TableIndex | null>(null)
const showRelationFormDialog = ref(false)
const currentEditRelation = ref<TableRelation | null>(null)
const selectedFieldId = ref<number | null>(null)
const selectedIndexId = ref<number | null>(null)
const selectedRelationId = ref<number | null>(null)
const indexForm = ref({
  index_name: '',
  is_unique: false,
  field_ids: [] as number[],
})
const relationForm = ref({
  related_table_id: null as number | null,
  reference_key: '',
  foreign_key: '',
  relation_type: SysTableRelationType.ONE_TO_ONE,
  many_table_code: '',
})

const DB_IDENTIFIER_MAX_LENGTH = 64
const dbIdentifierPattern = /^[A-Za-z_][A-Za-z0-9_]*$/

const validateDBIdentifier = (label: string, value: unknown, required = true) => {
  const text = String(value ?? '').trim()
  if (!text) return required ? `${label}不能为空` : ''
  if (text.length > DB_IDENTIFIER_MAX_LENGTH) {
    return `${label}长度不能超过${DB_IDENTIFIER_MAX_LENGTH}`
  }
  if (!dbIdentifierPattern.test(text)) {
    return `${label}只能包含字母、数字、下划线，且不能以数字开头`
  }
  return ''
}

const warnValidation = (message: string) => {
  if (!message) return false
  $q.notify({ type: 'warning', position: 'top-right', message })
  return true
}

const tableOptions = ref<Array<{ label: string; value: number }>>([])
const tableLabelMap = ref<Record<number, string>>({})
const relatedTableFields = ref<TableField[]>([])

const showInitDialog = ref(false)
const initTableCode = ref('')
const initTableCodeError = ref(false)
const initLoading = ref(false)

// 默认空查询
const emptyAdvancedQuery = (): Query => ({
  page: 1,
  num: 15,
  expressions: [
    {
      rules: [{ field: '', value: null }],
      nested: [],
    },
  ],
})

// 实际的查询条件对象
const query = ref<Query>({
  page: 1,
  num: 15,
  order: {
    field: '',
    is_asc: true,
  },
  table_code: 'sys_table',
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

// 使用 lodash 的 cloneDeep 替代 JSON.parse(JSON.stringify())
const tempAdvancedQuery = ref<Query>(cloneDeep(query.value))

// 跟踪已应用的高级查询条件
const appliedAdvancedQuery = ref<Query>(cloneDeep(emptyAdvancedQuery()))

// 判断是否存在已应用的高级查询条件
const hasAppliedAdvancedFilters = computed(() => hasEffectiveQueryRules(appliedAdvancedQuery.value))

// 计算活跃的筛选条件数量
const activeFilterCount = computed(() => countEffectiveQueryRules(appliedAdvancedQuery.value))

const collectTableMenuBindings = (menus: Menu[], target: Record<string, TableMenuBinding> = {}) => {
  menus.forEach((menu) => {
    const tableCode = menu.table_code?.trim()
    if (tableCode) {
      const binding = target[tableCode] ?? { lowCode: false }
      if (menu.page_type === 'low_code') {
        binding.lowCode = true
      } else if (menu.page_type === 'fixed') {
        binding.fixedMenuTitle = menu.title || menu.name
      }
      target[tableCode] = binding
    }
    if (menu.children?.length) {
      collectTableMenuBindings(menu.children, target)
    }
  })
  return target
}

const refreshTableMenuBindings = async () => {
  try {
    const res = await menuApi.queryMyMenu()
    tableMenuBindings.value =
      res.success && Array.isArray(res.data) ? collectTableMenuBindings(res.data) : {}
  } catch (error) {
    console.warn('获取发布状态失败', error)
    tableMenuBindings.value = {}
  }
}

const withPublishStatus = (items: Table[]): TableRow[] =>
  items.map((item) => {
    const binding = tableMenuBindings.value[item.table_code]
    const isPublished = !!binding?.lowCode
    const isFixedPage = !!binding?.fixedMenuTitle && !isPublished
    return {
      ...item,
      is_published: isPublished,
      is_low_code_publishable: !isFixedPage,
      publish_status: isFixedPage ? '固定页面' : isPublished ? '已发布' : '未发布',
      publish_block_reason: isFixedPage
        ? `已绑定固定菜单 ${binding.fixedMenuTitle}，不能发布成低代码页面`
        : '',
    }
  })

const publishStatusColor = (row: TableRow) => {
  if (row.is_published) return 'positive'
  if (row.is_low_code_publishable === false) return 'warning'
  return 'grey-6'
}

const indexFieldOptions = computed(() => {
  return fieldRows.value.map((field) => ({
    label: `${field.field_name} (${field.field_code})`,
    value: field.id,
  }))
})

const currentTableFieldOptions = computed(() => {
  return fieldRows.value.map((field) => ({
    label: `${field.field_name} (${field.field_code})`,
    value: field.field_code,
  }))
})

const relatedTableFieldOptions = computed(() => {
  return relatedTableFields.value.map((field) => ({
    label: `${field.field_name} (${field.field_code})`,
    value: field.field_code,
  }))
})

const relationTypeOptions = computed(() => {
  const rtField = relationFieldMeta.value.find((f) => f.field_code === 'relation_type')
  if (rtField?.dict_code) {
    return dictStore.getDictOptions(rtField.dict_code).map((item) => ({
      label: item.label,
      value: Number(item.value),
    }))
  }
  return []
})

const structureFieldColumns: QTableProps['columns'] = [
  { name: 'sequence', label: '序号', field: 'sequence', align: 'left', headerStyle: 'width: 72px' },
  { name: 'field', label: '字段', field: 'field_name', align: 'left' },
  { name: 'type', label: '类型', field: 'field_type', align: 'left', headerStyle: 'width: 130px' },
  {
    name: 'input',
    label: '输入控件',
    field: 'input_type',
    align: 'left',
    headerStyle: 'width: 130px',
  },
  { name: 'display', label: '页面显示', field: 'is_list_show', align: 'left' },
  { name: 'search', label: '查询能力', field: 'is_quick_search', align: 'left' },
  { name: 'constraints', label: '约束', field: 'is_primary_key', align: 'left' },
  {
    name: 'actions',
    label: '调整',
    field: 'actions',
    align: 'center',
    headerStyle: 'width: 96px',
  },
]

const structureIndexColumns: QTableProps['columns'] = [
  { name: 'name', label: '索引名称', field: 'index_name', align: 'left' },
  { name: 'type', label: '类型', field: 'is_unique', align: 'left', headerStyle: 'width: 130px' },
  { name: 'fields', label: '索引字段', field: 'index_fields', align: 'left' },
]

const structureRelationColumns: QTableProps['columns'] = [
  { name: 'relation', label: '关联关系', field: 'relation_type', align: 'left' },
  { name: 'mapping', label: '字段映射', field: 'reference_key', align: 'left' },
  {
    name: 'many',
    label: '中间表',
    field: 'many_table_code',
    align: 'left',
    headerStyle: 'width: 180px',
  },
]

const enumLabel = (map: Record<string, string>, value: unknown, fallback = '-') => {
  if (value === undefined || value === null || value === '') return fallback
  return map[String(value)] ?? String(value)
}

const fieldTypeLabel = (field: TableField) => {
  const base = enumLabel(SysTableFieldTypeMap as Record<string, string>, field.field_type)
  const length = Number(field.field_length || 0)
  const decimal = Number(field.field_decimal_length || 0)
  if (length > 0 && decimal > 0) return `${base}(${length},${decimal})`
  if (length > 0) return `${base}(${length})`
  return base
}

const inputTypeLabel = (field: TableField) =>
  enumLabel(SysTableFieldInputTypeMap as Record<string, string>, field.input_type)

const fieldCategoryLabel = (field: TableField) =>
  enumLabel(
    SysTableFieldCategoryMap as Record<string, string>,
    field.field_category || SysTableFieldCategory.NORMAL,
  )

const relationTypeLabel = (relation: TableRelation) =>
  enumLabel(SysTableRelationTypeMap as Record<string, string>, relation.relation_type)

const tableKindLabel = computed(() =>
  currentTable.value
    ? enumLabel(SysTableTypeMap as Record<string, string>, currentTable.value.table_type)
    : '-',
)

const hasLinkageConfig = (field: TableField) => {
  const raw = field.linkage_config as unknown
  if (raw === undefined || raw === null) return false
  if (typeof raw === 'string') {
    const value = raw.trim()
    return value !== '' && value !== '{}' && value !== 'null'
  }
  return true
}

const fieldDisplayTags = (field: TableField): StructureTag[] => {
  const tags: StructureTag[] = []
  if (field.is_list_show) tags.push({ label: '列表', tone: 'ok' })
  if (field.is_insert_show) tags.push({ label: '新增', tone: 'ok' })
  if (field.is_update_show) tags.push({ label: '编辑', tone: 'ok' })
  return tags.length > 0 ? tags : [{ label: '隐藏', tone: 'muted' }]
}

const fieldSearchTags = (field: TableField): StructureTag[] => {
  const tags: StructureTag[] = []
  if (field.is_quick_search) tags.push({ label: '快捷', tone: 'blue' })
  if (field.is_advanced_search) tags.push({ label: '高级', tone: 'warn' })
  if (field.is_sort) tags.push({ label: '排序', tone: 'blue' })
  return tags.length > 0 ? tags : [{ label: '无', tone: 'muted' }]
}

const fieldConstraintTags = (field: TableField): StructureTag[] => {
  const tags: StructureTag[] = []
  if (field.is_primary_key) tags.push({ label: '主键', tone: 'ok' })
  if (field.is_index) tags.push({ label: '索引', tone: 'blue' })
  if (!field.is_null) tags.push({ label: '非空', tone: 'soft' })
  if (field.dict_code) tags.push({ label: '字典', tone: 'ok' })
  if (hasLinkageConfig(field)) tags.push({ label: '联动', tone: 'warn' })
  if (field.field_category && field.field_category !== SysTableFieldCategory.NORMAL) {
    tags.push({ label: fieldCategoryLabel(field), tone: 'warn' })
  }
  return tags.length > 0 ? tags : [{ label: '普通', tone: 'muted' }]
}

const normalizeSearch = (value: unknown) =>
  String(value ?? '')
    .trim()
    .toLowerCase()

const structureSearchText = computed(() => normalizeSearch(structureKeyword.value))

const sortedFieldRows = computed(() =>
  [...fieldRows.value].sort((a, b) => (a.sequence ?? 0) - (b.sequence ?? 0) || a.id - b.id),
)

const sortedIndexRows = computed(() => [...indexRows.value].sort((a, b) => a.id - b.id))

const sortedRelationRows = computed(() => [...relationRows.value].sort((a, b) => a.id - b.id))

const filteredFieldRows = computed(() => {
  const keyword = structureSearchText.value
  if (!keyword) return sortedFieldRows.value
  return sortedFieldRows.value.filter((field) =>
    [
      field.field_name,
      field.field_code,
      fieldTypeLabel(field),
      inputTypeLabel(field),
      field.dict_code,
      fieldCategoryLabel(field),
    ]
      .map(normalizeSearch)
      .some((value) => value.includes(keyword)),
  )
})

const filteredIndexRows = computed(() => {
  const keyword = structureSearchText.value
  if (!keyword) return sortedIndexRows.value
  return sortedIndexRows.value.filter((index) =>
    [
      index.index_name,
      index.is_unique ? '唯一索引' : '普通索引',
      ...(index.index_fields || []).flatMap((field) => [field.field_name, field.field_code]),
    ]
      .map(normalizeSearch)
      .some((value) => value.includes(keyword)),
  )
})

const relatedTableLabel = (tableId: number) =>
  tableLabelMap.value[tableId] || String(tableId || '-')

const indexUniqueOptions = [
  { label: '普通索引', value: false },
  { label: '唯一索引', value: true },
]

const findOptionLabel = <T extends string | number | boolean | null>(
  options: Array<{ label: string; value: T }>,
  value: T,
  fallback = '-',
) => options.find((item) => item.value === value)?.label || fallback

const selectedIndexFieldLabels = computed(() =>
  indexForm.value.field_ids
    .map((fieldId) => findOptionLabel(indexFieldOptions.value, fieldId, ''))
    .filter(Boolean),
)

const indexFormStatusText = computed(() => {
  const missing: string[] = []
  if (!indexForm.value.index_name.trim()) missing.push('索引名称')
  if (indexForm.value.field_ids.length === 0) missing.push('索引字段')
  if (missing.length > 0) return `待完善：${missing.join('、')}`
  return `${indexForm.value.is_unique ? '唯一索引' : '普通索引'}，${indexForm.value.field_ids.length} 个字段`
})

const indexPreviewItems = computed(() => [
  { label: '所属表', value: currentTable.value?.table_code || '-' },
  { label: '索引名称', value: indexForm.value.index_name.trim() || '-' },
  { label: '索引类型', value: indexForm.value.is_unique ? '唯一索引' : '普通索引' },
  {
    label: '索引字段',
    value: selectedIndexFieldLabels.value.length ? selectedIndexFieldLabels.value.join('、') : '-',
  },
])

const relationTypeFormLabel = computed(
  () =>
    findOptionLabel(relationTypeOptions.value, relationForm.value.relation_type, '') ||
    enumLabel(SysTableRelationTypeMap as Record<string, string>, relationForm.value.relation_type),
)

const selectedRelationTableLabel = computed(() =>
  relationForm.value.related_table_id
    ? relatedTableLabel(relationForm.value.related_table_id)
    : '-',
)

const selectedReferenceFieldLabel = computed(() =>
  findOptionLabel(currentTableFieldOptions.value, relationForm.value.reference_key, '-'),
)

const selectedForeignFieldLabel = computed(() =>
  findOptionLabel(relatedTableFieldOptions.value, relationForm.value.foreign_key, '-'),
)

const relationFormStatusText = computed(() => {
  const missing: string[] = []
  if (!relationForm.value.related_table_id) missing.push('关联表')
  if (!relationForm.value.reference_key) missing.push('主表字段')
  if (!relationForm.value.foreign_key) missing.push('关联表字段')
  if (
    relationForm.value.relation_type === SysTableRelationType.MANY_TO_MANY &&
    !relationForm.value.many_table_code.trim()
  ) {
    missing.push('中间表')
  }
  return missing.length > 0 ? `待完善：${missing.join('、')}` : '关联关系已配置'
})

const relationPreviewItems = computed(() => [
  { label: '主表', value: currentTable.value?.table_code || '-' },
  { label: '关联表', value: selectedRelationTableLabel.value },
  { label: '关系类型', value: relationTypeFormLabel.value || '-' },
  {
    label: '字段映射',
    value:
      relationForm.value.reference_key && relationForm.value.foreign_key
        ? `${selectedReferenceFieldLabel.value} -> ${selectedForeignFieldLabel.value}`
        : '-',
  },
  {
    label: '中间表',
    value:
      relationForm.value.relation_type === SysTableRelationType.MANY_TO_MANY
        ? relationForm.value.many_table_code.trim() || '-'
        : '不需要',
  },
])

const filteredRelationRows = computed(() => {
  const keyword = structureSearchText.value
  if (!keyword) return sortedRelationRows.value
  return sortedRelationRows.value.filter((relation) =>
    [
      relationTypeLabel(relation),
      relatedTableLabel(relation.related_table_id),
      relation.reference_key,
      relation.foreign_key,
      relation.many_table_code,
    ]
      .map(normalizeSearch)
      .some((value) => value.includes(keyword)),
  )
})

const selectedField = computed(() => {
  const rows = structureSearchText.value ? filteredFieldRows.value : sortedFieldRows.value
  return rows.find((field) => field.id === selectedFieldId.value) ?? rows[0] ?? null
})

const selectedIndex = computed(() => {
  const rows = structureSearchText.value ? filteredIndexRows.value : sortedIndexRows.value
  return rows.find((index) => index.id === selectedIndexId.value) ?? rows[0] ?? null
})

const selectedRelation = computed(() => {
  const rows = structureSearchText.value ? filteredRelationRows.value : sortedRelationRows.value
  return rows.find((relation) => relation.id === selectedRelationId.value) ?? rows[0] ?? null
})

const structureSearchPlaceholder = computed(() => {
  if (activeMetaTab.value === 'indexes') return '搜索索引名称 / 字段'
  if (activeMetaTab.value === 'relations') return '搜索关联表 / 字段 / 关系类型'
  return '搜索字段名称 / 编码 / 类型'
})

const selectField = (row: TableField) => {
  selectedFieldId.value = row.id
}

const selectIndex = (row: TableIndex) => {
  selectedIndexId.value = row.id
}

const selectRelation = (row: TableRelation) => {
  selectedRelationId.value = row.id
}

const setMetaTab = (tab: StructureTab) => {
  activeMetaTab.value = tab
  structureKeyword.value = ''
}

const setMetaTabValue = (tab: string) => {
  if (tab === 'fields' || tab === 'indexes' || tab === 'relations') {
    setMetaTab(tab)
  }
}

const pagination = ref({
  page: query.value.page,
  rowsPerPage: 0,
  sortBy: '',
  descending: false,
})

// 初始化临时查询
const initTempQuery = () => {
  tempAdvancedQuery.value = cloneDeep(query.value)
}

const resetToFirstPageOrFetch = () => {
  if (pagination.value.page !== 1) {
    pagination.value.page = 1
    return
  }
  fetchData()
}

// 基础查询处理
const handleBasicSearch = () => {
  // 基本查询时重置高级查询部分，保留基本的关键字查询
  query.value.expressions = emptyAdvancedQuery().expressions
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })
  resetToFirstPageOrFetch()
}

// 高级查询处理
const handleAdvancedSearch = () => {
  // 应用临时查询条件到实际查询
  query.value.expressions = cloneDeep(tempAdvancedQuery.value.expressions)

  // 更新已应用的高级查询状态
  appliedAdvancedQuery.value = cloneDeep({
    expressions: query.value.expressions,
    page: query.value.page,
    num: query.value.num,
  })

  resetToFirstPageOrFetch()
  showAdvancedQuery.value = false
}

// 查询数据
const fetchData = async () => {
  await refreshTableMenuBindings()
  const res = await tableApi.queryTable(query.value)
  rows.value = withPublishStatus(Array.isArray(res.data) ? res.data : [])
  total.value = res.total || 0
}

const fetchTableFields = async () => {
  const res = await tableApi.queryTableByCode('sys_table')
  if (res.data && res.data.table_fields) {
    tableFields.value = res.data.table_fields.filter(
      (field) => field.is_insert_show || field.is_update_show,
    )
  }
  // 构建主表列
  const dictCodes0 = res.data.table_fields.map((f) => f.dict_code).filter((c): c is string => !!c)
  const [, mainRelationLookups] = await Promise.all([
    dictStore.loadDicts(dictCodes0),
    buildRelationLookups(res.data.table_fields),
  ])

  const { columns: mainCols, advancedFields } = buildTableColumns(res.data.table_fields, {
    getDictLabel: dictStore.getDictLabel,
    relationLookups: mainRelationLookups,
  })
  columns.value = [
    ...mainCols.filter((column) => column.name !== 'publish_status'),
    {
      name: 'publish_status',
      align: 'center',
      label: '发布状态',
      field: 'publish_status',
      sortable: false,
    },
  ]
  table_fields_advanced.value = advancedFields
  visibleColumns.value = columns.value.map((c) => c.name)
  if (!has_line_buttons.value) {
    visibleColumns.value = visibleColumns.value.filter((c) => c !== 'actions')
  }
  await fetchData()
}

const initialized = ref(false)

onMounted(async () => {
  await fetchTableFields()
  initialized.value = true
})

const openAddDialog = () => {
  currentEditData.value = null
  showFormDialog.value = true
}

const openEditDialog = (row: Table) => {
  currentEditData.value = cloneDeep(row)
  showFormDialog.value = true
}

const confirmDelete = (row: Table) => {
  confirmDanger({
    message: `确定要删除数据表 "${row.table_name}" 吗？`,
  }).onOk(() => {
    void (async () => {
      const result = await tableApi.deleteTable(row.id)
      if (result.success) {
        await fetchData()
      }
    })()
  })
}

const handleFormSubmit = async (formPayload: { data: Table; isEdit: boolean; id?: number }) => {
  const tableCode = String(formPayload.data.table_code ?? '').trim()
  if (warnValidation(validateDBIdentifier('表编码', tableCode))) return
  if (formPayload.isEdit && formPayload.id) {
    const result = await tableApi.updateTable({
      id: formPayload.id,
      table_name: formPayload.data.table_name,
      table_code: tableCode,
      table_type: formPayload.data.table_type,
      master_detail_mode: formPayload.data.master_detail_mode,
      form_open_mode: formPayload.data.form_open_mode,
      detail_open_mode: formPayload.data.detail_open_mode,
      parent_id: formPayload.data.parent_id ?? 0,
      sql: formPayload.data.sql ?? '',
    })
    if (result.success) {
      showFormDialog.value = false
      await fetchData()
    }
  } else {
    const result = await tableApi.createTable({
      table_name: formPayload.data.table_name,
      table_code: tableCode,
      table_type: formPayload.data.table_type,
      master_detail_mode: formPayload.data.master_detail_mode,
      form_open_mode: formPayload.data.form_open_mode,
      detail_open_mode: formPayload.data.detail_open_mode,
      parent_id: formPayload.data.parent_id ?? 0,
      sql: formPayload.data.sql ?? '',
    })
    if (result.success) {
      showFormDialog.value = false
      await fetchData()
    }
  }
}

/**
 * 统一加载子表（字段 / 索引 / 关联）的元数据、表单字段、列定义
 *
 * - sys_table_field：元数据驱动列 + 表单
 * - sys_table_relation：元数据驱动列 + 表单
 * - sys_table_index：元数据驱动列（index_fields 用 customFormats 处理嵌套数组）
 */
const fetchSubTableMeta = async () => {
  const [fieldRes, relationRes, indexRes] = await Promise.all([
    tableApi.queryTableByCode('sys_table_field'),
    tableApi.queryTableByCode('sys_table_relation'),
    tableApi.queryTableByCode('sys_table_index'),
  ])

  // ── 字段子表 ─────────────────────────────────
  if (fieldRes.data?.table_fields) {
    const fields = fieldRes.data.table_fields
    fieldFormFields.value = ensureFieldLayoutFormFields(fields)

    const dictCodes = fields.map((f) => f.dict_code).filter((c): c is string => !!c)
    await dictStore.loadDicts(dictCodes)
  }

  // ── 关联子表 ─────────────────────────────────
  if (relationRes.data?.table_fields) {
    const relFields = relationRes.data.table_fields
    relationFieldMeta.value = relFields

    const dictCodes = relFields.map((f) => f.dict_code).filter((c): c is string => !!c)
    if (dictCodes.length > 0) {
      await dictStore.loadDicts(dictCodes)
    }
  }

  // ── 索引子表 ─────────────────────────────────
  if (indexRes.data?.table_fields) {
    const idxFields = indexRes.data.table_fields

    const dictCodes = idxFields.map((f) => f.dict_code).filter((c): c is string => !!c)
    if (dictCodes.length > 0) {
      await dictStore.loadDicts(dictCodes)
    }
  }
}

const buildFieldLayoutMetaField = (
  base: TableField | undefined,
  fieldCode: 'form_span' | 'detail_span',
  fieldName: string,
  sequence: number,
) =>
  ({
    ...(base || {}),
    id: 0,
    table_id: base?.table_id || 0,
    field_name: fieldName,
    field_code: fieldCode,
    field_type: SysTableFieldType.INT,
    field_length: 0,
    field_decimal_length: 0,
    input_type: SysTableFieldInputType.INPUT_NUMBER,
    form_span: 1,
    detail_span: 1,
    default_value: '0',
    dict_code: '',
    is_primary_key: false,
    is_index: false,
    is_quick_search: false,
    is_advanced_search: false,
    is_sort: false,
    is_null: true,
    is_list_show: false,
    is_insert_show: true,
    is_update_show: true,
    sequence,
    original_field_id: 0,
    binding: '',
    field_category: SysTableFieldCategory.NORMAL,
    expression: '',
    linkage_config: '',
    state: true,
  }) as TableField

const ensureFieldLayoutFormFields = (fields: TableField[]) => {
  const result = fields.filter((field) => field.is_insert_show || field.is_update_show)
  const codes = new Set(result.map((field) => field.field_code))
  const base = fields[0]

  if (!codes.has('form_span')) {
    result.push(buildFieldLayoutMetaField(base, 'form_span', '表单占位', 251))
  }
  if (!codes.has('detail_span')) {
    result.push(buildFieldLayoutMetaField(base, 'detail_span', '详情占位', 252))
  }

  return result.sort((a, b) => (a.sequence || 0) - (b.sequence || 0))
}

const fetchIndexesByTableId = async (tableId: number) => {
  const res = await tableApi.queryTableIndexByTableId(tableId)
  indexRows.value = Array.isArray(res.data) ? res.data : []
  if (!indexRows.value.some((item) => item.id === selectedIndexId.value)) {
    selectedIndexId.value = sortedIndexRows.value[0]?.id ?? null
  }
}

const fetchRelationsByTableId = async (tableId: number) => {
  const res = await tableApi.queryTableRelationsByTableId(tableId)
  relationRows.value = Array.isArray(res.data) ? res.data : []
  if (!relationRows.value.some((item) => item.id === selectedRelationId.value)) {
    selectedRelationId.value = sortedRelationRows.value[0]?.id ?? null
  }
}

const fetchAllTables = async () => {
  const tableQuery: Query = {
    page: 1,
    num: 500,
    table_code: 'sys_table',
    expressions: emptyAdvancedQuery().expressions,
    quick_query: { keyword: '' },
    include_deleted: false,
  }
  const res = await tableApi.queryTable(tableQuery)
  const list = Array.isArray(res.data) ? res.data : []
  tableOptions.value = list.map((item) => ({
    label: `${item.table_name} (${item.table_code})`,
    value: item.id,
  }))
  const labelMap: Record<number, string> = {}
  list.forEach((item) => {
    labelMap[item.id] = `${item.table_name} (${item.table_code})`
  })
  tableLabelMap.value = labelMap
}

const fetchRelatedTableFields = async (tableId: number) => {
  const res = await tableApi.queryTableFieldsByTableId(tableId)
  relatedTableFields.value = Array.isArray(res.data) ? res.data : []
}

const fetchFieldsByTableId = async (tableId: number) => {
  const res = await tableApi.queryTableFieldsByTableId(tableId)
  fieldRows.value = Array.isArray(res.data) ? res.data : []
  if (!fieldRows.value.some((item) => item.id === selectedFieldId.value)) {
    selectedFieldId.value = sortedFieldRows.value[0]?.id ?? null
  }
}

const openFieldManager = async (row: Table) => {
  currentTable.value = row
  activeMetaTab.value = 'fields'
  structureKeyword.value = ''
  selectedFieldId.value = null
  selectedIndexId.value = null
  selectedRelationId.value = null
  showFieldDialog.value = true
  // 并行加载所有表选项 + 子表元数据
  await Promise.all([fetchAllTables(), fetchSubTableMeta()])
  // 并行加载当前表的字段、索引、关联数据
  await Promise.all([
    fetchFieldsByTableId(row.id),
    fetchIndexesByTableId(row.id),
    fetchRelationsByTableId(row.id),
  ])
}

const initTableMetaByCode = () => {
  initTableCode.value = ''
  initTableCodeError.value = false
  showInitDialog.value = true
}

const confirmInitTableMeta = async () => {
  const code = String(initTableCode.value || '').trim()
  if (!code) {
    initTableCodeError.value = true
    return
  }

  initTableCodeError.value = false
  initLoading.value = true
  try {
    const result = await tableApi.initTable(code)
    if (result.success) {
      await fetchData()
      showInitDialog.value = false
    }
  } catch (error) {
    console.error(error)
  } finally {
    initLoading.value = false
  }
}

const confirmSyncTableFields = (tableCode: string) => {
  $q.dialog({
    title: '确认同步',
    message: `确定要同步表 "${tableCode}" 的字段元数据吗？`,
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void (async () => {
      try {
        const result = await tableApi.syncTable(tableCode)
        if (result.success) {
          await fetchData()
          if (currentTable.value?.table_code === tableCode) {
            await fetchFieldsByTableId(currentTable.value.id)
          }
        }
      } catch (error) {
        console.error(error)
      }
    })()
  })
}

const confirmSyncCurrentTableFields = () => {
  if (!currentTable.value?.table_code) return
  confirmSyncTableFields(currentTable.value.table_code)
}

const publishParentQuery = (): Query => ({
  page: 1,
  num: 1000,
  order: {
    field: 'sequence',
    is_asc: true,
  },
  expressions: emptyAdvancedQuery().expressions,
  quick_query: {
    keyword: '',
  },
  include_deleted: false,
})

type PublishParentOption = { label: string; value: string; isDefault?: boolean }

const isDefaultPublishParent = (menu: Menu) =>
  menu.title === 'router.develop.default' || menu.name === 'develop' || menu.path === 'develop'

const getPublishMenuTitle = (menu: Menu) => {
  const title = menu.title || menu.name || menu.path || String(menu.id)
  return title.startsWith('router.') ? t(title) : title
}

const isPublishParentMenu = (menu: Menu) => {
  if (!menu.state || menu.is_hidden) return false
  if (menu.page_type) return menu.page_type === 'directory'
  return !menu.table_code && Boolean(menu.children?.length)
}

const collectPublishParentOptions = (menus: Menu[], level = 0): PublishParentOption[] => {
  const options: PublishParentOption[] = []
  menus.forEach((menu) => {
    if (isPublishParentMenu(menu)) {
      const prefix = level > 0 ? `${'　'.repeat(level)}└ ` : ''
      const isDefault = isDefaultPublishParent(menu)
      options.push({
        label: `${prefix}${getPublishMenuTitle(menu)}${isDefault ? '（默认）' : ''}`,
        value: String(menu.id),
        isDefault,
      })
    }
    if (menu.children?.length) {
      options.push(...collectPublishParentOptions(menu.children, level + 1))
    }
  })
  return options
}

const loadPublishParentOptions = async () => {
  const fallbackOption = { label: '开发管理（默认）', value: '0' }
  try {
    const res = await menuApi.queryMenu(publishParentQuery())
    if (!res.success || !Array.isArray(res.data)) {
      return {
        items: [fallbackOption],
        model: fallbackOption.value,
      }
    }
    const options = collectPublishParentOptions(res.data)
    const defaultOption = options.find((item) => item.isDefault)
    return {
      items: options.length ? options : [fallbackOption],
      model: defaultOption?.value ?? options[0]?.value ?? fallbackOption.value,
    }
  } catch (error) {
    console.error('加载发布目录失败', error)
    return {
      items: [fallbackOption],
      model: fallbackOption.value,
    }
  }
}

const confirmPublishTable = (row: TableRow) => {
  if (row.is_low_code_publishable === false) {
    $q.dialog({
      title: '不能发布',
      message: row.publish_block_reason || '当前表已绑定固定页面，不能发布成低代码页面。',
      ok: '知道了',
    })
    return
  }
  const tableCode = row.table_code
  void (async () => {
    const parentOptions = await loadPublishParentOptions()
    $q.dialog({
      title: '确认发布',
      message: `确定要将表 "${tableCode}" 发布为低代码菜单吗？`,
      options: {
        type: 'radio',
        model: parentOptions.model,
        items: parentOptions.items,
      },
      cancel: true,
      persistent: true,
    }).onOk((parentId) => {
      void (async () => {
        try {
          const result = await tableApi.publishTable(tableCode, {
            parent_id: Number(parentId) || 0,
          })
          if (result.success) {
            await fetchData()
          }
        } catch (error) {
          console.error(error)
        }
      })()
    })
  })()
}

const confirmUnpublishTable = (tableCode: string) => {
  $q.dialog({
    title: '确认下线',
    message: `确定要下线表 "${tableCode}" 的低代码菜单吗？`,
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void (async () => {
      try {
        const result = await tableApi.unpublishTable(tableCode)
        if (result.success) {
          await fetchData()
        }
      } catch (error) {
        console.error(error)
      }
    })()
  })
}

const confirmSyncTableIndexes = () => {
  if (!currentTable.value?.table_code) return
  const tableCode = currentTable.value.table_code
  $q.dialog({
    title: '确认同步',
    message: `确定要同步表 "${tableCode}" 的索引元数据吗？`,
    cancel: true,
    persistent: true,
  }).onOk(() => {
    void (async () => {
      try {
        const result = await tableApi.syncTableIndexes(tableCode)
        if (result.success) {
          if (currentTable.value?.id) {
            await fetchIndexesByTableId(currentTable.value.id)
          }
        }
      } catch (error) {
        console.error(error)
      }
    })()
  })
}

const refreshCurrentStructure = async () => {
  if (!currentTable.value?.id) return
  if (activeMetaTab.value === 'fields') {
    await fetchFieldsByTableId(currentTable.value.id)
    return
  }
  if (activeMetaTab.value === 'indexes') {
    await fetchIndexesByTableId(currentTable.value.id)
    return
  }
  await fetchRelationsByTableId(currentTable.value.id)
}

const openAddFieldDialog = () => {
  currentEditField.value = null
  showFieldFormDialog.value = true
}

const openEditFieldDialog = (row: TableField) => {
  currentEditField.value = cloneDeep(row)
  showFieldFormDialog.value = true
}

const confirmDeleteField = (row: TableField) => {
  confirmDanger({
    message: `确定要删除字段 "${row.field_name}" 吗？`,
  }).onOk(() => {
    void (async () => {
      const result = await tableApi.deleteTableField(row.id)
      if (result.success && currentTable.value?.id) {
        await fetchFieldsByTableId(currentTable.value.id)
      }
    })()
  })
}

const handleFieldFormSubmit = async (formPayload: {
  data: TableField
  isEdit: boolean
  id?: number
}) => {
  if (!currentTable.value) return
  const toNumber = (val: any, fallback = 0) => {
    if (val === '' || val === null || val === undefined) return fallback
    const num = Number(val)
    return Number.isNaN(num) ? fallback : num
  }
  const fieldType = (formPayload.data as any).type ?? formPayload.data.field_type
  const inputType = formPayload.data.input_type
  const fieldCode = String(formPayload.data.field_code ?? '').trim()
  if (warnValidation(validateDBIdentifier('字段编码', fieldCode))) return
  // 将 linkage_config 序列化为字符串（JsonEditor 会将 JSON 字符串解析为对象）
  const rawLinkage = (formPayload.data as any).linkage_config
  const linkageConfigStr =
    typeof rawLinkage === 'object' && rawLinkage !== null
      ? JSON.stringify(rawLinkage)
      : (rawLinkage ?? '')
  if (
    inputType === SysTableFieldInputType.SELECT &&
    !formPayload.data.dict_code &&
    !linkageConfigStr
  ) {
    $q.notify({
      type: 'warning',
      position: 'top-right',
      message: '选择“下拉选择”时必须指定字典编码或关联配置',
    })
    return
  }
  if (formPayload.isEdit && formPayload.id) {
    const req: TableFieldUpdateReq = {
      id: formPayload.id,
      table_id: currentTable.value.id,
      field_name: formPayload.data.field_name,
      field_code: fieldCode,
      type: fieldType,
      field_length: toNumber(formPayload.data.field_length, 0),
      field_decimal_length: toNumber(formPayload.data.field_decimal_length, 0),
      input_type: formPayload.data.input_type,
      form_span: toNumber((formPayload.data as any).form_span, 0),
      detail_span: toNumber((formPayload.data as any).detail_span, 0),
      default_value: formPayload.data.default_value ?? '',
      dict_code: formPayload.data.dict_code ?? '',
      is_primary_key: formPayload.data.is_primary_key ?? false,
      is_index: formPayload.data.is_index ?? false,
      is_quick_search: formPayload.data.is_quick_search ?? false,
      is_advanced_search: formPayload.data.is_advanced_search ?? false,
      is_sort: formPayload.data.is_sort ?? false,
      is_null: formPayload.data.is_null ?? true,
      is_list_show: formPayload.data.is_list_show ?? true,
      is_insert_show: formPayload.data.is_insert_show ?? true,
      is_update_show: formPayload.data.is_update_show ?? true,
      sequence: toNumber(formPayload.data.sequence, 0),
      original_field_id: toNumber(formPayload.data.original_field_id, 0),
      binding: formPayload.data.binding ?? '',
      field_category: formPayload.data.field_category || SysTableFieldCategory.NORMAL,
      expression: formPayload.data.expression ?? '',
      linkage_config: linkageConfigStr,
    }
    const result = await tableApi.updateTableField(req)
    if (result.success) {
      showFieldFormDialog.value = false
      await fetchFieldsByTableId(currentTable.value.id)
    }
  } else {
    const req: TableFieldCreateReq = {
      table_id: currentTable.value.id,
      field_name: formPayload.data.field_name,
      field_code: fieldCode,
      type: fieldType,
      field_length: toNumber(formPayload.data.field_length, 0),
      field_decimal_length: toNumber(formPayload.data.field_decimal_length, 0),
      input_type: formPayload.data.input_type,
      form_span: toNumber((formPayload.data as any).form_span, 0),
      detail_span: toNumber((formPayload.data as any).detail_span, 0),
      default_value: formPayload.data.default_value ?? '',
      dict_code: formPayload.data.dict_code ?? '',
      is_primary_key: formPayload.data.is_primary_key ?? false,
      is_index: formPayload.data.is_index ?? false,
      is_quick_search: formPayload.data.is_quick_search ?? false,
      is_advanced_search: formPayload.data.is_advanced_search ?? false,
      is_sort: formPayload.data.is_sort ?? false,
      is_null: formPayload.data.is_null ?? true,
      is_list_show: formPayload.data.is_list_show ?? true,
      is_insert_show: formPayload.data.is_insert_show ?? true,
      is_update_show: formPayload.data.is_update_show ?? true,
      sequence: toNumber(formPayload.data.sequence, 0),
      original_field_id: toNumber(formPayload.data.original_field_id, 0),
      binding: formPayload.data.binding ?? '',
      field_category: formPayload.data.field_category || SysTableFieldCategory.NORMAL,
      expression: formPayload.data.expression ?? '',
      linkage_config: linkageConfigStr,
    }
    const result = await tableApi.createTableField(req)
    if (result.success) {
      showFieldFormDialog.value = false
      await fetchFieldsByTableId(currentTable.value.id)
    }
  }
}

const resetIndexForm = () => {
  indexForm.value = {
    index_name: '',
    is_unique: false,
    field_ids: [],
  }
}

const openAddIndexDialog = () => {
  currentEditIndex.value = null
  resetIndexForm()
  showIndexFormDialog.value = true
}

const openEditIndexDialog = (row: TableIndex) => {
  currentEditIndex.value = cloneDeep(row)
  indexForm.value = {
    index_name: row.index_name,
    is_unique: row.is_unique,
    field_ids: Array.isArray(row.index_fields)
      ? row.index_fields.map((field) => field.id).filter((id) => typeof id === 'number')
      : [],
  }
  showIndexFormDialog.value = true
}

const buildIndexFieldReqs = (tableId: number, fieldIds: number[]) => {
  return fieldIds
    .map((id) => fieldRows.value.find((field) => field.id === id))
    .filter((field): field is TableField => Boolean(field))
    .map((field) => ({
      table_id: tableId,
      field_id: field.id,
      field_code: field.field_code,
    }))
}

const handleIndexFormSubmit = async () => {
  if (!currentTable.value) return
  const indexName = indexForm.value.index_name.trim()
  if (warnValidation(validateDBIdentifier('索引名称', indexName))) return
  if (indexForm.value.field_ids.length === 0) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请至少选择一个索引字段' })
    return
  }
  const indexFields = buildIndexFieldReqs(currentTable.value.id, indexForm.value.field_ids)
  if (indexFields.length !== indexForm.value.field_ids.length) {
    $q.notify({ type: 'warning', position: 'top-right', message: '索引字段无效，请重新选择' })
    return
  }

  if (currentEditIndex.value?.id) {
    const req: TableIndexUpdateReq = {
      id: currentEditIndex.value.id,
      table_id: currentTable.value.id,
      index_name: indexName,
      is_unique: indexForm.value.is_unique,
      index_fields: indexFields,
    }
    const result = await tableApi.updateTableIndex(req)
    if (result.success) {
      showIndexFormDialog.value = false
      await fetchIndexesByTableId(currentTable.value.id)
    }
  } else {
    const req: TableIndexCreateReq = {
      table_id: currentTable.value.id,
      index_name: indexName,
      is_unique: indexForm.value.is_unique,
      index_fields: indexFields,
    }
    const result = await tableApi.createTableIndex(req)
    if (result.success) {
      showIndexFormDialog.value = false
      await fetchIndexesByTableId(currentTable.value.id)
    }
  }
}

const confirmDeleteIndex = (row: TableIndex) => {
  confirmDanger({
    message: `确定要删除索引 "${row.index_name}" 吗？`,
  }).onOk(() => {
    void (async () => {
      const result = await tableApi.deleteTableIndex(row.id)
      if (result.success && currentTable.value?.id) {
        await fetchIndexesByTableId(currentTable.value.id)
      }
    })()
  })
}

const resetRelationForm = () => {
  relationForm.value = {
    related_table_id: null,
    reference_key: '',
    foreign_key: '',
    relation_type: SysTableRelationType.ONE_TO_ONE,
    many_table_code: '',
  }
}

const openAddRelationDialog = () => {
  currentEditRelation.value = null
  resetRelationForm()
  relatedTableFields.value = []
  showRelationFormDialog.value = true
}

const openEditRelationDialog = async (row: TableRelation) => {
  currentEditRelation.value = cloneDeep(row)
  relationForm.value = {
    related_table_id: row.related_table_id,
    reference_key: row.reference_key,
    foreign_key: row.foreign_key,
    relation_type: row.relation_type,
    many_table_code: row.many_table_code || '',
  }
  if (row.related_table_id) {
    await fetchRelatedTableFields(row.related_table_id)
  }
  showRelationFormDialog.value = true
}

const handleRelationFormSubmit = async () => {
  if (!currentTable.value) return
  const referenceKey = relationForm.value.reference_key.trim()
  const foreignKey = relationForm.value.foreign_key.trim()
  const manyTableCode = relationForm.value.many_table_code.trim()
  if (!relationForm.value.related_table_id) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择关联表' })
    return
  }
  if (!referenceKey) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择主表字段' })
    return
  }
  if (!foreignKey) {
    $q.notify({ type: 'warning', position: 'top-right', message: '请选择关联表字段' })
    return
  }
  if (warnValidation(validateDBIdentifier('主表字段', referenceKey))) return
  if (warnValidation(validateDBIdentifier('关联表字段', foreignKey))) return
  if (relationForm.value.relation_type === SysTableRelationType.MANY_TO_MANY) {
    if (warnValidation(validateDBIdentifier('中间表编码', manyTableCode))) return
  } else if (warnValidation(validateDBIdentifier('中间表编码', manyTableCode, false))) {
    return
  }

  if (currentEditRelation.value?.id) {
    const req: TableRelationUpdateReq = {
      id: currentEditRelation.value.id,
      table_id: currentTable.value.id,
      related_table_id: relationForm.value.related_table_id,
      reference_key: referenceKey,
      foreign_key: foreignKey,
      relation_type: relationForm.value.relation_type,
      manyTableCode: manyTableCode || '',
    }
    const result = await tableApi.updateTableRelation(req)
    if (result.success) {
      showRelationFormDialog.value = false
      await fetchRelationsByTableId(currentTable.value.id)
    }
  } else {
    const req: TableRelationCreateReq = {
      table_id: currentTable.value.id,
      related_table_id: relationForm.value.related_table_id,
      reference_key: referenceKey,
      foreign_key: foreignKey,
      relation_type: relationForm.value.relation_type,
      manyTableCode: manyTableCode || '',
    }
    const result = await tableApi.createTableRelation(req)
    if (result.success) {
      showRelationFormDialog.value = false
      await fetchRelationsByTableId(currentTable.value.id)
    }
  }
}

const confirmDeleteRelation = (row: TableRelation) => {
  confirmDanger({
    message: '确定要删除该关联关系吗？',
  }).onOk(() => {
    void (async () => {
      const result = await tableApi.deleteTableRelation(row.id)
      if (result.success && currentTable.value?.id) {
        await fetchRelationsByTableId(currentTable.value.id)
      }
    })()
  })
}

const buildFieldUpdateReq = (field: TableField, overrides: Partial<TableFieldUpdateReq>) => {
  const toNumber = (val: any, fallback = 0) => {
    if (val === '' || val === null || val === undefined) return fallback
    const num = Number(val)
    return Number.isNaN(num) ? fallback : num
  }
  return {
    id: field.id,
    table_id: field.table_id,
    field_name: field.field_name,
    field_code: field.field_code,
    type: (field as any).type ?? field.field_type,
    field_length: toNumber(field.field_length, 0),
    field_decimal_length: toNumber(field.field_decimal_length, 0),
    input_type: field.input_type,
    default_value: field.default_value ?? '',
    dict_code: field.dict_code ?? '',
    is_primary_key: field.is_primary_key ?? false,
    is_index: field.is_index ?? false,
    is_quick_search: field.is_quick_search ?? false,
    is_advanced_search: field.is_advanced_search ?? false,
    is_sort: field.is_sort ?? false,
    is_null: field.is_null ?? true,
    is_list_show: field.is_list_show ?? true,
    is_insert_show: field.is_insert_show ?? true,
    is_update_show: field.is_update_show ?? true,
    sequence: toNumber(field.sequence, 0),
    original_field_id: toNumber(field.original_field_id, 0),
    binding: field.binding ?? '',
    field_category: field.field_category || SysTableFieldCategory.NORMAL,
    expression: field.expression ?? '',
    linkage_config: (field as any).linkage_config ?? '',
    ...overrides,
  }
}

const moveField = async (row: TableField, direction: -1 | 1) => {
  const sorted = [...fieldRows.value].sort((a, b) => (a.sequence ?? 0) - (b.sequence ?? 0))
  const index = sorted.findIndex((item) => item.id === row.id)
  const targetIndex = index + direction
  if (index === -1 || targetIndex < 0 || targetIndex >= sorted.length) return

  const current = sorted[index]
  const target = sorted[targetIndex]
  if (!current || !target) return
  const currentSeq = current.sequence ?? 0
  const targetSeq = target.sequence ?? 0

  const result1 = await tableApi.updateTableField(
    buildFieldUpdateReq(current, { sequence: targetSeq }),
  )
  const result2 = await tableApi.updateTableField(
    buildFieldUpdateReq(target, { sequence: currentSeq }),
  )

  if (result1.success && result2.success && currentTable.value?.id) {
    await fetchFieldsByTableId(currentTable.value.id)
  }
}

watch(
  () => relationForm.value.related_table_id,
  async (tableId) => {
    if (tableId) {
      await fetchRelatedTableFields(tableId)
    } else {
      relatedTableFields.value = []
    }
  },
)

watch(
  () => relationForm.value.relation_type,
  (relationType) => {
    if (relationType !== SysTableRelationType.MANY_TO_MANY) {
      relationForm.value.many_table_code = ''
    }
  },
)

// 监听分页变化（底部分页组件会改变 query.page/query.num）
watch(
  () => [query.value.page, query.value.num] as const,
  ([page]) => {
    if (!initialized.value) return
    pagination.value.page = page
    fetchData()
  },
)

// 监听排序变化（表头点击会改变 pagination.sortBy/descending）
watch(
  () => [pagination.value.sortBy, pagination.value.descending] as const,
  ([sortBy, descending], [prevSortBy, prevDescending]) => {
    if (!initialized.value) return
    if (sortBy === prevSortBy && descending === prevDescending) return

    // 同步排序到 query.order
    query.value.order = query.value.order ?? { field: '', is_asc: false }
    query.value.order.field = sortBy || ''
    query.value.order.is_asc = sortBy ? !descending : false

    // 排序变化时，自动回到第1页
    if (query.value.page !== 1) {
      query.value.page = 1
      // 回到第1页会触发上面的 watch，所以这里 return
      return
    }

    fetchData()
  },
)

// 监听高级查询对话框打开状态，打开时初始化临时查询
watch(
  () => showAdvancedQuery.value,
  (isOpen) => {
    if (isOpen) {
      initTempQuery()
    }
  },
)
</script>

<style scoped>
.table-structure-workbench {
  height: 100vh;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
  background: #ffffff;
}

.structure-header {
  min-height: 92px;
  padding: 20px 24px;
  border-bottom: 1px solid #e5ebf6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.structure-title-area {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 14px;
}

.structure-avatar {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  flex: 0 0 auto;
  background: linear-gradient(145deg, #715df2, #326df3);
  color: #ffffff;
  font-size: 20px;
  font-weight: 800;
}

.structure-title-copy {
  min-width: 0;
}

.structure-title {
  color: #172033;
  font-size: 24px;
  line-height: 1.2;
  font-weight: 800;
}

.structure-subtitle {
  margin-top: 7px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: #64728a;
  font-size: 14px;
}

.structure-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 0 0 auto;
}

.structure-code {
  max-width: 220px;
  height: 24px;
  padding: 0 9px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  background: #f1efff;
  color: #6957ed;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.structure-body {
  min-height: 0;
  padding: 0;
  display: grid;
  grid-template-columns: 236px minmax(540px, 1fr) 340px;
  background: #ffffff;
}

.structure-nav-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.structure-list-panel {
  min-width: 0;
  min-height: 0;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  border-right: 1px solid #e7edf7;
}

.structure-list-toolbar {
  min-height: 72px;
  padding: 15px 16px;
  border-bottom: 1px solid #edf1f8;
  display: flex;
  align-items: center;
  gap: 10px;
}

.structure-search {
  flex: 1;
}

.structure-table {
  min-height: 0;
  height: 100%;
}

.structure-table :deep(.q-table__middle) {
  height: 100%;
  overflow: auto;
}

.structure-table :deep(thead tr th) {
  position: sticky;
  z-index: 1;
  top: 0;
  height: 44px;
  background: #f8f9fd;
  border-bottom: 1px solid #dfe6f1;
  color: #243047;
  font-size: 14px;
  font-weight: 800;
}

.structure-table :deep(tbody td) {
  height: 58px;
  border-bottom: 1px solid #edf1f6;
  color: #2b3548;
  font-size: 14px;
}

.structure-row {
  cursor: pointer;
}

.structure-row.is-selected > td {
  background: #f5f3ff;
}

.structure-row.is-selected > td:first-child {
  box-shadow: inset 4px 0 0 #7162ee;
}

.structure-primary-text {
  max-width: 100%;
  color: #1f2937;
  font-size: 15px;
  font-weight: 800;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.structure-secondary-text {
  margin-top: 4px;
  max-width: 100%;
  color: #71809a;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.structure-tag-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
}

.structure-tag {
  min-height: 22px;
  padding: 2px 7px;
  border-radius: 5px;
  display: inline-flex;
  align-items: center;
  color: #53627a;
  background: #f0f3f8;
  font-size: 12px;
  line-height: 1.2;
  font-weight: 700;
}

.structure-tag.is-ok {
  color: #138547;
  background: #e7f7ed;
}

.structure-tag.is-blue {
  color: #2563eb;
  background: #eaf1ff;
}

.structure-tag.is-warn {
  color: #9a6500;
  background: #fff3d6;
}

.structure-tag.is-soft {
  color: #6957ed;
  background: #f0efff;
}

.structure-tag.is-muted {
  color: #68778e;
  background: #eef2f7;
}

.structure-row-actions {
  white-space: nowrap;
}

.structure-detail-panel {
  min-width: 0;
  min-height: 0;
  background: #ffffff;
}

.structure-detail,
.structure-empty {
  height: 100%;
  min-height: 0;
}

.structure-detail {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
}

.structure-detail-head {
  padding: 18px 16px 15px;
  border-bottom: 1px solid #edf1f8;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.structure-detail-title {
  color: #1f2937;
  font-size: 18px;
  line-height: 1.25;
  font-weight: 800;
}

.structure-status {
  height: 25px;
  padding: 0 9px;
  border-radius: 14px;
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  color: #148848;
  background: #e8f8ee;
  font-size: 13px;
  font-weight: 800;
}

.structure-status.is-off {
  color: #68778e;
  background: #eef2f7;
}

.structure-detail-scroll {
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.structure-section {
  margin-bottom: 12px;
  border: 1px solid #e6ebf4;
  border-radius: 8px;
  overflow: hidden;
}

.structure-section-title {
  min-height: 40px;
  padding: 0 12px;
  display: flex;
  align-items: center;
  background: #f9fbff;
  color: #253047;
  font-weight: 800;
}

.structure-info-grid {
  padding: 12px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.structure-info-grid > div,
.structure-info-list > div {
  min-width: 0;
}

.structure-info-grid span,
.structure-info-list span {
  display: block;
  margin-bottom: 6px;
  color: #7b879a;
  font-size: 12px;
}

.structure-info-grid strong,
.structure-info-list strong {
  min-height: 32px;
  padding: 0 9px;
  border: 1px solid #dce2ee;
  border-radius: 6px;
  display: flex;
  align-items: center;
  color: #1f2937;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.35;
  word-break: break-all;
}

.structure-switch-grid {
  padding: 12px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}

.structure-switch {
  min-height: 34px;
  border-radius: 7px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: #f3f6fb;
  color: #65728a;
  font-size: 13px;
  font-weight: 800;
}

.structure-switch.is-ok,
.structure-switch.is-blue,
.structure-switch.is-warn,
.structure-switch.is-soft {
  border: 1px solid #d9d5ff;
  background: #eef0ff;
  color: #6957ed;
}

.structure-switch.is-muted {
  color: #68778e;
  background: #eef2f7;
}

.structure-info-list {
  padding: 0 12px 12px;
  display: grid;
  gap: 10px;
}

.structure-index-fields {
  padding: 12px;
  display: grid;
  gap: 8px;
}

.structure-index-fields span {
  min-height: 32px;
  padding: 0 10px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  background: #f4f6fb;
  color: #273246;
  font-weight: 700;
}

.structure-detail-actions {
  padding: 12px 16px;
  border-top: 1px solid #edf1f8;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.structure-empty {
  display: grid;
  place-items: center;
  color: #8a95aa;
}

.metadata-simple-form {
  min-height: 0;
}

.metadata-simple-section {
  padding: 16px;
  border: 1px solid #e4ebf7;
  border-radius: 8px;
  background: #ffffff;
}

.metadata-simple-section__head {
  margin-bottom: 14px;
}

.metadata-simple-section__title {
  color: #172033;
  font-size: 16px;
  line-height: 1.3;
  font-weight: 800;
}

.metadata-simple-section__desc {
  margin-top: 4px;
  color: #8290a6;
  font-size: 12px;
}

.metadata-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.metadata-form-wide {
  grid-column: 1 / -1;
}

.metadata-toggle-field {
  min-height: 40px;
  padding: 6px 8px 6px 12px;
  border: 1px solid #dce4f2;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  background: #fbfcff;
}

.metadata-toggle-field span {
  color: #26334d;
  font-size: 13px;
  font-weight: 800;
}

.metadata-form-preview {
  display: grid;
  gap: 8px;
}

.metadata-form-preview__title {
  margin-bottom: 6px;
  color: #172033;
  font-size: 15px;
  font-weight: 800;
}

.metadata-form-preview__row {
  min-width: 0;
  padding: 10px;
  border: 1px solid #e4ebf7;
  border-radius: 8px;
  background: #f8faff;
}

.metadata-form-preview__row span {
  display: block;
  margin-bottom: 5px;
  color: #8290a6;
  font-size: 12px;
}

.metadata-form-preview__row strong {
  display: block;
  color: #172033;
  font-size: 13px;
  line-height: 1.45;
  word-break: break-word;
}

@media (max-width: 1280px) {
  .structure-body {
    grid-template-columns: 210px minmax(420px, 1fr) 300px;
  }

  .structure-header {
    padding: 18px;
  }

  .structure-actions {
    gap: 8px;
  }
}
</style>
