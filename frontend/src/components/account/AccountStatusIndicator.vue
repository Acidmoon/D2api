<template>
  <div class="flex items-center gap-2">
    <!-- Rate Limit Display (429) - Two-line layout -->
    <div v-if="isRateLimited" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-warning">{{ t('admin.accounts.status.rateLimited') }}</span>
      <span class="status-meta">{{ rateLimitResumeText }}</span>
    </div>

    <!-- Overload Display (529) - Two-line layout -->
    <div v-else-if="isOverloaded" class="flex flex-col items-center gap-1">
      <span class="badge text-xs badge-danger">{{ t('admin.accounts.status.overloaded') }}</span>
      <span class="status-meta">{{ overloadCountdown }}</span>
    </div>

    <!-- Main Status Badge (shown when not rate limited/overloaded) -->
    <template v-else>
      <div v-if="isTempUnschedulable" class="flex flex-col items-center gap-1">
        <button
          type="button"
          :class="['badge text-xs', statusClass, 'cursor-pointer']"
          :title="t('admin.accounts.status.viewTempUnschedDetails')"
          @click="handleTempUnschedClick"
        >
          {{ statusText }}
        </button>
        <span class="max-w-[180px] text-center text-[11px] leading-4 text-gray-500 dark:text-gray-400">
          {{ tempUnschedRecoveryText }}
        </span>
      </div>
      <span v-else :class="['badge text-xs', statusClass]">
        {{ statusText }}
      </span>
    </template>

    <!-- Error Info Indicator -->
    <div v-if="hasError && account.error_message" class="group/error relative">
      <svg
        class="status-error-icon h-4 w-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z"
        />
      </svg>
      <!-- Tooltip - 向下显示 -->
      <div class="status-tooltip status-tooltip--error">
        <div class="status-tooltip-content">
          {{ account.error_message }}
        </div>
        <!-- 上方小三角 -->
        <div class="status-tooltip-arrow status-tooltip-arrow--top"></div>
      </div>
    </div>

    <!-- Rate Limit Indicator (429) -->
    <div v-if="isRateLimited" class="group relative">
      <span class="status-chip status-chip--warning">
        <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
        429
      </span>
      <!-- Tooltip -->
      <div class="status-tooltip status-tooltip--above">
        {{ t('admin.accounts.status.rateLimitedUntil', { time: formatDateTime(account.rate_limit_reset_at) }) }}
        <div class="status-tooltip-arrow status-tooltip-arrow--bottom"></div>
      </div>
    </div>

    <!-- Model Status Indicators (普通限流 / 超量请求中) -->
    <div
      v-if="activeModelStatuses.length > 0"
      :class="[
        activeModelStatuses.length <= 4
          ? 'flex flex-col gap-1'
          : activeModelStatuses.length <= 8
            ? 'columns-2 gap-x-2'
            : 'columns-3 gap-x-2'
      ]"
    >
      <div v-for="item in activeModelStatuses" :key="`${item.kind}-${item.model}`" class="group relative mb-1 break-inside-avoid">
        <!-- 积分已用尽 -->
        <span v-if="item.kind === 'credits_exhausted'" class="status-chip status-chip--danger">
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ t('admin.accounts.status.creditsExhausted') }}
          <span class="status-chip-time">{{ formatCountdown(item.reset_at) }}</span>
        </span>
        <!-- 正在走积分（模型限流但积分可用）-->
        <span v-else-if="item.kind === 'credits_active'" class="status-chip status-chip--warning">
          <span aria-hidden="true">⚡</span>
          {{ formatScopeName(item.model) }}
          <span class="status-chip-time">{{ formatCountdown(item.reset_at) }}</span>
        </span>
        <!-- 普通模型限流 -->
        <span v-else class="status-chip status-chip--info">
          <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
          {{ formatScopeName(item.model) }}
          <span class="status-chip-time">{{ formatCountdown(item.reset_at) }}</span>
        </span>
        <!-- Tooltip -->
        <div class="status-tooltip status-tooltip--above">
          {{
            item.kind === 'credits_exhausted'
              ? t('admin.accounts.status.creditsExhaustedUntil', { time: formatDateTimeToMinute(item.reset_at) })
              : item.kind === 'credits_active'
                ? t('admin.accounts.status.modelCreditOveragesUntil', { model: formatScopeName(item.model), time: formatDateTimeToMinute(item.reset_at) })
                : t('admin.accounts.status.modelRateLimitedUntil', { model: formatScopeName(item.model), time: formatDateTimeToMinute(item.reset_at) })
          }}
          <div class="status-tooltip-arrow status-tooltip-arrow--bottom"></div>
        </div>
      </div>
    </div>

    <!-- Overload Indicator (529) -->
    <div v-if="isOverloaded" class="group relative">
      <span class="status-chip status-chip--danger">
        <Icon name="exclamationTriangle" size="xs" :stroke-width="2" />
        529
      </span>
      <!-- Tooltip -->
      <div class="status-tooltip status-tooltip--above">
        {{ t('admin.accounts.status.overloadedUntil', { time: formatTime(account.overload_until) }) }}
        <div class="status-tooltip-arrow status-tooltip-arrow--bottom"></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { Account } from '@/types'
