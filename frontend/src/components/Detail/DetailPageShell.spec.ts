import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DetailPageShell from './DetailPageShell.vue'

const mountShell = (props: Record<string, unknown> = {}) =>
  mount(DetailPageShell, {
    props: {
      title: '审计详情',
      subtitle: '#51',
      icon: 'manage_search',
      ...props,
    },
    slots: {
      actions: '<button>刷新</button>',
      default: '<section class="detail-page-section"><h3>基础信息</h3></section>',
    },
    global: {
      stubs: {
        BaseContent: {
          name: 'BaseContent',
          props: { scrollable: Boolean },
          template: '<main><slot /></main>',
        },
        QIcon: { props: ['name'], template: '<i />' },
        QSpace: true,
        QSpinner: true,
        QInnerLoading: { props: ['showing'], template: '<div><slot /></div>' },
        QBanner: { template: '<div><slot name="avatar" /><slot /><slot name="action" /></div>' },
        QBtn: {
          props: ['label'],
          emits: ['click'],
          template: '<button @click="$emit(\'click\')">{{ label }}</button>',
        },
      },
    },
  })

describe('DetailPageShell', () => {
  it('统一承载标题、操作区、滚动容器和详情分区', () => {
    const wrapper = mountShell()

    expect(wrapper.findComponent({ name: 'BaseContent' }).props('scrollable')).toBe(true)
    expect(wrapper.text()).toContain('审计详情')
    expect(wrapper.text()).toContain('#51')
    expect(wrapper.text()).toContain('刷新')
    expect(wrapper.text()).toContain('基础信息')
  })

  it('错误状态只在允许时提供统一重试入口', async () => {
    const wrapper = mountShell({ error: '加载失败', retryable: true })

    expect(wrapper.text()).toContain('加载失败')
    const retryButton = wrapper.findAll('button').find((button) => button.text() === '重新加载')
    expect(retryButton).toBeDefined()
    await retryButton!.trigger('click')
    expect(wrapper.emitted('retry')).toHaveLength(1)
  })
})
