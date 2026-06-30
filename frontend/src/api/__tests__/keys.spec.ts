import { describe, expect, it, vi, beforeEach } from 'vitest'

const postMock = vi.fn()
const putMock = vi.fn()

vi.mock('@/api/client', () => ({
  apiClient: {
    post: postMock,
    put: putMock
  }
}))

describe('keys API', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    postMock.mockResolvedValue({ data: {} })
    putMock.mockResolvedValue({ data: {} })
  })

  it('sends primary and secondary group ids when creating a key', async () => {
    const { create } = await import('@/api/keys')

    await create('demo', 10, 20)

    expect(postMock).toHaveBeenCalledWith('/keys', {
      name: 'demo',
      primary_group_id: 10,
      group_id: 20
    })
  })

  it('preserves explicit null group ids when updating a key', async () => {
    const { update } = await import('@/api/keys')

    await update(7, { primary_group_id: null, group_id: 20 })

    expect(putMock).toHaveBeenCalledWith('/keys/7', {
      primary_group_id: null,
      group_id: 20
    })
  })
})