import { formatCountdown, formatDateTime, formatDateTimeToMinute, formatCountdownWithSuffix, formatTime } from '@/utils/format'

const { t } = useI18n()

const props = defineProps<{
  account: Account
}>()

const emit = defineEmits<{
  (e: 'show-temp-unsched', account: Account): void
}>()

// Computed: is rate limited (429)
const isRateLimited = computed(() => {
  if (!props.account.rate_limit_reset_at) return false
  return new Date(props.account.rate_limit_reset_at) > new Date()
})

type AccountModelStatusItem = {
  kind: 'rate_limit' | 'credits_exhausted' | 'credits_active'
  model: string
  reset_at: string
}

// Computed: active model statuses (普通模型限流 + 积分耗尽 + 走积分中)
const activeModelStatuses = computed<AccountModelStatusItem[]>(() => {
  const extra = props.account.extra as Record<string, unknown> | undefined
  const modelLimits = extra?.model_rate_limits as
    | Record<string, { rate_limited_at: string; rate_limit_reset_at: string }>
    | undefined
  const now = new Date()
  const items: AccountModelStatusItem[] = []

  if (!modelLimits) return items

  // 检查 AICredits key 是否生效（积分是否耗尽）
  const aiCreditsEntry = modelLimits['AICredits']
  const hasActiveAICredits = aiCreditsEntry && new Date(aiCreditsEntry.rate_limit_reset_at) > now
  const allowOverages = !!(extra?.allow_overages)

  for (const [model, info] of Object.entries(modelLimits)) {
    if (new Date(info.rate_limit_reset_at) <= now) continue

    if (model === 'AICredits') {
      // AICredits key → 积分已用尽
      items.push({ kind: 'credits_exhausted', model, reset_at: info.rate_limit_reset_at })
    } else if (allowOverages && !hasActiveAICredits) {
      // 普通模型限流 + overages 启用 + 积分可用 → 正在走积分
      items.push({ kind: 'credits_active', model, reset_at: info.rate_limit_reset_at })
    } else {
      // 普通模型限流
      items.push({ kind: 'rate_limit', model, reset_at: info.rate_limit_reset_at })
    }
  }

  return items
})

const formatScopeName = (scope: string): string => {
  const aliases: Record<string, string> = {
    // Claude 系列
    'claude-fable-5': 'CFable5',
    'claude-opus-4-6': 'COpus46',
    'claude-opus-4-6-thinking': 'COpus46T',
    'claude-opus-4-7': 'COpus47',
    'claude-opus-4-8': 'COpus48',
    'claude-opus-5': 'COpus5',
    'claude-sonnet-4-6': 'CSon46',
    'claude-sonnet-4-5': 'CSon45',
    'claude-sonnet-4-5-thinking': 'CSon45T',
    // Gemini 2.5 系列
    'gemini-2.5-flash': 'G25F',
    'gemini-2.5-flash-lite': 'G25FL',
    'gemini-2.5-flash-thinking': 'G25FT',
    'gemini-2.5-pro': 'G25P',
    'gemini-2.5-flash-image': 'G25I',
    // Gemini 3.5 系列
    'gemini-3.5-flash': 'G35F',
    // Gemini 3 系列
    'gemini-3-flash': 'G3F',
    'gemini-3.1-pro-high': 'G3PH',
    'gemini-3.1-pro-low': 'G3PL',
    'gemini-3-pro-image': 'G3PI',
    'gemini-3.1-flash-image': 'G31FI',
    // 其他
    'gpt-oss-120b-medium': 'GPT120',
    'tab_flash_lite_preview': 'TabFL',
    // 旧版 scope 别名（兼容）
    claude: 'Claude',
    claude_sonnet: 'CSon',
    claude_opus: 'COpus',
    claude_haiku: 'CHaiku',
    gemini_text: 'Gemini',
    gemini_image: 'GImg',
    gemini_flash: 'GFlash',
    gemini_pro: 'GPro',
  }
  return aliases[scope] || scope
}

// Computed: is overloaded (529)
const isOverloaded = computed(() => {
  if (!props.account.overload_until) return false
  return new Date(props.account.overload_until) > new Date()
})

// Computed: is temp unschedulable
const isTempUnschedulable = computed(() => {
  if (!props.account.temp_unschedulable_until) return false
  return new Date(props.account.temp_unschedulable_until) > new Date()
})

