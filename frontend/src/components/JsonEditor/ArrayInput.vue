<template>
  <div class="array-input">
    <q-card flat bordered class="q-pa-sm">
      <q-card-section class="q-py-xs">
        <div class="text-subtitle1">{{ displayLabel }}</div>
      </q-card-section>

      <q-separator />

      <q-card-section>
        <!-- 已添加的参数显示为 chips -->
        <div class="params-chips q-mb-md">
          <q-chip
            v-for="(item, index) in filteredItems"
            :key="`chip-${index}`"
            :removable="!disabled"
            @remove="removeItem(index)"
            color="primary"
            text-color="white"
            size="md"
            class="q-ma-xs"
          >
            {{ item }}
          </q-chip>

          <div v-if="filteredItems.length === 0" class="text-grey q-px-sm q-py-xs">
            {{ t('ui.noParametersAddBelow') }}
          </div>
        </div>

        <!-- 添加新参数区域 -->
        <div class="row q-col-gutter-sm">
          <div class="col">
            <q-input
              v-model="newItem"
              outlined
              dense
              :placeholder="t('ui.enterNamedItem', { label: displayLabel })"
              :disable="disabled"
              @keyup.enter="addItemFromInput"
              @blur="handleInputBlur"
            >
              <template v-slot:append>
                <q-btn
                  round
                  flat
                  dense
                  color="primary"
                  icon="add"
                  @click="addItemFromInput"
                  :disable="disabled || loading || !newItem.trim()"
                />
              </template>
            </q-input>
          </div>
        </div>
      </q-card-section>
    </q-card>
    <div v-if="errorMessage" class="text-negative text-caption q-mt-xs">{{ errorMessage }}</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

import { ref, watch, onMounted, computed } from 'vue'

const { t } = useI18n({ useScope: 'global' })

interface ArrayInputProps {
  modelValue?: string[] | string | null
  label?: string
  rules?: Array<(value: any) => boolean | string>
  disabled?: boolean
}

const props = withDefaults(defineProps<ArrayInputProps>(), {
  modelValue: () => [],
  label: '',
  rules: () => [],
  disabled: false,
})

const displayLabel = computed(() => props.label || t('ui.numericData'))

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

// 内部数组，避免直接操作响应式prop
const internalItems = ref<string[]>([])
// 新输入项
const newItem = ref('')
// 用于防止高频操作
const loading = ref(false)
const errorMessage = ref('')
// 防止循环更新
const isInternalUpdate = ref(false)

// 过滤掉空项目的计算属性
const filteredItems = computed(() => {
  return internalItems.value.filter((item) => item.trim() !== '')
})

// 初始化内部数组
const initInternalItems = () => {
  if (isInternalUpdate.value) return

  let resultArray: string[] = []

  try {
    if (props.modelValue === null || props.modelValue === undefined) {
      resultArray = [] // 空数组
    } else if (typeof props.modelValue === 'string') {
      if (!props.modelValue.trim()) {
        resultArray = [] // 空字符串情况
      } else {
        try {
          // 尝试解析为JSON
          const parsed = JSON.parse(props.modelValue)
          resultArray = Array.isArray(parsed) ? parsed.map(String) : []
        } catch {
          // 不是JSON，尝试以逗号分隔
          resultArray = props.modelValue
            .split(',')
            .map((item) => item.trim())
            .filter(Boolean)
        }
      }
    } else if (Array.isArray(props.modelValue)) {
      resultArray = props.modelValue.map(String).filter(Boolean)
    }

    internalItems.value = resultArray
  } catch (e) {
    console.error('Error initializing array input:', e)
    internalItems.value = [] // 发生错误时提供空数组
  }
}

// 安全发射更新事件，可以强制执行
const emitUpdate = (force = false) => {
  // 如果设置了内部更新标志且不是强制的，则跳过
  if (isInternalUpdate.value && !force) return

  // 发送更新
  emit('update:modelValue', [...filteredItems.value])
}

// 从输入框添加项目
const addItemFromInput = () => {
  if (props.disabled || loading.value || !newItem.value.trim()) return

  loading.value = true
  isInternalUpdate.value = true

  // 添加新项目
  if (newItem.value.trim()) {
    internalItems.value.push(newItem.value.trim())
    // 清空输入
    newItem.value = ''

    // 直接发送事件
    emitUpdate(true)
  }

  setTimeout(() => {
    loading.value = false
    setTimeout(() => {
      isInternalUpdate.value = false
    }, 50)
  }, 100)
}

// 删除项目
const removeItem = (index: number) => {
  if (props.disabled || loading.value) return

  loading.value = true
  isInternalUpdate.value = true

  // 删除项目
  internalItems.value.splice(index, 1)

  // 直接发送事件
  emitUpdate(true)

  setTimeout(() => {
    loading.value = false
    setTimeout(() => {
      isInternalUpdate.value = false
    }, 50)
  }, 100)
}

// 处理输入框失焦
const handleInputBlur = () => {
  if (!loading.value && !isInternalUpdate.value && newItem.value.trim()) {
    addItemFromInput()
  }
}

// 监听内部数组变化
let updateTimer: number | null = null
watch(
  () => filteredItems.value,
  () => {
    // 防止无限循环
    if (isInternalUpdate.value || loading.value) return

    // 使用防抖
    if (updateTimer) clearTimeout(updateTimer)
    updateTimer = window.setTimeout(() => {
      emitUpdate()
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
      initInternalItems()
    }
  },
  { deep: true },
)

// 组件挂载时初始化
onMounted(() => {
  initInternalItems()
})

const validate = () => {
  errorMessage.value = ''
  for (const rule of props.rules) {
    const result = rule(filteredItems.value)
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
.array-input {
  width: 100%;
}

.params-chips {
  display: flex;
  flex-wrap: wrap;
  min-height: 42px;
  border-radius: 4px;
  padding: 4px;
  background-color: rgba(0, 0, 0, 0.02);
}
</style>
