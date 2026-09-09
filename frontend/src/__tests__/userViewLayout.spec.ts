import { describe, expect, it } from 'vitest'

/**
 * 回归守卫：用户端页面必须包在 `<AppLayout>` 里，否则会丢掉左侧栏与顶栏，
 * 看起来像跳到了一个独立站点（生图创作中心首版就漏了这一层）。
 * 支付弹窗/回跳页天然是独立壳，列入白名单。
 */
const ALLOWLIST = new Set([
  'ChannelStatusView.vue',
  'PaymentResultView.vue',
  'StripePaymentView.vue',
  'StripePopupView.vue'
])

const userViews = import.meta.glob('../views/user/*.vue', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>

describe('user view layout', () => {
  it('wraps every user page in AppLayout', () => {
    const missing = Object.entries(userViews)
      .filter(([path]) => !ALLOWLIST.has(path.split('/').pop() || ''))
      .filter(([, source]) => !source.includes('<AppLayout>'))
      .map(([path]) => path)
    expect(missing).toEqual([])
  })
})
