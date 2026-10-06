import { afterEach, describe, expect, it, vi } from 'vitest'
import { encodeAudioReply } from './audioReply'

afterEach(() => vi.unstubAllGlobals())
describe('Audio reply encoding', () => {
  it('encodes stereo samples as a playable mono MP3 and releases the audio context', async () => {
    const close = vi.fn()
    const samples = Float32Array.from(
      { length: 4410 },
      (_, i) => Math.sin((i * 2 * Math.PI * 440) / 44100) * 0.2
    )
    vi.stubGlobal(
      'AudioContext',
      class {
        decodeAudioData = async () => ({
          numberOfChannels: 2,
          length: samples.length,
          sampleRate: 44100,
          getChannelData: () => samples
        })
        close = close
      }
    )
    const file = await encodeAudioReply(new Blob(['audio']))
    expect(file.type).toBe('audio/mpeg')
    expect(file.name).toMatch(/\.mp3$/)
    expect(file.size).toBeGreaterThan(1000)
    const bytes = new Uint8Array(await file.arrayBuffer())
    expect(bytes[0]).toBe(255)
    expect(bytes[1] & 224).toBe(224)
    expect(close).toHaveBeenCalledOnce()
  })
  it('releases the audio context if decoding fails', async () => {
    const close = vi.fn()
    vi.stubGlobal(
      'AudioContext',
      class {
        decodeAudioData = async () => {
          throw new Error('decode failed')
        }
        close = close
      }
    )
    await expect(encodeAudioReply(new Blob(['invalid']))).rejects.toThrow('decode failed')
    expect(close).toHaveBeenCalledOnce()
  })
})
