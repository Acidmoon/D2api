<template>
  <AppLayout>
    <div class="space-y-4">
      <header>
        <h1 class="text-[28px] font-semibold leading-9 tracking-[-0.01em] text-foreground">
          {{ t('imageStudio.title') }}
        </h1>
        <p class="mt-2 text-sm text-muted-foreground">{{ t('imageStudio.description') }}</p>
      </header>
    <!-- 顶部：选择创作分组的 API Key -->
    <div class="card">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div class="w-full lg:max-w-lg">
          <label class="input-label" for="image-studio-key">{{ t('imageStudio.key.label') }}</label>
          <Select
            id="image-studio-key"
            v-model="selectedKeyId"
            :options="keyOptions"
            :placeholder="t('imageStudio.key.placeholder')"
            :disabled="accessLoading || keyOptions.length === 0"
            :loading="accessLoading"
            searchable
            @change="handleKeyChange"
          />
          <p class="input-hint">
            {{ selectedKey ? t('imageStudio.key.hint', { group: selectedKey.group?.name || '-' }) : t('imageStudio.key.empty') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <span v-if="selectedKey" class="badge badge-primary">{{ selectedKey.group?.name || '-' }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="accessLoading" @click="reloadKeys">
            {{ t('imageStudio.key.refresh') }}
          </button>
          <router-link v-if="keyOptions.length === 0 && !accessLoading" to="/keys" class="btn btn-primary btn-sm">
            {{ t('imageStudio.key.create') }}
          </router-link>
        </div>
      </div>
    </div>

    <div class="grid gap-4 xl:grid-cols-[380px_minmax(0,1fr)]">
      <!-- 左侧：提示词 + 模型 -->
      <div class="card space-y-4 self-start">
        <div>
          <label class="input-label" for="image-studio-model">{{ t('imageStudio.form.model') }}</label>
          <Select
            id="image-studio-model"
            v-model="selectedModel"
            :options="modelOptions"
            :placeholder="modelsLoading ? t('imageStudio.form.modelsLoading') : t('imageStudio.form.modelPlaceholder')"
            :disabled="!selectedKey || modelsLoading || modelOptions.length === 0"
            :loading="modelsLoading"
            searchable
          />
          <p v-if="!modelsLoading && selectedKey && modelOptions.length === 0" class="input-hint text-warning">
            {{ t('imageStudio.form.modelsEmpty') }}
          </p>
        </div>

        <div>
          <label class="input-label" for="image-studio-prompt">{{ t('imageStudio.form.prompt') }}</label>
          <TextArea
            id="image-studio-prompt"
            v-model="prompt"
            :rows="8"
            :placeholder="t('imageStudio.form.promptPlaceholder')"
          />
          <p class="input-hint text-right">{{ prompt.length }}/4000</p>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="input-label" for="image-studio-size">{{ t('imageStudio.form.size') }}</label>
            <Select id="image-studio-size" v-model="size" :options="sizeOptions" />
          </div>
          <div>
            <label class="input-label" for="image-studio-count">{{ t('imageStudio.form.count') }}</label>
            <Select id="image-studio-count" v-model="count" :options="countOptions" />
          </div>
        </div>

        <button
          type="button"
          class="btn btn-primary w-full"
          :disabled="!canGenerate"
          @click="generate"
        >
          {{ generating ? t('imageStudio.form.generating') : t('imageStudio.form.generate') }}
        </button>
        <p class="text-xs text-muted-foreground">{{ t('imageStudio.form.costHint') }}</p>
      </div>

      <!-- 右侧：生成结果 -->
      <div class="card min-h-[480px]">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-base font-semibold text-foreground">{{ t('imageStudio.result.title') }}</h2>
          <button
            v-if="results.length > 0"
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="generating"
            @click="clearResults"
          >
            {{ t('imageStudio.result.clear') }}
          </button>
        </div>

        <div v-if="accessLoading || (modelsLoading && results.length === 0)" class="flex items-center justify-center py-20">
          <LoadingSpinner size="lg" />
        </div>

        <EmptyState
          v-else-if="keyOptions.length === 0"
          :title="t('imageStudio.result.noKeyTitle')"
          :description="t('imageStudio.result.noKeyDescription')"
          :action-text="t('imageStudio.key.create')"
          action-to="/keys"
        />

        <EmptyState
          v-else-if="results.length === 0 && !generating"
          :title="t('imageStudio.result.emptyTitle')"
          :description="t('imageStudio.result.emptyDescription')"
        />

        <div v-else class="space-y-4">
          <div v-if="generating" class="flex items-center gap-2 text-sm text-muted-foreground">
            <LoadingSpinner size="sm" />
            <span>{{ t('imageStudio.result.loading') }}</span>
          </div>

          <div v-if="lastError" class="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">
            {{ lastError }}
          </div>

          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
            <figure
              v-for="(item, index) in results"
              :key="item.id"
              class="group relative overflow-hidden rounded-xl border border-border bg-muted"
            >
              <button
                type="button"
                class="block w-full"
                :aria-label="t('imageStudio.result.open')"
                @click="openLightbox(index)"
              >
                <img
                  :src="item.previewUrl"
                  :alt="item.prompt"
                  class="aspect-square w-full cursor-zoom-in object-cover transition-transform duration-200 group-hover:scale-[1.02]"
                  loading="lazy"
                />
              </button>
              <div
                class="pointer-events-none absolute inset-x-0 bottom-0 flex items-center justify-between gap-2 bg-gradient-to-t from-black/70 to-transparent px-3 py-2 opacity-0 transition-opacity group-hover:opacity-100"
              >
                <span class="truncate text-xs text-white/90">{{ item.model }}</span>
                <span class="pointer-events-auto flex items-center gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="openLightbox(index)">
                    {{ t('imageStudio.result.open') }}
                  </button>
                  <button type="button" class="btn btn-primary btn-sm" :disabled="item.downloading" @click="downloadImage(index)">
                    {{ item.downloading ? t('imageStudio.result.downloading') : t('imageStudio.result.download') }}
                  </button>
                </span>
              </div>
            </figure>
          </div>

          <p v-if="revisedPrompts.length > 0" class="text-xs text-muted-foreground">
            {{ t('imageStudio.result.revisedPrompt') }}：{{ revisedPrompts.join(' / ') }}
          </p>
        </div>
      </div>
    </div>

    <!-- 大图查看 -->
    <BaseDialog
      :show="lightboxIndex !== null"
      :title="lightboxTitle"
      width="extra-wide"
      @close="closeLightbox"
    >
      <div v-if="lightboxItem" class="flex flex-col items-center gap-4">
        <img
          :src="lightboxItem.previewUrl"
          :alt="lightboxItem.prompt"
          class="max-h-[70vh] w-auto max-w-full rounded-lg object-contain"
        />
        <div class="flex w-full flex-wrap items-center justify-between gap-3">
          <p class="max-w-xl text-xs text-muted-foreground">{{ lightboxItem.prompt }}</p>
          <button type="button" class="btn btn-primary" :disabled="lightboxItem.downloading" @click="downloadImage(lightboxIndex)">
            {{ lightboxItem.downloading ? t('imageStudio.result.downloading') : t('imageStudio.result.downloadOriginal') }}
          </button>
        </div>
      </div>
    </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import TextArea from '@/components/common/TextArea.vue'
import {
  extensionForBlob,
  generateImages,
  imageToBlob,
  imageToDataUrl,
  listGatewayModels,
  type GatewayModel,
  type GeneratedImage,
} from '@/api/imageStudio'
import { saveBlob } from '@/api/batchImage'
import { useImageStudioAccess } from '@/composables/useImageStudioAccess'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'

interface StudioResult {
  id: string
  image: GeneratedImage
  previewUrl: string
  prompt: string
  model: string
  downloading: boolean
}

const { t } = useI18n()
const appStore = useAppStore()
const { imageStudioKeys, imageStudioAccessLoading, loadImageStudioAccess } = useImageStudioAccess()

const STORAGE_KEY_ID = 'image_studio_key_id'
const STORAGE_MODEL = 'image_studio_model'
const STORAGE_SIZE = 'image_studio_size'
const STORAGE_COUNT = 'image_studio_count'

const selectedKeyId = ref<number | null>(null)
const selectedModel = ref<string>('')
const prompt = ref('')
const size = ref<string>('')
const count = ref<number>(1)

const models = ref<GatewayModel[]>([])
const modelsLoading = ref(false)
const generating = ref(false)
const lastError = ref('')
const results = ref<StudioResult[]>([])
const lightboxIndex = ref<number | null>(null)

let modelsAbort: AbortController | null = null
let generateAbort: AbortController | null = null
let resultSeq = 0

const accessLoading = computed(() => imageStudioAccessLoading.value)

const keyOptions = computed<SelectOption[]>(() =>
  imageStudioKeys.value.map((key: ApiKey) => ({
    value: key.id,
    label: `${key.name} · ${key.group?.name || ''}`.trim(),
  }))
)

const selectedKey = computed<ApiKey | null>(
  () => imageStudioKeys.value.find((key: ApiKey) => key.id === selectedKeyId.value) || null
)

const modelOptions = computed<SelectOption[]>(() =>
  models.value.map((model) => ({
    value: model.id,
    label: model.display_name || model.id,
  }))
)

const sizeOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('imageStudio.form.sizeAuto') },
  { value: '1024x1024', label: '1024 × 1024' },
  { value: '1536x1024', label: '1536 × 1024' },
  { value: '1024x1536', label: '1024 × 1536' },
])

