import { describe, expect, it } from 'vitest'

import {
  IMAGE_SIZE_TIERS,
  estimateImageCost,
  filterImageModels,
  formatImagePrice,
  isImageGenerationModel,
  normalizeSizeTier,
  pickDefaultImageModel,
  unitPriceForSelection,
} from '../useImageStudioOptions'
import type { GatewayModel } from '@/api/imageStudio'
import type { Group } from '@/types'

function makeGroup(overrides: Partial<Group> = {}): Group {
  return {
    image_price_1k: 0.04,
    image_price_2k: 0.08,
    image_price_4k: 0.1,
    ...overrides,
  } as Group
}

function makeModels(...ids: string[]): GatewayModel[] {
  return ids.map((id) => ({ id }))
}

describe('isImageGenerationModel / filterImageModels', () => {
  it('accepts the model families the images endpoint allows', () => {
    for (const id of ['gpt-image-1', 'gpt-image-2', 'gpt-image-2.5-flare', 'grok-imagine', 'grok-imagine-image', 'grok-imagine-edit']) {
      expect(isImageGenerationModel(id)).toBe(true)
    }
  })

  it('rejects text models that would 404 on /v1/images/generations', () => {
    for (const id of ['gpt-5.5', 'gpt-6', 'grok-4.5', 'claude-3', '', '   ', 'image-proxy']) {
      expect(isImageGenerationModel(id)).toBe(false)
    }
  })

  it('drops text models from the group model list', () => {
    const list = makeModels('gpt-5.5', 'gpt-image-2', 'grok-4.5', 'gpt-image-2.5-sunburst')
    expect(filterImageModels(list).map((model) => model.id)).toEqual([
      'gpt-image-2',
      'gpt-image-2.5-sunburst',
    ])
  })

  it('survives a malformed list', () => {
    expect(filterImageModels([] as unknown as GatewayModel[])).toEqual([])
    expect(filterImageModels([{} as GatewayModel, { id: 'gpt-image-2' }])).toHaveLength(1)
  })
})

describe('pickDefaultImageModel', () => {
  it('prefers the newest 2.5 family, then gpt-image-2', () => {
    expect(pickDefaultImageModel(makeModels('gpt-image-2', 'gpt-image-2.5-flare'))).toBe('gpt-image-2.5-flare')
    expect(pickDefaultImageModel(makeModels('gpt-image-1', 'gpt-image-2'))).toBe('gpt-image-2')
  })

  it('falls back to any image model and then to empty', () => {
    expect(pickDefaultImageModel(makeModels('grok-imagine-image'))).toBe('grok-imagine-image')
    expect(pickDefaultImageModel(makeModels('gpt-5.5'))).toBe('')
    expect(pickDefaultImageModel([])).toBe('')
  })
})

describe('size tier pricing', () => {
  it('exposes 1K/2K/4K plus auto as the values the upstream accepts', () => {
    expect(IMAGE_SIZE_TIERS.map((tier) => tier.value)).toEqual(['', '1K', '2K', '4K'])
  })

  it('reads the unit price for the selected tier from the group config', () => {
    const group = makeGroup()
    expect(unitPriceForSelection(group, '1K')).toBe(0.04)
    expect(unitPriceForSelection(group, '2K')).toBe(0.08)
    expect(unitPriceForSelection(group, '4K')).toBe(0.1)
  })

  it('maps auto to the gateway default tier instead of showing no price', () => {
    // backend NormalizeImageBillingTierOrDefault('') == 2K
    expect(unitPriceForSelection(makeGroup(), '')).toBe(0.08)
  })

  it('returns null when the group has no price configured', () => {
    const group = makeGroup({ image_price_1k: null, image_price_2k: null, image_price_4k: null })
    expect(unitPriceForSelection(group, '1K')).toBeNull()
    expect(unitPriceForSelection(null, '1K')).toBeNull()
    expect(estimateImageCost(group, '1K', 2)).toBeNull()
  })

  it('estimates the total and clamps an out-of-range count', () => {
    const group = makeGroup()
    expect(estimateImageCost(group, '4K', 3)).toBeCloseTo(0.3)
    expect(estimateImageCost(group, '1K', 99)).toBe(0.04)
  })
})

describe('normalizeSizeTier', () => {
  it('keeps known tiers', () => {
    for (const value of ['', '1K', '2K', '4K']) {
      expect(normalizeSizeTier(value)).toBe(value)
    }
  })

  it('maps legacy pixel sizes to the same tier the backend classifies', () => {
    // 与 backend ClassifyImageBillingTier 一致：按长边 ≤1024 算 1K、≤2048 算 2K。
    expect(normalizeSizeTier('1024x1024')).toBe('1K')
    expect(normalizeSizeTier('1536x1024')).toBe('2K')
    expect(normalizeSizeTier('1024x1536')).toBe('2K')
    expect(normalizeSizeTier('2048x2048')).toBe('2K')
    expect(normalizeSizeTier('3840x2160')).toBe('4K')
  })

  it('falls back to auto for junk', () => {
    expect(normalizeSizeTier(null)).toBe('')
    expect(normalizeSizeTier('huge')).toBe('')
    expect(normalizeSizeTier('0x0')).toBe('')
  })
})

describe('formatImagePrice', () => {
  it('keeps sub-cent prices readable without trailing zero noise', () => {
    expect(formatImagePrice(0.04)).toBe('$0.04')
    expect(formatImagePrice(0.1)).toBe('$0.10')
    expect(formatImagePrice(0.0268)).toBe('$0.0268')
    expect(formatImagePrice(1.5)).toBe('$1.50')
    expect(formatImagePrice(0)).toBe('$0.00')
    expect(formatImagePrice(Number.NaN)).toBe('-')
  })
})
