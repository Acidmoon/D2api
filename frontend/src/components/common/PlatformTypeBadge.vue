<template>
  <div class="inline-flex flex-col gap-0.5 text-xs font-medium">
    <!-- Row 1: Platform + Type -->
    <div class="platform-type-row inline-flex items-center overflow-hidden">
      <span :class="['platform-type-segment inline-flex items-center gap-1 px-2 py-1', platformClass]">
        <PlatformIcon :platform="platform" size="xs" />
        <span>{{ platformLabel }}</span>
      </span>
      <span :class="['platform-type-segment inline-flex items-center gap-1 px-1.5 py-1', typeClass]">
        <!-- OAuth icon -->
        <svg
          v-if="type === 'oauth'"
          class="h-3 w-3"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
          />
        </svg>
        <!-- Setup Token icon -->
        <Icon v-else-if="type === 'setup-token'" name="shield" size="xs" />
        <!-- API Key icon -->
        <Icon v-else-if="type === 'service_account'" name="cloud" size="xs" />
        <Icon v-else name="key" size="xs" />
        <span>{{ typeLabel }}</span>
      </span>
    </div>
    <!-- Row 2: Plan type + Privacy mode (only if either exists) -->
    <div v-if="planLabel || privacyBadge" class="platform-type-row inline-flex items-center overflow-hidden">
      <span v-if="planLabel" :class="['platform-type-segment inline-flex items-center gap-1 px-1.5 py-1', planBadgeClass]">
        <GrokFreeIcon
          v-if="isGrokFreePlan"
          data-testid="grok-free-plan-icon"
        />
        <Icon
          v-else-if="planIconName"
          :name="planIconName"
          size="xs"
          data-testid="grok-plan-icon"
          aria-hidden="true"
        />
        <span>{{ planLabel }}</span>
      </span>
      <span
        v-if="privacyBadge"
        :class="['platform-type-segment inline-flex items-center gap-1 px-1.5 py-1', privacyBadge.class]"
        :title="privacyBadge.title"
      >
        <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" :d="privacyBadge.icon" />
        </svg>
        <span>{{ privacyBadge.label }}</span>
      </span>
    </div>
    <!-- Row 3: Subscription expiration (non-free paid accounts only) -->
    <div v-if="expiresLabel" class="platform-type-expiry pl-0.5 text-[10px] leading-tight" :title="subscriptionExpiresAt">
      {{ expiresLabel }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AccountPlatform, AccountType } from '@/types'
import GrokFreeIcon from './GrokFreeIcon.vue'
import PlatformIcon from './PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

interface Props {
  platform: AccountPlatform
  type: AccountType
  authMode?: string
  planType?: string
  privacyMode?: string
  subscriptionExpiresAt?: string
}

const props = defineProps<Props>()

const platformLabel = computed(() => {
  if (props.platform === 'anthropic') return 'Anthropic'
  if (props.platform === 'openai') return 'OpenAI'
  if (props.platform === 'antigravity') return 'Antigravity'
  if (props.platform === 'grok') return 'Grok'
  return 'Gemini'
})

const normalizedAuthMode = computed(() =>
  (props.authMode || '').trim().toLowerCase().replace(/[\s_-]+/g, '')
)

const typeLabel = computed(() => {
  // OpenAI OAuth 细分认证模式:Agent Identity / PAT 优先于通用 OAuth 标签。
  if (props.platform === 'openai' && props.type === 'oauth') {
    if (normalizedAuthMode.value === 'agentidentity') return 'Agent Identity'
    if (normalizedAuthMode.value === 'personalaccesstoken') return 'PAT'
  }
  switch (props.type) {
    case 'oauth':
      return 'OAuth'
    case 'setup-token':
      return 'Token'
    case 'apikey':
      return 'Key'
    case 'bedrock':
      return 'AWS'
    case 'service_account':
      return 'Vertex'
    default:
      return props.type
  }
})

const normalizedPlanType = computed(() =>
  (props.planType || '').trim().toLowerCase().replace(/[\s_-]+/g, '')
)

const planLabel = computed(() => {
  if (!normalizedPlanType.value) return ''
  switch (normalizedPlanType.value) {
    case 'plus':
      return 'Plus'
    case 'team':
      return 'Team'
    case 'chatgptpro':
    case 'pro':
      return 'Pro'
    case 'free':
    case 'basic':
      return props.platform === 'grok' ? 'Grok Free' : 'Free'
    case 'supergrok':
      return 'SuperGrok'
    case 'supergrokheavy':
      return 'SuperGrok Heavy'
    case 'heavy':
      return 'Heavy'
    case 'abnormal':
      return t('admin.accounts.subscriptionAbnormal')
    default:
      return props.planType
  }
})

const isGrokFreePlan = computed(() =>
  props.platform === 'grok' &&
  (normalizedPlanType.value === 'free' || normalizedPlanType.value === 'basic')
)

