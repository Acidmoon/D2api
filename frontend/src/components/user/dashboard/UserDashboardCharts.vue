<template>
  <section>
    <h2 class="section-title">{{ t('dashboard.activityAnalysis') }}</h2>

    <!-- Date Range Filter -->
    <div class="card mb-6 p-4">
      <div class="flex flex-wrap items-center gap-4">
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

    <!-- 左请求趋势(2/3) + 右模型排行(1/3) -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>
      <div class="lg:col-span-1">
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
import UserDashboardModels from '@/components/user/dashboard/UserDashboardModels.vue'
import type { TrendDataPoint, ModelStat } from '@/types'

defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()
</script>

<style scoped>
.section-title {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--nm-ink);
  margin-bottom: 1rem;
}
</style>
