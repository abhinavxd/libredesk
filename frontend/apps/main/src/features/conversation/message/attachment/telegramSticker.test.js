import { gzipSync } from 'node:zlib'
import { describe, expect, it } from 'vitest'
import { decodeTelegramSticker } from './telegramSticker'
const valid = { w: 512, h: 512, layers: [], assets: [] }
const encode = (data) => gzipSync(JSON.stringify(data))

describe('Telegram animated sticker decoding', () => {
  it('decodes a Telegram gzip animation', async () => {
    expect(await decodeTelegramSticker(encode(valid))).toEqual(valid)
  })
  it.each([
    { w: 0 },
    { h: 0 },
    { layers: null },
    { fonts: {} },
    { chars: [] },
    { assets: [{ p: 'external.png' }] },
    { assets: [{ u: 'https://example.com/' }] }
  ])('rejects unsupported content and external resources: %j', async (value) => {
    await expect(decodeTelegramSticker(encode({ ...valid, ...value }))).rejects.toThrow(
      'Unsupported sticker data'
    )
  })
  it('rejects oversized compressed and expanded files', async () => {
    await expect(decodeTelegramSticker(new Uint8Array(65537))).rejects.toThrow('64 KB')
    await expect(
      decodeTelegramSticker(encode({ ...valid, data: 'a'.repeat(4 * 1024 * 1024) }))
    ).rejects.toThrow('too large')
  })
  it('rejects invalid gzip and JSON', async () => {
    await expect(decodeTelegramSticker(new Uint8Array([1, 2, 3]))).rejects.toThrow()
    await expect(decodeTelegramSticker(gzipSync('not json'))).rejects.toThrow()
  })
})
