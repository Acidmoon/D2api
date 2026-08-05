<template>
  <div>
    <!-- 入口切换：测账号 / 测外部端点（§3.1 / §3.2） -->
    <div class="mb-5 inline-flex rounded-lg border border-gray-200 p-1 dark:border-dark-700">
      <button
        v-for="tab in targetTabs"
        :key="tab.value"
        type="button"
        class="rounded-md px-4 py-1.5 text-sm font-medium transition-colors"
        :class="targetType === tab.value
          ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
          : 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'"
        @click="targetType = tab.value"
      >
        {{ tab.label }}
      </button>
    </div>

    <form @submit.prevent="handleSubmit" class="grid grid-cols-1 gap-4 md:grid-cols-2">
      <!-- 测账号：选择系统内账号 -->
      <div v-if="targetType === 'account'">
        <label class="input-label">{{ t('admin.fingerprint.create.account') }} <span class="text-red-500">*</span></label>
        <Select
          v-model="form.account_id"
          :options="accountOptions"
          :placeholder="t('admin.fingerprint.create.accountPlaceholder')"
          searchable
        />
      </div>

      <!-- 测外部端点：BaseURL + API Key + provider（+ openai 的 api_mode） -->
      <template v-else>
        <div>
          <label class="input-label">{{ t('admin.fingerprint.create.baseUrl') }} <span class="text-red-500">*</span></label>
          <input v-model="form.base_url" type="text" class="input" :placeholder="t('admin.fingerprint.create.baseUrlPlaceholder')" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.fingerprint.create.apiKey') }} <span class="text-red-500">*</span></label>
          <input v-model="form.api_key" type="password" autocomplete="off" class="input" :placeholder="t('admin.fingerprint.create.apiKeyPlaceholder')" />
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.fingerprint.create.apiKeyHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.fingerprint.create.provider') }} <span class="text-red-500">*</span></label>
          <Select v-model="form.provider" :options="providerOptions" />
        </div>
        <div v-if="form.provider === 'openai'">
          <label class="input-label">{{ t('admin.fingerprint.create.apiMode') }}</label>
          <Select v-model="form.api_mode" :options="apiModeOptions" />
        </div>
      </template>

      <div>
        <label class="input-label">{{ t('admin.fingerprint.create.model') }} <span class="text-red-500">*</span></label>
        <input v-model="form.model" type="text" class="input" :placeholder="t('admin.fingerprint.create.modelPlaceholder')" />
      </div>

      <!-- 参考基准：已有参考 / 选可信账号现场注册（§3.3） -->
      <div class="md:col-span-2">
        <label class="input-label">{{ t('admin.fingerprint.create.referenceMode') }} <span class="text-red-500">*</span></label>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <button
            type="button"
            class="rounded-lg border-2 px-3 py-2 text-left text-sm transition-colors"
            :class="refModeButtonClass(referenceMode === 'existing')"
            @click="referenceMode = 'existing'"
          >
            {{ t('admin.fingerprint.create.referenceModeExisting') }}
          </button>
          <button
            type="button"
            class="rounded-lg border-2 px-3 py-2 text-left text-sm transition-colors"
            :class="refModeButtonClass(referenceMode === 'enroll')"
            @click="referenceMode = 'enroll'"
          >
            {{ t('admin.fingerprint.create.referenceModeEnroll') }}
          </button>
        </div>
        <div class="mt-3">
          <Select
            v-if="referenceMode === 'existing'"
            v-model="form.reference_model"
            :options="referenceOptions"
            :placeholder="t('admin.fingerprint.create.referenceExistingPlaceholder')"
            searchable
          />
          <div v-else>
            <Select
              v-model="form.reference_account_id"
              :options="accountOptions"
              :placeholder="t('admin.fingerprint.create.referenceAccountPlaceholder')"
              searchable
            />
            <p class="mt-1 text-xs text-gray-400">{{ t('admin.fingerprint.create.referenceEnrollHint') }}</p>
          </div>
        </div>
      </div>

      <!-- 高级选项：请求节奏（并发/间隔），留空用后端默认（2 并发 + 500ms） -->
      <div class="md:col-span-2">
        <details class="rounded-lg border border-gray-200 dark:border-dark-700">
          <summary class="cursor-pointer select-none px-3 py-2 text-sm text-gray-600 dark:text-gray-400">
            {{ t('admin.fingerprint.create.advanced') }}
          </summary>
          <div class="grid grid-cols-1 gap-4 px-3 pb-3 sm:grid-cols-2">
            <div>
              <label class="input-label">{{ t('admin.fingerprint.create.concurrency') }}</label>
              <input
                v-model.number="form.concurrency"
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
                v-model.number="form.interval_ms"
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
      </div>

      <div class="flex items-center justify-between md:col-span-2">
        <div>
          <label class="input-label mb-0">{{ t('admin.fingerprint.create.keepRaw') }}</label>
          <p class="mt-0.5 text-xs text-gray-400">{{ t('admin.fingerprint.create.keepRawHint') }}</p>
        </div>
        <Toggle v-model="form.keep_raw" />
      </div>

      <div class="md:col-span-2">
        <button type="submit" class="btn btn-primary" :disabled="submitting">
          {{ submitting ? t('admin.fingerprint.create.submitting') : t('admin.fingerprint.create.submit') }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type {
  CreateAuditParams,
  FingerprintAPIMode,
  FingerprintProvider,
  FingerprintReference,
  FingerprintTargetType,
} from '@/api/admin/fingerprint'
import type { Account } from '@/types'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useFingerprintFormat } from '@/composables/useFingerprintFormat'

