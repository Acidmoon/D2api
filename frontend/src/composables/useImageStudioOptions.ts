/**
 * 生图创作中心的参数逻辑。
 *
 * 这里的函数都是纯函数（不依赖响应式状态），便于单测；页面只负责把它们
 * 接到控件上。取值依据来自生产实测（创作分组 group 30，上游 wegoapi）：
 * - `/v1/images/generations` 只接受图像模型（`gpt-image-*` / `grok-imagine*`），
 *   选到文本模型必然 404 `model_not_found`；
 * - 上游按字面量接受 `size` 为 `1K/2K/4K`（也接受 `WxH`），并且会原样回显；
 * - 计费档位与 `size` 的关系由网关决定，不是「越大越清晰」的承诺，
 *   所以界面上把每档的**分组单价**直接标出来，让用户按价选档。
 */
import type { Group } from '@/types'
import type { GatewayModel } from '@/api/imageStudio'

/** 尺寸档位选项；value 即传给上游的 size 字段（空串表示不传，交给上游默认）。 */
export interface ImageSizeTier {
  value: string
  /** 分组图片价字段后缀；null 表示该档没有对应单价 */
  priceKey: 'image_price_1k' | 'image_price_2k' | 'image_price_4k' | null
  labelKey: string
}

export const IMAGE_SIZE_TIERS: ImageSizeTier[] = [
  // 不传 size 时网关按默认档计费（见 backend image_billing_size.go 的 2K 兜底），
  // 所以「自动」对齐 2K 单价展示，避免显示 $0 误导。
  { value: '', priceKey: 'image_price_2k', labelKey: 'imageStudio.form.sizeAuto' },
  { value: '1K', priceKey: 'image_price_1k', labelKey: 'imageStudio.form.size1k' },
  { value: '2K', priceKey: 'image_price_2k', labelKey: 'imageStudio.form.size2k' },
  { value: '4K', priceKey: 'image_price_4k', labelKey: 'imageStudio.form.size4k' },
]

/** 一次生成允许的张数（上游实测支持 n>1；再多收益递减且容易超时）。 */
export const IMAGE_COUNT_OPTIONS = [1, 2, 3, 4] as const

/** 兜底单价：与 backend NormalizeImageBillingTierOrDefault 的 2K 默认档一致。 */
const FALLBACK_PRICE_KEY: ImageSizeTier['priceKey'] = 'image_price_2k'

/** 是否图像生成模型；与后端 isOpenAIImageGenerationModel 的判据保持一致。 */
export function isImageGenerationModel(modelId: string): boolean {
  const id = (modelId || '').trim().toLowerCase()
  if (!id) return false
  if (id.startsWith('gpt-image-')) return true
  return id === 'grok-imagine' || id === 'grok-imagine-edit' || id.startsWith('grok-imagine-image')
}

/**
 * 从分组模型列表里筛出真正能在本页生图的模型。
 * 上游 /v1/models 可能混入文本模型（分组白名单与上游实际支持集合并不一致），
 * 选中文本模型只会得到 404 model_not_found。
 */
export function filterImageModels(models: GatewayModel[]): GatewayModel[] {
  return (models || []).filter((model) => isImageGenerationModel(model?.id || ''))
}

/** 在可用图像模型里挑默认值：优先较新的 2.5/2 代，其次任意图像模型。 */
export function pickDefaultImageModel(models: GatewayModel[]): string {
  const list = filterImageModels(models)
  if (list.length === 0) return ''
  const preferred = list.find((model) => /gpt-image-2\.5/.test(model.id))
    || list.find((model) => /gpt-image-2/.test(model.id))
  return (preferred || list[0]).id
}

/** 取某档位在当前分组下的单张价格；未配置返回 null（界面就不报价）。 */
export function priceForSizeTier(group: Group | null, tierValue: string): number | null {
  if (!group) return null
  const tier = IMAGE_SIZE_TIERS.find((item) => item.value === tierValue)
  const key = tier?.priceKey || FALLBACK_PRICE_KEY
  const raw = key ? group[key] : null
  if (raw === null || raw === undefined) return null
  const price = Number(raw)
  return Number.isFinite(price) && price >= 0 ? price : null
}

/** 本次选择的单张价格（空档位回落到分组兜底档/2K 档）。 */
export function unitPriceForSelection(group: Group | null, tierValue: string): number | null {
  return priceForSizeTier(group, tierValue)
}

/** 预估总价：任一环节缺价就返回 null，让界面显示「以实际账单为准」。 */
export function estimateImageCost(
  group: Group | null,
  tierValue: string,
  count: number
): number | null {
  const unit = unitPriceForSelection(group, tierValue)
  if (unit === null) return null
  const safeCount = IMAGE_COUNT_OPTIONS.includes(count as (typeof IMAGE_COUNT_OPTIONS)[number])
    ? count
    : 1
  return unit * safeCount
}

/** 校验持久化回来的档位（localStorage 可能被旧版本写过像素串）。 */
export function normalizeSizeTier(stored: string | null): string {
  if (!stored) return ''
  if (IMAGE_SIZE_TIERS.some((tier) => tier.value === stored)) return stored
  // 旧的像素写法（1024x1024 等）归一到最接近的档位，避免下拉显示空值。
  const match = /^(\d+)x(\d+)$/i.exec(stored.trim())
  if (match) {
    const maxEdge = Math.max(Number(match[1]) || 0, Number(match[2]) || 0)
    if (maxEdge <= 0) return ''
    if (maxEdge <= 1024) return '1K'
    if (maxEdge <= 2048) return '2K'
    return '4K'
  }
  return ''
}

/**
 * 金额展示：保留 4 位精度但去掉多余的 0，至少留 2 位小数，
 * 避免 $0.04000000 / $0.1 这种噪声。
 */
export function formatImagePrice(value: number): string {
  if (!Number.isFinite(value)) return '-'
  const rounded = Math.round(value * 10000) / 10000
  let text = rounded.toFixed(4)
  while (text.endsWith('0') && (text.split('.')[1] || '').length > 2) {
    text = text.slice(0, -1)
  }
  return `$${text}`
}
