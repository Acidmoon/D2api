<template>
  <section>
    <h2 class="section-title">{{ t('dashboard.overview') }}</h2>
    <div class="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
      <!-- 余额（简单模式隐藏） -->
      <div v-if="!isSimple" class="kpi-cell card">
        <p class="kpi-label">{{ t('dashboard.balance') }}</p>
        <p class="kpi-value" style="color: var(--nm-success-text)">${{ formatBalance(balance) }}</p>
        <p class="kpi-sub">{{ t('common.available') }}</p>
      </div>

      <!-- 请求次数 -->
      <div class="kpi-cell card">
        <p class="kpi-label">{{ t('dashboard.totalRequests') }}</p>
        <p class="kpi-value">{{ formatNumber(stats?.total_requests || 0) }}</p>
        <p class="kpi-sub">{{ t('dashboard.todayRequests') }}: {{ formatNumber(stats?.today_requests || 0) }}</p>
      </div>

      <!-- Input Tokens -->
      <div class="kpi-cell card">
        <p class="kpi-label">Input Tokens</p>
        <p class="kpi-value">{{ formatTokens(stats?.total_input_tokens || 0) }}</p>
        <p class="kpi-sub">{{ t('dashboard.today') }}: {{ formatTokens(stats?.today_input_tokens || 0) }}</p>
      </div>

      <!-- Output Tokens -->
      <div class="kpi-cell card">
        <p class="kpi-label">Output Tokens</p>
        <p class="kpi-value">{{ formatTokens(stats?.total_output_tokens || 0) }}</p>
        <p class="kpi-sub">{{ t('dashboard.today') }}: {{ formatTokens(stats?.today_output_tokens || 0) }}</p>
      </div>

      <!-- Cache Tokens -->
      <div class="kpi-cell card">
        <p class="kpi-label">Cache Tokens</p>
        <p class="kpi-value">{{ formatTokens(cacheTotal) }}</p>
        <p class="kpi-sub">{{ t('dashboard.today') }}: {{ formatTokens(cacheToday) }}</p>
      </div>

      <!-- 总 Tokens -->
      <div class="kpi-cell card">
        <p class="kpi-label">{{ t('dashboard.totalTokens') }}</p>
        <p class="kpi-value">{{ formatTokens(stats?.total_tokens || 0) }}</p>
        <p class="kpi-sub">{{ t('dashboard.todayTokens') }}: {{ formatTokens(stats?.today_tokens || 0) }}</p>
      </div>
    </div>

    <div v-if="!isSimple && platformCards.length > 0" class="platform-ledger">
      <div class="platform-ledger__header">
        <div>
          <p class="platform-ledger__eyebrow">{{ t('dashboard.actual') }}</p>
          <h3 class="platform-ledger__title">{{ t('dashboard.platformBreakdown') }}</h3>
        </div>
        <span class="platform-ledger__count">
          {{ t('dashboard.platformCount', { count: platformCount }) }}
        </span>
      </div>

      <div class="platform-grid">
        <article
          v-for="item in platformCards"
          :key="item.platform"
          class="platform-card card"
          :class="{ 'platform-card--other': item.isOther }"
          :data-platform="item.platform"
        >
          <header class="platform-card__header">
            <div class="platform-identity">
              <span class="platform-code">{{ platformCode(item.platform) }}</span>
              <span class="platform-name">
                {{ item.isOther ? t('dashboard.platformOther') : platformLabel(item.platform) }}
              </span>
            </div>
            <div class="platform-total">
              <span class="platform-total__label">{{ t('common.total') }}</span>
              <strong>${{ formatCost(item.total_actual_cost) }}</strong>
            </div>
          </header>

          <dl class="platform-metrics">
            <div>
              <dt>{{ t('dashboard.todayCost') }}</dt>
              <dd>${{ formatCost(item.today_actual_cost) }}</dd>
            </div>
            <div>
              <dt>{{ t('dashboard.requests') }}</dt>
              <dd>{{ item.total_requests > 0 ? formatNumber(item.total_requests) : '—' }}</dd>
            </div>
            <div>
              <dt>{{ t('dashboard.tokens') }}</dt>
              <dd>{{ item.total_tokens > 0 ? formatTokens(item.total_tokens) : '—' }}</dd>
            </div>
          </dl>

          <div v-if="hasAnyLimit(item.quota) && !item.isOther" class="quota-panel">
            <p class="quota-panel__title">{{ t('dashboard.platformQuota.title') }}</p>
            <template v-for="quotaWindow in quotaWindows" :key="quotaWindow">
              <div v-if="quotaLimit(item.quota, quotaWindow) !== null" class="quota-row">
                <div class="quota-row__summary">
                  <span>{{ t(`dashboard.platformQuota.${quotaWindow}`) }}</span>
                  <span
                    class="quota-row__value"
                    :class="{ 'quota-row__value--disabled': quotaLimit(item.quota, quotaWindow) === 0 }"
                  >
                    {{ formatQuota(item.quota, quotaWindow) }}
                  </span>
                </div>
                <div class="quota-track" aria-hidden="true">
                  <div
                    class="quota-fill"
                    :class="`quota-fill--${quotaTone(item.quota, quotaWindow)}`"
                    :style="{ width: `${quotaPercent(item.quota, quotaWindow)}%` }"
                  />
                </div>
                <p v-if="quotaReset(item.quota, quotaWindow)" class="quota-reset">
                  {{
                    t('dashboard.platformQuota.resetsAt', {
                      time: formatResetTime(quotaReset(item.quota, quotaWindow))
                    })
                  }}
                </p>
              </div>
            </template>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { PlatformQuotaItem } from '@/types'

