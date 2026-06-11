<template>
  <div class="card p-4">
    <div class="mb-4 flex items-center justify-between border-b pb-2" style="border-color: var(--nm-border)">
      <h3 class="text-xs font-bold uppercase" style="color: var(--nm-ink); letter-spacing: 0">
        {{ t('dashboard.activityHeatmap') }}
      </h3>
      <span class="text-xs" style="color: var(--nm-ink-faint)">{{ t('dashboard.last7Days') }}</span>
    </div>

    <div v-if="loading" class="flex h-48 items-center justify-center">
      <LoadingSpinner />
    </div>

    <div v-else-if="weeks.length === 0" class="flex h-48 items-center justify-center text-sm" style="color: var(--nm-ink-faint)">
      {{ t('dashboard.noDataAvailable') }}
    </div>

    <div v-else class="heatmap-wrap">
      <!-- 星期标签 + 网格 -->
      <div class="heatmap-body">
        <div class="heatmap-weekdays">
          <span v-for="(wd, i) in weekdayLabels" :key="i" class="heatmap-wd">{{ i % 2 === 1 ? wd : '' }}</span>
        </div>
        <div class="heatmap-grid">
          <div v-for="(week, wi) in weeks" :key="wi" class="heatmap-col">
            <div
              v-for="(cell, ci) in week"
              :key="ci"
              class="heatmap-cell"
              :class="cell ? `lvl-${cell.level}` : 'lvl-empty'"
              :title="cell ? `${cell.date}: ${formatTokens(cell.value)} tokens` : ''"
            />
          </div>
        </div>
      </div>

      <!-- 图例 -->
      <div class="heatmap-legend">
        <span class="heatmap-legend-label">{{ t('dashboard.heatmapLess') }}</span>
        <span class="heatmap-cell lvl-0" />
        <span class="heatmap-cell lvl-1" />
        <span class="heatmap-cell lvl-2" />
        <span class="heatmap-cell lvl-3" />
        <span class="heatmap-cell lvl-4" />
        <span class="heatmap-legend-label">{{ t('dashboard.heatmapMore') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { TrendDataPoint } from '@/types'

const { t } = useI18n()

const props = defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
}>()

const weekdayLabels = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

interface Cell {
  date: string
  value: number
  level: number
}

// 按 total_tokens 映射 0-4 档
function levelOf(value: number, max: number): number {
  if (value <= 0 || max <= 0) return 0
  const ratio = value / max
  if (ratio > 0.75) return 4
  if (ratio > 0.5) return 3
  if (ratio > 0.25) return 2
  return 1
}

// 把 trendData 排成「按周分列、每列 7 格（周日→周六）」的二维网格
const weeks = computed<(Cell | null)[][]>(() => {
  const data = props.trendData
  if (!data?.length) return []

  const max = Math.max(...data.map((d) => d.total_tokens), 0)
  const byDate = new Map<string, TrendDataPoint>()
  for (const d of data) byDate.set(d.date.slice(0, 10), d)

  // 区间：第一条到最后一条
  const dates = data.map((d) => d.date.slice(0, 10)).sort()
  const start = new Date(dates[0] + 'T00:00:00')
  const end = new Date(dates[dates.length - 1] + 'T00:00:00')

  // 对齐到周日起点
  const gridStart = new Date(start)
  gridStart.setDate(gridStart.getDate() - gridStart.getDay())

  const cols: (Cell | null)[][] = []
  let col: (Cell | null)[] = []
  const cursor = new Date(gridStart)

  while (cursor <= end) {
    const iso = cursor.toISOString().slice(0, 10)
    const dp = byDate.get(iso)
    if (dp) {
      col.push({ date: iso, value: dp.total_tokens, level: levelOf(dp.total_tokens, max) })
    } else if (cursor < start) {
      col.push(null) // 区间前的占位
    } else {
      col.push({ date: iso, value: 0, level: 0 })
    }
    if (col.length === 7) {
      cols.push(col)
      col = []
    }
    cursor.setDate(cursor.getDate() + 1)
  }
  if (col.length) {
    while (col.length < 7) col.push(null)
    cols.push(col)
  }
  return cols
})

const formatTokens = (value: number): string => {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toLocaleString()
}
</script>

<style scoped>
.heatmap-wrap {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.heatmap-body {
  display: flex;
  gap: 0.375rem;
  overflow-x: auto;
  padding-bottom: 0.5rem;
}

.heatmap-weekdays {
  display: flex;
  flex-direction: column;
  gap: 3px;
  flex-shrink: 0;
  padding-top: 0;
}

.heatmap-wd {
  height: 14px;
  font-size: 9px;
  line-height: 14px;
  color: var(--nm-ink-faint);
}

.heatmap-grid {
  display: flex;
  gap: 3px;
}

.heatmap-col {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.heatmap-cell {
  width: 14px;
  height: 14px;
  border-radius: 2px;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--nm-border) 70%, transparent);
}

/* 用量分档：nm accent 透明度梯度 */
.lvl-empty { background: transparent; }
.lvl-0 { background: var(--nm-surface-alt); }
.lvl-1 { background: color-mix(in srgb, var(--nm-accent) 30%, var(--nm-surface-alt)); }
.lvl-2 { background: color-mix(in srgb, var(--nm-accent) 55%, var(--nm-surface-alt)); }
.lvl-3 { background: color-mix(in srgb, var(--nm-accent) 78%, var(--nm-surface-alt)); }
.lvl-4 { background: var(--nm-accent); }

.heatmap-legend {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}

.heatmap-legend-label {
  font-size: 10px;
  color: var(--nm-ink-faint);
  margin: 0 2px;
}
</style>
