<template>
  <div>
    <!-- Window stats row (above progress bar) -->
    <div
      v-if="windowStats && (windowStats.requests > 0 || windowStats.tokens > 0)"
      class="mb-0.5 flex items-center"
    >
      <div class="usage-stats-row flex items-center gap-1.5 text-[9px]">
        <span class="usage-stat-chip px-1.5 py-0.5">
          {{ formatRequests }} req
        </span>
        <span class="usage-stat-chip px-1.5 py-0.5">
          {{ formatTokens }}
        </span>
        <span class="usage-stat-chip px-1.5 py-0.5" :title="t('usage.accountBilled')">
          A ${{ formatAccountCost }}
        </span>
        <span
          v-if="windowStats?.user_cost != null"
          class="usage-stat-chip px-1.5 py-0.5"
          :title="t('usage.userBilled')"
        >
          U ${{ formatUserCost }}
        </span>
      </div>
    </div>

    <!-- Progress bar row -->
    <div class="usage-bar-row flex items-center gap-1">
      <!-- Label badge (fixed width for alignment) -->
      <span
        :class="['usage-label w-[32px] shrink-0 px-1 text-center text-[10px] font-medium', labelClass]"
        :title="label"
      >
        {{ label }}
      </span>

      <!-- Progress bar container -->
      <div class="usage-track h-1.5 w-8 shrink-0 overflow-hidden">
        <div
          :class="['usage-fill h-full transition-all duration-300', barClass]"
          :style="{ width: barWidth }"
        ></div>
      </div>

      <!-- Percentage -->
      <span :class="['w-[32px] shrink-0 text-right text-[10px] font-medium', textClass]">
        {{ displayPercent }}
      </span>

      <!-- Reset time -->
      <span v-if="shouldShowResetTime" class="usage-reset shrink-0 text-[10px]">
        {{ formatResetTime }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
import { formatCompactNumber } from '@/utils/format'

const props = defineProps<{
  label: string
  utilization: number // Percentage (0-100+)
  resetsAt?: string | null
  color: 'indigo' | 'emerald' | 'purple' | 'amber'
  windowStats?: WindowStats | null
  showNowWhenIdle?: boolean
  remainingCapacity?: boolean
}>()

const { t } = useI18n()

// Reactive clock for countdown — only runs when a reset time is shown,
// to avoid creating many idle timers across large account lists.
const now = ref(new Date())
const { pause: pauseClock, resume: resumeClock } = useIntervalFn(
  () => {
    now.value = new Date()
  },
  60_000,
  { immediate: false },
)
if (props.resetsAt) resumeClock()
watch(
  () => props.resetsAt,
  (val) => {
    if (val) {
      now.value = new Date()
      resumeClock()
    } else {
      pauseClock()
    }
  },
)

// Label background colors
const labelClass = computed(() => {
  const colors = {
    indigo: 'usage-label-info',
    emerald: 'usage-label-success',
    purple: 'usage-label-accent',
    amber: 'usage-label-warning'
  }
  return colors[props.color]
})

// Progress bar color based on utilization
const barClass = computed(() => {
  if (props.remainingCapacity) {
    if (props.utilization <= 20) {
      return 'bg-red-500'
    } else if (props.utilization <= 50) {
      return 'bg-amber-500'
    }
    return 'bg-green-500'
  }
  if (props.utilization >= 100) {
    // 保留上游的 bg-red-500 语义类名（测试依赖），实际配色由 usage-fill-danger 的 scoped 样式覆盖
    return 'usage-fill-danger bg-red-500'
  } else if (props.utilization >= 80) {
    return 'usage-fill-warning'
  } else {
    return 'usage-fill-success'
  }
})

// Text color based on utilization
const textClass = computed(() => {
  if (props.remainingCapacity) {
    if (props.utilization <= 20) {
      return 'text-red-600 dark:text-red-400'
    } else if (props.utilization <= 50) {
      return 'text-amber-600 dark:text-amber-400'
    }
    return 'text-gray-600 dark:text-gray-400'
  }
  if (props.utilization >= 100) {
    return 'usage-text-danger'
  } else if (props.utilization >= 80) {
    return 'usage-text-warning'
  } else {
    return 'usage-text-muted'
  }
})

// Bar width (capped at 100%)
const barWidth = computed(() => {
  return `${Math.min(Math.max(props.utilization, 0), 100)}%`
})

// Display percentage (cap at 999% for readability)
const displayPercent = computed(() => {
  const percent = Math.round(
    props.remainingCapacity
      ? Math.min(Math.max(props.utilization, 0), 100)
      : props.utilization
  )
  return percent > 999 ? '>999%' : `${percent}%`
})

const shouldShowResetTime = computed(() => {
  if (props.resetsAt) return true
  return Boolean(props.showNowWhenIdle && props.utilization <= 0)
})

// Format reset time
const formatResetTime = computed(() => {
  // For rolling windows, when utilization is 0%, treat as immediately available.
  if (props.showNowWhenIdle && props.utilization <= 0) {
    return t('usage.resetNow')
  }

  if (!props.resetsAt) return '-'

  const date = new Date(props.resetsAt)
  const diffMs = date.getTime() - now.value.getTime()

  // resetsAt 已过期：utilization>0 说明后端窗口数据还没刷新（active poll 没回写），
  // 显示「待刷新」以区别于真正可用的「现在」。
  if (diffMs <= 0) {
    return props.utilization > 0 ? t('usage.resetPending') : t('usage.resetNow')
  }

  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffMins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))

  if (diffHours >= 24) {
    const days = Math.floor(diffHours / 24)
    return `${days}d ${diffHours % 24}h`
  } else if (diffHours > 0) {
    return `${diffHours}h ${diffMins}m`
  } else {
    return `${diffMins}m`
  }
})