interface FusedPlatformCard {
  platform: string
  total_actual_cost: number
  today_actual_cost: number
  total_requests: number
  total_tokens: number
  isOther?: boolean
  quota?: PlatformQuotaItem
}

type QuotaWindow = 'daily' | 'weekly' | 'monthly'

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  platformQuotas?: PlatformQuotaItem[] | null
}>()
const { t } = useI18n()

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok'
}

const PLATFORM_CODES: Record<string, string> = {
  anthropic: 'CL',
  openai: 'OA',
  gemini: 'GE',
  antigravity: 'AG',
  grok: 'GX',
  __other__: '··'
}

const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
const OTHER_THRESHOLD = 0.0001
const quotaWindows: QuotaWindow[] = ['daily', 'weekly', 'monthly']

const platformLabel = (platform: string) => PLATFORM_LABELS[platform] ?? platform
const platformCode = (platform: string) =>
  PLATFORM_CODES[platform] ?? platform.slice(0, 2).toUpperCase()

const platformCards = computed<FusedPlatformCard[]>(() => {
  const statsByPlatform = new Map(
    (props.stats?.by_platform ?? []).map((item) => [item.platform, item] as const)
  )
  const quotaByPlatform = new Map<string, PlatformQuotaItem>(
    (props.platformQuotas ?? []).map((item) => [item.platform, item] as const)
  )
  const platforms = new Set([...statsByPlatform.keys(), ...quotaByPlatform.keys()])

  const cards = [...platforms].map<FusedPlatformCard>((platform) => {
    const platformStats = statsByPlatform.get(platform)
    return {
      platform,
      total_actual_cost: platformStats?.total_actual_cost ?? 0,
      today_actual_cost: platformStats?.today_actual_cost ?? 0,
      total_requests: platformStats?.total_requests ?? 0,
      total_tokens: platformStats?.total_tokens ?? 0,
      quota: quotaByPlatform.get(platform)
    }
  })

  cards.sort((left, right) => {
    const leftIndex = PLATFORM_ORDER.indexOf(left.platform)
    const rightIndex = PLATFORM_ORDER.indexOf(right.platform)
    if (leftIndex === -1 && rightIndex === -1) return left.platform.localeCompare(right.platform)
    if (leftIndex === -1) return 1
    if (rightIndex === -1) return -1
    return leftIndex - rightIndex
  })

  // 后端无法归属平台的记录不会出现在 by_platform 中，因此显式补齐与总消费的差值。
  const accountedTotal = cards.reduce((sum, card) => sum + card.total_actual_cost, 0)
  const accountedToday = cards.reduce((sum, card) => sum + card.today_actual_cost, 0)
  const otherTotal = Math.max(0, (props.stats?.total_actual_cost ?? 0) - accountedTotal)
  const otherToday = Math.max(0, (props.stats?.today_actual_cost ?? 0) - accountedToday)

  if (otherTotal > OTHER_THRESHOLD || otherToday > OTHER_THRESHOLD) {
    cards.push({
      platform: '__other__',
      total_actual_cost: otherTotal,
      today_actual_cost: otherToday,
      total_requests: 0,
      total_tokens: 0,
      isOther: true
    })
  }

  return cards
})

