<template>
  <div v-if="hasActiveSubscriptions" class="relative" ref="containerRef">
    <!-- Mini Progress Display -->
    <button
      @click="toggleTooltip"
      class="subscription-trigger flex cursor-pointer items-center gap-2 px-3 py-1.5 transition-colors"
      :title="t('subscriptionProgress.viewDetails')"
    >
      <Icon name="creditCard" size="sm" style="color: var(--nm-accent-text)" />
      <div class="flex items-center gap-1.5">
        <!-- Combined progress indicator -->
        <div class="flex items-center gap-0.5">
          <div
            v-for="(sub, index) in displaySubscriptions.slice(0, 3)"
            :key="index"
            class="subscription-dot h-2 w-2"
            :class="getProgressDotClass(sub)"
          ></div>
        </div>
        <span class="text-xs font-medium" style="color: var(--nm-accent-text)">
          {{ activeSubscriptions.length }}
        </span>
      </div>
    </button>

    <!-- Hover/Click Tooltip -->
    <transition name="dropdown">
      <div
        v-if="tooltipOpen"
        class="dropdown right-0 mt-2 w-[340px] overflow-hidden"
      >
        <div class="border-b p-3" style="border-color: var(--nm-border-light)">
          <h3 class="text-sm font-semibold" style="color: var(--nm-ink)">
            {{ t('subscriptionProgress.title') }}
          </h3>
          <p class="mt-0.5 text-xs" style="color: var(--nm-ink-muted)">
            {{ t('subscriptionProgress.activeCount', { count: activeSubscriptions.length }) }}
          </p>
        </div>

        <div class="max-h-64 overflow-y-auto">
          <div
            v-for="subscription in displaySubscriptions"
            :key="subscription.id"
            class="border-b p-3 last:border-b-0"
            style="border-color: var(--nm-border-light)"
          >
            <div class="mb-2 flex items-center justify-between">
              <span class="text-sm font-medium" style="color: var(--nm-ink)">
                {{ subscription.plan_name || subscription.group?.name || `Subscription #${subscription.id}` }}
              </span>
              <span
                v-if="subscription.expires_at"
                class="text-xs"
                :class="getDaysRemainingClass(subscription.expires_at)"
              >
                {{ formatDaysRemaining(subscription.expires_at) }}
              </span>
            </div>

            <!-- Progress bars or Unlimited badge -->
            <div class="space-y-1.5">
              <!-- Unlimited subscription badge -->
              <div
                v-if="isUnlimited(subscription)"
                class="subscription-unlimited flex items-center gap-2 px-2.5 py-1.5"
              >
                <span class="text-lg" style="color: var(--nm-success-text)">∞</span>
                <span class="text-xs font-medium" style="color: var(--nm-success-text)">
                  {{ t('subscriptionProgress.unlimited') }}
                </span>
              </div>

              <!-- Progress bars for limited subscriptions -->
              <template v-else>
                <div v-if="subscriptionLimit(subscription, 'daily')" class="flex items-center gap-2">
                  <span class="w-8 flex-shrink-0 text-[10px]" style="color: var(--nm-ink-muted)">{{
                    t('subscriptionProgress.daily')
                  }}</span>
                  <div class="metric-progress h-1.5 min-w-0 flex-1">
                    <div
                      class="metric-progress-bar h-full transition-all"
                      :class="
                        getProgressBarClass(
                          subscription.daily_usage_usd,
                          subscriptionLimit(subscription, 'daily')
                        )
                      "
                      :style="{
                        width: getProgressWidth(
                          subscription.daily_usage_usd,
                          subscriptionLimit(subscription, 'daily')
                        )
                      }"
                    ></div>
                  </div>
                  <span class="w-24 flex-shrink-0 text-right text-[10px]" style="color: var(--nm-ink-muted)">
                    {{
                      formatUsage(subscription.daily_usage_usd, subscriptionLimit(subscription, 'daily'))
                    }}
                  </span>
                </div>

                <div v-if="subscriptionLimit(subscription, 'weekly')" class="flex items-center gap-2">
                  <span class="w-8 flex-shrink-0 text-[10px]" style="color: var(--nm-ink-muted)">{{
                    t('subscriptionProgress.weekly')
                  }}</span>
                  <div class="metric-progress h-1.5 min-w-0 flex-1">
                    <div
                      class="metric-progress-bar h-full transition-all"
                      :class="
                        getProgressBarClass(
                          subscription.weekly_usage_usd,
                          subscriptionLimit(subscription, 'weekly')
                        )
                      "
                      :style="{
                        width: getProgressWidth(
                          subscription.weekly_usage_usd,
                          subscriptionLimit(subscription, 'weekly')
                        )
                      }"
                    ></div>
                  </div>
                  <span class="w-24 flex-shrink-0 text-right text-[10px]" style="color: var(--nm-ink-muted)">
                    {{
                      formatUsage(subscription.weekly_usage_usd, subscriptionLimit(subscription, 'weekly'))
                    }}
                  </span>
                </div>

                <div v-if="subscriptionLimit(subscription, 'monthly')" class="flex items-center gap-2">
                  <span class="w-8 flex-shrink-0 text-[10px]" style="color: var(--nm-ink-muted)">{{
                    t('subscriptionProgress.monthly')
                  }}</span>
                  <div class="metric-progress h-1.5 min-w-0 flex-1">
                    <div
                      class="metric-progress-bar h-full transition-all"
                      :class="
                        getProgressBarClass(
                          subscription.monthly_usage_usd,
                          subscriptionLimit(subscription, 'monthly')
                        )
                      "
                      :style="{
                        width: getProgressWidth(
                          subscription.monthly_usage_usd,
                          subscriptionLimit(subscription, 'monthly')
                        )
                      }"
                    ></div>
                  </div>
                  <span class="w-24 flex-shrink-0 text-right text-[10px]" style="color: var(--nm-ink-muted)">
                    {{
                      formatUsage(
                        subscription.monthly_usage_usd,
                        subscriptionLimit(subscription, 'monthly')
                      )
                    }}
                  </span>
                </div>
              </template>
            </div>
          </div>
        </div>

        <div class="border-t p-2" style="border-color: var(--nm-border-light)">
          <router-link
            to="/subscriptions"
            @click="closeTooltip"
            class="subscription-link block w-full py-1 text-center text-xs"
          >
            {{ t('subscriptionProgress.viewAll') }}
          </router-link>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useSubscriptionStore } from '@/stores'
