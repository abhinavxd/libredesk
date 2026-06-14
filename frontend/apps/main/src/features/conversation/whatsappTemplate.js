export const WHATSAPP_CHANNEL = 'whatsapp'
export const WHATSAPP_WINDOW_MS = 24 * 60 * 60 * 1000
export const PLACEHOLDER_PATTERN = /\{\{([A-Za-z0-9_]+)\}\}/g

export const extractPlaceholders = (sources) => {
  const seen = new Set()
  const out = []
  for (const src of sources) {
    if (!src) continue
    for (const match of src.matchAll(PLACEHOLDER_PATTERN)) {
      if (!seen.has(match[1])) {
        seen.add(match[1])
        out.push(match[1])
      }
    }
  }
  return out
}

export const placeholderLabel = (key) => `{{${key}}}`
