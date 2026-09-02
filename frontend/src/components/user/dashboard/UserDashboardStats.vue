<template>
  <section class="grid gap-4 xl:grid-cols-3" data-testid="dashboard-stats">
    <!-- Left 2/3: balance / requests / spend / tokens -->
    <div class="grid gap-4 sm:grid-cols-2 xl:col-span-2">
      <!-- Balance (hidden in simple mode) -->
      <div v-if="!isSimple" class="card flex flex-col p-5" data-testid="dashboard-balance-card">
        <div class="flex items-center justify-between">
          <span class="stat-label">{{ t('dashboard.balance') }}</span>
          <span class="stat-icon stat-icon-success"><Icon name="dollar" size="sm" /></span>
        </div>
        <p class="stat-value mt-3 text-3xl" style="color: var(--nm-success-text)">
          ${{ formatBalance(balance) }}
        </p>
        <p class="mt-1 text-xs text-muted-foreground">{{ t('common.available') }}</p>
        <div class="mt-4 flex flex-wrap items-center gap-2">
          <RouterLink v-if="showPaymentLinks" to="/purchase" class="swiss-action text-brand">
            <Icon name="plus" size="xs" />
            {{ t('dashboard.topUp') }}
          </RouterLink>
          <RouterLink v-if="showPaymentLinks" to="/orders" class="swiss-action">
            <Icon name="document" size="xs" />
            {{ t('dashboard.viewOrders') }}
          </RouterLink>
        </div>
      </div>

      <!-- Requests -->
      <div class="card p-5" data-testid="dashboard-requests-card">
        <div class="flex items-center justify-between">
          <span class="stat-label">{{ t('dashboard.totalRequests') }}</span>
          <span class="stat-icon stat-icon-primary"><Icon name="bolt" size="sm" /></span>
        </div>
        <p class="stat-value mt-3">{{ formatNumber(stats.total_requests) }}</p>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ t('dashboard.todayRequests') }}: {{ formatNumber(stats.today_requests) }}
        </p>
      </div>

      <!-- Spend with 7-day mini trend -->
      <div class="card p-5" data-testid="dashboard-spend-card">
        <div class="flex items-center justify-between">
          <span class="stat-label">{{ t('dashboard.totalSpend') }}</span>
          <span class="stat-icon stat-icon-warning"><Icon name="trendingUp" size="sm" /></span>
        </div>
        <p class="stat-value mt-3">${{ formatCost(stats.total_actual_cost) }}</p>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ t('dashboard.todayCost') }}: ${{ formatCost(stats.today_actual_cost) }}
        </p>
        <div v-if="spendBars.length > 0" class="mt-4 flex h-10 items-end gap-1" aria-hidden="true">
          <div
            v-for="bar in spendBars"
            :key="bar.date"
            class="min-w-0 flex-1 rounded-full bg-brand"
            :style="{ height: bar.height + '%' }"
            :title="bar.title"
          />
        </div>
      </div>

      <!-- Tokens -->
      <div class="card p-5" data-testid="dashboard-tokens-card">
        <div class="flex items-center justify-between">
          <span class="stat-label">{{ t('dashboard.totalTokens') }}</span>
          <span class="stat-icon"><Icon name="database" size="sm" /></span>
        </div>
        <p class="stat-value mt-3">{{ formatTokens(stats.total_tokens) }}</p>
        <p class="mt-1 text-xs text-muted-foreground">
          {{ t('dashboard.todayTokens') }}: {{ formatTokens(stats.today_tokens) }}
        </p>
        <dl class="mt-3 space-y-1.5 text-xs">
          <div class="flex items-center justify-between gap-2">
            <dt class="shrink-0 text-muted-foreground">{{ t('dashboard.input') }}</dt>
            <dd class="truncate font-medium tabular-nums text-foreground">
              {{ formatTokens(stats.total_input_tokens) }}
              <span class="font-normal text-muted-foreground">/ {{ t('dashboard.today') }} {{ formatTokens(stats.today_input_tokens) }}</span>
            </dd>
          </div>
          <div class="flex items-center justify-between gap-2">
            <dt class="shrink-0 text-muted-foreground">{{ t('dashboard.output') }}</dt>
            <dd class="truncate font-medium tabular-nums text-foreground">
              {{ formatTokens(stats.total_output_tokens) }}
              <span class="font-normal text-muted-foreground">/ {{ t('dashboard.today') }} {{ formatTokens(stats.today_output_tokens) }}</span>
            </dd>
          </div>
          <div class="flex items-center justify-between gap-2">
            <dt class="shrink-0 text-muted-foreground">{{ t('dashboard.cache') }}</dt>
            <dd class="truncate font-medium tabular-nums text-foreground">
              {{ formatTokens(cacheTotal) }}
              <span class="font-normal text-muted-foreground">/ {{ t('dashboard.today') }} {{ formatTokens(cacheToday) }}</span>
            </dd>
          </div>
        </dl>
      </div>

      <!-- Platform benefits / quotas: hidden in simple mode; rendered only when at
           least one platform has quota limits configured (same condition as before). -->
      <div
        v-if="!isSimple && quotaCards.length > 0"
        class="card p-5 sm:col-span-2"
        data-testid="dashboard-benefits-card"
      >
        <div class="flex items-center justify-between">
          <span class="stat-label">{{ t('dashboard.platformBenefits') }}</span>
          <span class="stat-icon"><Icon name="gift" size="sm" /></span>
        </div>
        <div class="mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="quotaCard in quotaCards"
            :key="quotaCard.platform"
            class="rounded-xl bg-secondary p-3"
          >
            <p class="text-sm font-semibold text-foreground">{{ quotaCard.label }}</p>
            <div class="mt-2 space-y-2.5">
              <div v-for="row in quotaCard.windows" :key="row.window">
                <div class="flex items-center justify-between gap-2 text-xs">
                  <span class="shrink-0 text-muted-foreground">{{ t(`dashboard.platformQuota.${row.window}`) }}</span>
                  <span v-if="row.disabled" class="font-mono text-destructive">{{ t('dashboard.platformQuota.disabled') }}</span>
                  <span v-else class="truncate font-mono tabular-nums text-foreground">
                    ${{ formatUsd(row.usage) }} / ${{ formatUsd(row.limit) }}
                  </span>
                </div>
                <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-border" aria-hidden="true">
                  <div
                    class="h-full rounded-full transition-all"
                    :style="{
                      width: (row.disabled ? 100 : row.percent) + '%',
                      backgroundColor: row.disabled ? 'var(--nm-danger)' : quotaBarColor(row.percent)
                    }"
                  />
                </div>
                <p v-if="row.resetsAt" class="mt-0.5 text-[11px] text-muted-foreground">
                  {{ t('dashboard.platformQuota.resetsAt', { time: formatResetTime(row.resetsAt) }) }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Right 1/3: usage summary -->
    <div class="card flex flex-col p-5" data-testid="dashboard-usage-summary">
      <div class="flex items-center justify-between gap-2">
        <span class="stat-label">{{ t('dashboard.usageSummary') }}</span>
        <RouterLink to="/usage" class="swiss-action">
          {{ t('dashboard.viewAllUsage') }}
          <Icon name="arrowRight" size="xs" />
        </RouterLink>
      </div>
      <dl class="mt-4 grid flex-1 grid-cols-2 gap-3">
        <div class="rounded-xl bg-secondary p-3">
          <dt class="text-xs text-muted-foreground">{{ t('dashboard.todayRequests') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatNumber(stats.today_requests) }}
          </dd>
        </div>
        <div class="rounded-xl bg-secondary p-3">
          <dt class="text-xs text-muted-foreground">{{ t('dashboard.todayTokens') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatTokens(stats.today_tokens) }}
          </dd>
        </div>
        <div class="rounded-xl bg-secondary p-3">
          <dt class="text-xs text-muted-foreground">{{ t('dashboard.rpm') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatRate(stats.rpm) }}
          </dd>
        </div>
        <div class="rounded-xl bg-secondary p-3">
          <dt class="text-xs text-muted-foreground">{{ t('dashboard.tpm') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatTokens(stats.tpm) }}
          </dd>
        </div>
        <div class="col-span-2 rounded-xl bg-secondary p-3">
          <dt class="text-xs text-muted-foreground">{{ t('dashboard.avgResponse') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatDuration(stats.average_duration_ms) }}
          </dd>
        </div>
      </dl>
      <RouterLink to="/usage" class="btn btn-secondary btn-sm mt-4">
        {{ t('dashboard.viewUsage') }}
      </RouterLink>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useAuthStore } from '@/stores'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'
