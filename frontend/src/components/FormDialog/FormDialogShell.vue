<template>
  <div v-if="embedded && show" class="form-dialog-shell form-dialog-shell--embedded">
    <q-card-section class="form-dialog-shell__header">
      <div class="form-dialog-shell__mark">
        <q-icon :name="icon" size="22px" />
      </div>
      <div class="form-dialog-shell__title-area">
        <div class="form-dialog-shell__title-line">
          <div class="form-dialog-shell__title">{{ title }}</div>
          <slot name="title-extra" />
        </div>
        <div v-if="subtitle" class="form-dialog-shell__subtitle">{{ subtitle }}</div>
      </div>
      <q-space />
      <slot name="header-actions" />
      <q-btn icon="close" flat round dense class="form-dialog-shell__close" @click="show = false" />
    </q-card-section>

    <q-card-section class="form-dialog-shell__body" :class="{ 'has-preview': hasPreview }">
      <main class="form-dialog-shell__main">
        <slot />
      </main>
      <aside v-if="hasPreview" class="form-dialog-shell__preview">
        <slot name="preview" />
      </aside>
    </q-card-section>

    <div class="form-dialog-shell__footer">
      <div class="form-dialog-shell__footer-left">
        <slot name="footer-left" />
        <div class="form-dialog-shell__status">
          <slot name="footer-status" />
        </div>
      </div>
      <q-space />
      <q-btn
        :label="readonly ? '关闭' : cancelText"
        color="grey-7"
        :disable="loading"
        flat
        @click="show = false"
      />
      <q-btn
        v-if="!readonly && showSubmit"
        :label="submitText"
        color="primary"
        unelevated
        :loading="loading"
        class="form-dialog-shell__submit"
        @click="emit('submit')"
      >
        <template v-slot:loading>
          <q-spinner-dots />
        </template>
      </q-btn>
    </div>
  </div>

  <q-dialog v-else v-model="show" persistent transition-show="slide-up" transition-hide="slide-down">
    <q-card class="form-dialog-shell" :style="{ width, maxWidth: width, maxHeight }">
      <q-card-section class="form-dialog-shell__header">
        <div class="form-dialog-shell__mark">
          <q-icon :name="icon" size="22px" />
        </div>
        <div class="form-dialog-shell__title-area">
          <div class="form-dialog-shell__title-line">
            <div class="form-dialog-shell__title">{{ title }}</div>
            <slot name="title-extra" />
          </div>
          <div v-if="subtitle" class="form-dialog-shell__subtitle">{{ subtitle }}</div>
        </div>
        <q-space />
        <slot name="header-actions" />
        <q-btn icon="close" flat round dense class="form-dialog-shell__close" v-close-popup />
      </q-card-section>

      <q-card-section class="form-dialog-shell__body" :class="{ 'has-preview': hasPreview }">
        <main class="form-dialog-shell__main">
          <slot />
        </main>
        <aside v-if="hasPreview" class="form-dialog-shell__preview">
          <slot name="preview" />
        </aside>
      </q-card-section>

      <div class="form-dialog-shell__footer">
        <div class="form-dialog-shell__footer-left">
          <slot name="footer-left" />
          <div class="form-dialog-shell__status">
            <slot name="footer-status" />
          </div>
        </div>
        <q-space />
        <q-btn
          :label="readonly ? '关闭' : cancelText"
          color="grey-7"
          :disable="loading"
          flat
          v-close-popup
        />
        <q-btn
          v-if="!readonly && showSubmit"
          :label="submitText"
          color="primary"
          unelevated
          :loading="loading"
          class="form-dialog-shell__submit"
          @click="emit('submit')"
        >
          <template v-slot:loading>
            <q-spinner-dots />
          </template>
        </q-btn>
      </div>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, useSlots } from 'vue'

defineOptions({ name: 'FormDialogShell' })

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    subtitle?: string
    icon?: string
    submitText?: string
    cancelText?: string
    loading?: boolean
    readonly?: boolean
    showSubmit?: boolean
    showPreview?: boolean
    embedded?: boolean
    width?: string
    maxHeight?: string
  }>(),
  {
    subtitle: '',
    icon: 'edit_note',
    submitText: '保存',
    cancelText: '取消',
    loading: false,
    readonly: false,
    showSubmit: true,
    showPreview: true,
    embedded: false,
    width: 'min(1120px, calc(100vw - 48px))',
    maxHeight: 'min(88vh, 860px)',
  },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (event: 'submit'): void
}>()

const slots = useSlots()
const hasPreview = computed(() => props.showPreview && Boolean(slots.preview))

const show = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value),
})
</script>

<style scoped lang="scss">
.form-dialog-shell {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  border-radius: 9px;
  overflow: hidden;
  box-shadow: 0 18px 54px rgba(15, 23, 42, 0.2);
}

.form-dialog-shell--embedded {
  min-height: calc(100vh - 142px);
  max-height: calc(100vh - 142px);
  border: 1px solid rgba(115, 103, 240, 0.18);
  box-shadow: none;
  background: #fff;
}

.form-dialog-shell__header {
  min-height: 66px;
  padding: 14px 18px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
}

.form-dialog-shell__mark {
  width: 38px;
  height: 38px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #fff;
  background: linear-gradient(135deg, $primary, #3768f0);
  box-shadow: 0 10px 22px rgba($primary, 0.24);
}

.form-dialog-shell__title-area {
  min-width: 0;
}

.form-dialog-shell__title-line {
  display: flex;
  align-items: center;
  gap: 10px;
}

.form-dialog-shell__title {
  font-size: 19px;
  line-height: 1.2;
  font-weight: 800;
  color: #172033;
}

.form-dialog-shell__subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: #748098;
}

.form-dialog-shell__close {
  width: 34px;
  height: 34px;
  color: #111827;
  background: #f8fafc;
  border: 1px solid rgba(15, 23, 42, 0.1);
}

.form-dialog-shell__body {
  min-height: 0;
  padding: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  background: #f8faff;
}

.form-dialog-shell__body.has-preview {
  grid-template-columns: minmax(0, 1fr) 280px;
}

.form-dialog-shell__main {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.form-dialog-shell__preview {
  min-width: 0;
  overflow: auto;
  padding: 18px;
  border-left: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
}

.form-dialog-shell__footer {
  min-height: 62px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 18px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
  background: #fff;
}

.form-dialog-shell__footer-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.form-dialog-shell__status {
  font-size: 13px;
  color: #748098;
}

.form-dialog-shell__submit {
  min-width: 90px;
  height: 36px;
  border-radius: 7px;
  box-shadow: 0 8px 18px rgba($primary, 0.28);
}

@media (max-width: 900px) {
  .form-dialog-shell {
    width: calc(100vw - 24px) !important;
    max-height: calc(100vh - 24px) !important;
  }

  .form-dialog-shell__body.has-preview {
    grid-template-columns: 1fr;
  }

  .form-dialog-shell__preview {
    display: none;
  }
}
</style>
