import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import OrganizationReadOnlyTree from './OrganizationReadOnlyTree.vue'

const QTreeStub = defineComponent({
  name: 'QTree',
  props: {
    nodes: {
      type: Array,
      default: () => [],
    },
    selected: {
      type: Number,
      default: null,
    },
    expanded: {
      type: Array,
      default: () => [],
    },
    noConnectors: {
      type: Boolean,
      default: false,
    },
    dark: {
      type: Boolean,
      default: false,
    },
  },
  emits: ['update:selected', 'update:expanded'],
  setup(props, { slots }) {
    return () =>
      h(
        'div',
        { 'data-testid': 'q-tree' },
        (props.nodes as Record<string, unknown>[]).map((node) =>
          slots['default-header']?.({
            node,
            key: node.id,
            expanded: false,
          }),
        ),
      )
  },
})

const QIconStub = defineComponent({
  name: 'QIcon',
  props: {
    name: {
      type: String,
      default: '',
    },
  },
  setup(props) {
    return () => h('i', { 'data-icon': props.name })
  },
})

const mountTree = () =>
  mount(OrganizationReadOnlyTree, {
    props: {
      nodes: [
        {
          id: 1,
          code: 'ROOT',
          name: '集团总部',
          icon: 'corporate_fare',
          typeLabel: '中心',
          statusLabel: '启用',
          statusColor: 'positive',
          children: [
            {
              id: 2,
              code: 'CHILD',
              name: '财务中心',
              statusLabel: '启用',
              statusColor: 'positive',
              children: [],
            },
          ],
        },
      ],
      selectedId: 1,
    },
    global: {
      stubs: {
        QTree: QTreeStub,
        QLinearProgress: true,
        QIcon: QIconStub,
        QChip: true,
      },
    },
  })

describe('OrganizationReadOnlyTree', () => {
  it('renders the name as the primary identity and code/type as supporting metadata', () => {
    const wrapper = mountTree()
    const tree = wrapper.findComponent(QTreeStub)

    expect(wrapper.find('.organization-readonly-tree__name').text()).toBe('集团总部')
    expect(wrapper.find('.organization-readonly-tree__meta').text()).toContain('ROOT')
    expect(wrapper.find('.organization-readonly-tree__meta').text()).toContain('中心')
    expect(wrapper.find('[data-icon="corporate_fare"]').exists()).toBe(true)
    expect(tree.props('noConnectors')).toBe(false)
    expect(tree.props('dark')).toBe(false)
    expect(wrapper.text()).not.toContain('操作')
    expect(wrapper.text()).not.toContain('详情')
  })

  it('keeps expansion separate from node selection', async () => {
    const wrapper = mountTree()
    const tree = wrapper.findComponent(QTreeStub)

    tree.vm.$emit('update:expanded', [1])
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('select')).toBeUndefined()

    tree.vm.$emit('update:selected', 2)
    await wrapper.vm.$nextTick()
    expect(wrapper.emitted('select')).toEqual([[2]])
  })
})
