<template>
  <!-- QW 三列工作台：左（余额+权益）/ 中（近期支出趋势）/ 右（用量分析）；
       简单模式隐藏左列时退化为两列，避免第三列空置 -->
  <section class="grid gap-3 md:grid-cols-3" :class="{ 'md:grid-cols-2': isSimple }" data-testid="dashboard-stats">
    <!-- Left column: balance + benefits (hidden in simple mode) -->
    <div v-if="!isSimple" class="flex flex-col gap-2.5">
      <!-- Balance -->
      <div
        class="flex min-h-[177px] flex-1 flex-col rounded-[24px] bg-card p-7 shadow-card"
        data-testid="dashboard-balance-card"
      >
        <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.balance') }}</h2>
        <p class="mt-3 text-[32px] font-bold leading-10 tabular-nums text-foreground">
          ${{ formatBalance(balance) }}
        </p>
        <div v-if="showPaymentLinks" class="mt-auto flex flex-wrap items-center gap-3 pt-4">
          <RouterLink
            to="/purchase"
            class="flex items-center gap-1.5 text-[13px] font-medium text-[color:var(--nm-accent-text)] transition-opacity hover:opacity-80"
          >
            <Icon name="creditCard" size="xs" />
            {{ t('dashboard.topUp') }}
          </RouterLink>
          <span class="h-3.5 w-px bg-[color:var(--nm-border)]" aria-hidden="true"></span>
          <RouterLink
            to="/orders"
            class="flex items-center gap-1.5 text-[13px] font-medium text-foreground transition-opacity hover:opacity-80"
          >
            <Icon name="document" size="xs" />
            {{ t('dashboard.viewOrders') }}
          </RouterLink>
        </div>
      </div>

      <!-- Platform benefits / quotas: rendered only when at least one platform
           has quota limits configured (same condition as before). -->
      <div
        v-if="quotaCards.length > 0"
        class="flex min-h-[187px] flex-col rounded-[24px] bg-card p-7 shadow-card"
        data-testid="dashboard-benefits-card"
      >
        <div class="flex items-center justify-between gap-3">
          <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.platformBenefits') }}</h2>
          <RouterLink to="/usage" class="qw-pill">{{ t('dashboard.benefits.viewBenefits') }}</RouterLink>
        </div>
        <div class="mt-auto flex items-end justify-between gap-4 pt-4">
          <div class="flex gap-8" :title="benefitsSummary">
            <div>
              <p class="text-[28px] font-bold leading-9 tabular-nums text-foreground">
                {{ quotaCards.length }}
              </p>
              <p class="mt-1 text-[13px] text-[#8E96A7] dark:text-[#7A8290]">
                {{ t('dashboard.benefits.platforms') }}
              </p>
            </div>
            <div>
              <p class="text-[28px] font-bold leading-9 tabular-nums text-foreground">
                {{ totalQuotaWindows }}
              </p>
              <p class="mt-1 text-[13px] text-[#8E96A7] dark:text-[#7A8290]">
                {{ t('dashboard.benefits.quotaWindows') }}
              </p>
            </div>
          </div>
          <!-- 原创 CSS/SVG 点阵装饰：中性点阵 + 一段 accent 折线，随主题换色 -->
          <svg
            class="hidden h-[72px] w-[132px] shrink-0 sm:block"
            viewBox="0 0 132 72"
            fill="none"
            aria-hidden="true"
          >
            <defs>
              <pattern id="qw-benefits-dots" width="12" height="12" patternUnits="userSpaceOnUse">
                <circle cx="1.5" cy="1.5" r="1.5" fill="var(--nm-border)" />
              </pattern>
            </defs>
            <rect width="132" height="72" fill="url(#qw-benefits-dots)" />
            <path
              d="M6 62C36 62 46 22 76 22s44 16 50 16"
              :stroke="quotaPressureColor"
              stroke-width="2"
              stroke-linecap="round"
            />
            <circle cx="126" cy="38" r="3" :fill="quotaPressureColor" />
          </svg>
        </div>
      </div>
    </div>

    <!-- Middle column: recent spend with 7-day mini line chart -->
    <div
      class="flex min-h-[384px] flex-col rounded-[24px] bg-card p-7 shadow-card"
      data-testid="dashboard-spend-card"
    >
      <div class="flex items-center justify-between gap-3">
        <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.totalSpend') }}</h2>
        <RouterLink v-if="showPaymentLinks" to="/orders" class="qw-pill">
          {{ t('dashboard.viewOrders') }}
        </RouterLink>
      </div>
      <p class="mt-5 flex items-center gap-1 text-xs text-[#8E96A7] dark:text-[#7A8290]">
        {{ t('dashboard.last7Days') }}
        <Icon name="infoCircle" size="xs" />
      </p>
      <p class="mt-1 text-[32px] font-bold leading-10 tabular-nums text-foreground">
        ${{ formatCost(spend7dTotal) }}
      </p>
      <div v-if="spendChart" class="relative mt-4 min-h-0 flex-1">
        <svg
          class="h-full w-full"
          viewBox="0 0 315 120"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <defs>
            <linearGradient id="qw-spend-area" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="var(--nm-accent)" stop-opacity="0.16" />
              <stop offset="100%" stop-color="var(--nm-accent)" stop-opacity="0" />
            </linearGradient>
          </defs>
          <path :d="spendChart.areaPath" fill="url(#qw-spend-area)" />
          <path
            :d="spendChart.linePath"
            fill="none"
            stroke="var(--nm-accent)"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            vector-effect="non-scaling-stroke"
          />
        </svg>
        <span
          class="absolute right-0 -translate-y-1/2 rounded-full bg-[#5B58FF] px-2 py-0.5 text-xs font-medium leading-4 text-[#F0F3FF] dark:bg-[#7B78FF] dark:text-[#0B0C0F]"
          :style="{ top: `${spendChart.lastYPct}%` }"
        >
          ${{ formatCost(spendChart.lastValue) }}
        </span>
      </div>
      <div
        v-if="spendChart"
        class="mt-2 flex items-center justify-between text-xs text-[#8E96A7] dark:text-[#7A8290]"
      >
        <span>{{ axisStart }}</span>
        <span>{{ axisMid }}</span>
        <span>{{ t('dates.today') }}</span>
      </div>
    </div>

    <!-- Right column: usage analysis -->
    <div
      class="flex min-h-[384px] flex-col rounded-[24px] bg-card p-7 shadow-card"
      data-testid="dashboard-usage-summary"
    >
      <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.usageAnalysis') }}</h2>

      <!-- Promo banner (payment enabled only) -->
      <RouterLink
        v-if="showPaymentLinks"
        to="/purchase"
        class="promo-banner group relative mt-5 flex min-h-[96px] flex-col justify-center overflow-hidden rounded-[18px] p-5"
        data-testid="dashboard-promo-banner"
      >
        <svg
          class="absolute right-4 top-4 h-5 w-5 text-foreground transition-transform group-hover:-translate-y-0.5 group-hover:translate-x-0.5"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M7 17L17 7M7 7h10v10" />
        </svg>
        <span class="relative z-[1] flex min-w-0 flex-col items-start gap-2">
          <span class="text-lg font-semibold text-foreground">{{ t('dashboard.promo.title') }}</span>
          <span
            class="max-w-full truncate rounded-full bg-white/70 px-2.5 py-1 text-xs text-foreground dark:bg-black/30"
          >
            {{ t('dashboard.promo.desc') }}
          </span>
        </span>
      </RouterLink>

      <div class="mt-6 h-px shrink-0 bg-[color:var(--nm-border)]" aria-hidden="true"></div>

      <!-- Pay-as-you-go big number with prev/next pager -->
      <div class="mt-6 flex items-end justify-between gap-3" data-testid="dashboard-paygo">
        <div class="min-w-0">
          <p class="text-sm font-semibold text-foreground">{{ t('dashboard.payAsYouGo') }}</p>
          <p class="mt-2 text-[32px] font-bold leading-10 tabular-nums text-foreground">
            ${{ formatCost(currentSpendPage.value) }}
          </p>
          <p class="mt-1 truncate text-[13px] text-[#8E96A7] dark:text-[#7A8290]">
            {{ currentSpendPage.label }}
          </p>
        </div>
        <div class="flex shrink-0 items-center gap-2 pb-1">
          <button
            type="button"
            class="inline-flex h-9 w-9 items-center justify-center rounded-full border border-[#D1D7E2] text-muted-foreground transition-colors hover:bg-[#F2F4F8] hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 dark:border-[#2A2E37] dark:hover:bg-[#1D2026]"
            :disabled="spendPageIndex === 0"
            :aria-label="t('dashboard.prev')"
            data-testid="dashboard-paygo-prev"
            @click="spendPageIndex = Math.max(0, spendPageIndex - 1)"
          >
            <Icon name="chevronLeft" size="sm" />
          </button>
          <button
            type="button"
            class="inline-flex h-9 w-9 items-center justify-center rounded-full border border-[#D1D7E2] text-muted-foreground transition-colors hover:bg-[#F2F4F8] hover:text-foreground disabled:cursor-not-allowed disabled:opacity-40 dark:border-[#2A2E37] dark:hover:bg-[#1D2026]"
            :disabled="spendPageIndex >= spendPages.length - 1"
            :aria-label="t('dashboard.next')"
            data-testid="dashboard-paygo-next"
            @click="spendPageIndex = Math.min(spendPages.length - 1, spendPageIndex + 1)"
          >
            <Icon name="chevronRight" size="sm" />
          </button>
        </div>
      </div>

      <!-- Secondary today summary: keeps the requests/tokens cards addressable -->
      <dl class="mt-auto grid grid-cols-2 gap-3 border-t border-[color:var(--nm-border)] pt-4">
        <div data-testid="dashboard-requests-card">
          <dt class="text-xs text-[#8E96A7] dark:text-[#7A8290]">{{ t('dashboard.todayRequests') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatNumber(stats.today_requests) }}
          </dd>
        </div>
        <div data-testid="dashboard-tokens-card">
          <dt class="text-xs text-[#8E96A7] dark:text-[#7A8290]">{{ t('dashboard.todayTokens') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatTokens(stats.today_tokens) }}
          </dd>
        </div>
        <div>
          <dt class="text-xs text-[#8E96A7] dark:text-[#7A8290]">{{ t('dashboard.rpm') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatRate(stats.rpm) }}
          </dd>
        </div>
        <div>
          <dt class="text-xs text-[#8E96A7] dark:text-[#7A8290]">{{ t('dashboard.avgResponse') }}</dt>
          <dd class="mt-1 text-lg font-semibold tabular-nums text-foreground">
            {{ formatDuration(stats.average_duration_ms) }}
          </dd>
        </div>
      </dl>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
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

