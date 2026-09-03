<template>
  <!-- QW 最新模型卡：全宽白卡，标题 + 描边药丸，3 列模型卡；失败/空数据整卡隐藏 -->
  <section
    v-if="modelCards.length > 0"
    class="rounded-[24px] bg-card p-7 shadow-card"
    data-testid="dashboard-models"
  >
    <div class="flex items-center justify-between gap-3">
      <h2 class="text-xl font-semibold text-foreground">{{ t('dashboard.models.section') }}</h2>
      <RouterLink to="/available-channels" class="qw-pill">
        {{ t('dashboard.models.viewAll') }}
      </RouterLink>
    </div>
    <div class="mt-6 grid grid-cols-1 gap-6 md:grid-cols-3">
      <div
        v-for="model in modelCards"
        :key="model.key"
        class="flex min-h-[380px] flex-col gap-6 rounded-[18px] border border-[color:var(--nm-border)] p-6"
      >
        <div class="flex min-w-0 flex-col gap-2">
          <h3 class="truncate text-lg font-semibold text-foreground">{{ model.name }}</h3>
          <div v-if="model.tags.length > 0" class="flex flex-wrap gap-2">
            <span
              v-for="tag in model.tags"
              :key="tag"
              class="inline-flex h-6 items-center rounded-lg bg-[color:var(--nm-surface-soft)] px-2 text-xs text-foreground"
            >
              {{ tag }}
            </span>
          </div>
          <p
            v-if="model.desc"
            class="line-clamp-3 text-sm leading-5 text-[#7F8798] dark:text-[#B4BCC6]"
          >
            {{ model.desc }}
          </p>
        </div>

        <dl v-if="model.specs.length > 0" class="space-y-2">
          <div v-for="spec in model.specs" :key="spec.label" class="flex items-baseline justify-between gap-3">
            <dt class="shrink-0 text-xs text-[#8E96A7] dark:text-[#7A8290]">{{ spec.label }}</dt>
            <dd class="truncate text-sm font-medium text-foreground">{{ spec.value }}</dd>
          </div>
        </dl>

        <div class="mt-auto flex flex-wrap items-center gap-3">
          <RouterLink
            to="/keys"
            class="inline-flex h-10 items-center justify-center rounded-full bg-[#0B0C0F] px-5 text-sm font-medium text-[#F0F3FF] transition-opacity hover:opacity-90 dark:bg-[#F2F4F8] dark:text-[#0B0C0F]"
          >
            {{ t('dashboard.models.tryNow') }}
          </RouterLink>
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="qw-pill qw-pill-lg"
          >
            {{ t('dashboard.models.callApi') }}
          </a>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeUrl } from '@/utils/url'
import { formatScaled } from '@/utils/pricing'
import userChannelsAPI, { type UserSupportedModelPricing } from '@/api/channels'
import {
  BILLING_MODE_TOKEN,
  BILLING_MODE_PER_REQUEST,
  BILLING_MODE_IMAGE,
  BILLING_MODE_VIDEO,
  type BillingMode
} from '@/constants/channel'

interface ModelSpec {
  label: string
  value: string
}

interface ModelCard {
  key: string
  name: string
  tags: string[]
  desc: string
  specs: ModelSpec[]
}

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const docUrl = computed(() => sanitizeUrl(appStore.docUrl))

// 与侧栏/快捷入口一致：功能开关关闭或简单模式时不拉取、不渲染。
const enabled = computed(
  () => !authStore.isSimpleMode && isFeatureFlagEnabled(FeatureFlags.availableChannels)
)

const modelCards = ref<ModelCard[]>([])

const PLATFORM_LABELS: Record<string, string> = {
  anthropic: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  antigravity: 'Antigravity',
  grok: 'Grok'
}

const platformLabel = (p: string) => PLATFORM_LABELS[p] ?? p

function billingModeLabel(mode: BillingMode): string {
  switch (mode) {
    case BILLING_MODE_TOKEN:
      return t('availableChannels.pricing.billingModeToken')
    case BILLING_MODE_PER_REQUEST:
      return t('availableChannels.pricing.billingModePerRequest')
    case BILLING_MODE_IMAGE:
      return t('availableChannels.pricing.billingModeImage')
    case BILLING_MODE_VIDEO:
      return t('availableChannels.pricing.billingModeVideo')
    default:
      return ''
  }
}

