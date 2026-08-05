<template>
  <header class="sticky top-0 z-30 border-b border-border bg-background">
    <div class="flex h-16 items-center justify-between gap-2 px-2 sm:px-4 md:px-6">
      <!-- Left: Mobile Menu Toggle + Page Title -->
      <div class="flex shrink-0 items-center gap-2 sm:gap-4">
        <button
          @click="toggleMobileSidebar"
          class="btn-ghost btn-icon lg:hidden"
          :aria-label="t('common.toggleMenu')"
        >
          <Icon name="menu" size="md" />
        </button>

        <div class="hidden lg:block">
          <h1 class="text-lg font-semibold text-foreground">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="text-xs text-muted-foreground">
            {{ pageDescription }}
          </p>
        </div>
      </div>

      <!-- Right: Announcements + Docs + Language + Subscriptions + Balance + User Dropdown -->
      <div class="flex min-w-0 items-center gap-1 sm:gap-3">
        <!-- Announcement Bell -->
        <AnnouncementBell v-if="user" />

        <!-- Docs Link -->
        <a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-ghost btn-sm hidden sm:inline-flex"
        >
          <Icon name="book" size="sm" />
          <span class="hidden sm:inline">{{ t('nav.docs') }}</span>
        </a>

        <!-- Language Switcher -->
        <LocaleSwitcher />

        <!-- Subscription Progress (for users with active subscriptions) -->
        <SubscriptionProgressMini v-if="user" />

        <!-- Balance Display -->
        <div
          v-if="user"
          ref="walletRef"
          class="relative hidden sm:block"
        >
          <button
            class="flex min-h-11 items-center gap-2 rounded-md border border-border bg-card px-3 py-1.5 transition-colors"
            :title="t('subscriptionProgress.walletTitle')"
            @click="toggleWalletPanel"
          >
            <Icon name="dollar" size="sm" class="text-primary" />
            <span class="text-sm font-semibold text-foreground">
              {{ formatHeaderMoney(availableBalance) }}
            </span>
            <span
              v-if="frozenBalance > 0"
              class="rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-800 dark:bg-amber-900/40 dark:text-amber-300"
            >
              {{ t('common.frozenBalance') }} {{ formatHeaderMoney(frozenBalance) }}
            </span>
          </button>

          <transition name="dropdown">
            <div v-if="walletOpen" class="dropdown right-0 mt-2 w-[360px] overflow-hidden">
              <div class="border-b border-border p-4">
                <div class="flex items-center justify-between gap-3">
                  <div>
                    <h3 class="text-sm font-semibold text-foreground">
                      {{ t('subscriptionProgress.walletTitle') }}
                    </h3>
                    <p class="mt-1 text-xs leading-5 text-muted-foreground">
                      {{ t('subscriptionProgress.walletHint') }}
                    </p>
                  </div>
                  <button
                    type="button"
                    class="btn-ghost btn-icon"
                    :title="t('common.refresh')"
                    :disabled="subscriptionStore.loading"
                    @click="refreshWallet"
                  >
                    <Icon name="refresh" size="sm" :class="subscriptionStore.loading ? 'animate-spin' : ''" />
                  </button>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2 p-3">
                <div class="wallet-metric">
                  <span>{{ t('subscriptionProgress.subscriptionQuota') }}</span>
                  <strong>{{ subscriptionQuotaLabel }}</strong>
                </div>
                <div class="wallet-metric">
                  <span>{{ t('subscriptionProgress.accountBalance') }}</span>
                  <strong>{{ formatHeaderMoney(availableBalance) }}</strong>
                  <small v-if="frozenBalance > 0">
                    {{ t('common.frozenBalance') }} {{ formatHeaderMoney(frozenBalance) }}
                  </small>
                </div>
              </div>

              <div class="max-h-64 overflow-y-auto border-t border-border">
                <div
                  v-if="activeSubscriptions.length === 0"
                  class="px-4 py-5 text-center text-sm text-muted-foreground"
                >
                  {{ t('subscriptionProgress.noActiveWallets') }}
                </div>
                <div
                  v-for="subscription in activeSubscriptions.slice(0, 5)"
                  :key="subscription.id"
                  class="border-b border-border px-4 py-3 last:border-b-0"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="truncate text-sm font-medium text-foreground">
                      {{ subscription.plan_name || subscription.group?.name || `Subscription #${subscription.id}` }}
                    </span>
                    <span class="shrink-0 text-xs text-muted-foreground">
                      {{ formatSubscriptionExpiry(subscription.expires_at) }}
                    </span>
                  </div>
                  <div class="mt-2 space-y-1">
                    <div
                      v-for="period in walletPeriods(subscription)"
                      :key="period.label"
                      class="flex items-center gap-2 text-xs text-muted-foreground"
                    >
                      <span class="w-10 shrink-0">{{ period.label }}</span>
                      <div class="metric-progress h-1.5 min-w-0 flex-1">
                        <div
                          class="metric-progress-bar h-full"
                          :class="period.percent >= 90 ? 'metric-progress-bar-danger' : period.percent >= 70 ? 'metric-progress-bar-warning' : 'metric-progress-bar-success'"
                          :style="{ width: `${period.percent}%` }"
                        />
                      </div>
                      <span class="w-24 shrink-0 text-right">{{ period.text }}</span>
                    </div>
                    <div
                      v-if="walletPeriods(subscription).length === 0"
                      class="text-xs font-medium text-emerald-600 dark:text-emerald-400"
                    >
                      {{ t('subscriptionProgress.quotaUnlimited') }}
                    </div>
                  </div>
                </div>
              </div>

              <div class="grid grid-cols-2 gap-2 border-t border-border p-2">
                <router-link to="/subscriptions" class="subscription-link py-1.5 text-center text-xs" @click="closeWalletPanel">
                  {{ t('subscriptionProgress.viewAll') }}
                </router-link>
                <router-link to="/purchase" class="subscription-link py-1.5 text-center text-xs" @click="closeWalletPanel">
                  {{ t('subscriptionProgress.viewRecharge') }}
                </router-link>
              </div>
            </div>
          </transition>
        </div>

        <!-- User Dropdown -->
        <div v-if="user" class="relative" ref="dropdownRef">
          <button
            @click="toggleDropdown"
            class="user-menu-trigger flex min-h-11 items-center gap-2 rounded-md p-1.5 transition-colors"
            :aria-label="t('common.userMenu')"
          >
            <div class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-full bg-primary text-sm font-semibold text-primary-foreground">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="displayName"
                class="h-full w-full object-cover"
              >
              <span v-else>{{ userInitials }}</span>
            </div>
            <div class="hidden text-left md:block">
              <div class="text-sm font-medium text-foreground">
                {{ displayName }}
              </div>
              <div class="text-xs capitalize text-muted-foreground">
                {{ user.role }}
              </div>
            </div>
            <Icon name="chevronDown" size="sm" class="hidden text-muted-foreground md:block" />
          </button>

          <!-- Dropdown Menu -->
          <transition name="dropdown">
            <div v-if="dropdownOpen" class="dropdown right-0 mt-2 w-56">
              <!-- User Info -->
              <div class="border-b border-border px-4 py-3">
                <div class="text-sm font-medium text-foreground">
                  {{ displayName }}
                </div>
                <div class="text-xs text-muted-foreground">{{ user.email }}</div>
              </div>

              <!-- Balance (mobile only) -->
              <button class="block w-full border-b border-border px-4 py-2 text-left sm:hidden" @click="openWalletFromUserMenu">
                <div class="text-xs text-muted-foreground">
                  {{ t('common.balance') }}
                </div>
                <div class="text-sm font-semibold text-primary">
                  {{ formatHeaderMoney(availableBalance) }}
                </div>
                <div v-if="frozenBalance > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
                  {{ t('common.frozenBalance') }} {{ formatHeaderMoney(frozenBalance) }}
                </div>
              </button>

              <div class="py-1">
                <router-link to="/profile" @click="closeDropdown" class="dropdown-item">
                  <Icon name="user" size="sm" />
                  {{ t('nav.profile') }}
                </router-link>

                <router-link to="/keys" @click="closeDropdown" class="dropdown-item">
                  <Icon name="key" size="sm" />
                  {{ t('nav.apiKeys') }}
                </router-link>

                <a
                  v-if="authStore.isAdmin"
                  href="https://github.com/Wei-Shaw/sub2api"
                  target="_blank"
                  rel="noopener noreferrer"
                  @click="closeDropdown"
                  class="dropdown-item"
                >
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      fill-rule="evenodd"
                      clip-rule="evenodd"
                      d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                    />
                  </svg>
                  {{ t('nav.github') }}
                </a>

              </div>

              <!-- Contact Support (only show if configured) -->
              <div
                v-if="contactInfo"
                class="border-t border-border px-4 py-2.5"
              >
                <div class="flex items-center gap-2 text-xs text-muted-foreground">
                  <svg
                    class="h-3.5 w-3.5 flex-shrink-0"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M20.25 8.511c.884.284 1.5 1.128 1.5 2.097v4.286c0 1.136-.847 2.1-1.98 2.193-.34.027-.68.052-1.02.072v3.091l-3-3c-1.354 0-2.694-.055-4.02-.163a2.115 2.115 0 01-.825-.242m9.345-8.334a2.126 2.126 0 00-.476-.095 48.64 48.64 0 00-8.048 0c-1.131.094-1.976 1.057-1.976 2.192v4.286c0 .837.46 1.58 1.155 1.951m9.345-8.334V6.637c0-1.621-1.152-3.026-2.76-3.235A48.455 48.455 0 0011.25 3c-2.115 0-4.198.137-6.24.402-1.608.209-2.76 1.614-2.76 3.235v6.226c0 1.621 1.152 3.026 2.76 3.235.577.075 1.157.14 1.74.194V21l4.155-4.155"
                    />
                  </svg>
                  <span>{{ t('common.contactSupport') }}:</span>
                  <span class="font-medium text-foreground">{{
                    contactInfo
                  }}</span>
                </div>
              </div>

              <div v-if="showOnboardingButton" class="border-t border-border py-1">
                <button @click="handleReplayGuide" class="dropdown-item w-full">
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M12 2a10 10 0 100 20 10 10 0 000-20zm0 14a1 1 0 110 2 1 1 0 010-2zm1.07-7.75c0-.6-.49-1.25-1.32-1.25-.7 0-1.22.4-1.43 1.02a1 1 0 11-1.9-.62A3.41 3.41 0 0111.8 5c2.02 0 3.25 1.4 3.25 2.9 0 2-1.83 2.55-2.43 3.12-.43.4-.47.75-.47 1.23a1 1 0 01-2 0c0-1 .16-1.82 1.1-2.7.69-.64 1.82-1.05 1.82-2.06z"
                    />
                  </svg>
                  {{ $t('onboarding.restartTour') }}
                </button>
              </div>

              <div class="border-t border-border py-1">
                <button
                  @click="handleLogout"
                  class="dropdown-item logout-item w-full text-semantic-danger"
                >
                  <svg
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
                    />
                  </svg>
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useOnboardingStore, useSubscriptionStore } from '@/stores'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import SubscriptionProgressMini from '@/components/common/SubscriptionProgressMini.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'
import type { UserSubscription } from '@/types'
import { sanitizeUrl } from '@/utils/url'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const onboardingStore = useOnboardingStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)
const dropdownOpen = ref(false)
const walletOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const walletRef = ref<HTMLElement | null>(null)
const contactInfo = computed(() => appStore.contactInfo)
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const activeSubscriptions = computed(() => subscriptionStore.activeSubscriptions)
const subscriptionQuotaLabel = computed(() => {
  if (activeSubscriptions.value.some((subscription) => walletPeriods(subscription).length === 0)) {
    return t('subscriptionProgress.unlimited')
  }
  const total = activeSubscriptions.value.reduce((sum, subscription) => sum + subscriptionRemaining(subscription), 0)
  return `$${formatUSD(total)}`
})

