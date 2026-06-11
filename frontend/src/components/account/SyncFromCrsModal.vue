<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.syncFromCrsTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <!-- Step 1: Input credentials -->
    <form
      v-if="currentStep === 'input'"
      id="sync-from-crs-form"
      class="space-y-4"
      @submit.prevent="handlePreview"
    >
      <div class="crs-dialog-copy">
        {{ t('admin.accounts.syncFromCrsDesc') }}
      </div>
      <div class="crs-note">
        {{ t('admin.accounts.crsUpdateBehaviorNote') }}
      </div>
      <div class="crs-note crs-note--warning">
        {{ t('admin.accounts.crsVersionRequirement') }}
      </div>

      <div class="grid grid-cols-1 gap-4">
        <div>
          <label for="crs-base-url" class="input-label">{{ t('admin.accounts.crsBaseUrl') }}</label>
          <input
            id="crs-base-url"
            v-model="form.base_url"
            type="text"
            class="input"
            required
            :placeholder="t('admin.accounts.crsBaseUrlPlaceholder')"
          />
        </div>

        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label for="crs-username" class="input-label">{{ t('admin.accounts.crsUsername') }}</label>
            <input id="crs-username" v-model="form.username" type="text" class="input" required autocomplete="username" />
          </div>
          <div>
            <label for="crs-password" class="input-label">{{ t('admin.accounts.crsPassword') }}</label>
            <input
              id="crs-password"
              v-model="form.password"
              type="password"
              class="input"
              required
              autocomplete="current-password"
            />
          </div>
        </div>

        <label class="crs-checkbox-label">
          <input
            v-model="form.sync_proxies"
            type="checkbox"
            class="crs-checkbox"
          />
          {{ t('admin.accounts.syncProxies') }}
        </label>
      </div>
    </form>

    <!-- Step 2: Preview & select -->
    <div v-else-if="currentStep === 'preview' && previewResult" class="space-y-4">
      <!-- Existing accounts (read-only info) -->
      <div
        v-if="previewResult.existing_accounts.length"
        class="crs-panel"
      >
        <div class="crs-panel-title">
          {{ t('admin.accounts.crsExistingAccounts') }}
          <span class="crs-count">({{ previewResult.existing_accounts.length }})</span>
        </div>
        <div class="crs-compact-list">
          <div
            v-for="acc in previewResult.existing_accounts"
            :key="acc.crs_account_id"
            class="flex items-center gap-2 py-0.5"
          >
            <span class="crs-badge crs-badge--info">{{ acc.platform }} / {{ acc.type }}</span>
            <span class="truncate">{{ acc.name }}</span>
          </div>
        </div>
      </div>

      <!-- New accounts (selectable) -->
      <div v-if="previewResult.new_accounts.length">
        <div class="mb-2 flex items-center justify-between">
          <div class="crs-section-title">
            {{ t('admin.accounts.crsNewAccounts') }}
            <span class="crs-count">({{ previewResult.new_accounts.length }})</span>
          </div>
          <div class="flex gap-2">
            <button
              type="button"
              class="crs-link crs-link--primary"
              @click="selectAll"
            >{{ t('admin.accounts.crsSelectAll') }}</button>
            <button
              type="button"
              class="crs-link crs-link--neutral"
              @click="selectNone"
            >{{ t('admin.accounts.crsSelectNone') }}</button>
          </div>
        </div>
        <div class="crs-selection-list">
          <label
            v-for="acc in previewResult.new_accounts"
            :key="acc.crs_account_id"
            class="crs-select-row"
          >
            <input
              type="checkbox"
              :checked="selectedIds.has(acc.crs_account_id)"
              class="crs-checkbox"
              @change="toggleSelect(acc.crs_account_id)"
            />
            <span class="crs-badge crs-badge--success">{{ acc.platform }} / {{ acc.type }}</span>
            <span class="crs-row-name">{{ acc.name }}</span>
          </label>
        </div>
        <div class="crs-countline">
          {{ t('admin.accounts.crsSelectedCount', { count: selectedIds.size }) }}
        </div>
      </div>

      <!-- Sync options summary -->
      <div class="crs-sync-summary">
        <span>{{ t('admin.accounts.syncProxies') }}:</span>
        <span :class="['crs-sync-value', form.sync_proxies ? 'crs-sync-value--enabled' : 'crs-sync-value--disabled']">
          {{ form.sync_proxies ? t('common.yes') : t('common.no') }}
        </span>
      </div>

      <!-- No new accounts -->
      <div
        v-if="!previewResult.new_accounts.length"
        class="crs-empty"
      >
        {{ t('admin.accounts.crsNoNewAccounts') }}
        <span v-if="previewResult.existing_accounts.length">
          {{ t('admin.accounts.crsWillUpdate', { count: previewResult.existing_accounts.length }) }}
        </span>
      </div>
    </div>

    <!-- Step 3: Result -->
    <div v-else-if="currentStep === 'result' && result" class="space-y-4">
      <div class="crs-result-card">
        <div class="crs-section-title">
          {{ t('admin.accounts.syncResult') }}
        </div>
        <div class="crs-result-summary">
          {{ t('admin.accounts.syncResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="crs-error-title">
            {{ t('admin.accounts.syncErrors') }}
          </div>
          <div class="crs-error-log">
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.crs_account_id }} — {{ item.action
              }}{{ item.error ? `: ${item.error}` : '' }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <!-- Step 1: Input -->
        <template v-if="currentStep === 'input'">
          <button
            class="btn btn-secondary"
            type="button"
            :disabled="previewing"
            @click="handleClose"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            class="btn btn-primary"
            type="submit"
            form="sync-from-crs-form"
            :disabled="previewing"
          >
            {{ previewing ? t('admin.accounts.crsPreviewing') : t('admin.accounts.crsPreview') }}
          </button>
        </template>

        <!-- Step 2: Preview -->
        <template v-else-if="currentStep === 'preview'">
          <button
            class="btn btn-secondary"
            type="button"
            :disabled="syncing"
            @click="handleBack"
          >
            {{ t('admin.accounts.crsBack') }}
          </button>
          <button
            class="btn btn-primary"
            type="button"
            :disabled="syncing || hasNewButNoneSelected"
            @click="handleSync"
          >
            {{ syncing ? t('admin.accounts.syncing') : t('admin.accounts.syncNow') }}
          </button>
        </template>

        <!-- Step 3: Result -->
        <template v-else-if="currentStep === 'result'">
          <button class="btn btn-secondary" type="button" @click="handleClose">
            {{ t('common.close') }}
          </button>
        </template>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { PreviewFromCRSResult } from '@/api/admin/accounts'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'synced'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

type Step = 'input' | 'preview' | 'result'
const currentStep = ref<Step>('input')
const previewing = ref(false)
const syncing = ref(false)
const previewResult = ref<PreviewFromCRSResult | null>(null)
const selectedIds = ref(new Set<string>())
const result = ref<Awaited<ReturnType<typeof adminAPI.accounts.syncFromCrs>> | null>(null)

const form = reactive({
  base_url: '',
  username: '',
  password: '',
  sync_proxies: true
})

const hasNewButNoneSelected = computed(() => {
  if (!previewResult.value) return false
  return previewResult.value.new_accounts.length > 0 && selectedIds.value.size === 0
})

const errorItems = computed(() => {
  if (!result.value?.items) return []
  return result.value.items.filter(
    (i) => i.action === 'failed' || (i.action === 'skipped' && i.error !== 'not selected')
  )
})

watch(
  () => props.show,
  (open) => {
    if (open) {
      currentStep.value = 'input'
      previewResult.value = null
      selectedIds.value = new Set()
      result.value = null
      form.base_url = ''
      form.username = ''
      form.password = ''
      form.sync_proxies = true
    }
  }
)

const handleClose = () => {
  if (syncing.value || previewing.value) {
    return
  }
  emit('close')
}

const handleBack = () => {
  currentStep.value = 'input'
  previewResult.value = null
  selectedIds.value = new Set()
}

const selectAll = () => {
  if (!previewResult.value) return
  selectedIds.value = new Set(previewResult.value.new_accounts.map((a) => a.crs_account_id))
}

const selectNone = () => {
  selectedIds.value = new Set()
}

const toggleSelect = (id: string) => {
  const s = new Set(selectedIds.value)
  if (s.has(id)) {
    s.delete(id)
  } else {
    s.add(id)
  }
  selectedIds.value = s
}

const handlePreview = async () => {
  if (!form.base_url.trim() || !form.username.trim() || !form.password.trim()) {
    appStore.showError(t('admin.accounts.syncMissingFields'))
    return
  }

  previewing.value = true
  try {
    const res = await adminAPI.accounts.previewFromCrs({
      base_url: form.base_url.trim(),
      username: form.username.trim(),
      password: form.password
    })
    previewResult.value = res
    // Auto-select all new accounts
    selectedIds.value = new Set(res.new_accounts.map((a) => a.crs_account_id))
    currentStep.value = 'preview'
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.crsPreviewFailed'))
  } finally {
    previewing.value = false
  }
}

const handleSync = async () => {
  if (!form.base_url.trim() || !form.username.trim() || !form.password.trim()) {
    appStore.showError(t('admin.accounts.syncMissingFields'))
    return
  }

  syncing.value = true
  try {
    const res = await adminAPI.accounts.syncFromCrs({
      base_url: form.base_url.trim(),
      username: form.username.trim(),
      password: form.password,
      sync_proxies: form.sync_proxies,
      selected_account_ids: [...selectedIds.value]
    })
    result.value = res
    currentStep.value = 'result'

    if (res.failed > 0) {
      appStore.showError(t('admin.accounts.syncCompletedWithErrors', res))
    } else {
      appStore.showSuccess(t('admin.accounts.syncCompleted', res))
    }
    emit('synced')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.syncFailed'))
  } finally {
    syncing.value = false
  }
}
</script>

<style scoped>
.crs-dialog-copy,
.crs-result-summary,
.crs-checkbox-label,
.crs-row-name {
  color: var(--nm-ink-muted);
  font-size: 0.875rem;
  line-height: 1.5;
}

.crs-note,
.crs-panel,
.crs-empty,
.crs-result-card {
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius-lg);
  background: var(--nm-surface-soft);
}

.crs-note {
  padding: 0.75rem;
  color: var(--nm-ink-muted);
  font-size: 0.75rem;
  line-height: 1.5;
}

.crs-note--warning {
  border-color: color-mix(in srgb, var(--nm-warning) 34%, var(--nm-border-light));
  background: var(--nm-warning-soft);
  color: var(--nm-warning-text);
}

.crs-checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.crs-checkbox {
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-sm);
  accent-color: var(--nm-accent);
}