const platformCount = computed(() => platformCards.value.filter((item) => !item.isOther).length)

const cacheTotal = computed(
  () => (props.stats?.total_cache_creation_tokens || 0) + (props.stats?.total_cache_read_tokens || 0)
)
const cacheToday = computed(
  () => (props.stats?.today_cache_creation_tokens || 0) + (props.stats?.today_cache_read_tokens || 0)
)

function hasAnyLimit(quota: PlatformQuotaItem | undefined): boolean {
  if (!quota) return false
  return (
    quota.daily_limit_usd !== null ||
    quota.weekly_limit_usd !== null ||
    quota.monthly_limit_usd !== null
  )
}

function quotaLimit(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number | null {
  return quota?.[`${window}_limit_usd`] ?? null
}

function quotaUsage(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number {
  return quota?.[`${window}_usage_usd`] ?? 0
}

function quotaReset(quota: PlatformQuotaItem | undefined, window: QuotaWindow): string | null {
  return quota?.[`${window}_window_resets_at`] ?? null
}

function quotaPercent(quota: PlatformQuotaItem | undefined, window: QuotaWindow): number {
  const limit = quotaLimit(quota, window)
  if (limit === 0) return 100
  if (limit === null || limit < 0) return 0
  return Math.min(100, Math.max(0, Math.round((quotaUsage(quota, window) / limit) * 100)))
}

function quotaTone(
  quota: PlatformQuotaItem | undefined,
  window: QuotaWindow
): 'success' | 'warning' | 'danger' {
  const limit = quotaLimit(quota, window)
  if (limit === 0) return 'danger'
  const percentage = quotaPercent(quota, window)
  if (percentage >= 95) return 'danger'
  if (percentage >= 75) return 'warning'
  return 'success'
}

const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2
})

function formatUsd(value: number): string {
  return Number.isFinite(value) ? usdFormatter.format(value) : '0.00'
}

function formatQuota(quota: PlatformQuotaItem | undefined, window: QuotaWindow): string {
  const limit = quotaLimit(quota, window)
  if (limit === 0) return t('dashboard.platformQuota.disabled')
  if (limit === null) return ''
  return `$${formatUsd(quotaUsage(quota, window))} / $${formatUsd(limit)}`
}