// 只在标准模式的管理员下显示新手引导按钮
const showOnboardingButton = computed(() => {
  return !authStore.isSimpleMode && user.value?.role === 'admin'
})

const userInitials = computed(() => {
  if (!user.value) return ''
  // Prefer username, fallback to email
  if (user.value.username) {
    return user.value.username.substring(0, 2).toUpperCase()
  }
  if (user.value.email) {
    // Get the part before @ and take first 2 chars
    const localPart = user.value.email.split('@')[0]
    return localPart.substring(0, 2).toUpperCase()
  }
  return ''
})

const displayName = computed(() => {
  if (!user.value) return ''
  return user.value.username || user.value.email?.split('@')[0] || ''
})

const pageTitle = computed(() => {
  // For custom pages, use the menu item's label instead of generic "自定义页面"
  if (route.name === 'CustomPage') {
    const id = route.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) return menuItem.label
  }
  const titleKey = route.meta.titleKey as string
  if (titleKey) {
    return t(titleKey)
  }
  return (route.meta.title as string) || ''
})

const pageDescription = computed(() => {
  const descKey = route.meta.descriptionKey as string
  if (descKey) {
    return t(descKey)
  }
  return (route.meta.description as string) || ''
})

function toggleMobileSidebar() {
  appStore.toggleMobileSidebar()
}

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