.crs-checkbox:focus-visible {
  outline: 3px solid var(--nm-accent);
  outline-offset: 2px;
}

.crs-panel,
.crs-result-card {
  padding: 0.75rem;
}

.crs-result-card {
  display: grid;
  gap: 0.5rem;
  padding: 1rem;
}

.crs-panel-title,
.crs-section-title {
  color: var(--nm-ink);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.4;
}

.crs-panel-title {
  margin-bottom: 0.5rem;
}

.crs-count,
.crs-countline {
  color: var(--nm-ink-faint);
  font-size: 0.75rem;
  font-weight: 400;
}

.crs-count {
  margin-left: 0.25rem;
}

.crs-compact-list,
.crs-selection-list,
.crs-error-log {
  overflow: auto;
}

.crs-compact-list {
  max-height: 8rem;
  color: var(--nm-ink-faint);
  font-size: 0.75rem;
  line-height: 1.5;
}

.crs-selection-list {
  max-height: 12rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
  padding: 0.5rem;
}

.crs-select-row {
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--nm-radius);
  padding: 0.375rem 0.5rem;
  transition: background-color 160ms ease;
}

.crs-select-row:hover {
  background: var(--nm-surface-soft);
}

.crs-badge {
  display: inline-block;
  flex-shrink: 0;
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius-sm);
  padding: 0.125rem 0.375rem;
  font-size: 0.625rem;
  font-weight: 600;
  line-height: 1.2;
}

