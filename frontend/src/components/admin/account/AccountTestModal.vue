<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.testAccountConnection')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <!-- Account Info Card -->
      <div
        v-if="account"
        class="test-header flex items-center justify-between p-3"
      >
        <div class="flex items-center gap-3">
          <div class="test-header-icon flex h-10 w-10 items-center justify-center">
            <Icon name="play" size="md" :stroke-width="2" />
          </div>
          <div>
            <div class="test-title font-semibold">{{ account.name }}</div>
            <div class="test-muted flex items-center gap-1.5 text-xs">
              <span class="test-type-badge px-1.5 py-0.5 text-[10px] font-medium uppercase">
                {{ account.type }}
              </span>
              <span>{{ t('admin.accounts.account') }}</span>
            </div>
          </div>
        </div>
        <span
          :class="[
            'test-status px-2.5 py-1 text-xs font-semibold',
            account.status === 'active'
              ? 'test-status-active'
              : 'test-status-muted'
          ]"
        >
          {{ account.status }}
        </span>
      </div>

      <div class="space-y-1.5">
        <label class="test-field-label text-sm font-medium">
          {{ t('admin.accounts.selectTestModel') }}
        </label>
        <Select
          v-model="selectedModelId"
          :options="availableModels"
          :disabled="loadingModels || status === 'connecting'"
          value-key="id"
          label-key="display_name"
          :placeholder="loadingModels ? t('common.loading') + '...' : t('admin.accounts.selectTestModel')"
        />
      </div>

      <div v-if="isOpenAIAccount" class="space-y-1.5">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.openai.testMode') }}
        </label>
        <Select
          v-model="testMode"
          :options="openAITestModeOptions"
          :disabled="status === 'connecting'"
        />
      </div>

      <div v-if="supportsImageTest" class="space-y-1.5">
        <TextArea
          v-model="testPrompt"
          :label="t('admin.accounts.imagePromptLabel')"
          :placeholder="t('admin.accounts.imagePromptPlaceholder')"
          :hint="t('admin.accounts.imageTestHint')"
          :disabled="status === 'connecting'"
          rows="3"
        />
      </div>

      <!-- Terminal Output -->
      <div class="group relative">
        <div
          ref="terminalRef"
          class="test-terminal max-h-[240px] min-h-[120px] overflow-y-auto p-4 font-mono text-sm"
        >
          <!-- Status Line -->
          <div v-if="status === 'idle'" class="test-log-muted flex items-center gap-2">
            <Icon name="play" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.readyToTest') }}</span>
          </div>
          <div v-else-if="status === 'connecting'" class="test-log-warning flex items-center gap-2">
            <Icon name="refresh" size="sm" class="animate-spin" :stroke-width="2" />
            <span>{{ t('admin.accounts.connectingToApi') }}</span>
          </div>

          <!-- Output Lines -->
          <div v-for="(line, index) in outputLines" :key="index" :class="['test-log-line', line.class]">
            {{ line.text }}
          </div>

          <!-- Streaming Content -->
          <div v-if="streamingContent" class="test-log-success">
            {{ streamingContent }}<span class="animate-pulse">_</span>
          </div>

          <!-- Result Status -->
          <div
            v-if="status === 'success'"
            class="test-result-line test-log-success mt-3 flex items-center gap-2 pt-3"
          >
            <Icon name="check" size="sm" :stroke-width="2" />
            <span>{{ t('admin.accounts.testCompleted') }}</span>
          </div>
          <div
            v-else-if="status === 'error'"
            class="test-result-line test-log-danger mt-3 flex items-center gap-2 pt-3"
          >
            <Icon name="x" size="sm" :stroke-width="2" />
            <span>{{ errorMessage }}</span>
          </div>
        </div>

        <!-- Copy Button -->
        <button
          v-if="outputLines.length > 0"
          @click="copyOutput"
          class="test-copy-button absolute right-2 top-2 p-1.5 opacity-0 transition-all group-hover:opacity-100"
          :title="t('admin.accounts.copyOutput')"
          :aria-label="t('admin.accounts.copyOutput')"
        >
          <Icon name="link" size="sm" :stroke-width="2" />
        </button>
      </div>

      <div v-if="generatedImages.length > 0" class="space-y-2">
        <div class="test-field-label text-xs font-medium">
          {{ t('admin.accounts.imagePreview') }}
        </div>
        <div class="flex flex-wrap justify-center gap-3">
          <div
            v-for="(image, index) in generatedImages"
            :key="`${image.url}-${index}`"
            class="test-image-card group/img relative cursor-pointer overflow-hidden transition-colors"
            @click="previewImageUrl = image.url"
          >
            <img :src="image.url" :alt="`test-image-${index + 1}`" class="max-h-[360px] w-full object-contain" />
            <div class="test-image-overlay absolute inset-0 flex items-center justify-center opacity-0 transition-opacity group-hover/img:opacity-100">
              <Icon name="eye" size="lg" :stroke-width="2" />
            </div>
            <div class="test-image-meta px-3 py-1.5 text-xs">
              {{ image.mimeType || 'image/*' }}
            </div>
          </div>
        </div>
      </div>

      <!-- Image Lightbox -->
      <Teleport to="body">
        <Transition name="fade">
          <div
            v-if="previewImageUrl"
            class="test-lightbox fixed inset-0 z-[100] flex items-center justify-center p-4"
            @click.self="previewImageUrl = ''"
          >
            <button
              class="test-lightbox-close absolute right-4 top-4 p-2 transition-colors"
              @click="previewImageUrl = ''"
              :aria-label="t('common.close')"
            >
              <Icon name="x" size="lg" :stroke-width="2" />
            </button>
            <img
              :src="previewImageUrl"
              alt="preview"
              class="test-lightbox-image max-h-[90vh] max-w-[90vw] object-contain"
            />
          </div>
        </Transition>
      </Teleport>

      <!-- Test Info -->
      <div class="test-muted flex items-center justify-between px-1 text-xs">
        <div class="flex items-center gap-3">
          <span class="flex items-center gap-1">
            <Icon name="grid" size="sm" :stroke-width="2" />
            {{ t('admin.accounts.testModel') }}
          </span>
        </div>
        <span class="flex items-center gap-1">
          <Icon name="chat" size="sm" :stroke-width="2" />
          {{
            supportsImageTest
              ? t('admin.accounts.imageTestMode')
              : t('admin.accounts.testPrompt')
          }}
        </span>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="test-button test-button-secondary px-4 py-2 text-sm font-medium transition-colors"
        >
          {{ t('common.close') }}
        </button>
        <button
          @click="startTest"
          :disabled="status === 'connecting' || !selectedModelId"
          :class="[
            'test-button flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors',
            testActionClass
          ]"
        >
          <Icon
            v-if="status === 'connecting'"
            name="refresh"
            size="sm"
            class="animate-spin"
            :stroke-width="2"
          />
          <Icon v-else-if="status === 'idle'" name="play" size="sm" :stroke-width="2" />
          <Icon v-else name="refresh" size="sm" :stroke-width="2" />
          <span>
            {{
              status === 'connecting'
                ? t('admin.accounts.testing')
                : status === 'idle'
                  ? t('admin.accounts.startTest')
                  : t('admin.accounts.retry')
            }}
          </span>
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import { Icon } from '@/components/icons'
import { useClipboard } from '@/composables/useClipboard'
import { buildApiUrl } from '@/api/client'
import { ADMIN_UI_REQUEST_HEADER } from '@/api/adminUIRequest'
import { adminAPI } from '@/api/admin'
import type { Account, ClaudeModel } from '@/types'

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

