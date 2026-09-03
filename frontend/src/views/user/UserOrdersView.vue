<template>
  <AppLayout>
    <div class="flex flex-col">
      <!-- Page header: 28px/600 title + 14px muted description -->
      <header class="mb-6">
        <h1 class="text-[28px] font-semibold leading-9 tracking-[-0.01em] text-foreground">
          {{ t('nav.myOrders') }}
        </h1>
        <p class="qw-desc mt-2 text-sm">{{ t('payment.orders.description') }}</p>
      </header>

      <!-- Order history card -->
      <section class="qw-card p-7">
        <h2 class="text-xl font-semibold text-foreground">{{ t('payment.orders.historyTitle') }}</h2>

        <!-- Filter row: search pill + pill selects + clear-filters link -->
        <div class="mt-5 flex flex-wrap items-center gap-2">
          <SearchInput
            v-model="filterSearch"
            :placeholder="t('payment.orders.searchPlaceholder')"
            class="qw-search w-full sm:w-[200px]"
            pills
          />
          <Select
            v-model="currentFilter"
            :options="statusFilters"
            class="qw-filter-select w-[160px]"
            @change="fetchOrders"
          />
          <Select
            v-model="filterDays"
            :options="dateRangeOptions"
            class="qw-filter-select w-[150px]"
          />
          <button
            type="button"
            class="qw-icon-btn size-9"
            :disabled="loading"
            :title="t('common.refresh')"
            :aria-label="t('common.refresh')"
            @click="fetchOrders"
          >
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
          <button
            type="button"
            class="qw-learn-more ml-1 inline-flex items-center text-[13px] font-medium"
            @click="clearFilters"
          >
            {{ t('payment.orders.clearFilters') }}
          </button>
          <Button
            variant="outline"
            class="ml-auto h-9 rounded-full border-[color:var(--nm-border-strong)] px-[18px] text-[13px] font-medium"
            @click="router.push('/purchase')"
          >
            {{ t('payment.result.backToRecharge') }}
          </Button>
        </div>

        <!-- Table -->
        <div class="qw-table mt-3">
          <OrderTable :orders="displayedOrders" :loading="loading">
            <template #actions="{ row }">
              <div class="flex items-center gap-3 whitespace-nowrap text-sm">
                <button v-if="row.status === 'PENDING'" type="button" class="font-medium text-semantic-danger transition-opacity hover:underline hover:opacity-80" @click="handleCancel(row.id)">
                  {{ t('payment.orders.cancel') }}
                </button>
                <button v-if="canRequestRefund(row)" type="button" class="font-medium text-brand transition-opacity hover:underline hover:opacity-80" @click="openRefundDialog(row)">
                  {{ t('payment.orders.requestRefund') }}
                </button>
              </div>
            </template>
            <template #empty>
              <div class="flex flex-col items-center justify-center gap-2 py-12 text-center">
                <Icon name="document" class="h-10 w-10 text-[#C9CFDA] dark:text-[color:var(--nm-ink-faint)]" />
                <p class="qw-desc text-sm">{{ t('payment.orders.empty') }}</p>
              </div>
            </template>
          </OrderTable>
        </div>

        <!-- Pagination (hidden while the page-local search/time filter is active) -->
        <div v-if="isClientFilterActive && displayedOrders.length !== orders.length" class="qw-desc mt-2 text-xs">
          {{ t('payment.orders.localFilterHint', { shown: displayedOrders.length, page: pagination.page, total: pagination.total }) }}
        </div>
        <div v-if="pagination.total > 0 && !isClientFilterActive" class="mt-2">
          <Pagination
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </section>
    </div>

    <!-- Cancel Confirm Dialog -->
    <BaseDialog :show="!!cancelTargetId" :title="t('payment.orders.cancel')" width="narrow" @close="cancelTargetId = null">
      <p class="text-sm text-muted-foreground">{{ t('payment.confirmCancel') }}</p>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="cancelTargetId = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-danger" :disabled="actionLoading" @click="confirmCancel">{{ actionLoading ? t('common.processing') : t('payment.orders.cancel') }}</button>
        </div>
      </template>
    </BaseDialog>

    <!-- Refund Dialog -->
    <BaseDialog :show="!!refundTarget" :title="t('payment.orders.requestRefund')" @close="refundTarget = null">
      <div v-if="refundTarget" class="space-y-4">
        <div class="rounded-xl bg-secondary px-4 py-3">
          <div class="flex justify-between text-sm">
            <span class="text-muted-foreground">{{ t('payment.orders.orderId') }}</span>
            <span class="font-mono text-foreground">#{{ refundTarget.id }}</span>
          </div>
          <div class="mt-2 flex justify-between text-sm">
            <span class="text-muted-foreground">{{ t('payment.orders.amount') }}</span>
            <span class="text-foreground">${{ refundTarget.amount.toFixed(2) }}</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('payment.refundReason') }}</label>
          <textarea v-model="refundReason" rows="3" class="input mt-1 w-full" :placeholder="t('payment.refundReasonPlaceholder')" />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="refundTarget = null">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" :disabled="actionLoading || !refundReason.trim()" @click="confirmRefund">{{ actionLoading ? t('common.processing') : t('payment.orders.requestRefund') }}</button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import { paymentAPI } from '@/api/payment'
