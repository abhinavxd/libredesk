const normalizeLanguage = (requested) => {
  try {
    return new Intl.Locale(String(requested || '').trim().replaceAll('_', '-')).baseName
  } catch {
    return ''
  }
}

export const matchLanguage = (requested, availableCodes) => {
  const wanted = normalizeLanguage(requested)
  if (!wanted || !Array.isArray(availableCodes)) return ''
  const exact = availableCodes.find((code) => normalizeLanguage(code) === wanted)
  if (exact) return exact
  const locale = new Intl.Locale(wanted).maximize()
  return availableCodes.find((code) => {
    const normalized = normalizeLanguage(code)
    if (!normalized) return false
    const candidate = new Intl.Locale(normalized).maximize()
    return candidate.language === locale.language && candidate.script === locale.script
  }) || ''
}

export const resolveLanguage = (config, hostLanguage, browserLanguages, availableCodes) => {
  const fallback = normalizeLanguage(config.fallback_language) || 'en-US'
  if (config.language !== 'auto') {
    const locale = normalizeLanguage(config.language) || fallback
    return { locale, requestedLocale: locale }
  }
  const browserLanguage = browserLanguages[0] || ''
  const locale = matchLanguage(hostLanguage, availableCodes) ||
    (availableCodes.includes(browserLanguage) ? browserLanguage : fallback)
  const requestedLocale = normalizeLanguage(hostLanguage) || locale
  return { locale, requestedLocale }
}
