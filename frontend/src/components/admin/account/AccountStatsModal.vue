<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.usageStatistics')"
    width="extra-wide"
    @close="handleClose"
  >
    <div class="space-y-6">
      <!-- Account Info Header -->
      <div
        v-if="account"
        class="stats-header flex items-center justify-between p-3"
      >
        <div class="flex items-center gap-3">
          <div class="stats-header-icon flex h-10 w-10 items-center justify-center">
            <Icon name="chartBar" size="md" />
          </div>
          <div>
            <div class="stats-title font-semibold">{{ account.name }}</div>
            <div class="stats-muted text-xs">
              {{ t('admin.accounts.last30DaysUsage') }}
            </div>
          </div>
        </div>
        <span
          :class="[
            'stats-status px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'stats-status-active'
              : 'stats-status-muted'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else-if="stats">
        <!-- Row 1: Main Stats Cards -->
        <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
          <!-- 30-Day Total Cost -->
          <div class="card p-4">
            <div class="mb-2 flex items-center justify-between">
              <span class="stats-muted text-xs font-medium">{{
                t('admin.accounts.stats.totalCost')
              }}</span>
              <div class="stats-icon stats-icon-success p-1.5">
                <Icon name="dollar" size="sm" />
              </div>
            </div>
            <p class="stats-value text-2xl font-bold">
              ${{ formatCost(stats.summary.total_cost) }}
            </p>
            <p class="stats-muted mt-1 text-xs">
              {{ t('admin.accounts.stats.accumulatedCost') }}
              <span class="stats-subtle">
                ({{ t('usage.userBilled') }}: ${{ formatCost(stats.summary.total_user_cost) }} ·
                {{ t('admin.accounts.stats.standardCost') }}: ${{
                  formatCost(stats.summary.total_standard_cost)
                }})
              </span>
            </p>
          </div>

          <!-- 30-Day Total Requests -->
          <div class="card p-4">
            <div class="mb-2 flex items-center justify-between">
              <span class="stats-muted text-xs font-medium">{{
                t('admin.accounts.stats.totalRequests')
              }}</span>
              <div class="stats-icon stats-icon-info p-1.5">
                <Icon name="bolt" size="sm" />
              </div>
            </div>
            <p class="stats-value text-2xl font-bold">
              {{ formatNumber(stats.summary.total_requests) }}
            </p>
            <p class="stats-muted mt-1 text-xs">
              {{ t('admin.accounts.stats.totalCalls') }}
            </p>
          </div>

          <!-- Daily Average Cost -->
          <div class="card p-4">
            <div class="mb-2 flex items-center justify-between">
              <span class="stats-muted text-xs font-medium">{{
                t('admin.accounts.stats.avgDailyCost')
              }}</span>
              <div class="stats-icon stats-icon-warning p-1.5">
                <Icon name="calculator" size="sm" />
              </div>
            </div>
            <p class="stats-value text-2xl font-bold">
              ${{ formatCost(stats.summary.avg_daily_cost) }}
            </p>
             <p class="stats-muted mt-1 text-xs">
              {{
                t('admin.accounts.stats.basedOnActualDays', {
                  days: stats.summary.actual_days_used
                })
              }}
              <span class="stats-subtle">
                ({{ t('usage.userBilled') }}: ${{ formatCost(stats.summary.avg_daily_user_cost) }})
              </span>
            </p>
          </div>

          <!-- Daily Average Requests -->
          <div class="card p-4">
            <div class="mb-2 flex items-center justify-between">
              <span class="stats-muted text-xs font-medium">{{
                t('admin.accounts.stats.avgDailyRequests')
              }}</span>
              <div class="stats-icon stats-icon-accent p-1.5">
                <Icon name="trendingUp" size="sm" />
              </div>
            </div>
            <p class="stats-value text-2xl font-bold">
              {{ formatNumber(Math.round(stats.summary.avg_daily_requests)) }}
            </p>
            <p class="stats-muted mt-1 text-xs">
              {{ t('admin.accounts.stats.avgDailyUsage') }}
            </p>
          </div>
        </div>

        <!-- Row 2: Today, Highest Cost, Highest Requests -->
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <!-- Today Overview -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-info p-1.5">
                <Icon name="clock" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.todayOverview')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.accountBilled') }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.today?.cost || 0) }}</span
                >
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.userBilled') }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.today?.user_cost || 0) }}</span
                >
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.requests')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatNumber(stats.summary.today?.requests || 0)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.tokens')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatTokens(stats.summary.today?.tokens || 0)
                }}</span>
              </div>
            </div>
          </div>

          <!-- Highest Cost Day -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-warning p-1.5">
                <Icon name="fire" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.highestCostDay')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.date')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  stats.summary.highest_cost_day?.label || '-'
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.accountBilled') }}</span>
                <span class="stats-highlight-warning text-sm font-semibold"
                  >${{ formatCost(stats.summary.highest_cost_day?.cost || 0) }}</span
                >
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.userBilled') }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.highest_cost_day?.user_cost || 0) }}</span
                >
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.requests')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatNumber(stats.summary.highest_cost_day?.requests || 0)
                }}</span>
              </div>
            </div>
          </div>

          <!-- Highest Request Day -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-accent p-1.5">
                <Icon name="trendingUp" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.highestRequestDay')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.date')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  stats.summary.highest_request_day?.label || '-'
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.requests')
                }}</span>
                <span class="stats-highlight-accent text-sm font-semibold">{{
                  formatNumber(stats.summary.highest_request_day?.requests || 0)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.accountBilled') }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.highest_request_day?.cost || 0) }}</span
                >
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{ t('usage.userBilled') }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.highest_request_day?.user_cost || 0) }}</span
                >
              </div>
            </div>
          </div>
        </div>

        <!-- Row 3: Token Stats -->
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <!-- Accumulated Tokens -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-success p-1.5">
                <Icon name="cube" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.accumulatedTokens')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.totalTokens')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatTokens(stats.summary.total_tokens)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.dailyAvgTokens')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatTokens(Math.round(stats.summary.avg_daily_tokens))
                }}</span>
              </div>
            </div>
          </div>

          <!-- Performance -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-danger p-1.5">
                <Icon name="bolt" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.performance')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.avgResponseTime')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatDuration(stats.summary.avg_duration_ms)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.daysActive')
                }}</span>
                <span class="stats-value text-sm font-semibold"
                  >{{ stats.summary.actual_days_used }} / {{ stats.summary.days }}</span
                >
              </div>
            </div>
          </div>

          <!-- Recent Activity -->
          <div class="card p-4">
            <div class="mb-3 flex items-center gap-2">
              <div class="stats-icon stats-icon-success p-1.5">
                <Icon name="clipboard" size="sm" />
              </div>
              <span class="stats-title text-sm font-semibold">{{
                t('admin.accounts.stats.recentActivity')
              }}</span>
            </div>
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.todayRequests')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatNumber(stats.summary.today?.requests || 0)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.todayTokens')
                }}</span>
                <span class="stats-value text-sm font-semibold">{{
                  formatTokens(stats.summary.today?.tokens || 0)
                }}</span>
              </div>
              <div class="flex items-center justify-between">
                <span class="stats-muted text-xs">{{
                  t('admin.accounts.stats.todayCost')
                }}</span>
                <span class="stats-value text-sm font-semibold"
                  >${{ formatCost(stats.summary.today?.cost || 0) }}</span
                >
              </div>
            </div>
          </div>
        </div>

        <!-- Usage Trend Chart -->
        <div class="card p-4">
          <h3 class="stats-title mb-4 text-sm font-semibold">
            {{ t('admin.accounts.stats.usageTrend') }}
          </h3>
          <div class="h-64">
            <Line v-if="trendChartData" :data="trendChartData" :options="lineChartOptions" />
            <div
              v-else
              class="stats-muted flex h-full items-center justify-center text-sm"
            >
              {{ t('admin.dashboard.noDataAvailable') }}
            </div>
          </div>
        </div>

        <!-- Model Distribution -->
        <ModelDistributionChart :model-stats="stats.models" :loading="false" />

        <EndpointDistributionChart
          :endpoint-stats="stats.endpoints || []"
          :loading="false"
          :title="t('usage.inboundEndpoint')"
        />

        <EndpointDistributionChart
          :endpoint-stats="stats.upstream_endpoints || []"
          :loading="false"
          :title="t('usage.upstreamEndpoint')"
        />
      </template>

      <!-- No Data State -->
      <div
        v-else-if="!loading"
        class="stats-muted flex flex-col items-center justify-center py-12"
      >
        <Icon name="chartBar" size="xl" class="mb-4 h-12 w-12" />
        <p class="text-sm">{{ t('admin.accounts.stats.noData') }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button
          @click="handleClose"
          class="stats-button-secondary px-4 py-2 text-sm font-medium transition-colors"
        >
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js'
import { Line } from 'vue-chartjs'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import EndpointDistributionChart from '@/components/charts/EndpointDistributionChart.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import { getChartTheme, useChartDarkMode, withAlpha } from '@/utils/chartColors'
import type { Account, AccountUsageStatsResponse } from '@/types'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const loading = ref(false)
const stats = ref<AccountUsageStatsResponse | null>(null)

// Dark mode detection
const isDarkMode = useChartDarkMode()

// Chart colors
const chartColors = computed(() => {
  const theme = getChartTheme(isDarkMode.value)
  return {
    text: theme.text,
    grid: theme.grid
  }
})

// Line chart data
const trendChartData = computed(() => {
  if (!stats.value?.history?.length) return null

  const theme = getChartTheme(isDarkMode.value)

  return {
    labels: stats.value.history.map((h) => h.label),
    datasets: [
      {
        label: t('usage.accountBilled') + ' (USD)',
        data: stats.value.history.map((h) => h.actual_cost),
        borderColor: theme.primary,
        backgroundColor: withAlpha(theme.primary, 0.1),
        fill: true,
        tension: 0.3,
        yAxisID: 'y'
      },
      {
        label: t('usage.userBilled') + ' (USD)',
        data: stats.value.history.map((h) => h.user_cost),
        borderColor: theme.green,
        backgroundColor: withAlpha(theme.green, 0.08),
        fill: false,
        tension: 0.3,
        borderDash: [5, 5],
        yAxisID: 'y'
      },
      {
        label: t('admin.accounts.stats.requests'),
        data: stats.value.history.map((h) => h.requests),
        borderColor: theme.amber,
        backgroundColor: withAlpha(theme.amber, 0.1),
        fill: false,
        tension: 0.3,
        yAxisID: 'y1'
      }
    ]
  }
})

// Line chart options with dual Y-axis
const lineChartOptions = computed(() => {
  const theme = getChartTheme(isDarkMode.value)
  return {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      intersect: false,
      mode: 'index' as const
    },
    plugins: {
      legend: {
        position: 'top' as const,
        labels: {
          color: chartColors.value.text,
          usePointStyle: true,
          pointStyle: 'circle',
          padding: 15,
          font: {
            size: 11
          }
        }
      },
      tooltip: {
        callbacks: {
          label: (context: any) => {
            const label = context.dataset.label || ''
            const value = context.raw
            if (label.includes('USD')) {
              return `${label}: $${formatCost(value)}`
            }
            return `${label}: ${formatNumber(value)}`
          }
        }
      }
    },
    scales: {
      x: {
        grid: {
          color: chartColors.value.grid
        },
        ticks: {
          color: chartColors.value.text,
          font: {
            size: 10
          },
          maxRotation: 45,
          minRotation: 0
        }
      },
      y: {
        type: 'linear' as const,
        display: true,
        position: 'left' as const,
        grid: {
          color: chartColors.value.grid
        },
        ticks: {
          color: theme.primary,
          font: {
            size: 10
          },
          callback: (value: string | number) => '$' + formatCost(Number(value))
        },
        title: {
          display: true,
          text: t('usage.accountBilled') + ' (USD)',
          color: theme.primary,
          font: {
            size: 11
          }
        }
      },
      y1: {
        type: 'linear' as const,
        display: true,
        position: 'right' as const,
        grid: {
          drawOnChartArea: false
        },
        ticks: {
          color: theme.amber,
          font: {
            size: 10
          },
          callback: (value: string | number) => formatNumber(Number(value))
        },
        title: {
          display: true,
          text: t('admin.accounts.stats.requests'),
          color: theme.amber,
          font: {
            size: 11
          }
        }
      }
    }
  }
})

