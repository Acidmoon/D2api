<template>
  <DataTable :columns="columns" :data="audits" :loading="loading">
    <template #cell-target="{ row }">
      <div class="flex flex-col">
        <span class="font-medium text-gray-900 dark:text-white">{{ targetLabel(row) }}</span>
        <span v-if="row.target.type === 'external' && row.target.provider" class="text-xs text-gray-400">
          {{ row.target.provider }}
        </span>
      </div>
    </template>

    <template #cell-target_model="{ row }">
      <span class="text-sm text-gray-700 dark:text-gray-300">{{ row.target.model }}</span>
    </template>

    <template #cell-verdict="{ row }">
      <!-- 进行中/失败只显示状态徽章；完成后按 §7.4 四档显示可信程度徽章 -->
      <span
        v-if="row.status !== 'done'"
        class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
        :class="statusBadgeClass(row.status)"
      >
        {{ t(`admin.fingerprint.status.${row.status}`) }}
      </span>
      <span
        v-else
        class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium"
        :class="verdictBadgeClass(row.verdict)"
      >
        {{ verdictLabel(row.verdict) }}
      </span>
      <!-- 失败行直接给出原因，避免只看到一个 failed 徽章 -->
      <span
        v-if="row.status === 'failed' && row.error"
        class="mt-1 block max-w-56 truncate text-xs text-red-500 dark:text-red-400"
        :title="row.error"
      >
        {{ row.error }}
      </span>
    </template>

    <template #cell-score="{ row }">
      <span class="text-sm tabular-nums text-gray-900 dark:text-gray-100">{{ formatScore(row.score) }}</span>
    </template>

    <template #cell-progress="{ row }">
      <div v-if="row.status === 'running'" class="flex min-w-36 items-center gap-2">
        <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
          <div
            class="h-full rounded-full bg-blue-500 transition-all dark:bg-blue-400"
            :style="{ width: `${progressPercent(row)}%` }"
          />
        </div>
        <span class="text-xs tabular-nums text-gray-500 dark:text-gray-400">
          {{ row.progress.done }}/{{ row.progress.total }}
        </span>
      </div>
      <span v-else class="text-sm text-gray-400">{{ t('admin.fingerprint.detail.noData') }}</span>
    </template>

    <template #cell-created_at="{ row }">
      <div class="flex flex-col">
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTimeToMinute(row.created_at) }}</span>
        <span v-if="row.status === 'done' && row.duration_ms" class="text-xs text-gray-400">
          {{ t('admin.fingerprint.records.duration', { value: formatDuration(row.duration_ms) }) }}
        </span>
      </div>
    </template>

    <template #cell-actions="{ row }">
      <div class="flex items-center gap-2">
        <button type="button" class="btn btn-secondary px-2.5 py-1 text-xs" @click="emit('select', row.id)">
          {{ t('admin.fingerprint.records.detail') }}
        </button>
        <!-- running 中的任务后端拒绝删除，前端也不给入口 -->
        <button
          v-if="row.status !== 'running'"
          type="button"
          class="btn btn-secondary px-2.5 py-1 text-xs text-red-600 dark:text-red-400"
          @click="emit('remove', row.id)"
        >
          {{ t('admin.fingerprint.records.delete') }}
        </button>
      </div>
    </template>

    <template #empty>
      <EmptyState
        :title="t('admin.fingerprint.records.empty')"
        :description="t('admin.fingerprint.records.emptyHint')"
      />
    </template>
  </DataTable>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { FingerprintAuditSummary } from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import { useFingerprintFormat } from '@/composables/useFingerprintFormat'

const props = defineProps<{
  audits: FingerprintAuditSummary[]
  accounts: Account[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'remove', id: string): void
}>()

const { t } = useI18n()
const { verdictLabel, verdictBadgeClass, statusBadgeClass, formatScore, formatDuration } =
  useFingerprintFormat()

const columns = computed<Column[]>(() => [
  { key: 'target', label: t('admin.fingerprint.records.columns.target'), sortable: false },
  { key: 'target_model', label: t('admin.fingerprint.records.columns.model'), sortable: false },
  { key: 'verdict', label: t('admin.fingerprint.records.columns.verdict'), sortable: false },
  { key: 'score', label: t('admin.fingerprint.records.columns.score'), sortable: false },
  { key: 'progress', label: t('admin.fingerprint.records.columns.progress'), sortable: false },
  { key: 'created_at', label: t('admin.fingerprint.records.columns.createdAt'), sortable: false },
  { key: 'actions', label: t('admin.fingerprint.records.columns.actions'), sortable: false },
])

const accountNameById = computed(() => {
  const map = new Map<number, string>()
  for (const account of props.accounts) map.set(account.id, account.name)
  return map
})

function targetLabel(audit: FingerprintAuditSummary): string {
  const target = audit.target
  if (target.type === 'account') {
    const name = target.account_id != null ? accountNameById.value.get(target.account_id) : undefined
    if (name) return name
    return t('admin.fingerprint.records.accountTarget', { id: target.account_id ?? '?' })
  }
  return target.base_url || t('admin.fingerprint.detail.noData')
}

function progressPercent(audit: FingerprintAuditSummary): number {
  const { done, total } = audit.progress
  if (!total) return 0
  return Math.min(100, Math.round((done / total) * 100))
}
</script>