interface OutputLine {
  text: string
  class: string
}

interface PreviewImage {
  url: string
  mimeType?: string
}

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const terminalRef = ref<HTMLElement | null>(null)
const status = ref<'idle' | 'connecting' | 'success' | 'error'>('idle')
const outputLines = ref<OutputLine[]>([])
const streamingContent = ref('')
const errorMessage = ref('')
const availableModels = ref<ClaudeModel[]>([])
const selectedModelId = ref('')
const testPrompt = ref('')
const loadingModels = ref(false)
let abortController: AbortController | null = null
const generatedImages = ref<PreviewImage[]>([])
const previewImageUrl = ref('')
const testMode = ref<'default' | 'compact'>('default')
const isOpenAIAccount = computed(() => props.account?.platform === 'openai')
const openAITestModeOptions = computed(() => [
  { value: 'default', label: t('admin.accounts.openai.testModeDefault') },
  { value: 'compact', label: t('admin.accounts.openai.testModeCompact') }
])
const prioritizedGeminiModels = ['gemini-3.1-flash-image', 'gemini-2.5-flash-image', 'gemini-3.5-flash', 'gemini-2.5-flash', 'gemini-2.5-pro', 'gemini-3-flash-preview', 'gemini-3-pro-preview', 'gemini-2.0-flash']
const supportsGeminiImageTest = computed(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gemini-') || !modelID.includes('-image')) return false

  return props.account?.platform === 'gemini' || (props.account?.platform === 'antigravity' && props.account?.type === 'apikey')
})

