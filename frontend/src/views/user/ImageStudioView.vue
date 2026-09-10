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
            {{ selectedKey ? t('imageStudio.key.hint', { group: selectedKeyGroup?.name || '-' }) : t('imageStudio.key.empty') }}
          </p>
        </div>
        <div class="flex items-center gap-2">
          <span v-if="selectedKey" class="badge badge-primary">{{ selectedKeyGroup?.name || '-' }}</span>
          <button type="button" class="btn btn-secondary btn-sm" :disabled="savingHistory" @click="openHistory">
            {{ t('imageStudio.history.open') }}
          </button>
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
      <!-- 左侧：参数区 -->
      <div class="card self-start space-y-5">
        <!-- 模型：只列真正能生图的模型 -->
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
          <p v-else-if="hiddenModelCount > 0" class="input-hint">
            {{ t('imageStudio.form.modelsHidden', { count: hiddenModelCount }) }}
          </p>
        </div>

        <!-- 提示词 -->
        <div>
          <div class="flex items-center justify-between">
            <label class="input-label mb-0" for="image-studio-prompt">{{ t('imageStudio.form.prompt') }}</label>
            <button
              v-if="prompt"
              type="button"
              class="text-xs text-muted-foreground transition-colors hover:text-foreground"
              @click="prompt = ''"
            >
              {{ t('imageStudio.form.clearPrompt') }}
            </button>
          </div>
          <TextArea
            id="image-studio-prompt"
            v-model="prompt"
            :rows="7"
            :placeholder="t('imageStudio.form.promptPlaceholder')"
          />
          <p class="input-hint text-right">{{ prompt.length }}/4000</p>
        </div>

        <!-- 尺寸档位：分段选择，每档标出该分组单价 -->
        <div>
          <label class="input-label">{{ t('imageStudio.form.size') }}</label>
          <div class="grid grid-cols-4 gap-2" role="radiogroup" :aria-label="t('imageStudio.form.size')">
            <button
              v-for="tier in sizeTierList"
              :key="tier.value || 'auto'"
              type="button"
              role="radio"
              :aria-checked="size === tier.value"
              class="rounded-lg border px-2 py-2 text-center transition-colors"
              :class="size === tier.value
                ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200'
                : 'border-border bg-transparent text-muted-foreground hover:border-primary-300 hover:text-foreground dark:hover:border-primary-700'"
              @click="size = tier.value"
            >
              <span class="block text-sm font-medium">{{ tier.label }}</span>
              <span class="mt-0.5 block text-[11px] leading-4 opacity-80">{{ tier.priceText }}</span>
            </button>
          </div>
          <p class="input-hint">{{ t('imageStudio.form.sizeHint') }}</p>
        </div>

        <!-- 张数 -->
        <div>
          <label class="input-label">{{ t('imageStudio.form.count') }}</label>
          <div class="grid grid-cols-4 gap-2" role="radiogroup" :aria-label="t('imageStudio.form.count')">
            <button
              v-for="option in countOptions"
              :key="option.value"
              type="button"
              role="radio"
              :aria-checked="count === option.value"
              class="rounded-lg border px-2 py-2 text-sm transition-colors"
              :class="count === option.value
                ? 'border-primary-500 bg-primary-50 font-medium text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200'
                : 'border-border text-muted-foreground hover:border-primary-300 hover:text-foreground dark:hover:border-primary-700'"
              @click="count = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <!-- 费用预估 -->
        <div class="rounded-lg border border-border bg-muted/40 px-3 py-2.5 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-muted-foreground">{{ t('imageStudio.form.unitPrice') }}</span>
            <span class="font-medium text-foreground">{{ unitPriceText }}</span>
          </div>
          <div class="mt-1 flex items-center justify-between">
            <span class="text-muted-foreground">{{ t('imageStudio.form.estimatedTotal') }}</span>
            <span class="font-semibold text-foreground">{{ estimatedTotalText }}</span>
          </div>
        </div>

        <!-- 生成 / 取消 -->
        <div class="space-y-2">
          <button
            type="button"
            class="btn btn-primary w-full"
            :disabled="!canGenerate"
            @click="generate"
          >
            <LoadingSpinner v-if="generating" size="sm" class="mr-2" />
            {{ generating ? t('imageStudio.form.generating') : t('imageStudio.form.generate') }}
          </button>
          <button
            v-if="generating"
            type="button"
            class="btn btn-secondary w-full"
            @click="cancelGeneration"
          >
            {{ t('imageStudio.form.cancel') }}
          </button>
          <p v-if="generating" class="text-center text-xs text-muted-foreground">
            {{ t('imageStudio.form.elapsed', { seconds: elapsedSeconds }) }}
          </p>
          <p v-else class="text-xs text-muted-foreground">{{ t('imageStudio.form.costHint') }}</p>
        </div>
      </div>

      <!-- 右侧：生成结果 -->
      <div class="card min-h-[480px]">
        <div class="mb-4 flex items-center justify-between">
          <h2 class="text-base font-semibold text-foreground">{{ t('imageStudio.result.title') }}</h2>
          <div v-if="results.length > 0" class="flex items-center gap-2">
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="generating || downloadingAll"
              @click="downloadAll"
            >
              {{ downloadingAll ? t('imageStudio.result.downloading') : t('imageStudio.result.downloadAll') }}
            </button>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="generating"
              @click="clearResults"
            >
              {{ t('imageStudio.result.clear') }}
            </button>
          </div>
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
            <span class="tabular-nums">{{ elapsedSeconds }}s</span>
          </div>

          <div v-if="lastError" class="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">
            {{ lastError }}
          </div>

          <!-- 生成中占位：按张数先占位，避免新图插入时布局跳动 -->
          <div v-if="generating" class="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
            <div
              v-for="placeholder in pendingPlaceholders"
              :key="placeholder"
              class="aspect-square w-full animate-pulse rounded-xl border border-border bg-muted"
            />
          </div>

          <div v-if="results.length > 0" class="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
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
                  @load="handleResultImageLoad(item, $event)"
                />
              </button>
              <!-- 左上：真实输出尺寸（以字节为准，不看上游回显） -->
              <div class="pointer-events-none absolute left-2 top-2 flex flex-wrap gap-1">
                <span class="rounded bg-black/60 px-1.5 py-0.5 text-[11px] font-medium tabular-nums text-white/90">
                  {{ item.width ? `${item.width}×${item.height}` : t('imageStudio.result.measuring') }}
                </span>
                <span class="rounded bg-black/60 px-1.5 py-0.5 text-[11px] text-white/90">{{ item.model }}</span>
              </div>
              <div
                class="pointer-events-none absolute inset-x-0 bottom-0 flex items-center justify-between gap-2 bg-gradient-to-t from-black/70 to-transparent px-3 py-2 opacity-0 transition-opacity group-hover:opacity-100"
              >
                <span class="pointer-events-auto flex items-center gap-2">
                  <button type="button" class="btn btn-secondary btn-sm" @click="openLightbox(index)">
                    {{ t('imageStudio.result.open') }}
                  </button>
                  <button type="button" class="btn btn-secondary btn-sm" @click="reusePrompt(item)">
                    {{ t('imageStudio.result.reuse') }}
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

    <!-- 历史记录：列表 / 详情 -->
    <BaseDialog
      :show="historyOpen"
      :title="historyDetail ? t('imageStudio.history.detailTitle') : t('imageStudio.history.title')"
      width="extra-wide"
      @close="closeHistory"
    >
      <!-- 详情 -->
      <div v-if="historyDetail" class="space-y-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <button type="button" class="btn btn-secondary btn-sm" @click="closeHistoryDetail">
            {{ t('imageStudio.history.back') }}
          </button>
          <div class="flex items-center gap-2">
            <span class="badge badge-gray">{{ historyDetail.model }}</span>
            <span v-if="historyDetail.size" class="badge badge-gray">{{ historyDetail.size }}</span>
            <button type="button" class="btn btn-secondary btn-sm" @click="removeHistoryItem(historyDetail)">
              {{ t('imageStudio.history.delete') }}
            </button>
          </div>
        </div>

        <div class="rounded-lg border border-border bg-muted/40 p-3">
          <p class="text-xs text-muted-foreground">{{ t('imageStudio.history.prompt') }}</p>
          <p class="mt-1 whitespace-pre-wrap text-sm text-foreground">{{ historyDetail.prompt }}</p>
          <template v-if="historyDetail.revised_prompt && historyDetail.revised_prompt !== historyDetail.prompt">
            <p class="mt-3 text-xs text-muted-foreground">{{ t('imageStudio.history.revisedPrompt') }}</p>
            <p class="mt-1 whitespace-pre-wrap text-sm text-foreground">{{ historyDetail.revised_prompt }}</p>
          </template>
          <p class="mt-3 text-xs text-muted-foreground">
            {{ formatHistoryTime(historyDetail.created_at) }} · {{ t('imageStudio.history.expiresAt') }}{{ formatHistoryTime(historyDetail.expires_at) }}
          </p>
        </div>

        <div v-if="historyDetailLoading" class="flex items-center justify-center py-10">
          <LoadingSpinner size="md" />
        </div>
        <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 2xl:grid-cols-3">
          <figure
            v-for="(url, index) in historyDetailUrls"
            :key="index"
            class="group relative overflow-hidden rounded-xl border border-border bg-muted"
          >
            <img :src="url" :alt="historyDetail.prompt" class="aspect-square w-full object-cover" />
            <div class="pointer-events-none absolute inset-x-0 bottom-0 flex justify-end bg-gradient-to-t from-black/70 to-transparent px-3 py-2 opacity-0 transition-opacity group-hover:opacity-100">
              <button type="button" class="btn btn-primary btn-sm pointer-events-auto" @click="downloadHistoryImage(historyDetail, index)">
                {{ t('imageStudio.result.download') }}
              </button>
            </div>
          </figure>
        </div>
      </div>

      <!-- 列表 -->
      <div v-else class="space-y-3">
        <div v-if="historyLoading" class="flex items-center justify-center py-12">
          <LoadingSpinner size="md" />
        </div>
        <p v-else-if="historyError" class="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger">
          {{ historyError }}
        </p>
        <EmptyState
          v-else-if="historyItems.length === 0"
          :title="t('imageStudio.history.emptyTitle')"
          :description="t('imageStudio.history.emptyDescription')"
        />
        <template v-else>
          <ul class="divide-y divide-border">
            <li
              v-for="item in historyItems"
              :key="item.id"
              class="flex cursor-pointer items-start gap-3 py-3 transition-colors hover:bg-muted/40"
              @click="openHistoryDetail(item)"
            >
              <div class="min-w-0 flex-1">
                <p class="line-clamp-2 text-sm text-foreground">{{ item.prompt }}</p>
                <p class="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span>{{ formatHistoryTime(item.created_at) }}</span>
                  <span class="badge badge-gray">{{ item.model }}</span>
                  <span v-if="item.size" class="badge badge-gray">{{ item.size }}</span>
                  <span>{{ t('imageStudio.history.imageCount', { count: item.image_count }) }}</span>
                </p>
              </div>
              <span class="badge badge-primary shrink-0">{{ t('imageStudio.history.view') }}</span>
            </li>
          </ul>
          <div v-if="historyTotal > historyPageSize" class="flex items-center justify-between pt-2">
            <span class="text-xs text-muted-foreground">
              {{ t('imageStudio.history.pageInfo', { page: historyPage, total: Math.max(1, Math.ceil(historyTotal / historyPageSize)) }) }}
            </span>
            <div class="flex items-center gap-2">
              <button type="button" class="btn btn-secondary btn-sm" :disabled="historyPage <= 1" @click="loadHistory(historyPage - 1)">
                {{ t('imageStudio.history.prev') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="historyPage * historyPageSize >= historyTotal"
                @click="loadHistory(historyPage + 1)"
              >
                {{ t('imageStudio.history.next') }}
              </button>
            </div>
          </div>
        </template>
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
  imageToBase64Payload,
  imageToBlob,
  imageToDataUrl,
  listGatewayModels,
  type GatewayModel,
  type GeneratedImage,
} from '@/api/imageStudio'
import {
  deleteImageStudioHistory,
  getImageStudioHistoryImage,
  listImageStudioHistory,
  saveImageStudioHistory,
  type ImageStudioHistoryItem,
} from '@/api/imageStudioHistory'
import { saveBlob } from '@/api/batchImage'
import {
  IMAGE_COUNT_OPTIONS,
  IMAGE_SIZE_TIERS,
  estimateImageCost,
  filterImageModels,
  formatImagePrice,
  normalizeSizeTier,
  pickDefaultImageModel,
  unitPriceForSelection,
} from '@/composables/useImageStudioOptions'
import { creationGroupOf, useImageStudioAccess } from '@/composables/useImageStudioAccess'
import { useAppStore } from '@/stores/app'
import type { ApiKey } from '@/types'

interface StudioResult {
  id: string
  image: GeneratedImage
  previewUrl: string
  prompt: string
  model: string
  downloading: boolean
  /** 从图片字节实际解析出的尺寸（不信任上游回显的 size） */
  width: number
  height: number
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
const elapsedSeconds = ref(0)
const downloadingAll = ref(false)
const lastError = ref('')
const results = ref<StudioResult[]>([])
const lightboxIndex = ref<number | null>(null)

let elapsedTimer: ReturnType<typeof setInterval> | null = null

// ── 历史记录 ──
const savingHistory = ref(false)
const historyOpen = ref(false)
const historyLoading = ref(false)
const historyError = ref('')
const historyItems = ref<ImageStudioHistoryItem[]>([])
const historyTotal = ref(0)
const historyPage = ref(1)
const historyPageSize = 10
const historyDetail = ref<ImageStudioHistoryItem | null>(null)
const historyDetailUrls = ref<string[]>([])
const historyDetailLoading = ref(false)

let modelsAbort: AbortController | null = null
let generateAbort: AbortController | null = null
let resultSeq = 0

const accessLoading = computed(() => imageStudioAccessLoading.value)

const keyOptions = computed<SelectOption[]>(() =>
  imageStudioKeys.value.map((key: ApiKey) => ({
    value: key.id,
    label: `${key.name} · ${creationGroupOf(key)?.name || ''}`.trim(),
  }))
)

const selectedKey = computed<ApiKey | null>(
  () => imageStudioKeys.value.find((key: ApiKey) => key.id === selectedKeyId.value) || null
)

const selectedKeyGroup = computed(() =>
  selectedKey.value ? creationGroupOf(selectedKey.value) : null
)

/** /v1/models 可能混入不能走 images 端点的文本模型，这里过滤后单独计数。 */
const imageModelList = computed<GatewayModel[]>(() => filterImageModels(models.value))
const hiddenModelCount = computed(() => Math.max(0, models.value.length - imageModelList.value.length))

const modelOptions = computed<SelectOption[]>(() =>
  imageModelList.value.map((model) => ({
    value: model.id,
    label: model.display_name || model.id,
  }))
)

/** 尺寸档位（带当前分组的单张价，未配价时退到不显示）。 */
const sizeTierList = computed(() =>
  IMAGE_SIZE_TIERS.map((tier) => {
    const price = unitPriceForSelection(selectedKeyGroup.value, tier.value)
    return {
      value: tier.value,
      label: t(tier.labelKey),
      priceText: price === null ? '' : formatImagePrice(price),
    }
  })
)

const countOptions = computed<Array<{ value: number; label: string }>>(() =>
  IMAGE_COUNT_OPTIONS.map((value) => ({ value, label: String(value) }))
)

const pendingPlaceholders = computed(() => Array.from({ length: count.value }, (_, i) => i))

const unitPriceText = computed(() => {
  const price = unitPriceForSelection(selectedKeyGroup.value, size.value)
  return price === null ? t('imageStudio.form.priceUnknown') : `${formatImagePrice(price)} / ${t('imageStudio.form.perImage')}`
})

const estimatedTotalText = computed(() => {
  const total = estimateImageCost(selectedKeyGroup.value, size.value, count.value)
  return total === null ? t('imageStudio.form.priceUnknown') : formatImagePrice(total)
})

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
    size.value = normalizeSizeTier(localStorage.getItem(STORAGE_SIZE))
    const storedCount = Number(localStorage.getItem(STORAGE_COUNT) || '')
    if (Number.isFinite(storedCount) && storedCount >= 1 && storedCount <= 4) count.value = storedCount
  } catch {
    // 忽略
  }
}

