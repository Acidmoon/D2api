import { buildGatewayUrl } from './client'

/**
 * 生图创作中心使用的网关接口。
 *
 * 与批量生图一致，浏览器直接携带用户自己的 API Key 调用网关
 * （`Authorization: Bearer <key>`），不经过用户会话的 REST 层，
 * 这样计费/分组/权限完全走网关既有链路。
 */

export interface GatewayModel {
  id: string
  object?: string
  owned_by?: string
  type?: string
  display_name?: string
}

export interface GeneratedImage {
  b64_json?: string
  url?: string
  revised_prompt?: string
  mime_type?: string
}

export interface ImageGenerationResponse {
  created?: number
  data: GeneratedImage[]
}

export interface ImageGenerationPayload {
  model: string
  prompt: string
  /** 生成张数，默认 1 */
  n?: number
  /** 尺寸，如 1024x1024 / 1536x1024 / 1024x1536；留空交给上游默认 */
  size?: string
  /** 画质，如 low / medium / high */
  quality?: string
}

async function parseGatewayError(response: Response): Promise<Error> {
  try {
    const body = await response.json()
    const message = body?.error?.message || body?.message || response.statusText
    const error = new Error(message || `HTTP ${response.status}`)
    ;(error as any).code = body?.error?.code || response.status
    ;(error as any).status = response.status
    return error
  } catch {
    const error = new Error(response.statusText || `HTTP ${response.status}`)
    ;(error as any).code = response.status
    ;(error as any).status = response.status
    return error
  }
}

function authHeaders(apiKey: string, extra?: HeadersInit): HeadersInit {
  return {
    Authorization: `Bearer ${apiKey}`,
    ...extra,
  }
}

/** 拉取该 API Key 所属分组当前可用的模型列表（GET /v1/models）。 */
export async function listGatewayModels(apiKey: string, signal?: AbortSignal): Promise<GatewayModel[]> {
  const response = await fetch(buildGatewayUrl('/v1/models'), {
    headers: authHeaders(apiKey),
    signal,
  })
  if (!response.ok) throw await parseGatewayError(response)
  const body = await response.json()
  return Array.isArray(body?.data) ? (body.data as GatewayModel[]) : []
}

/** 同步文生图（POST /v1/images/generations）。 */
export async function generateImages(
  apiKey: string,
  payload: ImageGenerationPayload,
  signal?: AbortSignal
): Promise<ImageGenerationResponse> {
  const response = await fetch(buildGatewayUrl('/v1/images/generations'), {
    method: 'POST',
    headers: authHeaders(apiKey, { 'Content-Type': 'application/json' }),
    body: JSON.stringify(payload),
    signal,
  })
  if (!response.ok) throw await parseGatewayError(response)
  const body = await response.json()
  return {
    created: body?.created,
    data: Array.isArray(body?.data) ? (body.data as GeneratedImage[]) : [],
  }
}

function base64ToBlob(base64: string, mimeType: string): Blob {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new Blob([bytes], { type: mimeType })
}

/** 从 base64 头部魔数识别图片 MIME，避免上游未回传 mime_type 时误标扩展名。 */
function detectMimeFromBase64(base64: string): string {
  const head = base64.slice(0, 24)
  let raw = ''
  try {
    raw = atob(head + '='.repeat((-head.length % 4 + 4) % 4))
  } catch {
    return ''
  }
  if (raw.startsWith('\x89PNG\r\n\x1a\n')) return 'image/png'
  if (raw.startsWith('\xff\xd8\xff')) return 'image/jpeg'
  if (raw.startsWith('RIFF') && raw.slice(8, 12) === 'WEBP') return 'image/webp'
  return ''
}

function resolvedMime(image: GeneratedImage): string {
  if (image.mime_type) return image.mime_type
  if (image.b64_json) {
    const detected = detectMimeFromBase64(image.b64_json)
    if (detected) return detected
  }
  return 'image/png'
}

/** 把一张生成结果转成可在页面中展示的 data URL。 */
export function imageToDataUrl(image: GeneratedImage): string {
  if (image.b64_json) {
    return `data:${resolvedMime(image)};base64,${image.b64_json}`
  }
  return image.url || ''
}

/** 取回原图 Blob（b64 直接解码，url 走一次 fetch 以便本地下载）。 */
export async function imageToBlob(image: GeneratedImage): Promise<Blob> {
  if (image.b64_json) {
    return base64ToBlob(image.b64_json, resolvedMime(image))
  }
  if (!image.url) throw new Error('image has neither b64_json nor url')
  const response = await fetch(image.url)
  if (!response.ok) throw new Error(`failed to fetch image: HTTP ${response.status}`)
  return response.blob()
}

/** 根据 MIME 推断扩展名，用于下载文件名。 */
export function extensionForBlob(blob: Blob): string {
  const mime = (blob.type || '').toLowerCase()
  if (mime.includes('jpeg') || mime.includes('jpg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  return 'png'
}