// Computed: has error status
const hasError = computed(() => {
  return props.account.status === 'error'
})

const isQuotaExceeded = computed(() => {
  const exceeded = (used?: number | null, limit?: number | null) =>
    typeof limit === 'number' && limit > 0 && typeof used === 'number' && used >= limit
  return (
    exceeded(props.account.quota_used, props.account.quota_limit) ||
    exceeded(props.account.quota_daily_used, props.account.quota_daily_limit) ||
    exceeded(props.account.quota_weekly_used, props.account.quota_weekly_limit)
  )
})

// Computed: countdown text for rate limit (429)
const rateLimitCountdown = computed(() => {
  return formatCountdown(props.account.rate_limit_reset_at)
})

const rateLimitResumeText = computed(() => {
  if (!rateLimitCountdown.value) return ''
  return t('admin.accounts.status.rateLimitedAutoResume', { time: rateLimitCountdown.value })
})

// Computed: countdown text for overload (529)
const overloadCountdown = computed(() => {
  return formatCountdownWithSuffix(props.account.overload_until)
})

const tempUnschedRecoveryText = computed(() => {
  if (!isTempUnschedulable.value || !props.account.temp_unschedulable_until) return ''
  return t('admin.accounts.status.tempUnschedulableUntil', {
    time: formatDateTime(props.account.temp_unschedulable_until)
  })
})

// Computed: status badge class
const statusClass = computed(() => {
  if (hasError.value) {
    return 'badge-danger'
  }
  if (isTempUnschedulable.value) {
    return 'badge-warning'
  }
  if (props.account.status !== 'active') {
    return props.account.status === 'error' ? 'badge-danger' : 'badge-gray'
  }
  if (isQuotaExceeded.value) {
    return 'badge-warning'
  }
  if (!props.account.schedulable) {
    return 'badge-gray'
  }
  return 'badge-success'
})

// Computed: status text
const statusText = computed(() => {
  if (hasError.value) {
    return t('admin.accounts.status.error')
  }
  if (isTempUnschedulable.value) {
    return t('admin.accounts.status.tempUnschedulable')
  }
  if (props.account.status !== 'active') {
    return t(`admin.accounts.status.${props.account.status}`)
  }
  if (isQuotaExceeded.value) {
    return t('admin.accounts.status.quotaExceeded')
  }
  if (!props.account.schedulable) {
    return t('admin.accounts.status.paused')
  }
  return t(`admin.accounts.status.${props.account.status}`)
})

const handleTempUnschedClick = () => {
  if (!isTempUnschedulable.value) return
  emit('show-temp-unsched', props.account)
}
</script>

<style scoped>
.status-meta {
  color: var(--nm-ink-faint);
  font-size: 0.6875rem;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.status-error-icon {
  cursor: help;
  color: var(--nm-danger-text);
  transition: color 160ms ease;
}

.status-error-icon:hover {
  color: var(--nm-danger);
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  border: 1px solid currentColor;
  border-radius: var(--nm-radius-sm);
  padding: 0.125rem 0.375rem;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}

.status-chip--danger {
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}

.status-chip--warning {
  background: var(--nm-warning-soft);
  color: var(--nm-warning-text);
}

.status-chip--info {
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.status-chip-time {
  font-size: 0.625rem;
  font-weight: 500;
  opacity: 0.76;
}

.status-tooltip {
  pointer-events: none;
  position: absolute;
  z-index: 100;
  width: 14rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  background: var(--nm-ink);
  padding: 0.5rem 0.75rem;
  color: var(--nm-bg);
  font-size: 0.75rem;
  line-height: 1.5;
  opacity: 0;
  transition: opacity 160ms ease;
}

.status-tooltip--above {
  bottom: 100%;
  left: 50%;
  margin-bottom: 0.5rem;
  transform: translateX(-50%);
  text-align: center;
}

.status-tooltip--error {
  left: 0;
  top: 100%;
  margin-top: 0.375rem;
  min-width: 12.5rem;
  max-width: 18.75rem;
  width: max-content;
}

.group:hover .status-tooltip,
.group\/error:hover .status-tooltip {
  opacity: 1;
}

.status-tooltip-content {
  white-space: pre-wrap;
  overflow-wrap: break-word;
}

.status-tooltip-arrow {
  position: absolute;
  width: 0;
  height: 0;
}

.status-tooltip-arrow--top {
  bottom: 100%;
  left: 0.75rem;
  border-right: 0.375rem solid transparent;
  border-bottom: 0.375rem solid var(--nm-ink);
  border-left: 0.375rem solid transparent;
}

.status-tooltip-arrow--bottom {
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border-top: 0.25rem solid var(--nm-ink);
  border-right: 0.25rem solid transparent;
  border-left: 0.25rem solid transparent;
}
</style>
