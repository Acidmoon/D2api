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

    <!-- Charts Grid：左折线(2/3 主角) + 右环形(1/3 占比) -->
    <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
      <!-- Token Usage Trend Chart（主角，占 2 列） -->
      <div class="lg:col-span-2">
        <TokenUsageTrend :trend-data="trend" :loading="loading" />
      </div>

      <!-- Model Distribution Chart（占比，占 1 列，环上表下） -->
      <div class="card relative overflow-hidden p-4 lg:col-span-1">
        <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center" style="background: var(--nm-bg)">
          <LoadingSpinner size="md" />
        </div>
        <h3 class="mb-4 text-sm font-semibold" style="color: var(--nm-ink)">{{ t('dashboard.modelDistribution') }}</h3>
        <div class="flex flex-col items-center gap-4">
          <div class="h-44 w-44 flex-shrink-0">
            <Doughnut v-if="modelData" :data="modelData" :options="doughnutOptions" />
            <div v-else class="flex h-full items-center justify-center text-sm" style="color: var(--nm-ink-faint)">{{ t('dashboard.noDataAvailable') }}</div>
          </div>
          <div class="max-h-44 w-full overflow-y-auto">
            <table class="w-full text-xs">
              <thead>
                <tr style="color: var(--nm-ink-faint)">
                  <th class="pb-2 text-left">{{ t('dashboard.model') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.requests') }}</th>
                  <th class="pb-2 text-right">{{ t('dashboard.actual') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="model in models" :key="model.model" style="border-top: 1px solid var(--nm-border-light)">
                  <td class="max-w-[110px] truncate py-1.5 font-medium" style="color: var(--nm-ink)" :title="model.model">{{ model.model }}</td>
                  <td class="py-1.5 text-right" style="color: var(--nm-ink-muted)">{{ formatNumber(model.requests) }}</td>
                  <td class="py-1.5 text-right" style="color: var(--nm-success-text)">${{ formatCost(model.actual_cost) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Select from '@/components/common/Select.vue'
import { Doughnut } from 'vue-chartjs'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatNumberLocaleString as formatNumber, formatTokensK as formatTokens } from '@/utils/format'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler } from 'chart.js'
ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, Title, Tooltip, Legend, Filler)

const props = defineProps<{ loading: boolean, startDate: string, endDate: string, granularity: string, trend: TrendDataPoint[], models: ModelStat[] }>()
defineEmits(['update:startDate', 'update:endDate', 'update:granularity', 'dateRangeChange', 'granularityChange', 'refresh'])
const { t } = useI18n()

const modelData = computed(() => !props.models?.length ? null : {
  labels: props.models.map((m: ModelStat) => m.model),
  datasets: [{
    data: props.models.map((m: ModelStat) => m.total_tokens),
    backgroundColor: ['#236b66', '#2d4055', '#9a6700', '#b4232a', '#52616f', '#6d5c7a', '#4f6a86', '#6b7a40']
  }]
})

const doughnutOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${context.label}: ${formatTokens(context.parsed)} tokens`
      }
    }
  }
}
</script>
