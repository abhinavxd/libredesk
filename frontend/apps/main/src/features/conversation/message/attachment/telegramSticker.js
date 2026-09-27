export async function decodeTelegramSticker(buffer) {
  if (buffer.byteLength > 65536) throw new Error('Sticker exceeds 64 KB')
  const stream = new Blob([buffer]).stream().pipeThrough(new DecompressionStream('gzip'))
  const reader = stream.getReader()
  const chunks = []
  let size = 0
  try {
    for (let chunk = await reader.read(); !chunk.done; chunk = await reader.read()) {
      const { value } = chunk
      size += value.byteLength
      if (size > 4 * 1024 * 1024) throw new Error('Sticker data is too large')
      chunks.push(value)
    }
  } finally {
    await reader.cancel()
  }
  const data = JSON.parse(await new Blob(chunks).text())
  if (
    !Array.isArray(data.layers) ||
    data.w !== 512 ||
    data.h !== 512 ||
    data.fonts ||
    data.chars ||
    data.assets?.some((asset) => asset.p || asset.u)
  ) {
    throw new Error('Unsupported sticker data')
  }
  return data
}
