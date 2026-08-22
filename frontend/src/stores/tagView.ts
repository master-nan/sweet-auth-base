import { defineStore } from 'pinia'
import type { RouteLocationNormalized } from 'vue-router'
import { Router as router } from 'src/router/index'
import { LocalStorage } from 'quasar'
import type { RouteData } from 'src/types'
import { useUserStore } from 'stores/user'
enum removeType {
  Right,
  Left,
  Other,
}
export const useTagViewStore = defineStore('tagView', {
  state: () => ({
    tagView: [] as RouteData[],
  }),

  getters: {
    getTagView(state): RouteData[] {
      return state.tagView
    },
  },

  actions: {
    addTagView(to: RouteLocationNormalized) {
      const routeTitle = resolveRouteTitle(to)
      const tag: RouteData = {
        title: routeTitle,
        name: to.name,
        fullPath: to.fullPath,
        icon: to.meta.icon as string,
        keepAlive: to.meta.keepAlive as boolean,
        isHidden: to.meta.isHidden,
        showTag: to.meta.showTag === true,
      }

      if (tag.isHidden && !tag.showTag) {
        return
      }

      if (
        tag.title !== null &&
        tag.title !== undefined &&
        tag.fullPath !== '/admin/home' &&
        tag.fullPath.indexOf('#') === -1
      ) {
        const size = this.tagView.length
        // 首次进入或刷新非首页路由时建立第一个页签。
        if (!size && tag.fullPath !== '/admin/home') {
          this.tagView.push(tag)
          return
        }
        // fullPath是页签唯一身份，避免同一路由被重复加入。
        const t = []
        for (let i = 0; i < size; i++) {
          t.push(this.tagView[i]!.fullPath)
        }
        // 只有当前fullPath尚不存在时才追加页签。
        if (t.indexOf(tag.fullPath) === -1) {
          this.tagView.push(tag)
        }
      }
    },

    setTagView(tagView: RouteData[]) {
      this.tagView = tagView
    },

    updateTagViewTitle(fullPath: string, title: string) {
      if (!title) return
      const tag = this.tagView.find((item) => item.fullPath === fullPath)
      if (tag && tag.title !== title) {
        tag.title = title
      }
    },

    removeTagViewByFullPath(fullPath: string) {
      const index = this.tagView.findIndex((item) => item.fullPath === fullPath)
      if (index !== -1) {
        removeATagView(this, index)
      }
    },

    removeTagViewAt(index: number) {
      removeATagView(this, index)
    },

    removeTagViewOnLeft(index: number) {
      removeOnSide(this, removeType.Left, index)
    },

    removeTagViewOnRight(index: number) {
      removeOnSide(this, removeType.Right, index)
    },

    removeOtherTagView(index: number) {
      removeOnSide(this, removeType.Other, index)
    },

    removeAllTagView() {
      this.tagView = []
      LocalStorage.set('tagView', [])
      const userStore = useUserStore()
      const isLogin = userStore.isLogin
      if (isLogin) {
        router.push({ name: 'home' })
      }
    },
  },
})

function firstQueryValue(value: unknown): string {
  const rawValue = Array.isArray(value) ? value[0] : value
  if (rawValue === null || rawValue === undefined) return ''
  if (typeof rawValue === 'string') return rawValue
  if (typeof rawValue === 'number' || typeof rawValue === 'boolean') return String(rawValue)
  return ''
}

function resolveRouteTitle(to: RouteLocationNormalized): string {
  const itemLabel = firstQueryValue(to.query.item_label)
  const baseTitle = String(to.meta.title || '')
  if (itemLabel && baseTitle) return `${baseTitle}：${itemLabel}`
  return baseTitle
}

// 删除单个页签时，若删除的是当前页签，则选择相邻页面作为稳定回退目标。
export function removeATagView(state: any, index: any) {
  // 删除前保存fullPath，用于判断当前浏览器地址是否需要跳转。
  const removedTagView = state.tagView[index].fullPath
  state.tagView.splice(index, 1)
  // 无剩余页签时回到首页。
  if (state.tagView.length === 0) {
    LocalStorage.set('tagView', [])
    router.push({ name: 'home' })
  } else {
    // 删除末尾当前页签时跳到新的末尾页签。
    if (index === state.tagView.length && window.location.href.indexOf(removedTagView) !== -1) {
      router.push(state.tagView[index - 1].fullPath)
      return
    }
    // 删除首个当前页签时跳到新的首个页签。
    if (index === 0 && window.location.href.indexOf(removedTagView) !== -1) {
      router.push(state.tagView[0].fullPath)
      return
    }
    if (window.location.href.indexOf(removedTagView) !== -1) {
      router.push(state.tagView[index - 1].fullPath)
    }
  }
}

// 按当前位置关闭左侧、右侧或其他页签，并保持当前路由可达。
export function removeOnSide(state: any, type: removeType, index: number) {
  switch (type) {
    case removeType.Right:
      state.tagView = state.tagView.slice(0, index + 1)
      if (state.tagView.length === 1) {
        router.push(state.tagView[0].fullPath)
      }
      if (state.tagView.length === index + 1) {
        router.push(state.tagView[index].fullPath)
      }
      break
    case removeType.Left:
      state.tagView = state.tagView.slice(index, state.tagView.length)
      if (state.tagView.length === 1) {
        router.push(state.tagView[0].fullPath)
      }
      if (state.tagView.length <= index) {
        router.push(state.tagView[0].fullPath)
      }
      break
    case removeType.Other:
      state.tagView = state.tagView.splice(index, 1)
      router.push(state.tagView[0].fullPath)
      break
    default:
      break
  }
}
