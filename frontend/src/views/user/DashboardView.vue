<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
      <template v-else-if="stats">
        <div class="page-header">
          <h1 class="page-title">{{ t('dashboard.title') }}</h1>
          <p class="page-description">{{ t('dashboard.welcomeMessage') }}</p>
        </div>
        <UserDashboardQuickActions />
        <UserDashboardStats
          :stats="stats"
          :balance="user?.balance || 0"
          :is-simple="authStore.isSimpleMode"
          :trend="trendData"
          :platform-quotas="platformQuotas"
        />
        <UserDashboardCharts
          v-model:startDate="startDate"
          v-model:endDate="endDate"
          v-model:granularity="granularity"
          :loading="loadingCharts"
          :trend="trendData"
          :heatmap-trend="monthTrendData"
          :heatmap-loading="loadingMonthHeatmap"
          :heatmap-month="heatmapMonth"
          :models="modelStats"
          @dateRangeChange="loadCharts"
          @granularityChange="loadCharts"
          @heatmapMonthChange="onHeatmapMonthChange"
          @refresh="refreshAll"
        />
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import type { TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
const loadingCharts = ref(false)
const loadingMonthHeatmap = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const monthTrendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const heatmapMonth = ref(new Date())

const formatLD = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const startDate = ref(formatLD(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatLD(new Date()))
const granularity = ref('day')
const getMonthRange = (month: Date) => {
  return {
    start: formatLD(new Date(month.getFullYear(), month.getMonth(), 1)),
    end: formatLD(new Date(month.getFullYear(), month.getMonth() + 1, 0))
  }
}

const loadStats = async () => {
  loading.value = true
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch (error) {
    console.error('Failed to load dashboard stats:', error)
  } finally {
    loading.value = false
  }
}

const loadCharts = async () => {
  loadingCharts.value = true
  try {
    const res = await Promise.all([
      usageAPI.getDashboardTrend({
        start_date: startDate.value,
        end_date: endDate.value,
        granularity: granularity.value as any
      }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })
    ])
    trendData.value = res[0].trend || []
    modelStats.value = res[1].models || []
  } catch (error) {
    console.error('Failed to load charts:', error)
  } finally {
    loadingCharts.value = false
  }
}

const loadMonthHeatmap = async () => {
  loadingMonthHeatmap.value = true
  try {
    const range = getMonthRange(heatmapMonth.value)
    const res = await usageAPI.getDashboardTrend({
      start_date: range.start,
      end_date: range.end,
      granularity: 'day'
    })
    monthTrendData.value = res.trend || []
  } catch (error) {
    console.error('Failed to load month heatmap:', error)
    monthTrendData.value = []
  } finally {
    loadingMonthHeatmap.value = false
  }
}

const loadPlatformQuotas = async () => {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch (error) {
    console.warn('Failed to load platform quotas:', error)
    platformQuotas.value = []
  }
}

const onHeatmapMonthChange = (month: Date) => {
  heatmapMonth.value = month
  loadMonthHeatmap()
}

const refreshAll = () => {
  loadStats()
  loadCharts()
  loadMonthHeatmap()
  loadPlatformQuotas()
}

onMounted(() => { refreshAll() })
</script>
