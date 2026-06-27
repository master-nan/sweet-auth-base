import { ref, readonly } from 'vue'
import { useDictApi } from 'src/api/services/sys-dict'

// 创建一个全局单例字典缓存
const dictCache = ref<Record<string, any[]>>({})

export function useDictCache() {
  const dictApi = useDictApi()

  /**
   * 加载字典数据，如果缓存中已有则不重复加载
   * @param dictCode 字典代码
   * @returns 加载的字典选项
   */
  const loadDictData = async (dictCode: string): Promise<any[]> => {
    // 如果没有dictCode，返回空数组
    if (!dictCode) return []

    // 如果缓存中已有该字典数据，直接返回
    if (dictCache.value[dictCode]) {
      return dictCache.value[dictCode]
    }

    try {
      const result = await dictApi.queryDictByCode(dictCode)
      if (result.data && result.data.dict_items) {
        // 存入缓存
        dictCache.value[dictCode] = result.data.dict_items
        return result.data.dict_items
      }
    } catch (error) {
      console.error(`加载字典数据失败: ${dictCode}`, error)
      // 失败时存入空数组，避免重复请求失败的字典
      dictCache.value[dictCode] = []
    }

    return []
  }

  /**
   * 获取字典选项，从缓存中读取
   * @param dictCode 字典代码
   * @returns 字典选项数组
   */
  const getDictOptions = (dictCode: string): any[] => {
    return dictCache.value[dictCode] || []
  }

  /**
   * 清除特定字典的缓存
   * @param dictCode 字典代码
   */
  const clearDictCache = (dictCode?: string) => {
    if (dictCode) {
      delete dictCache.value[dictCode]
    } else {
      // 如果没有指定dictCode，清空所有缓存
      dictCache.value = {}
    }
  }

  /**
   * 获取字典项显示文本
   * @param dictCode 字典代码
   * @param value 字典项值
   * @returns 字典项显示文本
   */
  const getDictLabel = (dictCode: string, value: any): string => {
    const options = getDictOptions(dictCode)
    const found = options.find(item => item.value === value || item.dict_value === value)
    return found ? (found.label || found.dict_label || '') : ''
  }

  return {
    getDictOptions,
    loadDictData,
    clearDictCache,
    getDictLabel,
    // 导出只读的缓存，避免外部修改
    dictCache: readonly(dictCache)
  }
}
