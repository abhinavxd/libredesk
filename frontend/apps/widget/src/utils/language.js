// Exact code first (case-insensitive), else the first code with the same primary subtag (zh -> zh-CN).
export const matchLanguage = (requested, availableCodes) => {
  const wanted = String(requested || '')
    .trim()
    .toLowerCase()
    .replace('_', '-')
  if (!wanted || !Array.isArray(availableCodes)) return ''
  const exact = availableCodes.find((code) => code.toLowerCase() === wanted)
  if (exact) return exact
  const primary = wanted.split('-')[0]
  return availableCodes.find((code) => code.toLowerCase().split('-')[0] === primary) || ''
}
