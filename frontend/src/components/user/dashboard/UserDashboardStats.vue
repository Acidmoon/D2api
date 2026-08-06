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

  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'

const props = defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
}>()
const { t } = useI18n()

const cacheTotal = computed(
  () => (props.stats?.total_cache_creation_tokens || 0) + (props.stats?.total_cache_read_tokens || 0)
)
const cacheToday = computed(
  () => (props.stats?.today_cache_creation_tokens || 0) + (props.stats?.today_cache_read_tokens || 0)
)

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(b)
const formatNumber = (n: number) => n.toLocaleString()
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

  </style>
