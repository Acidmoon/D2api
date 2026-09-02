<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- In-content page header -->
      <div class="page-header">
        <h1 class="page-title">{{ t('leaderboard.title') }}</h1>
        <p class="page-description">{{ t('leaderboard.subtitle') }}</p>
      </div>

      <!-- Toolbar: date range + top N -->
      <div class="card p-4">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <DateRangePicker
            v-model:start-date="startDate"
            v-model:end-date="endDate"
            @change="load"
          />
          <div class="w-32">
            <Select v-model="limit" :options="limitOptions" @change="load" />
          </div>
        </div>
      </div>

      <!-- Summary KPI cards -->
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div class="card p-5">
          <div class="flex items-center justify-between">
            <span class="stat-label">{{ t('leaderboard.totalCost') }}</span>
            <span class="stat-icon stat-icon-success"><Icon name="dollar" size="sm" /></span>
          </div>
          <p class="stat-value mt-3 text-semantic-success">${{ fmtCost(totals.total_actual_cost) }}</p>
        </div>
        <div class="card p-5">
          <div class="flex items-center justify-between">
            <span class="stat-label">{{ t('leaderboard.totalRequests') }}</span>
            <span class="stat-icon stat-icon-primary"><Icon name="chart" size="sm" /></span>
          </div>
          <p class="stat-value mt-3">{{ formatNumber(totals.total_requests) }}</p>
        </div>
        <div class="card p-5">
          <div class="flex items-center justify-between">
            <span class="stat-label">{{ t('leaderboard.totalTokens') }}</span>
            <span class="stat-icon"><Icon name="grid" size="sm" /></span>
          </div>
          <p class="stat-value mt-3">{{ formatNumber(totals.total_tokens) }}</p>
        </div>
      </div>

      <!-- Ranking table -->
      <div class="card overflow-hidden">
        <div class="border-b border-border px-4 py-3 sm:px-6">
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs text-muted-foreground">{{ t('leaderboard.privacy') }}</p>
            <span v-if="!loading && ranking.length > 0" class="text-xs text-muted-foreground">
              {{ ranking.length }}
            </span>
          </div>
        </div>
        <div class="overflow-x-auto">
          <table class="table w-full min-w-max">
            <thead>
              <tr>
                <th class="w-16">{{ t('leaderboard.rank') }}</th>
                <th>{{ t('leaderboard.user') }}</th>
                <th class="text-right">{{ t('leaderboard.requests') }}</th>
                <th class="text-right">{{ t('leaderboard.tokens') }}</th>
                <th class="text-right">{{ t('leaderboard.cost') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="5" class="py-12 text-center">
                  <LoadingSpinner />
                </td>
              </tr>
              <tr v-else-if="unavailable">
                <td colspan="5" class="py-12">
                  <div class="empty-state">
                    <Icon name="infoCircle" size="xl" class="mb-3 text-muted-foreground/40" />
                    <p class="text-sm text-muted-foreground">{{ t('leaderboard.unavailable') }}</p>
                  </div>
                </td>
              </tr>
              <tr v-else-if="ranking.length === 0">
                <td colspan="5" class="py-12">
                  <div class="empty-state">
                    <Icon name="inbox" size="xl" class="mb-3 text-muted-foreground/40" />
                    <p class="text-sm text-muted-foreground">{{ t('leaderboard.empty') }}</p>
                  </div>
                </td>
              </tr>
              <tr v-for="item in ranking" v-else :key="item.rank">
                <td>
                  <span
                    v-if="item.rank <= 3"
                    class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold"
                    :class="RANK_BADGE_CLASSES[item.rank - 1]"
                  >{{ item.rank }}</span>
                  <span v-else class="inline-block w-6 text-center text-sm tabular-nums text-muted-foreground">
                    {{ item.rank }}
                  </span>
                </td>
                <td class="max-w-[240px] truncate text-sm font-medium">
                  {{ item.display_name }}
                </td>
                <td class="whitespace-nowrap text-right text-sm tabular-nums text-muted-foreground">
                  {{ formatNumber(item.requests) }}
                </td>
                <td class="whitespace-nowrap text-right text-sm tabular-nums text-muted-foreground">
                  {{ formatNumber(item.tokens) }}
                </td>
                <td class="whitespace-nowrap text-right text-sm font-medium tabular-nums text-semantic-success">
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
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPublicLeaderboard } from '@/api/usage'
import { formatCompactNumber, formatCostFixed } from '@/utils/format'

const { t } = useI18n()

const RANK_BADGE_CLASSES = [
  'bg-amber-100 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400',
  'bg-gray-200 text-gray-600 dark:bg-gray-500/20 dark:text-gray-300',
  'bg-orange-100 text-orange-700 dark:bg-orange-500/20 dark:text-orange-400'
]

const limitOptions = computed(() => [
  { value: 10, label: t('leaderboard.topN', { n: 10 }) },
  { value: 20, label: t('leaderboard.topN', { n: 20 }) },
  { value: 50, label: t('leaderboard.topN', { n: 50 }) },
  { value: 100, label: t('leaderboard.topN', { n: 100 }) }
])

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