async function toggleWalletPanel() {
  walletOpen.value = !walletOpen.value
  if (walletOpen.value) {
    closeDropdown()
    await refreshWallet()
  }
}

function closeWalletPanel() {
  walletOpen.value = false
}

async function openWalletFromUserMenu() {
  closeDropdown()
  await router.push('/subscriptions')
}

async function refreshWallet() {
  try {
    await subscriptionStore.fetchActiveSubscriptions(true)
  } catch (error) {
    console.error('Failed to refresh subscription wallet:', error)
  }
}

async function handleLogout() {
  closeDropdown()
  try {
    await authStore.logout()
  } catch (error) {
    // Ignore logout errors - still redirect to login
    console.error('Logout error:', error)
  }
  await router.push('/login')
}

function handleReplayGuide() {
  closeDropdown()
  onboardingStore.replay()
}

function formatHeaderMoney(value: number) {
  if (!Number.isFinite(value)) return '$0.00'
  return `$${value.toFixed(2)}`
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  if (dropdownRef.value && !dropdownRef.value.contains(target)) {
    closeDropdown()
  }
  if (walletRef.value && !walletRef.value.contains(target)) {
    closeWalletPanel()
  }
}

function formatUSD(value: number): string {
  return Number.isFinite(value) ? value.toFixed(2) : '0.00'
}

