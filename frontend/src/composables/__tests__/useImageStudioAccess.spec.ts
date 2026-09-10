import { describe, expect, it } from 'vitest'

import { creationGroupOf, isImageStudioKey } from '../useImageStudioAccess'
import type { ApiKey, Group } from '@/types'

function makeGroup(name: string, allowImageGeneration: boolean | null = true): Group {
  return {
    name,
    allow_image_generation: allowImageGeneration ?? undefined
  } as Group
}

function makeKey(overrides: {
  status?: ApiKey['status']
  primaryGroupName?: string | null
  groupName?: string | null
  allowImageGeneration?: boolean | null
}): ApiKey {
  const {
    status = 'active',
    primaryGroupName = '创作分组',
    groupName = null,
    allowImageGeneration = true
  } = overrides
  return {
    status,
    primary_group: primaryGroupName === null ? undefined : makeGroup(primaryGroupName, allowImageGeneration),
    group: groupName === null ? undefined : makeGroup(groupName, allowImageGeneration)
  } as ApiKey
}

describe('creationGroupOf / isImageStudioKey', () => {
  it('matches a key bound through primary_group (group_id is null)', () => {
    const key = makeKey({ primaryGroupName: '创作分组', groupName: null })
    expect(creationGroupOf(key)?.name).toBe('创作分组')
    expect(isImageStudioKey(key)).toBe(true)
  })

  it('matches a key bound through the secondary group as well', () => {
    const key = makeKey({ primaryGroupName: null, groupName: '创作分组' })
    expect(isImageStudioKey(key)).toBe(true)
  })

  it('picks the creation group when both groups are set', () => {
    const key = makeKey({ primaryGroupName: 'Plus（禁止破限禁止涩涩）', groupName: '创作分组' })
    expect(creationGroupOf(key)?.name).toBe('创作分组')
    expect(isImageStudioKey(key)).toBe(true)
  })

  it('rejects keys from other groups even when image generation is enabled', () => {
    expect(isImageStudioKey(makeKey({ primaryGroupName: 'Plus（禁止破限禁止涩涩）', groupName: 'Grok' }))).toBe(false)
  })

  it('rejects the creation group when image generation is disabled', () => {
    expect(isImageStudioKey(makeKey({ allowImageGeneration: false }))).toBe(false)
    expect(isImageStudioKey(makeKey({ allowImageGeneration: null }))).toBe(false)
  })

  it('rejects inactive keys and keys without any group', () => {
    expect(isImageStudioKey(makeKey({ status: 'inactive' }))).toBe(false)
    expect(isImageStudioKey(makeKey({ primaryGroupName: null, groupName: null }))).toBe(false)
  })
})
