<template>
  <AppLayout>
    <div v-if="loading" class="flex items-center justify-center py-12"><LoadingSpinner /></div>
    <div v-else-if="stats" class="space-y-3">
      <!-- QW 工作台：标题块 28/600，与下方卡片间距 12px -->
      <h1 class="text-[28px] font-semibold leading-9 text-foreground">{{ t('dashboard.title') }}</h1>
      <UserDashboardQuickActions />
      <UserDashboardStats
        :stats="stats"
        :balance="user?.balance || 0"
        :is-simple="authStore.isSimpleMode"
        :trend="trendData"
        :platform-quotas="platformQuotas"
      />
      <UserDashboardLearn />
      <UserDashboardModels />
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
import UserDashboardLearn from '@/components/user/dashboard/UserDashboardLearn.vue'
import UserDashboardModels from '@/components/user/dashboard/UserDashboardModels.vue'
import type { TrendDataPoint, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const loading = ref(false)
// 仅消费卡的 7 天迷你柱状图在用；完整趋势图/热力图/模型排行已统一收敛到 /usage。
const trendData = ref<TrendDataPoint[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const formatLD = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

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

// 消费卡迷你趋势：固定近 7 天、按天，不再提供时间范围/粒度工具栏。
const loadSpendTrend = async () => {
  try {
    const res = await usageAPI.getDashboardTrend({
      start_date: formatLD(new Date(Date.now() - 6 * 86400000)),
      end_date: formatLD(new Date()),
      granularity: 'day'
    })
    trendData.value = res.trend || []
  } catch (error) {
    console.error('Failed to load spend trend:', error)
    trendData.value = []
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

onMounted(() => {
  loadStats()
  loadSpendTrend()
  loadPlatformQuotas()
})
</script>
