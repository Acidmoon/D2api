import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DataTable from '../DataTable.vue'

const measureElement = vi.fn()
const virtualItems = [
  { index: 0, start: 0, end: 136, size: 136 },
  { index: 1, start: 136, end: 272, size: 136 },
]

vi.mock('@tanstack/vue-virtual', () => ({
  observeElementRect: vi.fn(),
  useVirtualizer: vi.fn(() => ({
    value: {
      getVirtualItems: () => virtualItems,
      getTotalSize: () => 272,
      measureElement,
    },
  })),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const mountTable = (fixedVirtualRowHeight: boolean) =>
  mount(DataTable, {
    props: {
      columns: [{ key: 'name', label: 'Name' }],
      data: [{ id: 1, name: 'Alpha' }, { id: 2, name: 'Beta' }],
      estimateRowHeight: 136,
      fixedVirtualRowHeight,
      // 合并上游 virtualizeThreshold(默认 100)后,小列表默认全量渲染、不走虚拟器;
      // 这里强制开启虚拟化,才能检验 fixedVirtualRowHeight 对测量行为的影响。
      virtualizeThreshold: 1,
    },
    global: {
      stubs: {
        Icon: true,
      },
    },
  })

describe('DataTable fixed virtual row height', () => {
  beforeEach(() => {
    measureElement.mockClear()
    window.matchMedia = vi.fn().mockReturnValue({
      matches: true,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
    })
  })

  it('does not re-measure row elements when fixed row height is enabled', () => {
    const wrapper = mountTable(true)

    expect(wrapper.find('.dt-fixed-row-height').exists()).toBe(true)
    expect(measureElement).not.toHaveBeenCalled()
  })

  it('keeps dynamic row measurement for existing tables by default', () => {
    mountTable(false)

    expect(measureElement).toHaveBeenCalled()
  })
})
