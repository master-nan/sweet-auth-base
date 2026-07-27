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

const mountTree = () =>
  mount(OrganizationReadOnlyTree, {
    props: {
      nodes: [
        {
          id: 1,
          code: 'ROOT',
          name: '集团总部',
          icon: 'apartment',
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
        QIcon: true,
        QChip: true,
      },
    },
  })

describe('OrganizationReadOnlyTree', () => {
  it('renders organization identity without an operation column', () => {
    const wrapper = mountTree()

    expect(wrapper.text()).toContain('集团总部')
    expect(wrapper.text()).toContain('ROOT')
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