const countOptions = computed<SelectOption[]>(() =>
  [1, 2, 3, 4].map((value) => ({ value, label: String(value) }))
)

const canGenerate = computed(
  () => Boolean(selectedKey.value && selectedModel.value && prompt.value.trim()) && !generating.value
)

const lightboxItem = computed<StudioResult | null>(() =>
  lightboxIndex.value === null ? null : results.value[lightboxIndex.value] || null
)

const lightboxTitle = computed(() => lightboxItem.value?.model || t('imageStudio.lightbox.title'))

const revisedPrompts = computed(() =>
  results.value.map((item) => item.image.revised_prompt).filter((value): value is string => Boolean(value))
)

function persistSelection() {
  try {
    if (selectedKeyId.value !== null) localStorage.setItem(STORAGE_KEY_ID, String(selectedKeyId.value))
    localStorage.setItem(STORAGE_MODEL, selectedModel.value)
    localStorage.setItem(STORAGE_SIZE, size.value)
    localStorage.setItem(STORAGE_COUNT, String(count.value))
  } catch {
    // localStorage 不可用时忽略
  }
}

function restoreSelection() {
  try {
    const keyId = Number(localStorage.getItem(STORAGE_KEY_ID) || '')
    if (Number.isFinite(keyId) && keyId > 0) selectedKeyId.value = keyId
    selectedModel.value = localStorage.getItem(STORAGE_MODEL) || ''
    size.value = localStorage.getItem(STORAGE_SIZE) || ''
    const storedCount = Number(localStorage.getItem(STORAGE_COUNT) || '')
    if (Number.isFinite(storedCount) && storedCount >= 1 && storedCount <= 4) count.value = storedCount
  } catch {
    // 忽略
  }
}

