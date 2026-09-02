<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <!-- In-content page header (route meta inPageHeader keeps the top bar quiet) -->
      <div class="page-header">
        <h1 class="page-title">{{ t('redeem.title') }}</h1>
        <p class="page-description">{{ t('redeem.description') }}</p>
      </div>

      <div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_360px] lg:items-start">
        <!-- 左侧主体：最近活动（白卡 + 发丝分隔行） -->
        <div class="order-2 min-w-0 lg:order-1">
          <section class="card p-5">
            <h2 class="text-base font-semibold text-foreground">
              {{ t('redeem.recentActivity') }}
            </h2>

            <!-- Loading State -->
            <div v-if="loadingHistory" class="flex items-center justify-center py-10">
              <div class="h-6 w-6 animate-spin rounded-full border-4 border-brand border-t-transparent"></div>
            </div>

            <!-- History List -->
            <div v-else-if="history.length > 0" class="mt-1 divide-y divide-border">
              <div
                v-for="item in history"
                :key="item.id"
                class="flex items-center justify-between gap-4 py-4"
              >
                <div class="flex min-w-0 items-center gap-3">
                  <div class="stat-icon" :class="historyIconClass(item)">
                    <Icon name="dollar" v-if="isBalanceType(item.type)" size="sm" />
                    <Icon name="badge" v-else-if="isSubscriptionType(item.type)" size="sm" />
                    <Icon name="bolt" v-else size="sm" />
                  </div>
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-foreground">
                      {{ getHistoryItemTitle(item) }}
                    </p>
                    <p class="text-xs text-muted-foreground">
                      {{ formatDateTime(item.used_at) }}
                    </p>
                  </div>
                </div>
                <div class="shrink-0 text-right">
                  <p :class="['text-sm font-semibold', historyValueClass(item)]">
                    {{ formatHistoryValue(item) }}
                  </p>
                  <p
                    v-if="!isAdminAdjustment(item.type)"
                    class="font-mono text-xs text-muted-foreground/70"
                  >
                    {{ item.code.slice(0, 8) }}...
                  </p>
                  <p v-else class="text-xs text-muted-foreground">
                    {{ t('redeem.adminAdjustment') }}
                  </p>
                  <!-- Display notes for admin adjustments -->
                  <p
                    v-if="item.notes"
                    class="mt-1 max-w-[200px] truncate text-xs italic text-muted-foreground"
                    :title="item.notes"
                  >
                    {{ item.notes }}
                  </p>
                </div>
              </div>
            </div>

            <!-- Empty State -->
            <div v-else class="empty-state">
              <Icon name="clock" size="xl" class="empty-state-icon" />
              <p class="empty-state-description">
                {{ t('redeem.historyWillAppear') }}
              </p>
            </div>
          </section>
        </div>

        <!-- 右侧栏：兑换表单 + 结果提示 + 余额卡 -->
        <div class="order-1 space-y-5 lg:order-2 lg:sticky lg:top-6">
          <!-- Redeem Form -->
          <section class="card p-5 sm:p-6">
            <form @submit.prevent="handleRedeem" class="space-y-4">
              <div>
                <label for="code" class="input-label">
                  {{ t('redeem.redeemCodeLabel') }}
                </label>
                <div class="relative mt-1">
                  <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
                    <Icon name="gift" size="md" class="text-muted-foreground" />
                  </div>
                  <input
                    id="code"
                    v-model="redeemCode"
                    type="text"
                    required
                    :placeholder="t('redeem.redeemCodePlaceholder')"
                    :disabled="submitting"
                    class="input pl-11"
                  />
                </div>
                <p class="input-hint">
                  {{ t('redeem.redeemCodeHint') }}
                </p>
              </div>

              <button
                type="submit"
                :disabled="!redeemCode || submitting"
                class="btn btn-primary w-full"
              >
                <span
                  v-if="submitting"
                  class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"
                ></span>
                <Icon v-else name="checkCircle" size="sm" />
                {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
              </button>
            </form>
          </section>

          <!-- Success Message -->
          <transition name="fade">
            <div
              v-if="redeemResult"
              class="rounded-xl px-4 py-4"
              style="background: var(--nm-success-soft)"
            >
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" style="background: hsl(var(--card))">
                  <Icon name="checkCircle" size="md" class="text-semantic-success" />
                </div>
                <div class="min-w-0 flex-1">
                  <h3 class="text-sm font-semibold text-semantic-success">
                    {{ t('redeem.redeemSuccess') }}
                  </h3>
                  <div class="mt-2 space-y-1 text-sm text-semantic-success opacity-90">
                    <p>{{ redeemResult.message }}</p>
                    <p v-if="redeemResult.type === 'balance'" class="font-medium">
                      {{ t('redeem.added') }}: ${{ redeemResult.value.toFixed(2) }}
                    </p>
                    <p v-else-if="redeemResult.type === 'concurrency'" class="font-medium">
                      {{ t('redeem.added') }}: {{ redeemResult.value }}
                      {{ t('redeem.concurrentRequests') }}
                    </p>
                    <p v-else-if="redeemResult.type === 'subscription'" class="font-medium">
                      {{ t('redeem.subscriptionAssigned') }}
                      <span v-if="redeemResult.group_name"> - {{ redeemResult.group_name }}</span>
                      <span v-if="redeemResult.validity_days">
                        ({{
                          t('redeem.subscriptionDays', { days: redeemResult.validity_days })
                        }})</span
                      >
                    </p>
                    <p v-if="redeemResult.new_balance !== undefined">
                      {{ t('redeem.newBalance') }}:
                      <span class="font-semibold">${{ redeemResult.new_balance.toFixed(2) }}</span>
                    </p>
                    <p v-if="redeemResult.new_concurrency !== undefined">
                      {{ t('redeem.newConcurrency') }}:
                      <span class="font-semibold"
                        >{{ redeemResult.new_concurrency }} {{ t('redeem.requests') }}</span
                      >
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </transition>

          <!-- Error Message -->
          <transition name="fade">
            <div
              v-if="errorMessage"
              class="rounded-xl px-4 py-4"
              style="background: var(--nm-danger-soft)"
            >
              <div class="flex items-start gap-3">
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" style="background: hsl(var(--card))">
                  <Icon name="exclamationCircle" size="md" class="text-semantic-danger" />
                </div>
                <div class="min-w-0 flex-1">
                  <h3 class="text-sm font-semibold text-semantic-danger">
                    {{ t('redeem.redeemFailed') }}
                  </h3>
                  <p class="mt-2 text-sm break-words text-semantic-danger opacity-90">
                    {{ errorMessage }}
                  </p>
                </div>
              </div>
            </div>
          </transition>

          <!-- Current Balance Card (console stat card) -->
          <section class="card p-5">
            <div class="flex items-center justify-between">
              <span class="stat-label">{{ t('redeem.currentBalance') }}</span>
              <span class="stat-icon stat-icon-primary">
                <Icon name="creditCard" size="sm" />
              </span>
            </div>
            <p class="stat-value mt-3 text-3xl">
              ${{ user?.balance?.toFixed(2) || '0.00' }}
            </p>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ t('redeem.concurrency') }}: {{ user?.concurrency || 0 }} {{ t('redeem.requests') }}
            </p>
          </section>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { redeemAPI, type RedeemHistoryItem } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<{
  message: string
  type: string
  value: number
  new_balance?: number
  new_concurrency?: number
  group_name?: string
  validity_days?: number
} | null>(null)
const errorMessage = ref('')