/** 只有仍然是图像模型的记忆值才保留，否则重新选默认。 */
function imageModelListSome(list: GatewayModel[], modelId: string): boolean {
  if (!modelId) return false
  return filterImageModels(list).some((model) => model.id === modelId)
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
    const stillValid = imageModelListSome(list, selectedModel.value)
    if (!stillValid) selectedModel.value = pickDefaultImageModel(list)
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

/** 生成成功后异步保存到历史（失败只提示，不影响本次生成体验）。 */
async function persistGeneration(items: StudioResult[], promptText: string) {
  const key = selectedKey.value
  if (!key || items.length === 0) return
  savingHistory.value = true
  try {
    const images: Array<{ mime_type?: string; data: string }> = []
    for (const item of items) {
      const encoded = await imageToBase64Payload(item.image)
      if (encoded) images.push({ mime_type: encoded.mime_type, data: encoded.data })
    }
    if (images.length === 0) return
    await saveImageStudioHistory({
      model: items[0]?.model || selectedModel.value,
      prompt: promptText,
      revised_prompt: items[0]?.image.revised_prompt || '',
      size: size.value,
      api_key_id: key.id,
      group_id: creationGroupOf(key)?.id ?? null,
      images,
    })
  } catch (error) {
    appStore.showError(t('imageStudio.history.saveFailed', { message: (error as Error).message }))
  } finally {
    savingHistory.value = false
  }
}

function formatHistoryTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function releaseHistoryDetailUrls() {
  for (const url of historyDetailUrls.value) URL.revokeObjectURL(url)
  historyDetailUrls.value = []
}

async function loadHistory(page: number) {
  historyLoading.value = true
  historyError.value = ''
  try {
    const data = await listImageStudioHistory(page, historyPageSize)
    historyItems.value = data.items || []
    historyTotal.value = data.total || 0
    historyPage.value = data.page || page
  } catch (error) {
    historyItems.value = []
    historyTotal.value = 0
    historyError.value = (error as Error).message
  } finally {
    historyLoading.value = false
  }
}

async function openHistory() {
  historyOpen.value = true
  historyDetail.value = null
  releaseHistoryDetailUrls()
  await loadHistory(1)
}

function closeHistory() {
  historyOpen.value = false
  historyDetail.value = null
  releaseHistoryDetailUrls()
}

function closeHistoryDetail() {
  historyDetail.value = null
  releaseHistoryDetailUrls()
}

async function openHistoryDetail(item: ImageStudioHistoryItem) {
  historyDetail.value = item
  releaseHistoryDetailUrls()
  historyDetailLoading.value = true
  try {
    const urls: string[] = []
    for (const image of item.images) {
      try {
        const blob = await getImageStudioHistoryImage(item.id, image.index)
        urls.push(URL.createObjectURL(blob))
      } catch {
        // 单张图片丢失不影响其余图片展示
      }
    }
    historyDetailUrls.value = urls
  } finally {
    historyDetailLoading.value = false
  }
}

async function downloadHistoryImage(item: ImageStudioHistoryItem, index: number) {
  try {
    const blob = await getImageStudioHistoryImage(item.id, index)
    const stamp = new Date(item.created_at).toISOString().replace(/[:.]/g, '-').slice(0, 19)
    saveBlob(blob, `${item.model || 'image'}-${stamp}.${extensionForBlob(blob)}`)
  } catch (error) {
    appStore.showError(t('imageStudio.errors.downloadFailed', { message: (error as Error).message }))
  }
}

async function removeHistoryItem(item: ImageStudioHistoryItem | null) {
  if (!item) return
  try {
    await deleteImageStudioHistory(item.id)
    appStore.showSuccess(t('imageStudio.history.deleted'))
    closeHistoryDetail()
    await loadHistory(1)
  } catch (error) {
    appStore.showError(t('imageStudio.history.deleteFailed', { message: (error as Error).message }))
  }
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
  startElapsedTimer()
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
      width: 0,
      height: 0,
    }))
    results.value = [...items, ...results.value]
    appStore.showSuccess(t('imageStudio.result.generated', { count: items.length }))
    void persistGeneration(items, text)
  } catch (error) {
    if (controller.signal.aborted) return
    lastError.value = describeGenerationError(error as Error)
    appStore.showError(t('imageStudio.errors.generateFailed', { message: lastError.value }))
  } finally {
    if (!controller.signal.aborted) {
      stopElapsedTimer()
      generating.value = false
    }
  }
}

