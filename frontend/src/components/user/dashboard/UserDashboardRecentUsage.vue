<template>
  <div class="card">
    <div class="flex items-center justify-between border-b px-6 py-4" style="border-color: var(--nm-border-light)">
      <h2 class="text-lg font-semibold" style="color: var(--nm-ink)">{{ t('dashboard.recentUsage') }}</h2>
      <span class="badge badge-gray">{{ t('dashboard.last7Days') }}</span>
    </div>
    <div v-if="loading" class="flex items-center justify-center py-12">
      <LoadingSpinner size="lg" />
    </div>
    <div v-else-if="data.length === 0" class="py-8">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>
    <div v-else>
      <div class="table-wrapper">
        <table class="table">
          <thead>
            <tr>
              <th>{{ t('dashboard.model') }}</th>
              <th>{{ t('dashboard.time') }}</th>
              <th class="text-right">{{ t('dashboard.tokens') }}</th>
              <th class="text-right">{{ t('dashboard.cost') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in data" :key="log.id">
              <td>
                <div class="flex items-center gap-3">
                  <span class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg" style="background: var(--nm-bg); box-shadow: var(--nm-shadow-raised-sm); color: var(--nm-accent-text)">
                    <Icon name="beaker" size="sm" />
                  </span>
                  <span class="font-medium" style="color: var(--nm-ink)">{{ log.model }}</span>
                </div>
              </td>
              <td style="color: var(--nm-ink-muted)">{{ formatDateTime(log.created_at) }}</td>
              <td class="text-right font-mono" style="color: var(--nm-ink-muted)">{{ (log.input_tokens + log.output_tokens).toLocaleString() }}</td>
              <td class="text-right">
                <span class="font-semibold" style="color: var(--nm-success-text)" :title="t('dashboard.actual')">${{ formatCost(log.actual_cost) }}</span>
                <span class="text-xs" style="color: var(--nm-ink-faint)" :title="t('dashboard.standard')"> / ${{ formatCost(log.total_cost) }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <router-link to="/usage" class="flex items-center justify-center gap-2 py-3 text-sm font-medium transition-colors hover:underline" style="color: var(--nm-accent-text)">
        {{ t('dashboard.viewAllUsage') }}
        <Icon name="arrowRight" size="sm" />
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{
  data: UsageLog[]
  loading: boolean
}>()
const { t } = useI18n()
const formatCost = (c: number) => c.toFixed(4)
</script>