import { extractI18nErrorMessage } from '@/utils/apiError'
import type { PaymentOrder } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Icon from '@/components/icons/Icon.vue'
import OrderTable from '@/components/payment/OrderTable.vue'
import { Button } from '@/components/ui/button'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const orders = ref<PaymentOrder[]>([])
const refundEligibleProviders = ref<Set<string>>(new Set())
const currentFilter = ref('')
const filterSearch = ref('')
const filterDays = ref('')
const cancelTargetId = ref<number | null>(null)
const refundTarget = ref<PaymentOrder | null>(null)
const refundReason = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const statusFilters = computed(() => [
  { value: '', label: t('common.all') },
  { value: 'PENDING', label: t('payment.status.pending') },
  { value: 'COMPLETED', label: t('payment.status.completed') },
  { value: 'FAILED', label: t('payment.status.failed') },
  { value: 'REFUNDED', label: t('payment.status.refunded') },
])

const dateRangeOptions = computed(() => [
  { value: '', label: t('payment.orders.dateAll') },
  { value: '7', label: t('payment.orders.dateLast7Days') },
  { value: '30', label: t('payment.orders.dateLast30Days') },
])

// Client-side refinement of the current page (server API behavior untouched).
// The search box and time window only see the loaded page, so while either is
// active the server pagination is hidden and a scope hint is shown instead —
// otherwise a match living on another page would read as a false "no orders".
const isClientFilterActive = computed(() => filterSearch.value.trim() !== '' || filterDays.value !== '')
const displayedOrders = computed(() => {
  let list = orders.value
  const query = filterSearch.value.trim().toLowerCase()
  if (query) {
    list = list.filter(
      (order) =>
        String(order.id).includes(query) ||
        (order.out_trade_no ?? '').toLowerCase().includes(query)
    )
  }
  if (filterDays.value) {
    const since = Date.now() - Number(filterDays.value) * 86400000
    list = list.filter((order) => {
      const ts = new Date(order.created_at).getTime()
      return Number.isNaN(ts) || ts >= since
    })
  }
  return list
})

function clearFilters() {
  filterSearch.value = ''
  filterDays.value = ''
  if (currentFilter.value !== '') {
    currentFilter.value = ''
    fetchOrders()
  }
}

