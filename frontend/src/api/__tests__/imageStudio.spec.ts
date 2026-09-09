import { describe, expect, it } from 'vitest'

import { extensionForBlob, imageToBlob, imageToDataUrl } from '../imageStudio'

/** 用字节构造 base64，便于验证魔数嗅探。 */
function base64FromBytes(bytes: number[]): string {
  return btoa(String.fromCharCode(...bytes))
}

const PNG = base64FromBytes([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0])
const JPEG = base64FromBytes([0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0])
const WEBP = btoa('RIFF' + String.fromCharCode(0, 0, 0, 0) + 'WEBPVP8 ')

describe('imageStudio helpers', () => {
  it('builds a PNG data URL and detects the mime from magic bytes', () => {
    expect(imageToDataUrl({ b64_json: PNG })).toBe(`data:image/png;base64,${PNG}`)
  })

  it('detects JPEG and WEBP magic bytes', () => {
    expect(imageToDataUrl({ b64_json: JPEG })).toBe(`data:image/jpeg;base64,${JPEG}`)
    expect(imageToDataUrl({ b64_json: WEBP })).toBe(`data:image/webp;base64,${WEBP}`)
  })

  it('prefers an explicit mime_type over sniffing', () => {
    expect(imageToDataUrl({ b64_json: PNG, mime_type: 'image/webp' })).toBe(
      `data:image/webp;base64,${PNG}`
    )
  })

  it('falls back to the upstream URL when no b64 is returned', () => {
    expect(imageToDataUrl({ url: 'https://example.com/a.png' })).toBe('https://example.com/a.png')
  })

  it('decodes b64 into a blob with the detected type', async () => {
    const blob = await imageToBlob({ b64_json: PNG })
    expect(blob.type).toBe('image/png')
    expect(blob.size).toBe(12)
    expect(extensionForBlob(blob)).toBe('png')
  })

  it('maps blob mime to a file extension', () => {
    expect(extensionForBlob(new Blob([], { type: 'image/jpeg' }))).toBe('jpg')
    expect(extensionForBlob(new Blob([], { type: 'image/webp' }))).toBe('webp')
    expect(extensionForBlob(new Blob([], { type: 'application/octet-stream' }))).toBe('png')
  })
})