// 按量付费大数字翻页：累计消费 ↔ 今日消费。
const spendPages = computed(() => [
  { label: t('dashboard.totalSpend'), value: props.stats?.total_actual_cost ?? 0 },
  { label: t('dashboard.todayCost'), value: props.stats?.today_actual_cost ?? 0 }
])
const spendPageIndex = ref(0)
const currentSpendPage = computed(
  () => spendPages.value[Math.min(spendPageIndex.value, spendPages.value.length - 1)]
)

// ---- 7 天支出迷你折线图（SVG polyline + 渐变面积 + 末端数值徽标） ----

const CHART_W = 315
const CHART_H = 120

interface SpendChart {
  linePath: string
  areaPath: string
  lastYPct: number
  lastValue: number
}

const spend7dTotal = computed(() =>
  (props.trend || []).slice(-7).reduce((sum, p) => sum + (p.actual_cost || 0), 0)
)

const spendChart = computed<SpendChart | null>(() => {
  const points = (props.trend || []).slice(-7)
  if (points.length === 0) return null
  const values = points.map((p) => p.actual_cost || 0)
  const max = Math.max(...values)
  const padTop = 8
  const usable = CHART_H - padTop
  const xAt = (i: number) =>
    points.length === 1 ? CHART_W / 2 : (i / (points.length - 1)) * CHART_W
  const yAt = (v: number) => (max > 0 ? padTop + usable - (v / max) * usable : CHART_H - 4)
  const coords = values.map((v, i) => `${xAt(i).toFixed(2)},${yAt(v).toFixed(2)}`)
  const linePath = `M${coords.join(' L')}`
  const areaPath = `${linePath} L${CHART_W},${CHART_H} L0,${CHART_H} Z`
  return {
    linePath,
    areaPath,
    lastYPct: (yAt(values[values.length - 1]) / CHART_H) * 100,
    lastValue: values[values.length - 1] || 0
  }
})