/** 按计费模式产出规格行；无定价信息时返回空数组（规格区整体隐藏）。 */
function buildSpecs(pricing: UserSupportedModelPricing | null): ModelSpec[] {
  if (!pricing) return []
  const specs: ModelSpec[] = []
  const modeLabel = billingModeLabel(pricing.billing_mode)
  if (modeLabel) {
    specs.push({ label: t('availableChannels.pricing.billingMode'), value: modeLabel })
  }
  const firstTier = pricing.intervals[0]
  if (pricing.billing_mode === BILLING_MODE_PER_REQUEST) {
    const price = firstTier ? firstTier.per_request_price : pricing.per_request_price
    if (price != null) {
      specs.push({
        label: t('availableChannels.pricing.perRequestPrice'),
        value: `${formatScaled(price, 1)} ${t('availableChannels.pricing.unitPerRequest')}`
      })
    }
  } else if (pricing.billing_mode === BILLING_MODE_TOKEN) {
    const input = firstTier ? firstTier.input_price : pricing.input_price
    const output = firstTier ? firstTier.output_price : pricing.output_price
    const unit = t('availableChannels.pricing.unitPerMillion')
    if (input != null) {
      specs.push({
        label: t('availableChannels.pricing.inputPrice'),
        value: `${formatScaled(input, 1_000_000)} ${unit}`
      })
    }
    if (output != null) {
      specs.push({
        label: t('availableChannels.pricing.outputPrice'),
        value: `${formatScaled(output, 1_000_000)} ${unit}`
      })
    }
  } else if (pricing.billing_mode === BILLING_MODE_IMAGE) {
    if (pricing.image_input_price != null) {
      specs.push({
        label: t('availableChannels.pricing.imageInputPrice'),
        value: formatScaled(pricing.image_input_price, 1)
      })
    }
    if (pricing.image_output_price != null) {
      specs.push({
        label: t('availableChannels.pricing.imageOutputPrice'),
        value: formatScaled(pricing.image_output_price, 1)
      })
    }
  }
  return specs
}

function buildModelCards(channels: Awaited<ReturnType<typeof userChannelsAPI.getAvailable>>): ModelCard[] {
  const cards: ModelCard[] = []
  for (const channel of channels) {
    for (const section of channel.platforms) {
      for (const model of section.supported_models) {
        const tags = [platformLabel(section.platform)]
        const modeLabel = model.pricing ? billingModeLabel(model.pricing.billing_mode) : ''
        if (modeLabel) tags.push(modeLabel)
        cards.push({
          // 渠道/平台/模型三级拼 key；同渠道同平台理论上可重复出现同名模型，
          // 追加自增序号兜底避免 Vue 重复 key 告警。
          key: `${cards.length}::${channel.name}::${section.platform}::${model.name}`,
          name: model.name,
          tags,
          desc: (channel.description || '').trim(),
          specs: buildSpecs(model.pricing)
        })
      }
    }
  }
  return cards
}

// 请求序号守卫：组件卸载（或 enabled 翻转）后到达的响应不得写入 ref。
let loadSeq = 0

async function loadModels() {
  if (!enabled.value) return
  const seq = ++loadSeq
  try {
    const channels = await userChannelsAPI.getAvailable()
    if (seq !== loadSeq) return
    // 纯展示：仅取列表前 3 个模型；失败或为空时整卡隐藏（v-if），绝不报错溢出。
    modelCards.value = buildModelCards(channels).slice(0, 3)
  } catch (error) {
    if (seq !== loadSeq) return
    console.warn('Failed to load latest models:', error)
    modelCards.value = []
  }
}

onMounted(loadModels)

onBeforeUnmount(() => {
  loadSeq++
})
</script>

<style scoped>
/* QW 描边药丸按钮：白底 1px #D1D7E2 描边，dark 下换用边框 token。 */
.qw-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  flex-shrink: 0;
  padding: 0 18px;
  border-radius: 999px;
  border: 1px solid #d1d7e2;
  font-size: 13px;
  font-weight: 500;
  line-height: 1;
  color: var(--nm-ink);
  white-space: nowrap;
  transition: background-color 150ms ease;
}

.qw-pill:hover {
  background-color: var(--nm-surface-soft);
}

:global(.dark) .qw-pill {
  border-color: var(--nm-border);
}

/* 模型卡底部主行动同款高度：h-10 / px-5 / 14px。 */
.qw-pill-lg {
  height: 40px;
  padding: 0 20px;
  font-size: 14px;
}
</style>
