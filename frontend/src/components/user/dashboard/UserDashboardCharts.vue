<template>
  <section>
    <h2 class="section-title">{{ t('dashboard.activityAnalysis') }}</h2>

    <!-- Date Range Filter -->
    <div class="card mb-4 p-4">
      <div class="flex flex-wrap items-center gap-3">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium" style="color: var(--nm-ink-muted)">{{ t('dashboard.timeRange') }}:</span>
          <DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="$emit('update:startDate', $event)" @update:endDate="$emit('update:endDate', $event)" @change="$emit('dateRangeChange', $event)" />
        </div>
        <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary">
          {{ t('common.refresh') }}
        </button>
        <div class="ml-auto flex items-center gap-2">
          <span class="text-sm font-medium" style="color: var(--nm-ink-muted)">{{ t('dashboard.granularity') }}:</span>
          <div class="w-28">
            <Select :model-value="granularity" :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]" @update:model-value="$emit('update:granularity', $event)" @change="$emit('granularityChange')" />
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.55fr)_minmax(320px,0.85fr)]">
      <div class="min-w-0">
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>
      <div class="min-w-0 space-y-4">
        <UserDashboardHeatmap :trend-data="heatmapTrend" :loading="heatmapLoading" />
        <UserDashboardModels :models="models" :loading="loading" />
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import UserDashboardHeatmap from '@/components/user/dashboard/UserDashboardHeatmap.vue'
import UserDashboardModels from '@/components/user/dashboard/UserDashboardModels.vue'
import type { TrendDataPoint, ModelStat } from '@/types'

withDefaults(defineProps<{
  loading: boolean
  startDate: string
  endDate: string
  granularity: string
  trend: TrendDataPoint[]
  heatmapTrend?: TrendDataPoint[]
  heatmapLoading?: boolean
  models: ModelStat[]
}>(), {
  heatmapTrend: () => [],
  heatmapLoading: false
})
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()
</script>

<style scoped>
.section-title {
  font-size: 0.8125rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0;
  color: var(--nm-ink);
  margin-bottom: 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--nm-border);
}
</style>