/**
 * 把上游/网关报错归成用户看得懂的一句话。
 * 实测最常见的是 model_not_found（分组模型表与上游实际支持不一致）。
 */
function describeGenerationError(error: Error): string {
  const message = error?.message || ''
  if (/model_not_found|not supported by any configured account/i.test(message)) {
    return t('imageStudio.errors.modelNotSupported')
  }
  if (/insufficient|balance|quota|余额|额度/i.test(message)) {
    return t('imageStudio.errors.insufficientBalance')
  }
  if (/permission|forbidden|not allowed|权限/i.test(message)) {
    return t('imageStudio.errors.permissionDenied')
  }
  if (/timeout|timed out|超时/i.test(message)) {
    return t('imageStudio.errors.timeout')
  }
  return message || t('imageStudio.errors.unknown')
}

function startElapsedTimer() {
  stopElapsedTimer()
  elapsedSeconds.value = 0
  elapsedTimer = setInterval(() => {
    elapsedSeconds.value += 1
  }, 1000)
}

function stopElapsedTimer() {
  if (elapsedTimer !== null) {
    clearInterval(elapsedTimer)
    elapsedTimer = null
  }
}

/** 用户主动取消：上游可能已经出图并计费，提示里说清楚。 */
function cancelGeneration() {
  generateAbort?.abort()
  generateAbort = null
  stopElapsedTimer()
  generating.value = false
  lastError.value = t('imageStudio.errors.cancelled')
  appStore.showWarning(t('imageStudio.errors.cancelledHint'))
}