import type { UserSubscription } from '@/types'

const { t } = useI18n()

const subscriptionStore = useSubscriptionStore()

const containerRef = ref<HTMLElement | null>(null)
const tooltipOpen = ref(false)

// Use store data instead of local state
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)
const hasActiveSubscriptions = computed(() => subscriptionStore.hasActiveSubscriptions)

const displaySubscriptions = computed(() => {
  // Sort by most usage (highest percentage first)
  return [...activeSubscriptions.value].sort((a, b) => {
    const aMax = getMaxUsagePercentage(a)
    const bMax = getMaxUsagePercentage(b)
    return bMax - aMax
  })
})

function getMaxUsagePercentage(sub: UserSubscription): number {
  const percentages: number[] = []
  const dailyLimit = subscriptionLimit(sub, 'daily')
  const weeklyLimit = subscriptionLimit(sub, 'weekly')
  const monthlyLimit = subscriptionLimit(sub, 'monthly')
  if (dailyLimit) {
    percentages.push(((sub.daily_usage_usd || 0) / dailyLimit) * 100)
  }
  if (weeklyLimit) {
    percentages.push(((sub.weekly_usage_usd || 0) / weeklyLimit) * 100)
  }
  if (monthlyLimit) {
    percentages.push(((sub.monthly_usage_usd || 0) / monthlyLimit) * 100)
  }
  return percentages.length > 0 ? Math.max(...percentages) : 0
}