import type { TrendDataPoint, PlatformQuotaItem } from '@/types'

const props = withDefaults(defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
  trend?: TrendDataPoint[]
  platformQuotas?: PlatformQuotaItem[] | null
}>(), {
  trend: () => [],
  platformQuotas: null
})
const { t, locale } = useI18n()
const authStore = useAuthStore()

// 与路由 requiresPayment 守卫及快捷操作卡保持一致：支付功能关闭时不渲染入口。
const showPaymentLinks = computed(
  () => !authStore.isSimpleMode && isFeatureFlagEnabled(FeatureFlags.payment)
)

const cacheTotal = computed(
  () => (props.stats?.total_cache_creation_tokens || 0) + (props.stats?.total_cache_read_tokens || 0)
)
const cacheToday = computed(
  () => (props.stats?.today_cache_creation_tokens || 0) + (props.stats?.today_cache_read_tokens || 0)
)

// Mini bar trend of actual spend: the last 7 points of the trend selection
// driving the charts below (range/granularity follow that control).
const spendBars = computed(() => {
  const points = (props.trend || []).slice(-7)
  if (points.length === 0) return []
  const max = Math.max(...points.map((p) => p.actual_cost || 0))
  return points.map((p) => ({
    date: p.date,
    height: max > 0 ? Math.max(8, Math.round(((p.actual_cost || 0) / max) * 100)) : 8,
    title: `${p.date} · $${formatCost(p.actual_cost || 0)}`
  }))
})