function formatResetTime(value: string | null): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(undefined, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(b)
const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
</script>

<style scoped>
.section-title {
  font-size: 0.8125rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0;
  color: hsl(var(--foreground));
  margin-bottom: 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid hsl(var(--border));
}

.kpi-cell {
  padding: 0.875rem 1rem;
  min-height: 7rem;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.kpi-label {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0;
  color: hsl(var(--muted-foreground));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.kpi-value {
  margin-top: 0.5rem;
  font-size: 1.5rem;
  font-weight: 600;
  color: hsl(var(--foreground));
  font-variant-numeric: tabular-nums;
  line-height: 1.1;
}

.kpi-sub {
  margin-top: 0.5rem;
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.platform-ledger {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid hsl(var(--border));
}

.platform-ledger__header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.75rem;
}

.platform-ledger__eyebrow {
  font-size: 0.625rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: hsl(var(--brand));
}

.platform-ledger__title {
  margin-top: 0.125rem;
  font-size: 1rem;
  font-weight: 700;
  color: hsl(var(--foreground));
}

.platform-ledger__count {
  flex: none;
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
}

.platform-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr));
  gap: 0.75rem;
}

.platform-card {
  position: relative;
  overflow: hidden;
  padding: 1rem;
  border-top: 3px solid hsl(var(--brand));
}

.platform-card::after {
  position: absolute;
  top: 0;
  right: 0;
  width: 3.5rem;
  height: 3.5rem;
  content: '';
  background: linear-gradient(135deg, transparent 49%, hsl(var(--muted)) 50%);
  pointer-events: none;
}

.platform-card--other {
  border-top-color: hsl(var(--border));
  border-style: dashed;
}

.platform-card__header,
.platform-identity,
.quota-row__summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.platform-identity {
  justify-content: flex-start;
  min-width: 0;
}

.platform-code {
  display: inline-flex;
  width: 2rem;
  height: 2rem;
  flex: none;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--foreground));
  background: hsl(var(--foreground));
  color: hsl(var(--background));
  font-size: 0.625rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.platform-name {
  overflow: hidden;
  color: hsl(var(--foreground));
  font-size: 0.875rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.platform-total {
  position: relative;
  z-index: 1;
  text-align: right;
}

.platform-total__label {
  display: block;
  font-size: 0.5625rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: hsl(var(--muted-foreground));
}

.platform-total strong {
  display: block;
  color: hsl(var(--brand));
  font-size: 0.9375rem;
  font-variant-numeric: tabular-nums;
}

.platform-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin-top: 1rem;
  border: 1px solid hsl(var(--border));
}

.platform-metrics > div {
  min-width: 0;
  padding: 0.625rem;
}

.platform-metrics > div + div {
  border-left: 1px solid hsl(var(--border));
}

.platform-metrics dt {
  overflow: hidden;
  font-size: 0.5625rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: hsl(var(--muted-foreground));
  text-overflow: ellipsis;
  white-space: nowrap;
}

.platform-metrics dd {
  margin-top: 0.25rem;
  overflow: hidden;
  color: hsl(var(--foreground));
  font-size: 0.75rem;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.quota-panel {
  margin-top: 0.875rem;
  padding-top: 0.75rem;
  border-top: 1px solid hsl(var(--border));
}

.quota-panel__title {
  margin-bottom: 0.625rem;
  font-size: 0.5625rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: hsl(var(--muted-foreground));
}

.quota-row + .quota-row {
  margin-top: 0.625rem;
}

.quota-row__summary {
  align-items: baseline;
  font-size: 0.6875rem;
  color: hsl(var(--muted-foreground));
}

.quota-row__value {
  color: hsl(var(--foreground));
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.quota-row__value--disabled {
  color: hsl(var(--destructive));
}

.quota-track {
  height: 0.3125rem;
  margin-top: 0.3125rem;
  overflow: hidden;
  background: hsl(var(--muted));
}

.quota-fill {
  height: 100%;
  transition: width 220ms ease;
}

.quota-fill--success {
  background: var(--nm-success);
}

.quota-fill--warning {
  background: var(--nm-warning);
}

.quota-fill--danger {
  background: var(--nm-danger);
}

.quota-reset {
  margin-top: 0.25rem;
  font-size: 0.5625rem;
  color: hsl(var(--muted-foreground));
  text-align: right;
}

@media (max-width: 639px) {
  .platform-ledger__header {
    align-items: flex-start;
  }

  .platform-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