const axisStart = computed(() => (props.trend?.[0]?.date || '').slice(5))
const axisMid = computed(() => {
  const points = props.trend || []
  if (points.length === 0) return ''
  return (points[Math.floor(points.length / 2)]?.date || '').slice(5)
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

const totalQuotaWindows = computed(() =>
  quotaCards.value.reduce((sum, card) => sum + card.windows.length, 0)
)

// 权益数字的悬停摘要：逐平台逐窗口给出用量/限额与重置时间（保留旧配额明细语义）。
const benefitsSummary = computed(() =>
  quotaCards.value
    .map((card) => {
      const windows = card.windows
        .map((row) => {
          const usageText = row.disabled
            ? t('dashboard.platformQuota.disabled')
            : `${formatUsd(row.usage)} / ${formatUsd(row.limit)}`
          const reset = row.resetsAt ? ` (${formatResetTime(row.resetsAt)})` : ''
          return `${t(`dashboard.platformQuota.${row.window}`)} ${usageText}${reset}`
        })
        .join(' · ')
      return `${card.label}: ${windows}`
    })
    .join('\n')
)

// 装饰折线颜色随最高配额压力变化（<75 成功色 / <75 警告色 / ≥95 危险色）。
const quotaPressureColor = computed(() => {
  const maxPercent = Math.max(
    0,
    ...quotaCards.value.flatMap((card) => card.windows.map((row) => row.percent))
  )
  return quotaBarColor(maxPercent)
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

<style scoped>
/* QW 描边药丸按钮：白底 1px #D1D7E2 描边，dark 下换用边框 token。 */
.qw-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  flex-shrink: 0;
  padding: 0 18px;
  border-radius: 999px;
  border: 1px solid #d1d7e2;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  color: var(--nm-ink);
  white-space: nowrap;
  transition: background-color 150ms ease;
}

.qw-pill:hover {
  background-color: var(--nm-surface-soft);
}

:global(.dark) .qw-pill {
  border-color: var(--nm-border);
}

/* Tailwind 的 /透明度修饰符对 hsl(var(--x)) 颜色不生效，横幅底色用
   light 固定值 + dark token 两段表达，随亮/暗主题自动适配。 */
.promo-banner {
  /* #efeffe = rgb(239, 239, 254)，QW 实测浅靛底 */
  background-color: #efeffe;
}

.promo-banner:hover {
  background-color: #e4e7fe;
}

:global(.dark) .promo-banner {
  background-color: var(--nm-accent-soft);
}

:global(.dark) .promo-banner:hover {
  background-color: var(--nm-accent-strong);
}
</style>
