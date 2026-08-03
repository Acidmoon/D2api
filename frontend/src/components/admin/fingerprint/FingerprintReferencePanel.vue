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
    </form>

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
        <button
          v-if="row.source === 'account_sampled' && row.source_account_id"
          type="button"
          class="btn btn-secondary px-2.5 py-1 text-xs"
          :disabled="registering"
          @click="handleReRegister(row)"
        >
          {{ t('admin.fingerprint.references.reRegister') }}
        </button>
      </template>

      <template #empty>
        <EmptyState
          :title="t('admin.fingerprint.references.empty')"
          :description="t('admin.fingerprint.references.emptyHint')"
        />
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type { FingerprintReference } from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import type { Column } from '@/components/common/types'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Select from '@/components/common/Select.vue'
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
        appStore.showError(detail.error || t('admin.fingerprint.references.registerFailed'))
      }
    } catch (err: unknown) {
      appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.references.registerFailed')))
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
  try {
    const task = await adminAPI.fingerprint.registerReference({ account_id: accountId, model })
    appStore.showSuccess(t('admin.fingerprint.references.registered'))
    pollRegisterTask(task.task_id)
  } catch (err: unknown) {
    registering.value = false
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.references.registerFailed')))
  }
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
