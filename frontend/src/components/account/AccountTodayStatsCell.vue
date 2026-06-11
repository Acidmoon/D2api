<template>
  <div>
    <!-- Loading state -->
    <div v-if="props.loading && !props.stats" class="today-stats-skeleton-stack">
      <div class="today-stats-skeleton today-stats-skeleton--sm"></div>
      <div class="today-stats-skeleton today-stats-skeleton--md"></div>
      <div class="today-stats-skeleton today-stats-skeleton--xs"></div>
    </div>

    <!-- Error state -->
    <div v-else-if="props.error && !props.stats" class="today-stats-error" role="alert">
      {{ props.error }}
    </div>

    <!-- Stats data -->
    <div v-else-if="props.stats" class="today-stats-list">
      <!-- Requests -->
      <div class="today-stats-row">
        <span class="today-stats-label"
          >{{ t('admin.accounts.stats.requests') }}:</span
        >
        <span class="today-stats-value">{{
          formatNumber(props.stats.requests)
        }}</span>
      </div>
      <!-- Tokens -->
      <div class="today-stats-row">
        <span class="today-stats-label"
          >{{ t('admin.accounts.stats.tokens') }}:</span
        >
        <span class="today-stats-value">{{
          formatTokens(props.stats.tokens)
        }}</span>
      </div>
      <!-- Cost (Account) -->
      <div class="today-stats-row">
        <span class="today-stats-label">{{ t('usage.accountBilled') }}:</span>
        <span class="today-stats-cost">{{
          formatCurrency(props.stats.cost)
        }}</span>
      </div>
      <!-- Cost (User/API Key) -->
      <div v-if="props.stats.user_cost != null" class="today-stats-row">
        <span class="today-stats-label">{{ t('usage.userBilled') }}:</span>
        <span class="today-stats-value">{{
          formatCurrency(props.stats.user_cost)
        }}</span>
      </div>
    </div>

    <!-- No data -->
    <div v-else class="today-stats-empty">-</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
import { formatNumber, formatCurrency } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    stats?: WindowStats | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    stats: null,
    loading: false,
    error: null
  }
)

const { t } = useI18n()

// Format large token numbers (e.g., 1234567 -> 1.23M)
const formatTokens = (tokens: number): string => {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(2)}M`
  } else if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`
  }
  return tokens.toString()
}
</script>

<style scoped>
.today-stats-skeleton-stack,
.today-stats-list {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.today-stats-skeleton {
  height: 0.75rem;
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface-alt);
  animation: today-stats-pulse 1.35s ease-in-out infinite;
}

.today-stats-skeleton--xs {
  width: 2.5rem;
}

.today-stats-skeleton--sm {
  width: 3rem;
}

.today-stats-skeleton--md {
  width: 4rem;
}

.today-stats-error,
.today-stats-empty,
.today-stats-list {
  font-size: 0.75rem;
  line-height: 1.25;
}

.today-stats-error {
  color: var(--nm-danger-text);
}

.today-stats-row {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-width: 0;
}

.today-stats-label {
  color: var(--nm-ink-faint);
}

.today-stats-value,
.today-stats-cost {
  color: var(--nm-ink-muted);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.today-stats-cost {
  color: var(--nm-success-text);
}

.today-stats-empty {
  color: var(--nm-ink-faint);
}

@keyframes today-stats-pulse {
  0% {
    opacity: 0.52;
  }

  50% {
    opacity: 1;
  }

  100% {
    opacity: 0.52;
  }
}
</style>