const supportsOpenAIImageTest = computed(() => {
  const modelID = selectedModelId.value.toLowerCase()
  if (!modelID.startsWith('gpt-image-')) return false
  return props.account?.platform === 'openai'
})

const supportsImageTest = computed(() => supportsGeminiImageTest.value || supportsOpenAIImageTest.value)

const testActionClass = computed(() => {
  if (status.value === 'connecting' || !selectedModelId.value) return 'test-button-disabled'
  if (status.value === 'success') return 'test-button-success'
  if (status.value === 'error') return 'test-button-warning'
  return 'test-button-primary'
})

const sortTestModels = (models: ClaudeModel[]) => {
  const priorityMap = new Map(prioritizedGeminiModels.map((id, index) => [id, index]))

  return [...models].sort((a, b) => {
    const aPriority = priorityMap.get(a.id) ?? Number.MAX_SAFE_INTEGER
    const bPriority = priorityMap.get(b.id) ?? Number.MAX_SAFE_INTEGER
    if (aPriority !== bPriority) return aPriority - bPriority
    return 0
  })
}

// Load available models when modal opens
watch(
  () => props.show,
  async (newVal) => {
    if (newVal && props.account) {
      testPrompt.value = ''
      testMode.value = 'default'
      resetState()
      await loadAvailableModels()
    } else {
      abortStream()
    }
  }
)

watch(selectedModelId, () => {
  if (supportsImageTest.value && !testPrompt.value.trim()) {
    testPrompt.value = t('admin.accounts.imagePromptDefault')
  }
})

const loadAvailableModels = async () => {
  if (!props.account) return

  loadingModels.value = true
  selectedModelId.value = '' // Reset selection before loading
  try {
    const models = await adminAPI.accounts.getAvailableModels(props.account.id)
    availableModels.value = props.account.platform === 'gemini' || props.account.platform === 'antigravity'
      ? sortTestModels(models)
      : models
    // Default selection by platform
    if (availableModels.value.length > 0) {
      if (props.account.platform === 'gemini') {
        selectedModelId.value = availableModels.value[0].id
      } else {
        // Try to select Sonnet as default, otherwise use first model
        const sonnetModel = availableModels.value.find((m) => m.id.includes('sonnet'))
        selectedModelId.value = sonnetModel?.id || availableModels.value[0].id
      }
    }
  } catch (error) {
    console.error('Failed to load available models:', error)
    // Fallback to empty list
    availableModels.value = []
    selectedModelId.value = ''
  } finally {
    loadingModels.value = false
  }
}

const resetState = () => {
  status.value = 'idle'
  outputLines.value = []
  streamingContent.value = ''
  errorMessage.value = ''
  generatedImages.value = []
  previewImageUrl.value = ''
}