/** 分组可用模型里优先挑图片模型作为默认值。 */
function pickDefaultModel(list: GatewayModel[]): string {
  const imageModel = list.find((model) => /image/i.test(model.id))
  return (imageModel || list[0])?.id || ''
}

async function loadModels() {
  modelsAbort?.abort()
  models.value = []
  if (!selectedKey.value) {
    selectedModel.value = ''
    return
  }

  const controller = new AbortController()
  modelsAbort = controller
  modelsLoading.value = true
  try {
    const list = await listGatewayModels(selectedKey.value.key, controller.signal)
    if (controller.signal.aborted) return
    models.value = list
    const stillValid = list.some((model) => model.id === selectedModel.value)
    if (!stillValid) selectedModel.value = pickDefaultModel(list)
  } catch (error) {
    if (controller.signal.aborted) return
    models.value = []
    selectedModel.value = ''
    appStore.showError(t('imageStudio.errors.modelsFailed', { message: (error as Error).message }))
  } finally {
    if (!controller.signal.aborted) modelsLoading.value = false
  }
}

async function reloadKeys() {
  const previous = selectedKeyId.value
  await loadImageStudioAccess(true)
  const stillValid = previous !== null && imageStudioKeys.value.some((key: ApiKey) => key.id === previous)
  if (stillValid) {
    selectedKeyId.value = previous
  } else if (imageStudioKeys.value.length > 0) {
    selectedKeyId.value = imageStudioKeys.value[0].id
  } else {
    selectedKeyId.value = null
  }
  if (selectedKeyId.value && selectedKeyId.value !== previous) {
    await loadModels()
  }
}

