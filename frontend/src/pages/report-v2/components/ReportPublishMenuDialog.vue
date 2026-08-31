<template>
  <q-dialog
    :model-value="modelValue"
    persistent
    @hide="clearState"
    @update:model-value="handleDialogValue"
  >
    <q-card class="publish-menu-dialog">
      <q-card-section class="dialog-head">
        <div>
          <div class="dialog-title">{{ t('ui.releaseToMenu') }}</div>
          <div class="dialog-caption">
            {{ t('ui.loadsPublishedReportsToTheLeftMenuAndBusinessUsers') }}
          </div>
        </div>
      </q-card-section>

      <q-form ref="formRef" @submit.prevent="submit">
        <q-card-section class="dialog-body">
          <q-banner rounded class="q-mb-md bg-blue-1 text-blue-10">
            <template #avatar>
              <q-icon name="info" color="primary" />
            </template>
            {{ t('ui.theFirstStageIsTheDefaultAuthorizationOfSuperAdmin') }}
          </q-banner>

          <div class="row q-col-gutter-md">
            <div class="col-12">
              <q-select
                v-model="parentMenuId"
                outlined
                dense
                emit-value
                map-options
                use-input
                input-debounce="0"
                :label="t('ui.parentMenu')"
                :options="filteredParentOptions"
                :loading="menuLoading"
                :rules="[(val) => !!val || t('ui.pleaseSelectAParentMenu')]"
                @filter="filterParentOptions"
              >
                <template #no-option>
                  <q-item>
                    <q-item-section class="text-grey-6">
                      {{
                        menuLoading ? t('ui.loadingMenuDirectory') : t('ui.temporaryDirectoryMenu')
                      }}
                    </q-item-section>
                  </q-item>
                </template>
              </q-select>
            </div>

            <div class="col-12 col-md-6">
              <q-input
                v-model.trim="title"
                outlined
                dense
                :label="t('ui.menuName')"
                :rules="[(val) => !!val || t('ui.pleaseEnterAMenuName')]"
              />
            </div>

            <div class="col-12 col-md-6">
              <q-input
                v-model.trim="icon"
                outlined
                dense
                :label="t('ui.icon')"
                placeholder="assessment"
              />
            </div>

            <div class="col-12">
              <q-input
                v-model.trim="path"
                outlined
                dense
                :label="t('ui.runPath')"
                :rules="[(val) => !!val || t('ui.pleaseEnterTheRunningPath')]"
              >
                <template #hint> {{ t('ui.defaultFormatReportRuntime') }} </template>
              </q-input>
            </div>

            <div class="col-12 col-md-6">
              <q-input v-model.number="sort" outlined dense type="number" :label="t('ui.sort')" />
            </div>

            <div class="col-12 col-md-6 flex items-center">
              <q-toggle v-model="visible" :label="t('ui.showToMenu')" />
            </div>
          </div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn
            flat
            :label="t('ui.cancel')"
            :disable="submitting"
            @click="handleDialogValue(false)"
          />
          <q-btn
            unelevated
            color="primary"
            type="submit"
            :label="t('ui.confirmRelease')"
            :loading="submitting"
          />
        </q-card-actions>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { ref, watch } from 'vue'
import { useQuasar, type QForm } from 'quasar'
import type { Query } from 'src/types/global'
import { useMenuApi, type Menu } from 'src/api/services/sys-menu'
import { useReportApi, type Report } from 'src/api/services/report'

const { t } = useI18n({ useScope: 'global' })

type ParentMenuOption = {
  label: string
  value: number
  caption?: string
}

const props = defineProps<{
  modelValue: boolean
  report: Report | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: []
}>()

const $q = useQuasar()
const menuApi = useMenuApi()
const reportApi = useReportApi()

const formRef = ref<QForm | null>(null)
const parentMenuId = ref<number | null>(null)
const title = ref('')
const path = ref('')
const icon = ref('assessment')
const sort = ref(30)
const visible = ref(true)
const menuLoading = ref(false)
const submitting = ref(false)
const parentOptions = ref<ParentMenuOption[]>([])
const filteredParentOptions = ref<ParentMenuOption[]>([])

watch(
  () => props.modelValue,
  (opened) => {
    if (opened) {
      resetForm()
      void loadParentMenus()
    } else {
      clearState()
    }
  },
)

