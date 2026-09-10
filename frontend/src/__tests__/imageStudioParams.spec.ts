import { describe, expect, it } from 'vitest'

/**
 * 回归守卫：生图创作中心的参数区只放上游真正吃得下的开关。
 *
 * 生产实测（docs/IMAGE_STUDIO.md）：创作分组的上游会原样回显 size/quality/output_format
 * 却全都不执行——`1K`/`4K`/`2048x2048`/非法值都输出约 1.57MP，请求 jpeg 回来的仍是 PNG。
 * 所以：
 * 1. 尺寸必须走 IMAGE_SIZE_TIERS 档位，不许再写死 `1024x1024` 这类像素常量；
 * 2. quality / output_format / aspect_ratio 这类假开关不得回到请求体里。
 */
const viewSource = import.meta.glob('../views/user/ImageStudioView.vue', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>

const optionsSource = import.meta.glob('../composables/useImageStudioOptions.ts', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>

function first(sources: Record<string, string>): string {
  const entries = Object.values(sources)
  expect(entries.length).toBe(1)
  return entries[0]
}

describe('image studio parameter guards', () => {
  const view = first(viewSource)
  const options = first(optionsSource)

  it('drives size from the shared tier list instead of inline pixel values', () => {
    expect(view).toContain('IMAGE_SIZE_TIERS')
    for (const pixel of ['1024x1024', '1536x1024', '1024x1536', '2048x2048']) {
      expect(view).not.toContain(pixel)
    }
  })

  it('keeps the tiers the upstream accepts (1K/2K/4K plus auto)', () => {
    expect(options).toMatch(/value:\s*'1K'/)
    expect(options).toMatch(/value:\s*'2K'/)
    expect(options).toMatch(/value:\s*'4K'/)
    expect(options).toMatch(/value:\s*''/)
  })

  it('does not send switches the upstream ignores', () => {
    // 参数区如果哪天要放开这些，必须先在目标上游实测出可观测差异。
    expect(view).not.toMatch(/quality\s*:/)
    expect(view).not.toMatch(/output_format\s*:/)
    expect(view).not.toMatch(/aspect_ratio\s*:/)
  })

  it('filters the model list to image-capable models', () => {
    expect(view).toContain('filterImageModels')
    expect(options).toMatch(/gpt-image-/)
  })
})
