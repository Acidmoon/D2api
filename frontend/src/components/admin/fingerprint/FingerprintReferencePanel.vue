<template>
  <div>
    <!-- 注册新参考：选可信账号 + 模型，提交后异步采样，轮询到完成再刷新列表 -->
    <form @submit.prevent="handleRegister" class="mb-5 grid grid-cols-1 gap-3 md:grid-cols-[1fr_1fr_auto] md:items-end">
      <div>
        <label class="input-label">{{ t('admin.fingerprint.references.account') }} <span class="text-red-500">*</span></label>
        <Select
          v-model="registerAccountId"
          :options="accountOptions"
          :placeholder="t('admin.fingerprint.references.accountPlaceholder')"
          searchable
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.fingerprint.references.model') }} <span class="text-red-500">*</span></label>
        <input v-model="registerModel" type="text" class="input" :placeholder="t('admin.fingerprint.references.modelPlaceholder')" />
      </div>
      <button type="submit" class="btn btn-primary whitespace-nowrap" :disabled="registering">
        {{ registering ? t('admin.fingerprint.references.submitting') : t('admin.fingerprint.references.submit') }}
      </button>

      <!-- 高级选项：请求节奏（并发/间隔），留空用后端默认（2 并发 + 500ms） -->
      <details class="rounded-lg border border-gray-200 dark:border-dark-700 md:col-span-3">
        <summary class="cursor-pointer select-none px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
          {{ t('admin.fingerprint.create.advanced') }}
        </summary>
        <div class="grid grid-cols-1 gap-4 px-3 pb-3 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.fingerprint.create.concurrency') }}</label>
            <input
              v-model.number="registerConcurrency"
              type="number"
              min="1"
              max="16"
              class="input"
              :placeholder="t('admin.fingerprint.create.concurrencyPlaceholder')"
            />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.fingerprint.create.concurrencyHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.fingerprint.create.intervalMs') }}</label>
            <input
              v-model.number="registerIntervalMs"
              type="number"
              min="0"
              max="60000"
              step="100"
              class="input"
              :placeholder="t('admin.fingerprint.create.intervalMsPlaceholder')"
            />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.fingerprint.create.intervalMsHint') }}</p>
          </div>
        </div>
      </details>
    </form>

    <!-- 注册失败原因持久展示（轮询拿到 failed 后不再只是一个提示弹窗） -->
    <p
      v-if="registerError"
      class="mb-5 -mt-2 rounded-md border border-red-100 bg-red-50/50 px-3 py-2 text-xs text-red-600 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-400"
    >
      {{ registerError }}
    </p>

    <DataTable :columns="columns" :data="references" :loading="loading">
      <template #cell-model="{ value }">
        <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
      </template>

      <template #cell-source="{ row }">
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ sourceLabel(row.source) }}</span>
      </template>

      <template #cell-enrolled_at="{ row }">
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTimeToMinute(row.enrolled_at) }}</span>
          <span
            v-if="isReferenceStale(row.enrolled_at)"
            class="inline-flex items-center rounded-md bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
          >
            {{ t('admin.fingerprint.references.stale') }}
          </span>
        </div>
      </template>

      <template #cell-cells="{ row }">
        <span class="text-sm text-gray-700 dark:text-gray-300">
          {{ t('admin.fingerprint.references.cellCount', { count: Object.keys(row.cells || {}).length }) }}
        </span>
      </template>

      <template #cell-actions="{ row }">
        <div class="flex items-center gap-2">
          <button
            v-if="row.source === 'account_sampled' && row.source_account_id"
            type="button"
            class="btn btn-secondary px-2.5 py-1 text-xs"
            :disabled="registering"
            @click="handleReRegister(row)"
          >
            {{ t('admin.fingerprint.references.reRegister') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary px-2.5 py-1 text-xs text-red-600 dark:text-red-400"
            :disabled="registering"
            @click="askDelete(row)"
          >
            {{ t('admin.fingerprint.references.delete') }}
          </button>
        </div>
      </template>

      <template #empty>
        <EmptyState
          :title="t('admin.fingerprint.references.empty')"
          :description="t('admin.fingerprint.references.emptyHint')"
        />
      </template>
    </DataTable>

    <ConfirmDialog
      :show="showDeleteConfirm"
      :title="t('admin.fingerprint.references.deleteConfirmTitle')"
      :message="t('admin.fingerprint.references.deleteConfirmMessage', { model: deleteTarget?.model ?? '' })"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteConfirm = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type { FingerprintReference, RegisterReferenceParams } from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { formatDateTimeToMinute } from '@/utils/format'
import { useFingerprintFormat } from '@/composables/useFingerprintFormat'

const props = defineProps<{
  references: FingerprintReference[]
  accounts: Account[]
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'changed'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { sourceLabel, isReferenceStale } = useFingerprintFormat()

const registerAccountId = ref<number | null>(null)
const registerModel = ref('')
const registering = ref(false)
// 注册任务的失败原因（成功后清空，下一次发起时也清空）
const registerError = ref('')
// 高级选项：留空（null）时后端用默认值
const registerConcurrency = ref<number | null>(null)
const registerIntervalMs = ref<number | null>(null)

// 删除参考：ConfirmDialog 确认后调用 DELETE /references/:model
const showDeleteConfirm = ref(false)
const deleteTarget = ref<FingerprintReference | null>(null)

function askDelete(row: FingerprintReference) {
  deleteTarget.value = row
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  showDeleteConfirm.value = false
  const target = deleteTarget.value
  if (!target) return
  try {
    await adminAPI.fingerprint.deleteReference(target.model)
    appStore.showSuccess(t('admin.fingerprint.references.deleted'))
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.references.deleteFailed')))
  } finally {
    deleteTarget.value = null
    emit('changed')
  }
}

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.fingerprint.references.columns.model'), sortable: false },
  { key: 'source', label: t('admin.fingerprint.references.columns.source'), sortable: false },
  { key: 'enrolled_at', label: t('admin.fingerprint.references.columns.enrolledAt'), sortable: false },
  { key: 'cells', label: t('admin.fingerprint.references.columns.cells'), sortable: false },
  { key: 'actions', label: t('admin.fingerprint.references.columns.actions'), sortable: false },
])

const accountOptions = computed(() =>
  props.accounts.map((account) => ({
    value: account.id,
    label: `${account.name} (#${account.id} · ${account.platform})`,
  }))
)

let pollTimer: ReturnType<typeof setTimeout> | null = null

function stopPolling() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

onUnmounted(stopPolling)

/** 注册任务是异步的：轮询任务状态，完成/失败后停止并刷新参考列表 */
function pollRegisterTask(taskId: string) {
  stopPolling()
  const tick = async () => {
    try {
      const detail = await adminAPI.fingerprint.getAudit(taskId)
      if (detail.status === 'running') {
        pollTimer = setTimeout(tick, 2500)
        return
      }
      if (detail.status === 'failed') {
        registerError.value = detail.error || t('admin.fingerprint.references.registerFailed')
        appStore.showError(registerError.value)
      } else {
        registerError.value = ''
      }
    } catch (err: unknown) {
      registerError.value = extractApiErrorMessage(err, t('admin.fingerprint.references.registerFailed'))
      appStore.showError(registerError.value)
    } finally {
      registering.value = false
    }
    emit('changed')
  }
  pollTimer = setTimeout(tick, 2500)
}

async function startRegistration(accountId: number, model: string) {
  if (registering.value) return
  registering.value = true
  registerError.value = ''
  try {
    const task = await adminAPI.fingerprint.registerReference(buildRegisterParams(accountId, model))
    appStore.showSuccess(t('admin.fingerprint.references.registered'))
    pollRegisterTask(task.task_id)
  } catch (err: unknown) {
    registering.value = false
    registerError.value = extractApiErrorMessage(err, t('admin.fingerprint.references.registerFailed'))
    appStore.showError(registerError.value)
  }
}

/** 组装注册参数：高级选项只在填了数字时携带（清空后端用默认值） */
function buildRegisterParams(accountId: number, model: string): RegisterReferenceParams {
  const params: RegisterReferenceParams = { account_id: accountId, model }
  if (typeof registerConcurrency.value === 'number' && !Number.isNaN(registerConcurrency.value)) {
    params.concurrency = registerConcurrency.value
  }
  if (typeof registerIntervalMs.value === 'number' && !Number.isNaN(registerIntervalMs.value)) {
    params.interval_ms = registerIntervalMs.value
  }
  return params
}

async function handleRegister() {
  if (registerAccountId.value == null) {
    appStore.showError(t('admin.fingerprint.references.accountRequired'))
    return
  }
  const model = registerModel.value.trim()
  if (!model) {
    appStore.showError(t('admin.fingerprint.references.modelRequired'))
    return
  }
  await startRegistration(registerAccountId.value, model)
}

async function handleReRegister(row: FingerprintReference) {
  if (!row.source_account_id) return
  appStore.showSuccess(t('admin.fingerprint.references.reRegisterStarted'))
  await startRegistration(row.source_account_id, row.model)
}
</script>