function isUnlimited(sub: UserSubscription): boolean {
  return (
    !subscriptionLimit(sub, 'daily') &&
    !subscriptionLimit(sub, 'weekly') &&
    !subscriptionLimit(sub, 'monthly')
  )
}

function subscriptionLimit(
  sub: UserSubscription,
  period: 'daily' | 'weekly' | 'monthly'
): number | null {
  const direct = sub[`${period}_limit_usd` as keyof UserSubscription] as number | null | undefined
  const legacy = sub.group?.[`${period}_limit_usd` as keyof NonNullable<UserSubscription['group']>] as number | null | undefined
  const value = direct ?? legacy ?? null
  return value && value > 0 ? value : null
}

function getProgressDotClass(sub: UserSubscription): string {
  // Unlimited subscriptions get a special color
  if (isUnlimited(sub)) {
    return 'subscription-dot-success'
  }
  const maxPercentage = getMaxUsagePercentage(sub)
  if (maxPercentage >= 90) return 'subscription-dot-danger'
  if (maxPercentage >= 70) return 'subscription-dot-warning'
  return 'subscription-dot-success'
}

function getProgressBarClass(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return ''
  const percentage = ((used || 0) / limit) * 100
  if (percentage >= 90) return 'metric-progress-bar-danger'
  if (percentage >= 70) return 'metric-progress-bar-warning'
  return 'metric-progress-bar-success'
}

function getProgressWidth(used: number | undefined, limit: number | null | undefined): string {
  if (!limit || limit === 0) return '0%'
  const percentage = Math.min(((used || 0) / limit) * 100, 100)
  return `${percentage}%`
}

function formatUsage(used: number | undefined, limit: number | null | undefined): string {
  const usedValue = (used || 0).toFixed(2)
  const limitValue = limit?.toFixed(2) || '∞'
  return `$${usedValue}/$${limitValue}`
}

function formatDaysRemaining(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  if (diff < 0) return t('subscriptionProgress.expired')
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))
  if (days === 0) return t('subscriptionProgress.expiresToday')
  if (days === 1) return t('subscriptionProgress.expiresTomorrow')
  return t('subscriptionProgress.daysRemaining', { days })
}

function getDaysRemainingClass(expiresAt: string): string {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diff = expires.getTime() - now.getTime()
  const days = Math.ceil(diff / (1000 * 60 * 60 * 24))
  if (days <= 3) return 'text-semantic-danger'
  if (days <= 7) return 'text-semantic-warning'
  return 'subscription-muted'
}

function toggleTooltip() {
  tooltipOpen.value = !tooltipOpen.value
}

function closeTooltip() {
  tooltipOpen.value = false
}

function handleClickOutside(event: MouseEvent) {
  if (containerRef.value && !containerRef.value.contains(event.target as Node)) {
    closeTooltip()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  // Trigger initial fetch if not already loaded
  // The actual data loading is handled by App.vue globally
  subscriptionStore.fetchActiveSubscriptions().catch((error) => {
    console.error('Failed to load subscriptions in SubscriptionProgressMini:', error)
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}

.subscription-trigger {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface);
  border-radius: var(--nm-radius);
}

.subscription-trigger:hover {
  background: var(--nm-surface-soft);
}

.subscription-dot {
  border-radius: 999px;
}

.subscription-dot-success {
  background: var(--nm-success);
}

.subscription-dot-warning {
  background: var(--nm-warning);
}

.subscription-dot-danger {
  background: var(--nm-danger);
}

.subscription-unlimited {
  border: 1px solid var(--nm-success);
  background: var(--nm-success-soft);
  border-radius: var(--nm-radius-sm);
}

.subscription-link {
  color: var(--nm-accent-text);
  border-radius: var(--nm-radius-sm);
}

.subscription-link:hover {
  background: var(--nm-surface-soft);
}

.subscription-muted {
  color: var(--nm-ink-muted);
}
</style>
