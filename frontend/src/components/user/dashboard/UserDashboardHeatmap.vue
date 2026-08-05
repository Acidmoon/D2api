<template>
  <div :class="['heatmap-card', embedded ? 'heatmap-card--embedded' : 'card p-4']">
    <div class="mb-4 flex items-center justify-between gap-3 border-b border-border pb-2">
      <h3 class="text-sm font-semibold text-foreground">
        {{ t('dashboard.activityHeatmap') }}
      </h3>
      <div class="month-switcher">
        <button type="button" class="month-btn" :disabled="loading" :title="t('dates.lastMonth')" @click="shiftMonth(-1)">
          ‹
        </button>
        <span class="month-label">{{ monthLabel }}</span>
        <button
          type="button"
          class="month-btn"
          :disabled="loading || isCurrentOrFutureMonth"
          :title="t('dates.nextMonth')"
          @click="shiftMonth(1)"
        >
          ›
        </button>
      </div>
    </div>

    <div v-if="loading" class="flex h-[210px] items-center justify-center">
      <LoadingSpinner />
    </div>

    <div v-else class="heatmap-wrap">
      <div class="heatmap-content">
        <div class="heatmap-body">
          <div class="heatmap-weekdays">
            <span v-for="wd in weekdayLabels" :key="wd" class="heatmap-wd">{{ wd }}</span>
          </div>
          <div class="heatmap-grid">
            <div v-for="(week, wi) in weeks" :key="wi" class="heatmap-col">
              <button
                v-for="(cell, ci) in week"
                :key="ci"
                type="button"
                class="heatmap-cell"
                :class="cell ? [`lvl-${cell.level}`, { 'lvl-future': cell.isFuture }] : 'lvl-empty'"
                :title="cell ? `${cell.date}: ${formatTokens(cell.value)} tokens` : ''"
                :disabled="!cell"
              ></button>
            </div>
          </div>
        </div>

        <div class="heatmap-summary">
          <div class="summary-item">
            <span class="summary-value">{{ formatTokens(totalTokens) }}</span>
            <span class="summary-label">{{ t('dashboard.heatmapTotal') }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-value">{{ formatTokens(peakTokens) }}</span>
            <span class="summary-label">{{ t('dashboard.heatmapPeak') }}</span>
          </div>
          <div class="summary-item">
            <span class="summary-value">{{ activeDays }}</span>
            <span class="summary-label">{{ t('dashboard.heatmapActiveDays') }}</span>
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

const { t, locale } = useI18n()

const props = withDefaults(defineProps<{
  trendData: TrendDataPoint[]
  loading?: boolean
  month: Date
  embedded?: boolean
}>(), {
  embedded: false
})

const embedded = computed(() => props.embedded)

const emit = defineEmits<{
  monthChange: [month: Date]
}>()

const weekdayLabels = computed(() => {
  return locale.value === 'zh'
    ? ['一', '二', '三', '四', '五', '六', '日']
    : ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
})

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

const mondayIndex = (date: Date): number => (date.getDay() + 6) % 7

const monthLabel = computed(() => {
  return `${props.month.getFullYear()}-${String(props.month.getMonth() + 1).padStart(2, '0')}`
})

const isCurrentOrFutureMonth = computed(() => {
  const now = new Date()
  const selected = props.month.getFullYear() * 12 + props.month.getMonth()
  const current = now.getFullYear() * 12 + now.getMonth()
  return selected >= current
})

const weeks = computed<(Cell | null)[][]>(() => {
  const data = props.trendData
  const max = Math.max(...(data ?? []).map((d) => d.total_tokens), 0)
  const byDate = new Map<string, TrendDataPoint>()
  for (const d of data ?? []) byDate.set(d.date.slice(0, 10), d)

  const now = new Date()
  const monthStart = new Date(props.month.getFullYear(), props.month.getMonth(), 1)
  const monthEnd = new Date(props.month.getFullYear(), props.month.getMonth() + 1, 0)
  const todayKey = formatDateKey(now)

  const gridStart = new Date(monthStart)
  gridStart.setDate(gridStart.getDate() - mondayIndex(gridStart))
  const gridEnd = new Date(monthEnd)
  gridEnd.setDate(gridEnd.getDate() + (6 - mondayIndex(gridEnd)))

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

const currentMonthPoints = computed(() => {
  const monthKey = `${props.month.getFullYear()}-${String(props.month.getMonth() + 1).padStart(2, '0')}`
  return (props.trendData ?? []).filter((d) => d.date.slice(0, 7) === monthKey)
})

const totalTokens = computed(() => currentMonthPoints.value.reduce((sum, d) => sum + d.total_tokens, 0))
const peakTokens = computed(() => Math.max(...currentMonthPoints.value.map((d) => d.total_tokens), 0))
const activeDays = computed(() => currentMonthPoints.value.filter((d) => d.total_tokens > 0).length)

const shiftMonth = (delta: number) => {
  const next = new Date(props.month.getFullYear(), props.month.getMonth() + delta, 1)
  emit('monthChange', next)
}

const formatTokens = (value: number): string => {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return value.toLocaleString()
}
</script>

<style scoped>
.heatmap-card {
  min-height: 100%;
}

.heatmap-card--embedded {
  height: 100%;
}

.month-switcher {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  flex-shrink: 0;
}

.month-btn {
  display: inline-flex;
  width: 1.5rem;
  height: 1.5rem;
  align-items: center;
  justify-content: center;
  border-radius: var(--nm-radius-sm);
  border: 1px solid var(--nm-border);
  background: var(--nm-surface-soft);
  color: var(--nm-ink);
  font-size: 1.125rem;
  line-height: 1;
}

.month-btn:hover:not(:disabled) {
  border-color: var(--nm-accent);
  color: var(--nm-accent-text);
}

.month-btn:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.month-label {
  min-width: 4.75rem;
  text-align: center;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--nm-ink-muted);
}

.heatmap-wrap {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.heatmap-content {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 8.5rem;
  align-items: center;
  gap: 1rem;
}

.heatmap-body {
  display: flex;
  justify-content: center;
  gap: 0.625rem;
  overflow-x: hidden;
  padding: 0.125rem 0 0.25rem;
}

.heatmap-weekdays {
  display: flex;
  flex-direction: column;
  gap: clamp(4px, 0.9vw, 7px);
  flex-shrink: 0;
  padding-top: 0;
}

.heatmap-wd {
  height: clamp(18px, 2.6vw, 24px);
  font-size: 9px;
  line-height: clamp(18px, 2.6vw, 24px);
  color: var(--nm-ink-faint);
}

.heatmap-grid {
  display: flex;
  gap: clamp(4px, 0.9vw, 7px);
  flex: 1;
  justify-content: center;
  max-width: 100%;
}

.heatmap-col {
  display: flex;
  flex-direction: column;
  gap: clamp(4px, 0.9vw, 7px);
  flex: 1 1 0;
  max-width: 32px;
}

.heatmap-cell {
  width: 100%;
  aspect-ratio: 1;
  min-width: 18px;
  min-height: 18px;
  border-radius: 4px;
  flex-shrink: 0;
  border: 1px solid color-mix(in srgb, var(--nm-border) 72%, transparent);
  box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.12);
  transition: transform 120ms ease, border-color 120ms ease;
}

button.heatmap-cell:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: color-mix(in srgb, var(--nm-ink) 45%, var(--nm-border));
}

.lvl-empty { background: transparent; }
.lvl-0 { background: var(--nm-surface-alt); }
.lvl-1 { background: #9be9a8; }
.lvl-2 { background: #40c463; }
.lvl-3 { background: #30a14e; }
.lvl-4 { background: #216e39; }
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

.heatmap-summary {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.summary-item {
  display: flex;
  min-height: 3.6rem;
  flex-direction: column;
  justify-content: center;
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface-soft);
  border: 1px solid color-mix(in srgb, var(--nm-border) 70%, transparent);
  padding: 0.625rem 0.75rem;
}

.summary-value {
  color: var(--nm-ink);
  font-size: 1.125rem;
  font-weight: 700;
  line-height: 1.2;
}

.summary-label {
  margin-top: 0.125rem;
  color: var(--nm-ink-faint);
  font-size: 0.6875rem;
}

.heatmap-legend {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 5px;
}

.heatmap-legend .heatmap-cell {
  width: 15px;
  height: 15px;
  min-width: 15px;
  min-height: 15px;
}

.heatmap-legend-label {
  font-size: 10px;
  color: var(--nm-ink-faint);
  margin: 0 2px;
}

:global(.dark) .lvl-0 { background: #161b22; }
:global(.dark) .lvl-1 { background: #0e4429; }
:global(.dark) .lvl-2 { background: #006d32; }
:global(.dark) .lvl-3 { background: #26a641; }
:global(.dark) .lvl-4 { background: #39d353; }

@media (max-width: 720px) {
  .heatmap-content {
    grid-template-columns: 1fr;
  }

  .heatmap-summary {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