// Load stats when modal opens
watch(
  () => props.show,
  async (newVal) => {
    if (newVal && props.account) {
      await loadStats()
    } else {
      stats.value = null
    }
  }
)

const loadStats = async () => {
  if (!props.account) return

  loading.value = true
  try {
    stats.value = await adminAPI.accounts.getStats(props.account.id, 30)
  } catch (error) {
    console.error('Failed to load account stats:', error)
    stats.value = null
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('close')
}

// Format helpers
const formatCost = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else if (value >= 1) {
    return value.toFixed(2)
  } else if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}

const formatNumber = (value: number): string => {
  if (value >= 1_000_000) {
    return (value / 1_000_000).toFixed(2) + 'M'
  } else if (value >= 1_000) {
    return (value / 1_000).toFixed(2) + 'K'
  }
  return value.toLocaleString()
}

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`
  } else if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`
  } else if (value >= 1_000) {
    return `${(value / 1_000).toFixed(2)}K`
  }
  return value.toLocaleString()
}

const formatDuration = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)}s`
  }
  return `${Math.round(ms)}ms`
}
</script>

<style scoped>
.stats-header {
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
}

.stats-header-icon,
.stats-icon,
.stats-button-secondary,
.stats-status {
  border-radius: var(--nm-radius-sm);
}

.stats-header-icon {
  background: var(--nm-accent);
  color: var(--nm-on-accent);
}

.stats-title,
.stats-value {
  color: var(--nm-ink);
}

.stats-muted {
  color: var(--nm-ink-faint);
}

.stats-subtle {
  color: var(--nm-ink-faint);
}

.stats-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
}

.stats-icon-accent {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.stats-icon-success {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.stats-icon-info {
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.stats-icon-warning {
  background: var(--nm-warning-soft);
  color: var(--nm-warning-text);
}

.stats-icon-danger {
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}

.stats-status {
  border: 1px solid var(--nm-border-light);
}

.stats-status-active {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.stats-status-muted {
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
}

.stats-highlight-warning {
  color: var(--nm-warning-text);
}

.stats-highlight-accent {
  color: var(--nm-accent-text);
}

.stats-button-secondary {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
  cursor: pointer;
}

.stats-button-secondary:hover {
  border-color: var(--nm-ink-muted);
  color: var(--nm-ink);
}
</style>
