<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-20">
        <div class="h-8 w-8 animate-spin rounded-full border-4 border-brand border-t-transparent"></div>
      </div>

      <template v-else-if="detail">
        <!-- In-content page header -->
        <div class="page-header">
          <h1 class="page-title">{{ t('affiliate.title') }}</h1>
          <p class="page-description">{{ t('affiliate.description') }}</p>
        </div>

        <!-- Summary KPI cards -->
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <div class="flex items-center justify-between">
              <span class="stat-label">{{ t('affiliate.stats.rebateRate') }}</span>
              <span class="stat-icon stat-icon-primary"><Icon name="dollar" size="sm" /></span>
            </div>
            <p class="stat-value mt-3 text-brand">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-muted-foreground">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <div class="flex items-center justify-between">
              <span class="stat-label">{{ t('affiliate.stats.invitedUsers') }}</span>
              <span class="stat-icon"><Icon name="users" size="sm" /></span>
            </div>
            <p class="stat-value mt-3">{{ formatCount(detail.aff_count) }}</p>
          </div>
          <div class="card p-5">
            <div class="flex items-center justify-between">
              <span class="stat-label">{{ t('affiliate.stats.availableQuota') }}</span>
              <span class="stat-icon stat-icon-success"><Icon name="gift" size="sm" /></span>
            </div>
            <p class="stat-value mt-3 text-semantic-success">{{ formatCurrency(detail.aff_quota) }}</p>
          </div>
          <div class="card p-5">
            <div class="flex items-center justify-between">
              <span class="stat-label">{{ t('affiliate.stats.totalQuota') }}</span>
              <span class="stat-icon"><Icon name="document" size="sm" /></span>
            </div>
            <p class="stat-value mt-3">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-semantic-warning">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <!-- Invite code / link -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-foreground">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-muted-foreground">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-foreground">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-2xl bg-secondary px-3.5 py-2.5 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-foreground sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-foreground">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-2xl bg-secondary px-3.5 py-2.5 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-muted-foreground sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-2xl bg-secondary p-4">
            <p class="text-sm font-medium text-foreground">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-muted-foreground">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <!-- Transfer -->
        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-foreground">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-muted-foreground">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-semantic-warning">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <!-- Invitees -->
        <div class="card p-6">
          <h3 class="text-base font-semibold text-foreground">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="empty-state">
            <Icon name="users" size="xl" class="mb-3 text-muted-foreground/40" />
            <p class="text-sm text-muted-foreground">{{ t('affiliate.invitees.empty') }}</p>
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="table w-full min-w-[560px]">
              <thead>
                <tr>
                  <th>{{ t('affiliate.invitees.columns.email') }}</th>
                  <th>{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th>{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id">
                  <td>{{ item.email || '-' }}</td>
                  <td class="text-muted-foreground">{{ item.username || '-' }}</td>
                  <td class="text-right font-medium text-semantic-success">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="text-muted-foreground">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
