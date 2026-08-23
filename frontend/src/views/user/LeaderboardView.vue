<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Header -->
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
            {{ t('leaderboard.title') }}
          </h1>
          <p class="mt-1 text-sm text-muted-foreground">
            {{ t('leaderboard.subtitle') }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3">
          <DateRangePicker
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="load"
          />
          <div class="w-28">
            <Select v-model="limit" :options="limitOptions" @change="load" />
          </div>
        </div>
      </div>

      <!-- Summary -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('leaderboard.totalCost') }}
          </p>
          <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            ${{ fmtCost(totals.total_actual_cost) }}
          </p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('leaderboard.totalRequests') }}
          </p>
          <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            {{ formatNumber(totals.total_requests) }}
          </p>
        </div>
        <div class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('leaderboard.totalTokens') }}
          </p>
          <p class="mt-1 text-xl font-semibold text-gray-900 dark:text-white">
            {{ formatNumber(totals.total_tokens) }}
          </p>
        </div>
      </div>

      <!-- Ranking table -->
      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-4 py-3 sm:px-6 dark:border-dark-700/50">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-gray-400 dark:text-gray-500">{{ t('leaderboard.privacy') }}</p>
            <span v-if="!loading && ranking.length > 0" class="text-xs text-gray-400 dark:text-gray-500">
              {{ ranking.length }}
            </span>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full min-w-max divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="w-16 px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 sm:px-6 dark:text-dark-400">
                  {{ t('leaderboard.rank') }}
                </th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('leaderboard.user') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('leaderboard.requests') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('leaderboard.tokens') }}
                </th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-dark-400">
                  {{ t('leaderboard.cost') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-if="loading">
                <td colspan="5" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="unavailable">
                <td colspan="5" class="py-12 text-center text-sm text-gray-400">
                  {{ t('leaderboard.unavailable') }}
                </td>
              </tr>
              <tr v-else-if="ranking.length === 0">
                <td colspan="5" class="py-12 text-center text-sm text-gray-400">
                  {{ t('leaderboard.empty') }}
                </td>
              </tr>
              <tr v-for="item in ranking" v-else :key="item.rank">
                <td class="px-4 py-3 sm:px-6">
                  <span
                    v-if="item.rank <= 3"
                    class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                    :class="RANK_BADGE_CLASSES[item.rank - 1]"
                  >{{ item.rank }}</span>
                  <span v-else class="inline-block w-6 text-center text-sm tabular-nums text-gray-400">
                    {{ item.rank }}
                  </span>
                </td>
                <td class="max-w-[240px] truncate px-4 py-3 text-sm font-medium text-gray-700 dark:text-gray-200">
                  {{ item.display_name }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">
                  {{ formatNumber(item.requests) }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-500 dark:text-gray-400">
                  {{ formatNumber(item.tokens) }}
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm font-medium tabular-nums text-green-600 dark:text-green-400">
                  ${{ fmtCost(item.actual_cost) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { getPublicLeaderboard } from '@/api/usage'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'

const { t } = useI18n()

const RANK_BADGE_CLASSES = [
  'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400',
  'bg-gray-200 text-gray-600 dark:bg-gray-500/20 dark:text-gray-300',
  'bg-orange-100 text-orange-700 dark:bg-orange-500/20 dark:text-orange-400'
]

const limitOptions = [
  { value: 10, label: 'Top 10' },
  { value: 20, label: 'Top 20' },
  { value: 50, label: 'Top 50' },
  { value: 100, label: 'Top 100' }
]

const fmtCost = (v: number) => formatCostFixed(v, 4)
const formatNumber = (v: number) => formatCompactNumber(v)

function toISODate(d: Date): string {
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

const today = new Date()
const weekAgo = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000)
const startDate = ref(toISODate(weekAgo))
const endDate = ref(toISODate(today))
const limit = ref(50)

const ranking = ref<Awaited<ReturnType<typeof getPublicLeaderboard>>['ranking']>([])
const totals = ref({
  total_actual_cost: 0,
  total_requests: 0,
  total_tokens: 0
})
const loading = ref(false)
const unavailable = ref(false)
let reqSeq = 0

const load = async () => {
  const seq = ++reqSeq
  loading.value = true
  unavailable.value = false
  try {
    const res = await getPublicLeaderboard({
      start_date: startDate.value,
      end_date: endDate.value,
      limit: limit.value
    })
    if (seq !== reqSeq) return
    ranking.value = res.ranking || []
    totals.value = {
      total_actual_cost: res.total_actual_cost || 0,
      total_requests: res.total_requests || 0,
      total_tokens: res.total_tokens || 0
    }
  } catch (error: any) {
    if (seq !== reqSeq) return
    ranking.value = []
    totals.value = { total_actual_cost: 0, total_requests: 0, total_tokens: 0 }
    unavailable.value = error?.response?.status === 404
  } finally {
    if (seq === reqSeq) loading.value = false
  }
}

watch([startDate, endDate, limit], load, { immediate: true })
</script>
