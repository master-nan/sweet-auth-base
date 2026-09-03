<template>
  <div class="tag-view-wrap row">
    <q-tabs
      class="tagViewBase col-12"
      align="left"
      active-color="primary"
      active-class="tagActive"
      dense
      swipeable
      inline-label
      indicator-color="transparent"
      :breakpoint="0"
    >
      <q-route-tab :to="'/admin/home'" :class="tagViewClass('/admin/home')" flat dense no-caps>
        <q-icon size="1.1rem" name="home" />
        <div class="line-limit-length">{{ t('router.home') }}</div>
      </q-route-tab>
      <template v-for="(tag, i) in tagViewStore.tagView" :key="tag.fullPath + i">
        <q-route-tab :to="tag.fullPath" :class="tagViewClass(tag.fullPath)" flat dense no-caps>
          <q-icon size="1.1rem" :name="tag.icon" />
          <div class="line-limit-length">{{ formatTagTitle(tag.title) }}</div>
          <q-btn
            class="tagView-remove-icon"
            style="display: inline-flex"
            round
            size="0.45em"
            flat
            icon="close"
            @click.prevent.stop="removeTagViewAt(i)"
          />
          <q-menu touch-position context-menu>
            <q-list dense>
              <q-item clickable v-close-popup @click="refreshTagView(tag)">
                <q-item-section side>
                  <q-icon name="refresh" size="xs" />
                </q-item-section>
                <q-item-section>{{ t('tagView.reload') }}</q-item-section>
              </q-item>
              <q-separator />
              <q-item clickable v-close-popup @click="removeTagViewOnRight(i)">
                <q-item-section side>
                  <q-icon name="chevron_right" size="xs" />
                </q-item-section>
                <q-item-section>{{ t('tagView.closeRight') }}</q-item-section>
              </q-item>
              <q-item clickable v-close-popup @click="removeTagViewOnLeft(i)">
                <q-item-section side>
                  <q-icon name="chevron_left" size="xs" />
                </q-item-section>
                <q-item-section>{{ t('tagView.closeLeft') }}</q-item-section>
              </q-item>
              <q-item clickable v-close-popup @click="removeOtherTagView(i)">
                <q-item-section side>
                  <q-icon name="highlight_off" size="xs" />
                </q-item-section>
                <q-item-section>{{ t('tagView.closeOthers') }}</q-item-section>
              </q-item>
              <q-item clickable v-close-popup @click="removeAllTagView()">
                <q-item-section side>
                  <q-icon name="delete_sweep" size="xs" />
                </q-item-section>
                <q-item-section>{{ t('tagView.closeAll') }}</q-item-section>
              </q-item>
            </q-list>
          </q-menu>
        </q-route-tab>
      </template>
    </q-tabs>
  </div>
</template>

<script lang="ts" setup>
import { computed, onUnmounted } from 'vue'
import { LocalStorage } from 'quasar'
import { useRoute, useRouter } from 'vue-router'
import { useTagViewStore } from '@/stores/tagView'
import { useKeepAliveStore } from '@/stores/keep-alive'
import { useAppStore } from '@/stores/app'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineOptions({ name: 'TagView' })

const route = useRoute()
const router = useRouter()
const tagViewStore = useTagViewStore()
const keepAliveStore = useKeepAliveStore()
const appStore = useAppStore()

const removeAllTagView = () => {
  tagViewStore.removeAllTagView()
}

const removeTagViewAt = (i: number) => {
  tagViewStore.removeTagViewAt(i)
}

const removeTagViewOnRight = (i: number) => {
  tagViewStore.removeTagViewOnRight(i)
}

const removeTagViewOnLeft = (i: number) => {
  tagViewStore.removeTagViewOnLeft(i)
}

const removeOtherTagView = (i: number) => {
  tagViewStore.removeOtherTagView(i)
}

const refreshTagView = async (tag: {
  fullPath: string
  name?: string | symbol | null | undefined
}) => {
  // 先导航到目标标签（如果不是当前页）
  if (route.fullPath !== tag.fullPath) {
    await router.push(tag.fullPath)
  }
  // 从 keepAlive 缓存中移除该组件，再重新加载
  if (tag.name) {
    keepAliveStore.keepAliveList = keepAliveStore.keepAliveList.filter((name) => name !== tag.name)
  }
  await appStore.reloadPage()
  // 恢复 keepAlive
  if (tag.name) {
    keepAliveStore.setKeepAliveList(tagViewStore.getTagView)
  }
}

const tagViewClass = computed(() => {
  // 是否当前路由
  return (path: string) => {
    return route.fullPath === path ? 'tagView tagActive' : 'tagView'
  }
})

const formatTagTitle = (title: string) => {
  return title.startsWith('router.') ? t(title) : title
}

onUnmounted(() => {
  unSubscribe()
})

const unSubscribe = tagViewStore.$subscribe((mutation, state) => {
  keepAliveStore.setKeepAliveList(state.tagView)
  LocalStorage.set('tagView', state.tagView)
})
</script>

<style lang="scss" scoped>
.tag-view-wrap {
  width: 100%;
  min-width: 0;
}

.tagViewBase {
  min-height: 42px;

  :deep(.q-tabs__content) {
    gap: 0;
  }

  .tagView {
    margin: 0;
    min-height: 42px;
    height: 42px;
    padding: 0 14px;
    border-right: 1px solid var(--app-header-border);
    border-radius: 0;
    max-width: 210px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--app-header-muted);
    background: transparent;
    transition:
      color 0.18s ease,
      background-color 0.18s ease,
      box-shadow 0.18s ease;

    &:hover {
      color: var(--app-header-text);
      background: var(--app-primary-soft);
    }

    :deep(.q-tab__content) {
      min-width: 0;
      flex-wrap: nowrap;
    }

    :deep(.q-icon) {
      flex: 0 0 auto;
    }
  }

  .tagActive {
    font-weight: 800;
    color: var(--q-primary) !important;
    background: var(--app-primary-soft) !important;
    box-shadow: inset 0 -3px 0 var(--q-primary);
  }
}

.tagView-remove-icon {
  // font-size: .7rem;
  // border-radius: 0.2rem;
  font-weight: bold;
  opacity: 0.58;
  transition: all 0.3s;

  &:hover {
    opacity: 1;
  }
}

.line-limit-length {
  margin: 0px 5px 0px 7px;
  overflow: hidden;
  max-width: 158px;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