// History data
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)

// Helper functions for history display
const isBalanceType = (type: string) => {
  return type === 'balance' || type === 'admin_balance'
}

const isSubscriptionType = (type: string) => {
  return type === 'subscription'
}

const isAdminAdjustment = (type: string) => {
  return type === 'admin_balance' || type === 'admin_concurrency'
}

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') {
    return t('redeem.balanceAddedRedeem')
  } else if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  } else if (item.type === 'concurrency') {
    return t('redeem.concurrencyAddedRedeem')
  } else if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  } else if (item.type === 'subscription') {
    return t('redeem.subscriptionAssigned')
  }
  return t('common.unknown')
}

// Console semantic tones: green gain / red loss / indigo subscription / info & warning concurrency
const historyValueClass = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0 ? 'text-semantic-success' : 'text-semantic-danger'
  }
  if (isSubscriptionType(item.type)) {
    return 'text-brand'
  }
  return item.value >= 0 ? 'text-semantic-info' : 'text-semantic-warning'
}

const historyIconClass = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    return item.value >= 0 ? 'stat-icon-success' : 'stat-icon-danger'
  }
  if (isSubscriptionType(item.type)) {
    return 'stat-icon-primary'
  }
  return item.value >= 0 ? 'stat-icon-primary' : 'stat-icon-warning'
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  } else if (isSubscriptionType(item.type)) {
    // 订阅类型显示有效天数和分组名称
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  } else {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}${item.value} ${t('redeem.requests')}`
  }
}

const fetchHistory = async () => {
  loadingHistory.value = true
  try {
    history.value = await redeemAPI.getHistory()
  } catch (error) {
    console.error('Failed to fetch history:', error)
  } finally {
    loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (!redeemCode.value.trim()) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(redeemCode.value.trim())

    redeemResult.value = result

    // Refresh user data to get updated balance/concurrency
    await authStore.refreshUser()

    // If subscription type, immediately refresh subscription status
    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true) // force refresh
      } catch (error) {
        console.error('Failed to refresh subscriptions after redeem:', error)
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    // Clear the input
    redeemCode.value = ''

    // Refresh history
    await fetchHistory()

    // Show success toast
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error: any) {
    errorMessage.value = error.response?.data?.detail || t('redeem.failedToRedeem')

    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchHistory()
})
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
