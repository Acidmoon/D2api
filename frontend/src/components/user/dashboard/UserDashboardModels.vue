<template>
  <div class="card model-rank-card flex flex-col p-4">
    <h3 class="mb-3 border-b pb-2 text-xs font-bold uppercase" style="color: var(--nm-ink); border-color: var(--nm-border); letter-spacing: 0">
      {{ t('dashboard.modelRanking') }} Top 3
    </h3>

    <div v-if="loading" class="flex flex-1 items-center justify-center py-8">
      <LoadingSpinner size="md" />
    </div>

    <div v-else-if="ranked.length === 0" class="flex flex-1 items-center justify-center py-8 text-sm" style="color: var(--nm-ink-faint)">
      {{ t('dashboard.noDataAvailable') }}
    </div>

    <ol v-else class="divide-y" style="border-color: var(--nm-border-light)">
      <li v-for="(m, i) in ranked" :key="m.model" class="rank-row">
        <span class="rank-no" :class="`rank-${i + 1}`">{{ i + 1 }}</span>
        <div class="min-w-0 flex-1">
          <div class="flex items-baseline justify-between gap-2">
            <span class="truncate text-sm font-medium" style="color: var(--nm-ink)" :title="m.model">{{ m.model }}</span>
            <span class="flex-shrink-0 font-mono text-xs" style="color: var(--nm-success-text)">${{ formatCost(m.actual_cost) }}</span>
          </div>
          <div class="mt-1 flex items-center gap-2">
            <div class="rank-bar-track">
              <div class="rank-bar-fill" :style="{ width: pct(m.total_tokens) + '%' }" />
            </div>
            <span class="flex-shrink-0 text-[11px]" style="color: var(--nm-ink-faint)">{{ formatTokens(m.total_tokens) }}</span>
          </div>
        </div>
      </li>
    </ol>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { ModelStat } from '@/types'
import { formatCostFixed as formatCost, formatTokensK as formatTokens } from '@/utils/format'

const props = defineProps<{
  models: ModelStat[]
  loading?: boolean
}>()
const { t } = useI18n()

const ranked = computed(() =>
  [...(props.models ?? [])].sort((a, b) => b.total_tokens - a.total_tokens).slice(0, 3)
)

const maxTokens = computed(() => Math.max(...ranked.value.map((m) => m.total_tokens), 1))
const pct = (v: number) => Math.max(2, Math.round((v / maxTokens.value) * 100))
</script>

<style scoped>
.model-rank-card {
  height: auto;
}

.rank-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.625rem 0;
}

.rank-no {
  display: flex;
  height: 1.5rem;
  width: 1.5rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface-soft);
  border: 1px solid var(--nm-border);
  box-shadow: none;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--nm-ink-muted);
}

.rank-1 { color: #fff; background: var(--nm-accent); }
.rank-2,
.rank-3 { color: var(--nm-accent-text); }

.rank-bar-track {
  flex: 1;
  height: 4px;
  border-radius: var(--nm-radius-sm);
  background: var(--nm-surface-soft);
  box-shadow: none;
  overflow: hidden;
}

.rank-bar-fill {
  height: 100%;
  border-radius: var(--nm-radius-sm);
  background: var(--nm-accent);
  transition: width 300ms ease;
}
</style>
