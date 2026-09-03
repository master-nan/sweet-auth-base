import { defineStore } from 'pinia'
import { useDictApi } from '@/api/services/sys-dict'

interface DictOption {
  label: string
  value: string | number
  [key: string]: any
}

interface DictState {
  dictCache: Record<string, DictOption[]>
  loading: Record<string, boolean>
}

export const useDictStore = defineStore('dict', {
  state: (): DictState => ({
    dictCache: {},
    loading: {}
  }),

  getters: {
    /**
     * 获取指定字典的选项
     * @param state 状态
     * @returns 函数，接收字典代码，返回字典选项数组
     */
    getDictOptions: (state) => (dictCode: string): DictOption[] => {
      return state.dictCache[dictCode] || []
    },

    /**
     * 获取字典项的文本（根据值查找标签）
     * @param state 状态
     * @returns 函数，接收字典代码和值，返回对应的标签
     */
    getDictLabel: (state) => (dictCode: string, value: any): string => {
      const options = state.dictCache[dictCode] || []
      // 字典 item_value 始终为字符串，传入的 value 可能是数字/布尔，用 String() 统一比较
      const strValue = String(value)
      const found = options.find(item =>
        String(item.value) === strValue ||
        String(item.dict_value) === strValue ||
        String(item.item_code) === strValue
      )
      return found ? (found.label || found.dict_label || '') : ''
    },

    /**
     * 判断字典是否已加载
     * @param state 状态
     * @returns 函数，接收字典代码，返回是否已加载
     */
    isDictLoaded: (state) => (dictCode: string): boolean => {
      return !!state.dictCache[dictCode]
    },

    /**
     * 判断字典是否正在加载
     * @param state 状态
     * @returns 函数，接收字典代码，返回是否正在加载
     */
    isDictLoading: (state) => (dictCode: string): boolean => {
      return !!state.loading[dictCode]
    },
  },

  actions: {
    /**
     * 加载字典数据
     * @param dictCode 字典代码
     * @returns Promise，加载成功返回字典选项数组
     */
    async loadDict(dictCode: string): Promise<DictOption[]> {
      // 如果已经在缓存中存在，直接返回
      if (this.dictCache[dictCode]) {
        return this.dictCache[dictCode]
      }

      // 如果正在加载，等待加载完成
      if (this.loading[dictCode]) {
        return new Promise((resolve) => {
          const checkCache = () => {
            if (this.dictCache[dictCode]) {
              resolve(this.dictCache[dictCode])
            } else if (!this.loading[dictCode]) {
              resolve([])
            } else {
              setTimeout(checkCache, 50)
            }
          }
          setTimeout(checkCache, 50)
        })
      }

      // 开始加载
      this.loading[dictCode] = true

      try {
        const dictApi = useDictApi()
        const result = await dictApi.queryRuntimeDictByCode(dictCode)

        if (result.data && result.data.dict_items) {
          const formattedOptions = result.data.dict_items.map((item) => ({
            label: item.item_name,
            value: item.item_value,
            ...item // 保留原有属性
          }))

          // 存入缓存
          this.dictCache[dictCode] = formattedOptions
          return formattedOptions
        }

        // 如果没有数据，设置为空数组
        this.dictCache[dictCode] = []
        return []
      } catch (error) {
        console.error(`加载字典数据失败: ${dictCode}`, error)
        this.dictCache[dictCode] = []
        return []
      } finally {
        // 无论成功失败，都标记加载完成
        this.loading[dictCode] = false
      }
    },

    /**
     * 批量加载多个字典
     * @param dictCodes 字典代码数组
     */
    async loadDicts(dictCodes: string[]): Promise<void> {
      if (!dictCodes || dictCodes.length === 0) return

      const promises = dictCodes.map(code => this.loadDict(code))
      await Promise.all(promises)
    },

    /**
     * 刷新字典数据（强制重新加载）
     * @param dictCode 字典代码
     */
    async refreshDict(dictCode: string): Promise<DictOption[]> {
      // 移除缓存
      delete this.dictCache[dictCode]
      return this.loadDict(dictCode)
    },

    /**
     * 清除特定字典缓存或所有字典缓存
     * @param dictCode 可选的字典代码，不传则清除所有缓存
     */
    clearDict(dictCode?: string) {
      if (dictCode) {
        delete this.dictCache[dictCode]
      } else {
        this.dictCache = {}
      }
    }
  }
})