const handleClose = () => {
  abortStream()
  emit('close')
}

const abortStream = () => {
  if (abortController) {
    abortController.abort()
    abortController = null
  }
}

const addLine = (text: string = '', className: string = 'test-log-default') => {
  outputLines.value.push({ text, class: className })
  scrollToBottom()
}

const scrollToBottom = async () => {
  await nextTick()
  if (terminalRef.value) {
    terminalRef.value.scrollTop = terminalRef.value.scrollHeight
  }
}

const startTest = async () => {
  if (!props.account || !selectedModelId.value) return

  resetState()
  status.value = 'connecting'
  addLine(t('admin.accounts.startingTestForAccount', { name: props.account.name }), 'test-log-info')
  addLine(t('admin.accounts.testAccountTypeLabel', { type: props.account.type }), 'test-log-muted')
  addLine()

  abortStream()

  abortController = new AbortController()

  try {
    const requestBody: {
      model_id: string
      prompt: string
      mode?: 'default' | 'compact'
    } = {
      model_id: selectedModelId.value,
      prompt: supportsImageTest.value ? testPrompt.value.trim() : ''
    }
    if (isOpenAIAccount.value) {
      requestBody.mode = testMode.value
    }

    // Use the configured API base; EventSource does not support POST.
    const url = buildApiUrl(`/admin/accounts/${props.account.id}/test`)

    // Use fetch with streaming for SSE since EventSource doesn't support POST
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
        'Content-Type': 'application/json',
        [ADMIN_UI_REQUEST_HEADER]: '1'
      },
      body: JSON.stringify(requestBody),
      signal: abortController.signal
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const jsonStr = line.slice(6).trim()
          if (jsonStr) {
            try {
              const event = JSON.parse(jsonStr)
              handleEvent(event)
            } catch (e) {
              console.error('Failed to parse SSE event:', e)
            }
          }
        }
      }
    }
  } catch (error: unknown) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      status.value = 'idle'
      return
    }
    status.value = 'error'
    const msg = error instanceof Error ? error.message : 'Unknown error'
    errorMessage.value = msg
    addLine(`Error: ${msg}`, 'test-log-danger')
  }
}

const handleEvent = (event: {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
  image_url?: string
  mime_type?: string
}) => {
  switch (event.type) {
    case 'test_start':
      addLine(t('admin.accounts.connectedToApi'), 'test-log-success')
      if (event.model) {
        addLine(t('admin.accounts.usingModel', { model: event.model }), 'test-log-info')
      }
      addLine(
        supportsImageTest.value
            ? t('admin.accounts.sendingImageRequest')
            : t('admin.accounts.sendingTestMessage'),
        'test-log-muted'
      )
      addLine()
      addLine(t('admin.accounts.response'), 'test-log-warning')
      break

    case 'content':
      if (event.text) {
        streamingContent.value += event.text
        scrollToBottom()
      }
      break

    case 'image':
      if (event.image_url) {
        generatedImages.value.push({
          url: event.image_url,
          mimeType: event.mime_type
        })
        addLine(t('admin.accounts.imageReceived', { count: generatedImages.value.length }), 'test-log-info')
      }
      break

    case 'status':
      if (event.text) {
        addLine(event.text, 'text-cyan-300')
      }
      break

    case 'test_complete':
      // Move streaming content to output lines
      if (streamingContent.value) {
        addLine(streamingContent.value, 'test-log-success')
        streamingContent.value = ''
      }
      if (event.success) {
        status.value = 'success'
      } else {
        status.value = 'error'
        errorMessage.value = event.error || 'Test failed'
      }
      break

    case 'error':
      status.value = 'error'
      errorMessage.value = event.error || 'Unknown error'
      if (streamingContent.value) {
        addLine(streamingContent.value, 'test-log-success')
        streamingContent.value = ''
      }
      break
  }
}

const copyOutput = () => {
  const text = outputLines.value.map((l) => l.text).join('\n')
  copyToClipboard(text, t('admin.accounts.outputCopied'))
}
</script>

