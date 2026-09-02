<template>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
    <!-- Requests -->
    <div class="card p-5">
      <div class="flex items-center justify-between">
        <span class="stat-label">{{ t('usage.totalRequests') }}</span>
        <span class="stat-icon stat-icon-primary">
          <Icon name="document" size="sm" />
        </span>
      </div>
      <p class="stat-value mt-3">{{ stats?.total_requests?.toLocaleString() || '0' }}</p>
      <p class="mt-1 text-xs text-muted-foreground">{{ t('usage.inSelectedRange') }}</p>
    </div>

    <!-- Tokens -->
    <div class="card p-5">
      <div class="flex items-center justify-between">
        <span class="stat-label">{{ t('usage.totalTokens') }}</span>
        <span class="stat-icon"><svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" /></svg></span>
      </div>
      <p class="stat-value mt-3">{{ formatTokens(stats?.total_tokens || 0) }}</p>
      <p class="mt-1 flex flex-wrap items-center gap-x-1 text-xs text-muted-foreground">
        <span>{{ t('usage.in') }}: {{ formatTokens(stats?.total_input_tokens || 0) }}</span>
        <span>/</span>
        <span>{{ t('usage.out') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}</span>
        <span>/</span>
        <span class="group relative inline-flex cursor-help items-center gap-0.5" tabindex="0">
          <span>{{ cacheLabel() }}: {{ formatTokens(stats?.total_cache_tokens || 0) }}</span>
          <svg
            class="h-3.5 w-3.5 text-muted-foreground/70"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span
            class="pointer-events-none absolute left-1/2 top-full z-30 mt-2 w-56 -translate-x-1/2 rounded-lg border border-border bg-popover p-3 text-left text-xs text-popover-foreground opacity-0 shadow-lg transition-opacity duration-150 group-hover:opacity-100 group-focus:opacity-100"
          >
            <span class="mb-2 block font-medium text-foreground">
              {{ cacheDetailLabel() }}
            </span>
            <span class="flex items-center justify-between gap-3">
              <span class="text-muted-foreground">{{ t('usage.cacheCreationTokensLabel') }}</span>
              <span class="tabular-nums">
                {{ formatTokens(stats?.total_cache_creation_tokens || 0) }}
              </span>
            </span>
            <span class="mt-1 flex items-center justify-between gap-3">
              <span class="text-muted-foreground">{{ t('usage.cacheReadTokensLabel') }}</span>
              <span class="tabular-nums">
                {{ formatTokens(stats?.total_cache_read_tokens || 0) }}
              </span>
            </span>
          </span>
        </span>
      </p>
    </div>

    <!-- Cost -->
    <div class="card p-5">
      <div class="flex items-center justify-between">
        <span class="stat-label">{{ t('usage.totalCost') }}</span>
        <span class="stat-icon stat-icon-success">
          <Icon name="dollar" size="sm" />
        </span>
      </div>
      <p class="stat-value mt-3" style="color: var(--nm-success-text)">
        ${{ (stats?.total_actual_cost || 0).toFixed(4) }}
      </p>
      <p class="mt-1 text-xs text-muted-foreground">
        <template v-if="showAccountCost && totalAccountCost != null">
          <span class="text-semantic-warning">{{ t('usage.accountCost') }} ${{ totalAccountCost.toFixed(4) }}</span>
          <span> · </span>
        </template>
        <span>
          {{ t('usage.standardCost') }}
          <span :class="{ 'line-through': strikeStandardCost }">${{ (stats?.total_cost || 0).toFixed(4) }}</span>
        </span>
      </p>
    </div>

    <!-- Avg duration -->
    <div class="card p-5">
      <div class="flex items-center justify-between">
        <span class="stat-label">{{ t('usage.avgDuration') }}</span>
        <span class="stat-icon stat-icon-warning">
          <Icon name="clock" size="sm" />
        </span>
      </div>
      <p class="stat-value mt-3">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'
import type { UsageStatsResponse } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(defineProps<{
  stats: (AdminUsageStatsResponse | UsageStatsResponse) | null
  showAccountCost?: boolean
  strikeStandardCost?: boolean
}>(), {
  showAccountCost: true,
  strikeStandardCost: false,
})

const { t } = useI18n()

const totalAccountCost = computed(() => {
  const stats = props.stats as (AdminUsageStatsResponse & { total_account_cost?: number }) | null
  return stats?.total_account_cost ?? null
})
const showAccountCost = computed(() => props.showAccountCost)
const strikeStandardCost = computed(() => props.strikeStandardCost)

const formatDuration = (ms: number) =>
  ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`

const formatTokens = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + 'B'
  if (value >= 1e6) return (value / 1e6).toFixed(2) + 'M'
  if (value >= 1e3) return (value / 1e3).toFixed(2) + 'K'
  return value.toLocaleString()
}

const cacheLabel = () => t('usage.cacheTotal')
const cacheDetailLabel = () => t('usage.cacheBreakdown')
</script>