// ---- Platform quotas (benefits card) ----

type QuotaWindow = 'daily' | 'weekly' | 'monthly'
const QUOTA_WINDOWS: readonly QuotaWindow[] = ['daily', 'weekly', 'monthly']

interface QuotaWindowRow {
  window: QuotaWindow
  disabled: boolean
  usage: number
  limit: number
  percent: number
  resetsAt: string | null
}

interface QuotaCard {
  platform: string
  label: string
  windows: QuotaWindowRow[]
}

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok'
}

const PLATFORM_ORDER = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

function hasAnyLimit(q: PlatformQuotaItem): boolean {
  return q.daily_limit_usd != null || q.weekly_limit_usd != null || q.monthly_limit_usd != null
}

function calcPercent(usage: number, limit: number): number {
  if (!limit || limit <= 0) return 0
  return Math.min(100, Math.max(0, Math.round((usage / limit) * 100)))
}

// 与 formatBalance 一致使用 Intl.NumberFormat 做半偶舍入，避免 toFixed 在不同 JS 引擎
// 下偶发截断而非四舍五入（与后端展示精度不一致）。
const usdFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2
})
const formatUsd = (n: number) => (Number.isFinite(n) ? usdFormatter.format(n) : '0.00')

function quotaBarColor(p: number): string {
  if (p >= 95) return 'var(--nm-danger)'
  if (p >= 75) return 'var(--nm-warning)'
  return 'var(--nm-success)'
}

function formatResetTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(locale.value, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false
  })
}

// 仅展示配置了限额的平台（与旧版一致：未配置限额的平台不显示配额区，
// limit=0 视为禁用而非不限制）。按 platform 去重（与旧版 Map 语义一致，后者覆盖前者）。
const quotaCards = computed<QuotaCard[]>(() => {
  const byQuota = new Map<string, PlatformQuotaItem>()
  for (const quota of props.platformQuotas ?? []) {
    if (hasAnyLimit(quota)) byQuota.set(quota.platform, quota)
  }

  const cards = [...byQuota.values()].map((quota) => ({
    platform: quota.platform,
    label: platformLabel(quota.platform),
    windows: QUOTA_WINDOWS
      .filter((w) => quota[`${w}_limit_usd`] != null)
      .map((w): QuotaWindowRow => {
        const limit = quota[`${w}_limit_usd`] as number
        const disabled = limit === 0
        const usage = quota[`${w}_usage_usd`] ?? 0
        return {
          window: w,
          disabled,
          usage,
          limit,
          percent: calcPercent(usage, limit),
          // 刻意差异：禁用（limit=0）窗口不展示重置时间，旧版会展示但意义不大
          resetsAt: disabled ? null : (quota[`${w}_window_resets_at`] ?? null)
        }
      })
  }))

  cards.sort((a, b) => {
    const ai = PLATFORM_ORDER.indexOf(a.platform)
    const bi = PLATFORM_ORDER.indexOf(b.platform)
    if (ai === -1 && bi === -1) return a.platform.localeCompare(b.platform)
    if (ai === -1) return 1
    if (bi === -1) return -1
    return ai - bi
  })

  return cards
})

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(b)
const formatCost = (c: number) =>
  new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(c || 0)
const formatNumber = (n: number) => (n || 0).toLocaleString()
const formatRate = (n: number) =>
  n > 0 && !Number.isInteger(n) ? n.toFixed(1) : (n || 0).toLocaleString()
const formatDuration = (ms: number) => {
  if (!ms || ms <= 0) return '0ms'
  return ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${Math.round(ms)}ms`
}
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return (t || 0).toString()
}
</script>
