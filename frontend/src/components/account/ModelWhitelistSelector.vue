<template>
  <div>
    <!-- Multi-select Dropdown -->
    <div class="relative mb-3">
      <div
        @click="toggleDropdown"
        class="model-selector-trigger cursor-pointer"
      >
        <div class="grid grid-cols-2 gap-1.5">
          <span
            v-for="model in modelValue"
            :key="model"
            class="model-chip"
          >
            <span class="flex items-center gap-1 truncate">
              <ModelIcon :model="model" size="14px" />
              <span class="truncate">{{ model }}</span>
            </span>
            <button
              type="button"
              @click.stop="removeModel(model)"
              class="model-chip-remove"
            >
              <Icon name="x" size="xs" class="h-3.5 w-3.5" :stroke-width="2" />
            </button>
          </span>
        </div>
        <div class="model-selector-footer">
          <span class="model-count">{{ t('admin.accounts.modelCount', { count: modelValue.length }) }}</span>
          <svg class="model-chevron h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>
      <!-- Dropdown List -->
      <div
        v-if="showDropdown"
        class="model-dropdown"
      >
        <div class="model-dropdown-search">
          <input
            v-model="searchQuery"
            type="text"
            class="input w-full text-sm"
            :placeholder="t('admin.accounts.searchModels')"
            @click.stop
          />
        </div>
        <div class="max-h-52 overflow-auto">
          <button
            v-for="model in filteredModels"
            :key="model.value"
            type="button"
            @click="toggleModel(model.value)"
            class="model-option"
          >
            <span
              :class="[
                'model-checkmark',
                modelValue.includes(model.value) ? 'model-checkmark--selected' : ''
              ]"
            >
              <svg v-if="modelValue.includes(model.value)" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
              </svg>
            </span>
            <ModelIcon :model="model.value" size="18px" />
            <span class="model-option-name">{{ model.value }}</span>
          </button>
          <div v-if="filteredModels.length === 0" class="model-empty">
            {{ t('admin.accounts.noMatchingModels') }}
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div class="mb-4 flex flex-wrap gap-2">
      <button
        type="button"
        @click="fillRelated"
        class="model-action model-action--neutral"
      >
        {{ t('admin.accounts.fillRelatedModels') }}
      </button>
      <button
        v-if="canSyncUpstream"
        type="button"
        @click="syncUpstreamModels"
        :disabled="isSyncingUpstream"
        class="model-action model-action--success"
      >
        {{ isSyncingUpstream ? t('admin.accounts.syncUpstreamModelsLoading') : t('admin.accounts.syncUpstreamModels') }}
      </button>
      <button
        type="button"
        @click="clearAll"
        class="model-action model-action--danger"
      >
        {{ t('admin.accounts.clearAllModels') }}
      </button>
    </div>

    <!-- Custom Model Input -->
    <div class="mb-3">
      <label class="model-field-label">{{ t('admin.accounts.customModelName') }}</label>
      <div class="flex gap-2">
        <input
          v-model="customModel"
          type="text"
          class="input flex-1"
          :placeholder="t('admin.accounts.enterCustomModelName')"
          @keydown.enter.prevent="handleEnter"
          @compositionstart="isComposing = true"
          @compositionend="isComposing = false"
        />
        <button
          type="button"
          @click="addCustom"
          class="model-add-button"
        >
          {{ t('admin.accounts.addModel') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { accountsAPI } from '@/api/admin/accounts'
import type { SyncUpstreamPreviewParams } from '@/api/admin/accounts'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { allModels, getModelsByPlatform } from '@/composables/useModelWhitelist'

const { t } = useI18n()

const props = defineProps<{
  modelValue: string[]
  platform?: string
  platforms?: string[]
  accountId?: number
  syncCredentials?: {
    platform: string
    type: string
    base_url?: string
    api_key: string
  }
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const appStore = useAppStore()

const showDropdown = ref(false)
const searchQuery = ref('')
const customModel = ref('')
const isComposing = ref(false)
const isSyncingUpstream = ref(false)
const normalizedPlatforms = computed(() => {
  const rawPlatforms =
    props.platforms && props.platforms.length > 0
      ? props.platforms
      : props.platform
        ? [props.platform]
        : []

  return Array.from(
    new Set(
      rawPlatforms
        .map(platform => platform?.trim())
        .filter((platform): platform is string => Boolean(platform))
    )
  )
})

const upstreamSyncPlatforms = new Set([
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'grok',
  'kimi',
  'zhipu',
  'deepseek'
])
const canSyncUpstream = computed(() => {
  if (props.accountId) {
    if (normalizedPlatforms.value.length === 0) return true
    return normalizedPlatforms.value.some(platform => upstreamSyncPlatforms.has(platform.toLowerCase()))
  }
  if (props.syncCredentials) {
    return upstreamSyncPlatforms.has(props.syncCredentials.platform.toLowerCase())
  }
  return false
})

const availableOptions = computed(() => {
  if (normalizedPlatforms.value.length === 0) {
    return allModels
  }

  const allowedModels = new Set<string>()
  for (const platform of normalizedPlatforms.value) {
    for (const model of getModelsByPlatform(platform)) {
      allowedModels.add(model)
    }
  }

  return allModels.filter(model => allowedModels.has(model.value))
})

const filteredModels = computed(() => {
  const query = searchQuery.value.toLowerCase().trim()
  if (!query) return availableOptions.value
  return availableOptions.value.filter(
    m => m.value.toLowerCase().includes(query) || m.label.toLowerCase().includes(query)
  )
})

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
  if (!showDropdown.value) searchQuery.value = ''
}

const removeModel = (model: string) => {
  emit('update:modelValue', props.modelValue.filter(m => m !== model))
}

const toggleModel = (model: string) => {
  if (props.modelValue.includes(model)) {
    removeModel(model)
  } else {
    emit('update:modelValue', [...props.modelValue, model])
  }
}

const addCustom = () => {
  const model = customModel.value.trim()
  if (!model) return
  if (props.modelValue.includes(model)) {
    appStore.showInfo(t('admin.accounts.modelExists'))
    return
  }
  emit('update:modelValue', [...props.modelValue, model])
  customModel.value = ''
}

const handleEnter = () => {
  if (!isComposing.value) addCustom()
}

const fillRelated = () => {
  const newModels = [...props.modelValue]
  for (const platform of normalizedPlatforms.value) {
    for (const model of getModelsByPlatform(platform)) {
      if (!newModels.includes(model)) {
        newModels.push(model)
      }
    }
  }
  emit('update:modelValue', newModels)
}

const syncUpstreamModels = async () => {
  if (isSyncingUpstream.value) return
  if (!props.accountId && !props.syncCredentials) return

  isSyncingUpstream.value = true
  try {
    let result
    if (props.accountId) {
      result = await accountsAPI.syncUpstreamModels(props.accountId)
    } else if (props.syncCredentials) {
      result = await accountsAPI.syncUpstreamModelsPreview(props.syncCredentials as SyncUpstreamPreviewParams)
    } else {
      return
    }

    const upstreamModels = result.models.map(model => model.trim()).filter(Boolean)
    if (upstreamModels.length === 0) {
      appStore.showInfo(t('admin.accounts.syncUpstreamModelsEmpty'))
      return
    }

    const newModels = [...props.modelValue]
    let addedCount = 0
    for (const model of upstreamModels) {
      if (!newModels.includes(model)) {
        newModels.push(model)
        addedCount += 1
      }
    }

    emit('update:modelValue', newModels)
    if (addedCount > 0) {
      appStore.showSuccess(t('admin.accounts.syncUpstreamModelsSuccess', { count: addedCount, total: upstreamModels.length }))
    } else {
      appStore.showInfo(t('admin.accounts.syncUpstreamModelsNoChanges', { count: upstreamModels.length }))
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : t('admin.accounts.syncUpstreamModelsFailed')
    appStore.showError(t('admin.accounts.syncUpstreamModelsError', { message }))
  } finally {
    isSyncingUpstream.value = false
  }
}

const clearAll = () => {
  emit('update:modelValue', [])
}

</script>

<style scoped>
.model-selector-trigger {
  cursor: pointer;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
  background: var(--nm-surface);
  padding: 0.5rem 0.75rem;
}

.model-chip {
  display: inline-flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.25rem;
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
  padding: 0.25rem 0.5rem;
  color: var(--nm-ink-muted);
  font-size: 0.75rem;
  line-height: 1.25;
}

.model-chip-remove {
  flex-shrink: 0;
  border-radius: var(--nm-radius-sm);
  color: var(--nm-ink-faint);
  transition: background-color 160ms ease, color 160ms ease;
}

.model-chip-remove:hover {
  background: var(--nm-surface-alt);
  color: var(--nm-ink);
}

.model-selector-footer {
  margin-top: 0.5rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--nm-border-light);
  padding-top: 0.5rem;
}

.model-count,
.model-chevron,
.model-empty {
  color: var(--nm-ink-faint);
}

.model-count {
  font-size: 0.75rem;
  line-height: 1.25;
}

.model-dropdown {
  position: absolute;
  left: 0;
  right: 0;
  top: 100%;
  z-index: 50;
  margin-top: 0.25rem;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-lg);
  background: var(--nm-surface);
}

.model-dropdown-search {
  position: sticky;
  top: 0;
  border-bottom: 1px solid var(--nm-border-light);
  background: var(--nm-surface);
  padding: 0.5rem;
}

.model-option {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  text-align: left;
  color: var(--nm-ink);
  font-size: 0.875rem;
  transition: background-color 160ms ease;
}

.model-option:hover {
  background: var(--nm-surface-soft);
}

.model-checkmark {
  display: flex;
  height: 1rem;
  width: 1rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius-sm);
  color: var(--nm-on-accent);
}

.model-checkmark--selected {
  border-color: var(--nm-accent);
  background: var(--nm-accent);
}

.model-option-name {
  min-width: 0;
  overflow: hidden;
  color: var(--nm-ink);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.model-empty {
  padding: 1rem 0.75rem;
  text-align: center;
  font-size: 0.875rem;
}

.model-action,
.model-add-button {
  border: 1px solid var(--nm-border);
  border-radius: var(--nm-radius);
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 600;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.model-action:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.model-action--neutral,
.model-add-button {
  color: var(--nm-accent-text);
}

.model-action--neutral:hover,
.model-add-button:hover {
  border-color: var(--nm-accent);
  background: var(--nm-accent-soft);
}

.model-action--success {
  border-color: var(--nm-success);
  color: var(--nm-success-text);
}

.model-action--success:hover {
  background: var(--nm-success-soft);
}

.model-action--danger {
  border-color: var(--nm-danger);
  color: var(--nm-danger-text);
}

.model-action--danger:hover {
  background: var(--nm-danger-soft);
}

.model-field-label {
  margin-bottom: 0.375rem;
  display: block;
  color: var(--nm-ink-muted);
  font-size: 0.875rem;
  font-weight: 600;
  line-height: 1.4;
}
</style>
