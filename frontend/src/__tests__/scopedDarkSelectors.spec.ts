import { describe, expect, it } from 'vitest'

/**
 * 回归守卫：Vue 3.5 起 `:global(.dark) .foo` 会被 scoped 编译器压成裸 `.dark`，
 * 后代选择器整段丢失，暗色覆盖静默失效（曾导致首页充值横幅、密钥页表头在
 * 暗色下变白）。正确写法是直接 `.dark .foo`（Vue 只会给最后一个复合选择器
 * 加 data-v 属性，`.dark` 仍作为全局祖先匹配）。
 */
const vueSources = import.meta.glob('../**/*.vue', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>

describe('scoped dark selectors', () => {
  it('never uses :global(.dark) before a descendant selector', () => {
    const offenders = Object.entries(vueSources)
      .filter(([path]) => !path.includes('/__tests__/'))
      .filter(([, source]) => /:global\(\.dark\)\s+\S/.test(source))
      .map(([path]) => path)
    expect(offenders).toEqual([])
  })
})
