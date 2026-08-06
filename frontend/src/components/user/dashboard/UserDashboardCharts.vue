<template>
  <section>
    <!-- Date Range Filter -->
    <div class="card mb-4 p-4">
      <div class="flex flex-wrap items-center gap-3">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-muted-foreground">{{ t('dashboard.timeRange') }}:</span>
          <DateRangePicker :start-date="startDate" :end-date="endDate" @update:startDate="$emit('update:startDate', $event)" @update:endDate="$emit('update:endDate', $event)" @change="$emit('dateRangeChange', $event)" />
        </div>
        <button @click="$emit('refresh')" :disabled="loading" class="btn btn-secondary">
          {{ t('common.refresh') }}
        </button>
        <div class="ml-auto flex items-center gap-2">
          <span class="text-sm font-medium text-muted-foreground">{{ t('dashboard.granularity') }}:</span>
          <div class="w-28">
            <Select :model-value="granularity" :options="[{value:'day', label:t('dashboard.day')}, {value:'hour', label:t('dashboard.hour')}]" @update:model-value="$emit('update:granularity', $event)" @change="$emit('granularityChange')" />
          </div>
        </div>
      </div>
    </div>

    <div class="card usage-detail-panel p-4">
      <div class="usage-detail-grid">
        <div class="trend-pane">
          <TokenUsageTrend :trend-data="trend" :loading="loading" embedded />
        </div>
        <div class="heatmap-pane">
          <UserDashboardHeatmap
            :trend-data="heatmapTrend"
            :loading="heatmapLoading"
            :month="heatmapMonth"
            embedded
            @monthChange="$emit('heatmapMonthChange', $event)"
          />
        </div>
      </div>
    </div>

    <div class="mt-4">
      <UserDashboardModels :models="models" :loading="loading" />
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
  heatmapMonth?: Date
  models: ModelStat[]
}>(), {
  heatmapTrend: () => [],
  heatmapLoading: false,
  heatmapMonth: () => new Date()
})
defineEmits([
  'update:startDate',
  'update:endDate',
  'update:granularity',
  'dateRangeChange',
  'granularityChange',
  'heatmapMonthChange',
  'refresh'
])
const { t } = useI18n()
</script>

<style scoped>
.usage-detail-panel {
  overflow: hidden;
}

.usage-detail-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(430px, 0.92fr);
  gap: 1.25rem;
  align-items: stretch;
}

.trend-pane,
.heatmap-pane {
  min-width: 0;
}

.heatmap-pane {
  border-left: 1px solid hsl(var(--border));
  padding-left: 1.25rem;
}

@media (max-width: 1180px) {
  .usage-detail-grid {
    grid-template-columns: 1fr;
  }

  .heatmap-pane {
    border-left: 0;
    border-top: 1px solid hsl(var(--border));
    padding-left: 0;
    padding-top: 1.25rem;
  }
}
</style>