/** Select 的 change 只在用户手动切换时触发，避免与 onMounted 的初始化抢跑。 */
async function handleKeyChange() {
  persistSelection()
  await loadModels()
}

async function generate() {
  const key = selectedKey.value
  if (!key) {
    appStore.showError(t('imageStudio.errors.keyRequired'))
    return
  }
  if (!selectedModel.value) {
    appStore.showError(t('imageStudio.errors.modelRequired'))
    return
  }
  const text = prompt.value.trim()
  if (!text) {
    appStore.showError(t('imageStudio.errors.promptRequired'))
    return
  }

  generating.value = true
  lastError.value = ''
  generateAbort?.abort()
  const controller = new AbortController()
  generateAbort = controller

  try {
    const response = await generateImages(
      key.key,
      {
        model: selectedModel.value,
        prompt: text,
        n: count.value,
        ...(size.value ? { size: size.value } : {}),
      },
      controller.signal
    )
    if (controller.signal.aborted) return
    if (response.data.length === 0) {
      lastError.value = t('imageStudio.errors.emptyResult')
      return
    }
    const items: StudioResult[] = response.data.map((image) => ({
      id: `img-${Date.now()}-${(resultSeq += 1)}`,
      image,
      previewUrl: imageToDataUrl(image),
      prompt: image.revised_prompt || text,
      model: selectedModel.value,
      downloading: false,
    }))
    results.value = [...items, ...results.value]
    appStore.showSuccess(t('imageStudio.result.generated', { count: items.length }))
  } catch (error) {
    if (controller.signal.aborted) return
    lastError.value = (error as Error).message
    appStore.showError(t('imageStudio.errors.generateFailed', { message: (error as Error).message }))
  } finally {
    if (!controller.signal.aborted) generating.value = false
  }
}

function openLightbox(index: number) {
  lightboxIndex.value = index
}

function closeLightbox() {
  lightboxIndex.value = null
}

function clearResults() {
  results.value = []
  lastError.value = ''
}

async function downloadImage(index: number | null) {
  if (index === null) return
  const item = results.value[index]
  if (!item || item.downloading) return
  item.downloading = true
  try {
    const blob = await imageToBlob(item.image)
    const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    saveBlob(blob, `${item.model || 'image'}-${stamp}.${extensionForBlob(blob)}`)
  } catch (error) {
    appStore.showError(t('imageStudio.errors.downloadFailed', { message: (error as Error).message }))
  } finally {
    item.downloading = false
  }
}

watch([selectedModel, size, count], persistSelection)

onMounted(async () => {
  restoreSelection()
  await loadImageStudioAccess()
  if (selectedKeyId.value && !imageStudioKeys.value.some((key: ApiKey) => key.id === selectedKeyId.value)) {
    selectedKeyId.value = null
  }
  if (!selectedKeyId.value && imageStudioKeys.value.length > 0) {
    selectedKeyId.value = imageStudioKeys.value[0].id
  }
  if (selectedKeyId.value) {
    await loadModels()
  }
})

onBeforeUnmount(() => {
  modelsAbort?.abort()
  generateAbort?.abort()
})
</script>
