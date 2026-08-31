<template>
  <div class="key-value-editor">
    <q-card flat bordered class="q-pa-sm">
      <q-card-section class="q-py-xs">
        <div class="row items-center">
          <div class="text-subtitle1">{{ displayLabel }}</div>
          <q-space />
          <q-btn
            flat
            round
            dense
            color="primary"
            icon="add"
            @click="addKeyValue"
            :disable="disabled || loading"
          />
        </div>
      </q-card-section>
      <q-separator />
      <q-card-section>
        <div v-if="entries.length === 0" class="text-center q-py-md text-grey">
          {{ t('ui.noDataAvailableClickOnTheTopPlusSignTo') }}
        </div>
        <div
          v-for="(entry, index) in entries"
          :key="`entry-${index}`"
          class="row items-center q-mb-sm"
        >
          <q-input
            v-model="entries[index]!.key"
            outlined
            dense
            class="col-5"
            :label="t('ui.keys')"
            :disable="disabled"
            @blur="handleBlur"
          />
          <q-input
            v-model="entries[index]!.value"
            outlined
            dense
            class="col-5 q-mx-sm"
            :label="t('ui.value')"
            :disable="disabled"
            @blur="handleBlur"
          />
          <div class="col-1">
            <q-btn
              flat
              round
              dense
              color="negative"
              icon="remove"
              @click="removeEntry(index)"
              :disable="disabled || loading"
            />
          </div>
        </div>
      </q-card-section>
    </q-card>
    <div v-if="errorMessage" class="text-negative text-caption q-mt-xs">{{ errorMessage }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { computed, ref, watch, onMounted } from 'vue'

const { t } = useI18n({ useScope: 'global' })

// 键值对条目接口
interface KeyValueEntry {
  key: string
  value: string
}

interface KeyValueEditorProps {
  modelValue?: Record<string, any> | string | null
  label?: string
  rules?: Array<(value: any) => boolean | string>
  disabled?: boolean
}

const props = withDefaults(defineProps<KeyValueEditorProps>(), {
  modelValue: () => ({}),
  label: '',
  rules: () => [],
  disabled: false,
})

const displayLabel = computed(() => props.label || t('ui.keyPair'))

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>]
}>()

const entries = ref<KeyValueEntry[]>([])
const loading = ref(false)
const errorMessage = ref('')
const isInternalUpdate = ref(false)

// 初始化键值对数组
const initEntries = () => {
  if (isInternalUpdate.value) return

  try {
    let dataObject: Record<string, any> = {}

    if (!props.modelValue) {
      dataObject = {}
    } else if (typeof props.modelValue === 'string') {
      try {
        // 尝试解析JSON字符串
        const parsed = JSON.parse(props.modelValue)
        dataObject = typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {}
      } catch (e) {
        console.warn('Failed to parse JSON string:', e)
        dataObject = {}
      }
    } else {
      dataObject = { ...props.modelValue }
    }

    // 转换为数组形式
    entries.value = Object.entries(dataObject).map(([key, value]) => ({
      key,
      value: typeof value === 'string' ? value : JSON.stringify(value),
    }))

    // 如果为空，添加一个空条目
    if (entries.value.length === 0) {
      entries.value.push({ key: '', value: '' })
    }
  } catch (e) {
    console.error('Error initializing key-value editor:', e)
    entries.value = [{ key: '', value: '' }]
  }
}

// 发送更新事件，可以强制执行
const emitUpdate = (force = false) => {
  // 如果设置了内部更新标志且不是强制的，则跳过
  if (isInternalUpdate.value && !force) return

  // 标记为内部更新
  isInternalUpdate.value = true

  // 创建一个新对象，仅包含有效的键值对
  const result: Record<string, any> = {}

  entries.value.forEach((entry) => {
    // 只有键不为空时才添加
    if (entry.key.trim()) {
      // 尝试解析值是否为JSON
      try {
        result[entry.key] = JSON.parse(entry.value)
      } catch {
        // 非JSON则作为字符串保存
        result[entry.key] = entry.value
      }
    }
  })

  // 发送更新
  emit('update:modelValue', Object.keys(result).length ? result : {})

  // 延迟恢复非更新状态
  setTimeout(() => {
    isInternalUpdate.value = false
  }, 100)
}

// 添加键值对
const addKeyValue = () => {
  if (props.disabled || loading.value) return

  loading.value = true
  isInternalUpdate.value = true

  entries.value.push({ key: '', value: '' })

  // 不立即触发更新，等用户输入后再更新
  setTimeout(() => {
    loading.value = false
    setTimeout(() => {
      isInternalUpdate.value = false
    }, 50)
  }, 100)
}

// 删除条目
const removeEntry = (index: number) => {
  if (props.disabled || loading.value) return

  loading.value = true
  isInternalUpdate.value = true

  // 确保至少有一个条目
  if (entries.value.length === 1) {
    entries.value = [{ key: '', value: '' }]
  } else {
    entries.value.splice(index, 1)
  }

  // 直接发送更新
  emitUpdate(true)

  // 延迟恢复状态
  setTimeout(() => {
    loading.value = false
    setTimeout(() => {
      isInternalUpdate.value = false
    }, 50)
  }, 100)
}

// 处理输入框失焦事件
const handleBlur = () => {
  if (!isInternalUpdate.value && !loading.value) {
    emitUpdate(true)
  }
}

// 使用防抖的监听
let updateTimer: number | null = null
watch(
  () => entries.value,
  () => {
    // 防止循环更新
    if (isInternalUpdate.value || loading.value) return

    if (updateTimer) clearTimeout(updateTimer)
    updateTimer = window.setTimeout(() => {
      if (!isInternalUpdate.value) {
        emitUpdate()
      }
    }, 300)
  },
  { deep: true },
)

// 监听外部值变化
watch(
  () => props.modelValue,
  () => {
    // 如果是内部更新导致的变化，不再重新初始化
    if (!isInternalUpdate.value && !loading.value) {
      initEntries()
    }
  },
  { deep: true },
)

// 组件挂载时初始化
onMounted(() => {
  initEntries()
})

const currentValue = () => {
  const result: Record<string, any> = {}
  for (const entry of entries.value) {
    if (entry.key.trim()) result[entry.key] = entry.value
  }
  return result
}

const validate = () => {
  errorMessage.value = ''
  for (const rule of props.rules) {
    const result = rule(currentValue())
    if (result !== true) {
      errorMessage.value = typeof result === 'string' ? result : t('ui.invalidFieldValue')
      return false
    }
  }
  return true
}

const resetValidation = () => {
  errorMessage.value = ''
}

defineExpose({ validate, resetValidation })
</script>

<style scoped>
.key-value-editor {
  width: 100%;
}

/* 添加一些键值对条目动画效果 */
.row-enter-active,
.row-leave-active {
  transition: all 0.3s;
}

.row-enter-from,
.row-leave-to {
  opacity: 0;
  transform: translateX(15px);
}
</style>