.crs-badge--info {
  background: var(--nm-info-soft);
  color: var(--nm-info-text);
}

.crs-badge--success {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.crs-row-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.crs-link {
  font-size: 0.75rem;
  line-height: 1.3;
  transition: color 160ms ease;
}

.crs-link--primary {
  color: var(--nm-accent-text);
}

.crs-link--primary:hover {
  color: var(--nm-accent-strong);
}

.crs-link--neutral {
  color: var(--nm-ink-faint);
}

.crs-link--neutral:hover {
  color: var(--nm-ink-muted);
}

.crs-countline {
  margin-top: 0.25rem;
}

.crs-sync-summary {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--nm-ink-faint);
  font-size: 0.75rem;
  line-height: 1.4;
}

.crs-sync-value--enabled {
  color: var(--nm-success-text);
}

.crs-sync-value--disabled {
  color: var(--nm-ink-faint);
}

.crs-empty {
  padding: 1rem;
  text-align: center;
  color: var(--nm-ink-faint);
  font-size: 0.875rem;
  line-height: 1.5;
}

.crs-error-title {
  color: var(--nm-danger-text);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.4;
}

.crs-error-log {
  margin-top: 0.5rem;
  max-height: 12rem;
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius-lg);
  background: var(--nm-surface-soft);
  padding: 0.75rem;
  color: var(--nm-ink-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 0.75rem;
  line-height: 1.5;
}
</style>