function handleDialogValue(value: boolean) {
  emit('update:modelValue', value)
  if (!value) clearState()
}

function resetForm() {
  const reportCode = props.report?.report_code || props.report?.code || ''
  title.value = props.report?.report_name || props.report?.name || ''
  path.value = reportCode ? `report/runtime/${reportCode}` : ''
  icon.value = 'assessment'
  sort.value = 30
  visible.value = true
  parentMenuId.value = null
}

function clearState() {
  submitting.value = false
  menuLoading.value = false
  parentMenuId.value = null
  parentOptions.value = []
  filteredParentOptions.value = []
  title.value = ''
  path.value = ''
  icon.value = 'assessment'
  sort.value = 30
  visible.value = true
}

function parentMenuQuery(): Query {
  return {
    page: 1,
    num: 1000,
    order: { field: 'sequence', is_asc: true },
    expressions: [],
    quick_query: { keyword: '' },
    include_deleted: false,
  }
}

function menuTitle(menu: Menu) {
  return menu.title || menu.name || menu.path || String(menu.id)
}

function isDirectoryMenu(menu: Menu) {
  if (!menu.state || menu.is_hidden) return false
  if (menu.page_type) return menu.page_type === 'directory'
  return !menu.table_code && Boolean(menu.children?.length)
}

function collectParentOptions(menus: Menu[], level = 0): ParentMenuOption[] {
  const options: ParentMenuOption[] = []
  menus.forEach((menu) => {
    if (isDirectoryMenu(menu)) {
      const prefix = level > 0 ? `${'　'.repeat(level)}└ ` : ''
      options.push({
        label: `${prefix}${menuTitle(menu)}`,
        value: menu.id,
        caption: menu.path,
      })
    }
    if (menu.children?.length) {
      options.push(...collectParentOptions(menu.children, level + 1))
    }
  })
  return options
}

async function loadParentMenus() {
  menuLoading.value = true
  try {
    const res = await menuApi.queryMenu(parentMenuQuery())
    const options = res.success && Array.isArray(res.data) ? collectParentOptions(res.data) : []
    parentOptions.value = options
    filteredParentOptions.value = options
    parentMenuId.value = options[0]?.value || null
  } catch (error) {
    parentOptions.value = []
    filteredParentOptions.value = []
    $q.notify({
      type: 'negative',
      position: 'top-right',
      get message() {
        return error instanceof Error && error.message
          ? error.message
          : t('ui.fatherMenuLoadedFailed')
      },
    })
  } finally {
    menuLoading.value = false
  }
}

function filterParentOptions(value: string, update: (callback: () => void) => void) {
  update(() => {
    const keyword = value.trim().toLowerCase()
    filteredParentOptions.value = keyword
      ? parentOptions.value.filter((option) =>
          [option.label, option.caption].join(' ').toLowerCase().includes(keyword),
        )
      : parentOptions.value
  })
}

async function submit() {
  if (!props.report?.id) return
  const valid = await formRef.value?.validate()
  if (!valid || !parentMenuId.value) return

  submitting.value = true
  try {
    await reportApi.publishReportMenu(props.report.id, {
      parent_menu_id: parentMenuId.value,
      title: title.value,
      path: path.value,
      icon: icon.value || 'assessment',
      sort: Number(sort.value || 0),
      visible: visible.value,
    })
    $q.notify({
      type: 'positive',
      position: 'top-right',
      get message() {
        return t('ui.reportPublishedToMenu')
      },
    })
    emit('success')
    handleDialogValue(false)
  } catch (error) {
    $q.notify({
      type: 'negative',
      position: 'top-right',
      get message() {
        return error instanceof Error && error.message
          ? error.message
          : t('ui.failedToPublishToMenu')
      },
    })
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped lang="scss">
.publish-menu-dialog {
  width: min(720px, 94vw);
  max-height: min(760px, 88vh);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dialog-head {
  flex: 0 0 auto;
  padding: 24px 28px;
  border-bottom: 1px solid #e7ecf6;
}

.dialog-title {
  font-size: 20px;
  font-weight: 900;
}

.dialog-caption {
  margin-top: 4px;
  color: #667085;
}

.dialog-body {
  padding: 20px 28px;
}
</style>
