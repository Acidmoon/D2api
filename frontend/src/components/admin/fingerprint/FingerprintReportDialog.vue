<template>
  <BaseDialog
    :show="show"
    :title="t('admin.fingerprint.detail.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div v-if="loading && !detail" class="flex justify-center py-12">
      <LoadingSpinner />
    </div>

    <template v-else-if="detail">
      <!-- 元信息 -->
      <div class="mb-5 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
        <div>
          <div class="text-xs text-gray-400">{{ t('admin.fingerprint.detail.target') }}</div>
          <div class="font-medium text-gray-900 dark:text-white">{{ targetLabel }}</div>
        </div>
        <div>
          <div class="text-xs text-gray-400">{{ t('admin.fingerprint.detail.model') }}</div>
          <div class="font-medium text-gray-900 dark:text-white">{{ targetModel }}</div>
        </div>
        <div>
          <div class="text-xs text-gray-400">{{ t('admin.fingerprint.detail.createdAt') }}</div>
          <div class="font-medium text-gray-900 dark:text-white">{{ formatDateTimeToMinute(detail.created_at) }}</div>
        </div>
      </div>

      <!-- 进行中：进度条，轮询到完成自动刷新 -->
      <div v-if="detail.status === 'running'" class="rounded-lg border border-blue-100 bg-blue-50/50 p-4 dark:border-blue-500/20 dark:bg-blue-500/10">
        <div class="mb-2 flex items-center gap-3">
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-blue-100 dark:bg-blue-500/20">
            <div class="h-full rounded-full bg-blue-500 transition-all dark:bg-blue-400" :style="{ width: `${progressPercent}%` }" />
          </div>
          <span class="text-xs tabular-nums text-blue-600 dark:text-blue-300">{{ detail.progress.done }}/{{ detail.progress.total }}</span>
        </div>
        <p class="text-xs text-blue-600 dark:text-blue-300">{{ t('admin.fingerprint.detail.running') }}</p>
      </div>

      <!-- 失败 -->
      <div v-else-if="detail.status === 'failed'" class="rounded-lg border border-red-100 bg-red-50/50 p-4 dark:border-red-500/20 dark:bg-red-500/10">
        <p class="text-sm font-medium text-red-700 dark:text-red-300">{{ t('admin.fingerprint.detail.failed') }}</p>
        <p v-if="detail.error" class="mt-1 text-xs text-red-600 dark:text-red-400">{{ detail.error }}</p>
      </div>

      <!-- 完成：判定 + 置信要素 + flags + cell 对比 -->
      <template v-else-if="report">
        <!-- 判定徽章与解释（§7.4 口径） -->
        <div class="mb-4 rounded-lg border border-gray-200 p-4 dark:border-dark-700">
          <div class="flex flex-wrap items-center gap-3">
            <span
              class="inline-flex items-center rounded-md px-3 py-1 text-sm font-medium"
              :class="verdictBadgeClass(report.verdict)"
            >
              {{ verdictBadgeText(report.verdict, report.band) }}
            </span>
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ t('admin.fingerprint.detail.score') }}:
              <span class="font-semibold tabular-nums">{{ formatScore(report.score) }}</span>
            </span>
          </div>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
            {{ verdictExplain(report.verdict, report.band, report.cell_count, report.cells?.length || 16) }}
          </p>
        </div>

        <!-- 置信要素：k/n、固有误判率、参考基准来源与时间、劈半自校准、T=0 快信号 -->
        <ul class="mb-4 space-y-1.5 text-xs text-gray-500 dark:text-gray-400">
          <li>{{ t('admin.fingerprint.detail.kn', { k: report.cell_count, n: Math.round(report.avg_samples) }) }}</li>
          <li>{{ t('admin.fingerprint.detail.eerNote') }}</li>
          <li>
            {{ t('admin.fingerprint.detail.referenceLine', {
              source: sourceLabel(report.reference.source),
              time: formatDateTimeToMinute(report.reference.enrolled_at),
            }) }}
          </li>
          <li v-if="report.split_half_jsd !== null">
            {{ t('admin.fingerprint.detail.splitHalf', { value: report.split_half_jsd.toFixed(3) }) }}
            <span v-if="showSplitHalfNote" class="text-gray-700 dark:text-gray-300">
              — {{ t('admin.fingerprint.detail.splitHalfNote') }}
            </span>
          </li>
          <li v-if="report.t0_mismatch_cells > 0">
            {{ t('admin.fingerprint.detail.t0Mismatch', { count: report.t0_mismatch_cells }) }}
          </li>
          <li>{{ t('admin.fingerprint.detail.duration') }}: {{ formatDuration(report.duration_ms) }}</li>
        </ul>

        <!-- 异常标记 -->
        <div v-if="report.flags && report.flags.length > 0" class="mb-4 rounded-lg border border-amber-100 bg-amber-50/50 p-4 dark:border-amber-500/20 dark:bg-amber-500/10">
          <p class="mb-2 text-sm font-medium text-amber-700 dark:text-amber-300">{{ t('admin.fingerprint.flags.title') }}</p>
          <div v-for="flag in report.flags" :key="flag" class="mb-2 last:mb-0">
            <p class="text-sm font-medium text-amber-700 dark:text-amber-300">{{ flagLabel(flag) }}</p>
            <p v-if="flagDesc(flag)" class="text-xs text-amber-600 dark:text-amber-400">{{ flagDesc(flag) }}</p>
          </div>
        </div>

        <!-- 各 cell 分布对比 -->
        <p class="mb-2 text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.fingerprint.detail.cellsTitle') }}</p>
        <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
          <table class="w-full text-sm">
            <thead>
              <tr class="bg-gray-50 text-left text-xs text-gray-500 dark:bg-dark-800 dark:text-gray-400">
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.task') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.language') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.jsd') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.valid') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.compare') }}</th>
                <th class="px-3 py-2 font-medium">{{ t('admin.fingerprint.detail.cellColumns.note') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="cell in report.cells"
                :key="`${cell.task}|${cell.language}`"
                class="border-t border-gray-100 align-top dark:border-dark-700"
              >
                <td class="px-3 py-2 text-gray-900 dark:text-gray-100">{{ taskLabel(cell.task) }}</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ languageLabel(cell.language) }}</td>
                <td class="px-3 py-2 tabular-nums text-gray-900 dark:text-gray-100">
                  {{ cell.jsd === null ? t('admin.fingerprint.detail.noData') : cell.jsd.toFixed(3) }}
                </td>
                <td class="px-3 py-2 tabular-nums text-gray-600 dark:text-gray-400">
                  {{ cell.valid }}
                  <span v-if="cell.invalid || cell.refusal || cell.empty" class="block text-xs text-gray-400">
                    {{ cell.invalid }}/{{ cell.refusal }}/{{ cell.empty }}
                  </span>
                </td>
                <td class="px-3 py-2">
                  <div class="text-xs text-gray-700 dark:text-gray-300">{{ formatDistribution(cell.top_answers) }}</div>
                  <div class="text-xs text-gray-400">{{ formatDistribution(cell.reference_top_answers) }}</div>
                </td>
                <td class="px-3 py-2 text-xs text-gray-400">
                  <div v-if="cell.excluded">{{ excludedLabel(cell.excluded) }}</div>
                  <div v-if="cell.t0_answers && cell.t0_answers.length > 0">
                    {{ t('admin.fingerprint.detail.t0Answers', { value: cell.t0_answers.join('，') }) }}
                  </div>
                  <details v-if="cell.samples && cell.samples.length > 0" class="mt-1">
                    <summary class="cursor-pointer text-gray-500 dark:text-gray-400">samples ×{{ cell.samples.length }}</summary>
                    <div class="mt-1 max-h-32 overflow-y-auto whitespace-pre-wrap break-all">{{ cell.samples.join('\n') }}</div>
                  </details>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import { isFingerprintReport } from '@/api/admin/fingerprint'