<style scoped>
.test-header {
  border: 1px solid var(--nm-border-light);
  border-radius: var(--nm-radius);
  background: var(--nm-surface-soft);
}

.test-header-icon,
.test-type-badge,
.test-status,
.test-terminal,
.test-copy-button,
.test-image-card,
.test-lightbox-close,
.test-lightbox-image,
.test-button {
  border-radius: var(--nm-radius-sm);
}

.test-header-icon {
  background: var(--nm-accent);
  color: var(--nm-on-accent);
}

.test-title {
  color: var(--nm-ink);
}

.test-muted {
  color: var(--nm-ink-faint);
}

.test-field-label {
  color: var(--nm-ink-muted);
}

.test-type-badge,
.test-status {
  border: 1px solid var(--nm-border-light);
}

.test-type-badge {
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
}

.test-status-active {
  background: var(--nm-success-soft);
  color: var(--nm-success-text);
}

.test-status-muted {
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
}

.test-terminal {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface-alt);
  color: var(--nm-ink-muted);
}

.test-log-line {
  min-height: 1.25rem;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.test-log-default {
  color: var(--nm-ink-muted);
}

.test-log-muted {
  color: var(--nm-ink-faint);
}

.test-log-info {
  color: var(--nm-info-text);
}

.test-log-success {
  color: var(--nm-success-text);
}

.test-log-warning {
  color: var(--nm-warning-text);
}

.test-log-danger {
  color: var(--nm-danger-text);
}

.test-result-line {
  border-top: 1px solid var(--nm-border-light);
}

.test-copy-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--nm-border-light);
  background: color-mix(in srgb, var(--nm-surface) 88%, transparent);
  color: var(--nm-ink-faint);
  cursor: pointer;
}

.test-copy-button:hover {
  background: var(--nm-accent-soft);
  color: var(--nm-accent-text);
}

.test-image-card {
  border: 1px solid var(--nm-border);
  background: var(--nm-surface);
}

.test-image-card:hover {
  border-color: var(--nm-accent);
}

.test-image-overlay {
  background: color-mix(in srgb, var(--nm-ink) 24%, transparent);
  color: var(--nm-bg);
}

.test-image-meta {
  border-top: 1px solid var(--nm-border-light);
  color: var(--nm-ink-faint);
}

.test-lightbox {
  background: color-mix(in srgb, var(--nm-ink) 82%, transparent);
}

.test-lightbox-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in srgb, var(--nm-bg) 35%, transparent);
  background: color-mix(in srgb, var(--nm-ink) 55%, transparent);
  color: var(--nm-bg);
  cursor: pointer;
}

.test-lightbox-close:hover {
  background: color-mix(in srgb, var(--nm-ink) 72%, transparent);
}

.test-lightbox-image {
  border: 1px solid color-mix(in srgb, var(--nm-bg) 25%, transparent);
  background: var(--nm-surface);
}

.test-button {
  border: 1px solid transparent;
  cursor: pointer;
}

.test-button-secondary {
  border-color: var(--nm-border);
  background: var(--nm-surface);
  color: var(--nm-ink-muted);
}

.test-button-secondary:hover {
  border-color: var(--nm-ink-muted);
  color: var(--nm-ink);
}

.test-button-primary {
  border-color: var(--nm-accent);
  background: var(--nm-accent);
  color: var(--nm-on-accent);
}

.test-button-primary:hover {
  border-color: var(--nm-accent-strong);
  background: var(--nm-accent-strong);
}

.test-button-success {
  border-color: var(--nm-success);
  background: var(--nm-success);
  color: var(--nm-on-accent);
}

.test-button-warning {
  border-color: var(--nm-warning);
  background: var(--nm-warning);
  color: var(--nm-bg);
}

.test-button-disabled {
  border-color: var(--nm-border);
  background: var(--nm-surface-soft);
  color: var(--nm-ink-faint);
  cursor: not-allowed;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
