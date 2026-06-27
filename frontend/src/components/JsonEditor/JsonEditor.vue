<template>
  <div class="json-editor">
    <q-input
      v-model="jsonText"
      type="textarea"
      outlined
      :label="label"
      rows="8"
      class="full-width"
      :error="!!error"
      :error-message="error"
      @blur="validateJson"
    />
    <div v-if="!error && jsonText" class="q-mt-xs text-caption text-green">有效的JSON格式</div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

// TypeScript 类型方式定义 props
interface JsonEditorProps {
  modelValue?: string | object | null
  label?: string
}

const props = withDefaults(defineProps<JsonEditorProps>(), {
  modelValue: null,
  label: 'JSON数据',
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const jsonText = ref('')
const error = ref('')
const loading = ref(false)
const isInternalUpdate = ref(false)

// 初始化JSON文本
const initJsonText = () => {
  if (isInternalUpdate.value) return

  try {
    if (props.modelValue === null || props.modelValue === undefined) {
      jsonText.value = ''
      return
    }

    if (typeof props.modelValue === 'string') {
      if (!props.modelValue.trim()) {
        jsonText.value = ''
        return
      }

      try {
        // 尝试解析字符串为JSON并格式化
        const parsed = JSON.parse(props.modelValue)
        jsonText.value = JSON.stringify(parsed, null, 2)
      } catch {
        // 不是有效JSON，保持原样
        jsonText.value = props.modelValue
      }
    } else {
      // 对象或数组直接转换为格式化的JSON字符串
      jsonText.value = JSON.stringify(props.modelValue, null, 2)
    }

    error.value = ''
  } catch (e) {
    console.error('Error initializing JSON editor:', e)
    error.value = '无法解析JSON数据'
    jsonText.value =
      typeof props.modelValue === 'string'
        ? props.modelValue
        : JSON.stringify(props.modelValue || {})
  }
}

// 验证JSON并更新
const validateJson = () => {
  if (isInternalUpdate.value) return

  isInternalUpdate.value = true

  if (!jsonText.value.trim()) {
    emit('update:modelValue', null)
    error.value = ''

    setTimeout(() => {
      isInternalUpdate.value = false
    }, 100)
    return
  }

  try {
    const parsed = JSON.parse(jsonText.value)
    // 只接受对象或数组，裸数字/字符串/布尔值不算有效的JSON配置
    if (typeof parsed !== 'object' || parsed === null) {
      error.value = '请输入JSON对象 {} 或数组 []'
      return
    }
    emit('update:modelValue', parsed)
    error.value = ''
  } catch (e) {
    console.error('Error parsing JSON:', e)
    error.value = '无效的JSON格式'
  }

  setTimeout(() => {
    isInternalUpdate.value = false
  }, 100)
}

// 监听外部值变化
watch(
  () => props.modelValue,
  () => {
    if (!isInternalUpdate.value && !loading.value) {
      loading.value = true
      initJsonText()
      setTimeout(() => {
        loading.value = false
      }, 100)
    }
  },
  { deep: true },
)

// 组件挂载时初始化
onMounted(() => {
  initJsonText()
})
</script>

<style scoped>
.json-editor {
  width: 100%;
}
</style>