interface WalletPeriod {
  label: string
  remaining: number
  percent: number
  text: string
}

function subscriptionLimit(subscription: UserSubscription, period: 'daily' | 'weekly' | 'monthly'): number | null {
  const direct = subscription?.[`${period}_limit_usd`]
  const legacy = subscription?.group?.[`${period}_limit_usd`]
  const value = direct ?? legacy ?? null
  return value && value > 0 ? Number(value) : null
}

function subscriptionRemaining(subscription: UserSubscription): number {
  const remaining = walletPeriods(subscription).map((period) => period.remaining)
  return remaining.length > 0 ? Math.min(...remaining) : Number.POSITIVE_INFINITY
}

function walletPeriods(subscription: UserSubscription): WalletPeriod[] {
  return (['daily', 'weekly', 'monthly'] as const)
    .map((period) => {
      const limit = subscriptionLimit(subscription, period)
      if (!limit) return null
      const used = Number(subscription?.[`${period}_usage_usd`] || 0)
      const remaining = Math.max(limit - used, 0)
      return {
        label: t(`subscriptionProgress.${period}`),
        remaining,
        percent: Math.min((used / limit) * 100, 100),
        text: `$${formatUSD(remaining)}`
      }
    })
    .filter((period): period is WalletPeriod => Boolean(period))
}

function formatSubscriptionExpiry(expiresAt?: string | null): string {
  if (!expiresAt) return ''
  const days = Math.ceil((new Date(expiresAt).getTime() - Date.now()) / (1000 * 60 * 60 * 24))
  if (days <= 0) return t('subscriptionProgress.expiresToday')
  return t('subscriptionProgress.daysRemaining', { days })
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
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

.user-menu-trigger:hover {
  background: hsl(var(--accent));
}

.logout-item:hover {
  background: hsl(var(--destructive) / 0.1);
}

.wallet-metric {
  border: 1px solid hsl(var(--border));
  border-radius: calc(var(--radius) - 2px);
  background: hsl(var(--muted));
  padding: 0.75rem;
}

.wallet-metric span {
  display: block;
  font-size: 0.75rem;
  color: hsl(var(--muted-foreground));
}

.wallet-metric strong {
  display: block;
  margin-top: 0.25rem;
  color: hsl(var(--foreground));
  font-size: 1rem;
}
</style>
