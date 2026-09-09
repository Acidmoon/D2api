import { describe, expect, it } from 'vitest'

import { isImageStudioKey } from '../useImageStudioAccess'
import type { ApiKey } from '@/types'

function makeKey(overrides: {
  status?: ApiKey['status']
  groupName?: string | null
  allowImageGeneration?: boolean | null
}): ApiKey {
  const { status = 'active', groupName = '创作分组', allowImageGeneration = true } = overrides
  return {
    status,
    group:
      groupName === null
        ? undefined
        : ({ name: groupName, allow_image_generation: allowImageGeneration ?? undefined } as ApiKey['group'])
  } as ApiKey
}

describe('isImageStudioKey', () => {
  it('accepts active keys in the creation group with image generation enabled', () => {
    expect(isImageStudioKey(makeKey({}))).toBe(true)
  })

  it('rejects keys from other groups even when image generation is enabled', () => {
    expect(isImageStudioKey(makeKey({ groupName: 'Plus（禁止破限禁止涩涩）' }))).toBe(false)
    expect(isImageStudioKey(makeKey({ groupName: 'Grok' }))).toBe(false)
  })

  it('rejects the creation group when image generation is disabled', () => {
    expect(isImageStudioKey(makeKey({ allowImageGeneration: false }))).toBe(false)
    expect(isImageStudioKey(makeKey({ allowImageGeneration: null }))).toBe(false)
  })

  it('rejects inactive keys and keys without a group', () => {
    expect(isImageStudioKey(makeKey({ status: 'inactive' }))).toBe(false)
    expect(isImageStudioKey(makeKey({ groupName: null }))).toBe(false)
  })
})
