import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: { count?: number; time?: string }) =>
        params?.count !== undefined ? `${key}:${params.count}` : key
    })
  }
})

import UserDashboardStats from '../UserDashboardStats.vue'
import type { UserDashboardStats as DashboardStats } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

const baseStats: DashboardStats = {
  total_api_keys: 2,
  active_api_keys: 2,
  total_requests: 30,
  total_input_tokens: 300,
  total_output_tokens: 150,
  total_cache_creation_tokens: 20,
  total_cache_read_tokens: 10,
  total_tokens: 480,
  total_cost: 8,
  total_actual_cost: 5,
  today_requests: 12,
  today_input_tokens: 120,
  today_output_tokens: 60,
  today_cache_creation_tokens: 8,
  today_cache_read_tokens: 4,
  today_tokens: 192,
  today_cost: 3,
  today_actual_cost: 2,
  average_duration_ms: 400,
  rpm: 2,
  tpm: 64,
  by_platform: [
    {
      platform: 'grok',
      total_requests: 10,
      total_tokens: 180,
      total_actual_cost: 2,
      today_requests: 4,
      today_tokens: 72,
      today_actual_cost: 0.8
    },
    {
      platform: 'openai',
      total_requests: 20,
      total_tokens: 300,
      total_actual_cost: 3,
      today_requests: 8,
      today_tokens: 120,
      today_actual_cost: 1.2
    }
  ]
}

function quota(overrides: Partial<PlatformQuotaItem> & Pick<PlatformQuotaItem, 'platform'>) {
  return {
    daily_limit_usd: null,
    weekly_limit_usd: null,
    monthly_limit_usd: null,
    daily_usage_usd: 0,
    weekly_usage_usd: 0,
    monthly_usage_usd: 0,
    ...overrides
  } as PlatformQuotaItem
}

describe('UserDashboardStats platform ledger', () => {
  it('restores ordered platform usage and quota windows including Grok', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        stats: baseStats,
        balance: 20,
        isSimple: false,
        platformQuotas: [
          quota({ platform: 'grok', daily_limit_usd: 10, daily_usage_usd: 4 }),
          quota({ platform: 'openai', daily_limit_usd: 0 })
        ]
      }
    })

    const text = wrapper.text()
    expect(text.indexOf('OpenAI')).toBeLessThan(text.indexOf('Grok'))
    expect(text).toContain('dashboard.platformCount:2')
    expect(text).toContain('$4.00 / $10.00')
    expect(text).toContain('dashboard.platformQuota.disabled')
    expect(wrapper.find('[data-platform="grok"] .quota-fill').attributes('style')).toContain('40%')
  })

  it('shows unassigned spend as an explicit Other card', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        stats: {
          ...baseStats,
          total_actual_cost: 6,
          today_actual_cost: 2.5
        },
        balance: 20,
        isSimple: false,
        platformQuotas: []
      }
    })

    const other = wrapper.find('[data-platform="__other__"]')
    expect(other.exists()).toBe(true)
    expect(other.text()).toContain('dashboard.platformOther')
    expect(other.text()).toContain('$1.0000')
  })

  it('keeps the platform ledger hidden in simple mode', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        stats: baseStats,
        balance: 20,
        isSimple: true,
        platformQuotas: [quota({ platform: 'grok', daily_limit_usd: 10 })]
      }
    })

    expect(wrapper.find('.platform-ledger').exists()).toBe(false)
  })
})