import type {
  FingerprintReport,
  FingerprintTaskStatus,
} from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import { useFingerprintFormat } from '@/composables/useFingerprintFormat'

const props = defineProps<{
  show: boolean
  auditId: string | null
  accounts: Account[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t, te } = useI18n()
const appStore = useAppStore()
const {
  verdictBadgeText,
  verdictExplain,
  verdictBadgeClass,
  formatScore,
  formatDuration,
  taskLabel,
  languageLabel,
  sourceLabel,
  formatDistribution,
} = useFingerprintFormat()

const detail = ref<FingerprintTaskStatus | FingerprintReport | null>(null)
const loading = ref(false)

let pollTimer: ReturnType<typeof setTimeout> | null = null

const report = computed<FingerprintReport | null>(() =>
  detail.value && isFingerprintReport(detail.value) ? detail.value : null
)

const progressPercent = computed(() => {
  const progress = detail.value?.progress
  if (!progress || !progress.total) return 0
  return Math.min(100, Math.round((progress.done / progress.total) * 100))
})

const targetLabel = computed(() => {
  const d = detail.value
  if (!d) return ''
  if (!isFingerprintReport(d)) return d.model
  const target = d.target
  if (target.type === 'account') {
    const account = target.account_id != null
      ? props.accounts.find((a) => a.id === target.account_id)
      : undefined
    return account
      ? account.name
      : t('admin.fingerprint.records.accountTarget', { id: target.account_id ?? '?' })
  }
  return target.base_url || t('admin.fingerprint.detail.noData')
})

const targetModel = computed(() => {
  const d = detail.value
  if (!d) return ''
  return isFingerprintReport(d) ? d.target.model : d.model
})

/** 自身稳定（劈半 JSD 低）但对参考偏离很远时的补充说明（§7.4 置信要素） */
const showSplitHalfNote = computed(() => {
  const r = report.value
  if (!r || r.split_half_jsd === null || r.score === null) return false
  return r.split_half_jsd <= 0.15 && r.score > 0.3
})

function flagLabel(flag: string): string {
  const key = `admin.fingerprint.flags.${flag}`
  return te(key) ? t(key) : flag
}

function flagDesc(flag: string): string {
  const key = `admin.fingerprint.flags.${flag}Desc`
  return te(key) ? t(key) : ''
}

function excludedLabel(excluded: string): string {
  const key = `admin.fingerprint.detail.excluded.${excluded}`
  return te(key) ? t(key) : excluded
}

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

async function load() {
  if (!props.auditId) return
  loading.value = true
  try {
    const d = await adminAPI.fingerprint.getAudit(props.auditId)
    detail.value = d
    // 进行中继续轮询（2.5s），完成/失败停止
    if (d.status === 'running' && props.show) {
      stopPolling()
      pollTimer = setTimeout(load, 2500)
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.detail.loadFailed')))
  } finally {
    loading.value = false
  }
}

watch(
  () => [props.show, props.auditId] as const,
  ([show, id]) => {
    stopPolling()
    detail.value = null
    if (show && id) void load()
  },
  { immediate: true }
)

function handleClose() {
  stopPolling()
  emit('close')
}

onUnmounted(stopPolling)
</script>
