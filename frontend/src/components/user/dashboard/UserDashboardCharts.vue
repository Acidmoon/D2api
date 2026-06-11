<template>
  <div class="space-y-6">
    <!-- Date Range Filter -->
    <div class="card p-4">
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

    <!-- Charts Grid：左折线(2/3 趋势) + 右热力图(1/3 用量分布) -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>
      <div class="lg:col-span-1">
        <UserDashboardHeatmap :trend-data="trend" :loading="loading" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import UserDashboardHeatmap from '@/components/user/dashboard/UserDashboardHeatmap.vue'
import type { TrendDataPoint } from '@/types'

defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()
</script>
