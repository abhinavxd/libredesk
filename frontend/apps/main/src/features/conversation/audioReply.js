export async function encodeAudioReply(blob) {
  const { default: lamejs } = await import('@breezystack/lamejs')
  const context = new AudioContext({ sampleRate: 44100 })
  try {
    const decoded = await context.decodeAudioData(await blob.arrayBuffer())
    const channels = Array.from({ length: decoded.numberOfChannels }, (_, index) =>
      decoded.getChannelData(index)
    )
    const samples = new Int16Array(decoded.length)
    for (let i = 0; i < decoded.length; i++) {
      const value = Math.max(
        -1,
        Math.min(1, channels.reduce((sum, channel) => sum + channel[i], 0) / channels.length)
      )
      samples[i] = value < 0 ? value * 32768 : value * 32767
    }
    const encoder = new lamejs.Mp3Encoder(1, decoded.sampleRate, 128)
    const chunks = []
    for (let offset = 0; offset < samples.length; offset += 1152) {
      const chunk = encoder.encodeBuffer(samples.subarray(offset, offset + 1152))
      if (chunk.length) chunks.push(new Int8Array(chunk))
    }
    chunks.push(new Int8Array(encoder.flush()))
    return new File(chunks, `recording-${Date.now()}.mp3`, { type: 'audio/mpeg' })
  } finally {
    await context.close()
  }
}