const planIconName = computed<'bolt' | null>(() => {
  if (props.platform !== 'grok') return null
  // Paid Grok tiers (SuperGrok / Heavy) share the bolt mark; free uses GrokFreeIcon.
  if (
    normalizedPlanType.value === 'supergrok' ||
    normalizedPlanType.value === 'supergrokheavy' ||
    normalizedPlanType.value === 'heavy' ||
    normalizedPlanType.value.includes('heavy')
  ) {
    return 'bolt'
  }
  return null
})

const platformClass = computed(() => {
  return props.platform === 'anthropic' || props.platform === 'openai' || props.platform === 'antigravity'
    ? 'platform-type-primary'
    : 'platform-type-info'
})

const typeClass = computed(() => {
  return 'platform-type-muted'
})

const planBadgeClass = computed(() => {
  if (normalizedPlanType.value === 'abnormal') {
    return 'platform-type-danger'
  }
  // Free stays muted gray; paid Grok tiers get distinct colors.
  if (normalizedPlanType.value === 'free' || normalizedPlanType.value === 'basic') {
    return 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300'
  }
  if (props.platform === 'grok' && normalizedPlanType.value) {
    // Heavy / SuperGrok Heavy → purple
    if (normalizedPlanType.value.includes('heavy')) {
      return 'bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-300'
    }
    // SuperGrok → cyan
    if (normalizedPlanType.value.includes('supergrok')) {
      return 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300'
    }
    // Any other non-free Grok plan (future tiers) → amber so it still stands out
    return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  }
  // OpenAI / other paid plan labels: keep readable distinction from free gray
  if (normalizedPlanType.value === 'plus') {
    return 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
  }
  if (normalizedPlanType.value === 'team') {
    return 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
  }
  if (normalizedPlanType.value === 'pro' || normalizedPlanType.value === 'chatgptpro') {
    return 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300'
  }
  return typeClass.value
})

// Subscription expiration label (non-free only)
const expiresLabel = computed(() => {
  if (!props.subscriptionExpiresAt || !props.planType) return ''
  if (normalizedPlanType.value === 'free' || normalizedPlanType.value === 'basic') return ''
  try {
    const d = new Date(props.subscriptionExpiresAt)
    if (isNaN(d.getTime())) return ''
    const yyyy = d.getFullYear()
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    return `${t('admin.accounts.subscriptionExpires')} ${yyyy}-${mm}-${dd}`
  } catch {
    return ''
  }
})

// Privacy badge — shows different states for OpenAI/Antigravity OAuth privacy setting
const privacyBadge = computed(() => {
  if (props.type !== 'oauth' || !props.privacyMode) return null
  // 支持 OpenAI 和 Antigravity 平台
  if (props.platform !== 'openai' && props.platform !== 'antigravity') return null

  const shieldCheck = 'M9 12.75L11.25 15 15 9.75m-3-7.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285z'
  const shieldX = 'M12 9v3.75m0-10.036A11.959 11.959 0 013.598 6 11.99 11.99 0 003 9.749c0 5.592 3.824 10.29 9 11.623 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285zM12 18h.008v.008H12V18z'
  switch (props.privacyMode) {
    // OpenAI states
    case 'training_off':
      return { label: 'Private', icon: shieldCheck, title: t('admin.accounts.privacyTrainingOff'), class: 'platform-type-success' }
    case 'training_set_cf_blocked':
      return { label: 'CF', icon: shieldX, title: t('admin.accounts.privacyCfBlocked'), class: 'platform-type-warning' }
    case 'training_set_failed':
      return { label: 'Fail', icon: shieldX, title: t('admin.accounts.privacyFailed'), class: 'platform-type-danger' }
    // Antigravity states
    case 'privacy_set':
      return { label: 'Private', icon: shieldCheck, title: t('admin.accounts.privacyAntigravitySet'), class: 'platform-type-success' }
    case 'privacy_set_failed':
      return { label: 'Fail', icon: shieldX, title: t('admin.accounts.privacyAntigravityFailed'), class: 'platform-type-danger' }
    default:
      return null
  }
})
</script>

<style scoped>
.platform-type-row {
  max-width: 100%;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface);
}

.platform-type-segment {
  border-right: 1px solid var(--nm-border-light);
}

.platform-type-segment:last-child {
  border-right: 0;
}

.platform-type-primary {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.platform-type-info {
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.platform-type-muted {
  background: var(--nm-surface-soft);
  color: var(--nm-ink-muted);
}

.platform-type-success {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.platform-type-warning {
  background: var(--nm-warning-soft);
  color: var(--nm-warning-text);
}

.platform-type-danger {
  background: var(--nm-danger-soft);
  color: var(--nm-danger-text);
}

.platform-type-expiry {
  color: var(--nm-ink-faint);
}
</style>