const props = defineProps<{
  accounts: Account[]
  references: FingerprintReference[]
}>()

const emit = defineEmits<{
  (e: 'created'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { sourceLabel } = useFingerprintFormat()

const targetType = ref<FingerprintTargetType>('account')
const referenceMode = ref<'existing' | 'enroll'>('existing')
const submitting = ref(false)

const form = reactive({
  account_id: null as number | null,
  base_url: '',
  api_key: '',
  provider: 'openai' as FingerprintProvider,
  api_mode: 'chat_completions' as FingerprintAPIMode,
  model: '',
  reference_model: '',
  reference_account_id: null as number | null,
  keep_raw: false,
  concurrency: null as number | null,
  interval_ms: null as number | null,
})

const targetTabs = computed<{ value: FingerprintTargetType; label: string }[]>(() => [
  { value: 'account', label: t('admin.fingerprint.create.tabAccount') },
  { value: 'external', label: t('admin.fingerprint.create.tabExternal') },
])

const accountOptions = computed(() =>
  props.accounts.map((account) => ({
    value: account.id,
    label: `${account.name} (#${account.id} · ${account.platform})`,
  }))
)

const referenceOptions = computed(() =>
  props.references.map((reference) => ({
    value: reference.model,
    label: `${reference.model} · ${sourceLabel(reference.source)}`,
  }))
)

const providerOptions = computed<{ value: FingerprintProvider; label: string }[]>(() => [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'grok', label: 'Grok' },
])

const apiModeOptions = [
  { value: 'chat_completions', label: 'chat_completions' },
  { value: 'responses', label: 'responses' },
]

function refModeButtonClass(active: boolean): string {
  if (active) {
    return 'border-brand bg-white text-brand shadow-sm dark:border-brand dark:bg-brand/15 dark:text-brand'
  }
  return 'border-gray-200 bg-white/70 text-gray-600 hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400'
}

function validate(): string | null {
  if (targetType.value === 'account' && form.account_id == null) {
    return t('admin.fingerprint.create.accountRequired')
  }
  if (targetType.value === 'external') {
    if (!form.base_url.trim()) return t('admin.fingerprint.create.baseUrlRequired')
    if (!form.api_key.trim()) return t('admin.fingerprint.create.apiKeyRequired')
  }
  if (!form.model.trim()) return t('admin.fingerprint.create.modelRequired')
  if (referenceMode.value === 'existing' && !form.reference_model) {
    return t('admin.fingerprint.create.referenceRequired')
  }
  if (referenceMode.value === 'enroll' && form.reference_account_id == null) {
    return t('admin.fingerprint.create.referenceAccountRequired')
  }
  return null
}

function buildPayload(): CreateAuditParams {
  const payload: CreateAuditParams = {
    target_type: targetType.value,
    model: form.model.trim(),
    // 现场注册时参考与被测同名模型（§3.3：先注册该模型的参考再测）
    reference_model:
      referenceMode.value === 'existing' ? form.reference_model : form.model.trim(),
    keep_raw: form.keep_raw,
  }
  if (targetType.value === 'account') {
    payload.account_id = form.account_id ?? undefined
  } else {
    payload.base_url = form.base_url.trim()
    payload.api_key = form.api_key.trim()
    payload.provider = form.provider
    if (form.provider === 'openai') payload.api_mode = form.api_mode
  }
  if (referenceMode.value === 'enroll') {
    payload.reference_account_id = form.reference_account_id ?? undefined
  }
  // number input 清空时值为 null/''，只在填了数字时才带字段（后端用默认值）
  if (typeof form.concurrency === 'number' && !Number.isNaN(form.concurrency)) {
    payload.concurrency = form.concurrency
  }
  if (typeof form.interval_ms === 'number' && !Number.isNaN(form.interval_ms)) {
    payload.interval_ms = form.interval_ms
  }
  return payload
}

async function handleSubmit() {
  if (submitting.value) return
  const invalid = validate()
  if (invalid) {
    appStore.showError(invalid)
    return
  }
  submitting.value = true
  try {
    await adminAPI.fingerprint.createAudit(buildPayload())
    appStore.showSuccess(t('admin.fingerprint.create.created'))
    // API Key 用完即清，不在表单里残留
    form.api_key = ''
    emit('created')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.fingerprint.create.createFailed')))
  } finally {
    submitting.value = false
  }
}
</script>
