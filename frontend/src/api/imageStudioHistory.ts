import apiClient from './client'

/**
 * 生图创作中心历史接口。
 *
 * 与生成接口不同，历史走用户会话（apiClient 自动带 JWT），
 * 服务端按 user_id 隔离；图片字节通过专用接口读取，不暴露直链。
 */

export interface ImageStudioHistoryImage {
  index: number
  mime_type: string
  bytes: number
}

export interface ImageStudioHistoryItem {
  id: number
  model: string
  prompt: string
  revised_prompt: string
  size: string
  image_count: number
  created_at: string
  expires_at: string
  images: ImageStudioHistoryImage[]
}

export interface ImageStudioHistoryListResponse {
  items: ImageStudioHistoryItem[]
  total: number
  page: number
  page_size: number
}

export interface SaveImageStudioHistoryPayload {
  model: string
  prompt: string
  revised_prompt?: string
  size?: string
  api_key_id?: number | null
  group_id?: number | null
  images: Array<{ mime_type?: string; data: string }>
}

/** 保存一次生成结果（异步、失败不影响生成体验）。 */
export async function saveImageStudioHistory(
  payload: SaveImageStudioHistoryPayload
): Promise<ImageStudioHistoryItem> {
  const response = await apiClient.post<ImageStudioHistoryItem>('/image-studio/history', payload, {
    // base64 图片体积较大，给足超时
    timeout: 120000
  })
  return response.data
}

export async function listImageStudioHistory(
  page = 1,
  pageSize = 20
): Promise<ImageStudioHistoryListResponse> {
  const response = await apiClient.get<ImageStudioHistoryListResponse>('/image-studio/history', {
    params: { page, page_size: pageSize }
  })
  return response.data
}

export async function getImageStudioHistory(
  id: number
): Promise<ImageStudioHistoryItem> {
  const response = await apiClient.get<ImageStudioHistoryItem>(`/image-studio/history/${id}`)
  return response.data
}

export async function getImageStudioHistoryImage(id: number, index: number): Promise<Blob> {
  const response = await apiClient.get<Blob>(`/image-studio/history/${id}/images/${index}`, {
    responseType: 'blob',
    timeout: 60000
  })
  return response.data
}

export async function deleteImageStudioHistory(id: number): Promise<void> {
  await apiClient.delete(`/image-studio/history/${id}`)
}