/** 结果图加载完成后读真实像素；尺寸看字节，不信上游回显。 */
function handleResultImageLoad(item: StudioResult, event: Event) {
  const img = event.target as HTMLImageElement | null
  if (!img) return
  item.width = img.naturalWidth || 0
  item.height = img.naturalHeight || 0
}

/** 用该图的提示词回填输入区，便于微调重生。 */
function reusePrompt(item: StudioResult) {
  prompt.value = item.prompt
  selectedModel.value = item.model || selectedModel.value
  appStore.showSuccess(t('imageStudio.result.reused'))
}

/** 逐张下载（浏览器会要求允许多文件）；间隔开避免被截断。 */
async function downloadAll() {
  if (results.value.length === 0 || downloadingAll.value) return
  downloadingAll.value = true
  try {
    for (let index = 0; index < results.value.length; index += 1) {
      await downloadImage(index, true)
      await new Promise((resolve) => setTimeout(resolve, 300))
    }
  } finally {
    downloadingAll.value = false
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

async function downloadImage(index: number | null, silent = false) {
  if (index === null) return
  const item = results.value[index]
  if (!item || item.downloading) return
  item.downloading = true
  try {
    const blob = await imageToBlob(item.image)
    const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19)
    saveBlob(blob, `${item.model || 'image'}-${stamp}-${index + 1}.${extensionForBlob(blob)}`)
  } catch (error) {
    // 批量下载时只报一次，不刷一屏 toast
    if (!silent) {
      appStore.showError(t('imageStudio.errors.downloadFailed', { message: (error as Error).message }))
    }
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
  stopElapsedTimer()
  releaseHistoryDetailUrls()
})
</script>