// Window stats formatters
const formatRequests = computed(() => {
  if (!props.windowStats) return ''
  return formatCompactNumber(props.windowStats.requests, { allowBillions: false })
})

const formatTokens = computed(() => {
  if (!props.windowStats) return ''
  return formatCompactNumber(props.windowStats.tokens)
})

const formatAccountCost = computed(() => {
  if (!props.windowStats) return '0.00'
  return props.windowStats.cost.toFixed(2)
})

const formatUserCost = computed(() => {
  if (!props.windowStats || props.windowStats.user_cost == null) return '0.00'
  return props.windowStats.user_cost.toFixed(2)
})

</script>

<style scoped>
.usage-stats-row {
  color: var(--nm-ink-faint);
}

.usage-bar-row,
.usage-stats-row {
  flex-wrap: nowrap;
  white-space: nowrap;
}

.usage-stat-chip,
.usage-label,
.usage-track {
  border-radius: var(--nm-radius-sm);
}

.usage-stat-chip {
  border: 1px solid var(--nm-border-light);
  background: var(--nm-surface-soft);
  color: var(--nm-ink-muted);
}

.usage-label {
  border: 1px solid transparent;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.usage-label-info {
  border-color: color-mix(in srgb, var(--nm-info-text) 22%, var(--nm-border-light));
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.usage-label-success {
  border-color: color-mix(in srgb, var(--nm-success-text) 22%, var(--nm-border-light));
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.usage-label-accent {
  border-color: color-mix(in srgb, var(--nm-accent-text) 22%, var(--nm-border-light));
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.usage-label-warning {
  border-color: color-mix(in srgb, var(--nm-warning-text) 22%, var(--nm-border-light));
  background: var(--nm-warning-soft);
  color: var(--nm-warning-text);
}

.usage-track {
  background: var(--nm-surface-soft);
  border: 1px solid var(--nm-border-light);
}

.usage-fill {
  min-width: 1px;
}

.usage-fill-success {
  background: var(--nm-success);
}

.usage-fill-warning {
  background: var(--nm-warning);
}

.usage-fill-danger {
  background: var(--nm-danger);
}

.usage-text-muted {
  color: var(--nm-ink-muted);
}

.usage-text-warning {
  color: var(--nm-warning-text);
}

.usage-text-danger {
  color: var(--nm-danger-text);
}

.usage-reset {
  color: var(--nm-ink-faint);
  white-space: nowrap;
}
</style>