async function fetchOrders() {
  loading.value = true
  try {
    const res = await paymentAPI.getMyOrders({
      page: pagination.page,
      page_size: pagination.page_size,
      status: currentFilter.value || undefined,
    })
    orders.value = res.data.items || []
    pagination.total = res.data.total || 0
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) { pagination.page = page; fetchOrders() }
function handlePageSizeChange(size: number) { pagination.page_size = size; pagination.page = 1; fetchOrders() }

function handleCancel(orderId: number) { cancelTargetId.value = orderId }

async function confirmCancel() {
  if (!cancelTargetId.value) return
  actionLoading.value = true
  try {
    await paymentAPI.cancelOrder(cancelTargetId.value)
    appStore.showSuccess(t('common.success'))
    cancelTargetId.value = null
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function openRefundDialog(order: PaymentOrder) { refundTarget.value = order; refundReason.value = '' }

async function confirmRefund() {
  if (!refundTarget.value || !refundReason.value.trim()) return
  actionLoading.value = true
  try {
    await paymentAPI.requestRefund(refundTarget.value.id, { reason: refundReason.value.trim() })
    appStore.showSuccess(t('common.success'))
    refundTarget.value = null
    refundReason.value = ''
    await fetchOrders()
  } catch (err: unknown) {
    appStore.showError(extractI18nErrorMessage(err, t, 'payment.errors', t('common.error')))
  } finally {
    actionLoading.value = false
  }
}

function canRequestRefund(order: PaymentOrder): boolean {
  if (order.status !== 'COMPLETED') return false
  if (!order.provider_instance_id) return false
  return refundEligibleProviders.value.has(order.provider_instance_id)
}

async function loadRefundEligibility() {
  try {
    const res = await paymentAPI.getRefundEligibleProviders()
    refundEligibleProviders.value = new Set(res.data.provider_instance_ids || [])
  } catch { /* ignore — default to hiding refund button */ }
}

onMounted(() => { fetchOrders(); loadRefundEligibility() })
</script>

<style scoped>
/* ===== QW pixel spec (light hex from design-ref; dark via --nm-* vars) ===== */
.qw-desc {
  color: #7f8798;
}
:global(.dark) .qw-desc {
  color: var(--nm-ink-muted);
}

/* Page-level card: white, 24px radius, hairline border */
.qw-card {
  background-color: hsl(var(--card));
  border: 1px solid var(--nm-border);
  border-radius: 24px;
}

/* 36x36 quiet icon button */
.qw-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: hsl(var(--muted-foreground));
  border-radius: 9999px;
  transition: background-color 160ms ease, color 160ms ease;
}
.qw-icon-btn:hover {
  background-color: var(--nm-surface-soft);
  color: hsl(var(--foreground));
}
.qw-icon-btn:disabled {
  opacity: 0.5;
  pointer-events: none;
}

/* Accent text button/link */
.qw-learn-more {
  color: hsl(var(--brand));
}
.qw-learn-more:hover {
  text-decoration: underline;
}

/* Search pill: h-10 rounded-full hairline border, 13px placeholder */
.qw-search :deep(input) {
  height: 40px;
  border: 1px solid var(--nm-border);
  border-radius: 9999px;
  background-color: hsl(var(--card));
  box-shadow: none;
}
.qw-search :deep(input)::placeholder {
  color: #8e96a7;
  font-size: 13px;
}
:global(.dark) .qw-search :deep(input) {
  background-color: hsl(var(--card));
  border-color: var(--nm-border);
}

/* Filter select pill: h-10 rounded-full hairline border, 13px label */
.qw-filter-select :deep(.select-trigger) {
  height: 40px;
  min-height: 40px;
  border: 1px solid var(--nm-border);
  border-radius: 9999px;
  background-color: hsl(var(--card));
}
.qw-filter-select :deep(.select-value) {
  font-size: 13px;
}

/* ===== Table spec: header row h-11 rounded-lg on #F9FAFD, th 12px/500 #707889;
     body rows h-14 with #EEF1F5 hairline, td 14px ===== */
.qw-table :deep(thead),
.qw-table :deep(.table-wrapper .table-header) {
  background: transparent;
}
.qw-table :deep(.sticky-header-cell) {
  height: 44px;
  padding-top: 0;
  padding-bottom: 0;
  background: #f9fafd;
  color: #707889;
  font-size: 12px;
  font-weight: 500;
  text-transform: none;
  letter-spacing: 0;
}
.qw-table :deep(thead tr th:first-child) {
  border-top-left-radius: 8px;
  border-bottom-left-radius: 8px;
}
.qw-table :deep(thead tr th:last-child) {
  border-top-right-radius: 8px;
  border-bottom-right-radius: 8px;
}
.qw-table :deep(tbody tr.dt-row) {
  height: 56px;
}
.qw-table :deep(tbody tr.dt-row td) {
  font-size: 14px;
  border-bottom: 1px solid #eef1f5;
}
.qw-table :deep(tbody tr.dt-row:last-child td) {
  border-bottom: none;
}

:global(.dark) .qw-table :deep(thead),
:global(.dark) .qw-table :deep(.table-wrapper .table-header),
:global(.dark) .qw-table :deep(.sticky-header-cell) {
  background: var(--nm-surface-soft);
}
:global(.dark) .qw-table :deep(.sticky-header-cell) {
  color: var(--nm-ink-faint);
}
:global(.dark) .qw-table :deep(tbody tr.dt-row td) {
  border-bottom-color: var(--nm-border-light);
}
</style>
