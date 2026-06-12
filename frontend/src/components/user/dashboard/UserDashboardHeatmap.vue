<template>
  <div class="card p-4">
    <div class="mb-4 flex items-center justify-between border-b pb-2" style="border-color: var(--nm-border)">
      <h3 class="text-xs font-bold uppercase" style="color: var(--nm-ink); letter-spacing: 0">
        {{ t('dashboard.activityHeatmap') }}
      </h3>
      <span class="text-xs" style="color: var(--nm-ink-faint)">
        {{ t('dates.thisMonth') }} - {{ monthLabel }}
      </span>
    </div>

    <div v-if="loading" class="flex h-48 items-center justify-center">
      <LoadingSpinner />
    </div>

    <div v-else class="heatmap-wrap">
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
              :class="cell ? [`lvl-${cell.level}`, { 'lvl-future': cell.isFuture }] : 'lvl-empty'"
              :title="cell ? `${cell.date}: ${formatTokens(cell.value)} tokens` : ''"
            />
          </div>
        </div>
      </div>

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
  isFuture?: boolean
}

function levelOf(value: number, max: number): number {
  if (value <= 0 || max <= 0) return 0
  const ratio = value / max
  if (ratio > 0.75) return 4
  if (ratio > 0.5) return 3
  if (ratio > 0.25) return 2
  return 1
}

const formatDateKey = (date: Date): string => {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const monthLabel = computed(() => {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
})

const weeks = computed<(Cell | null)[][]>(() => {
  const data = props.trendData
  const max = Math.max(...(data ?? []).map((d) => d.total_tokens), 0)
  const byDate = new Map<string, TrendDataPoint>()
  for (const d of data ?? []) byDate.set(d.date.slice(0, 10), d)

  const now = new Date()
  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1)
  const monthEnd = new Date(now.getFullYear(), now.getMonth() + 1, 0)
  const todayKey = formatDateKey(now)

  const gridStart = new Date(monthStart)
  gridStart.setDate(gridStart.getDate() - gridStart.getDay())
  const gridEnd = new Date(monthEnd)
  gridEnd.setDate(gridEnd.getDate() + (6 - gridEnd.getDay()))

  const cols: (Cell | null)[][] = []
  let col: (Cell | null)[] = []
  const cursor = new Date(gridStart)

  while (cursor <= gridEnd) {
    const iso = formatDateKey(cursor)
    const inMonth = cursor.getMonth() === monthStart.getMonth()
    if (!inMonth) {
      col.push(null)
    } else {
      const dp = byDate.get(iso)
      const value = dp?.total_tokens ?? 0
      col.push({
        date: iso,
        value,
        level: levelOf(value, max),
        isFuture: iso > todayKey
      })
    }
    if (col.length === 7) {
      cols.push(col)
      col = []
    }
    cursor.setDate(cursor.getDate() + 1)
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
  gap: 0.875rem;
}

.heatmap-body {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  overflow-x: hidden;
  padding: 0.25rem 0 0.125rem;
}

.heatmap-weekdays {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex-shrink: 0;
  padding-top: 0;
}

.heatmap-wd {
  height: 18px;
  font-size: 9px;
  line-height: 18px;
  color: var(--nm-ink-faint);
}

.heatmap-grid {
  display: flex;
  gap: 4px;
}

.heatmap-col {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.heatmap-cell {
  width: 18px;
  height: 18px;
  border-radius: 3px;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--nm-border) 80%, transparent);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.16);
}

.lvl-empty { background: transparent; }
.lvl-0 { background: #e8ece7; }
.lvl-1 { background: #7dd3fc; }
.lvl-2 { background: #22c55e; }
.lvl-3 { background: #f59e0b; }
.lvl-4 { background: #dc2626; }
.lvl-future {
  background: repeating-linear-gradient(
    135deg,
    var(--nm-surface-alt),
    var(--nm-surface-alt) 4px,
    color-mix(in srgb, var(--nm-border) 55%, var(--nm-surface-alt)) 4px,
    color-mix(in srgb, var(--nm-border) 55%, var(--nm-surface-alt)) 6px
  );
  opacity: 0.62;
}

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

:global(.dark) .lvl-0 { background: #2d332f; }
:global(.dark) .lvl-1 { background: #0284c7; }
:global(.dark) .lvl-2 { background: #16a34a; }
:global(.dark) .lvl-3 { background: #d97706; }
:global(.dark) .lvl-4 { background: #ef4444; }
</style>
