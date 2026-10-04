/**
 * Resolves a requested language code against a list of available language codes.
 *
 * Matching rules:
 * 1. Exact match (case-insensitive): e.g. "zh-CN" or "zh-cn" matches "zh-CN"
 * 2. Bare primary tag match: e.g. "zh" matches "zh-CN", "de" matches "de-DE"
 * 3. Falls back to null if no match is found.
 *
 * @param {string} candidate - The language code requested (e.g. from config, query param, or event).
 * @param {string[]} availableCodes - The available language codes from GET /api/v1/lang.
 * @returns {string|null} The matched available language code, or null.
 */
export function resolveLanguage(candidate, availableCodes) {
  if (!candidate || typeof candidate !== 'string' || !Array.isArray(availableCodes)) {
    return null
  }

  const normalized = candidate.trim().toLowerCase()
  if (!normalized) {
    return null
  }

  // 1. Exact match (case-insensitive)
  const exact = availableCodes.find((code) => code.toLowerCase() === normalized)
  if (exact) {
    return exact
  }

  // 2. Primary tag match (e.g. "zh" or "zh-TW" -> "zh-CN")
  const primary = normalized.split('-')[0]
  const primaryMatch = availableCodes.find((code) => code.toLowerCase().split('-')[0] === primary)
  if (primaryMatch) {
    return primaryMatch
  }

  return null
}

/**
 * Determines the initial widget language based on widget configuration,
 * host-provided URL language parameter, and browser language detection.
 *
 * @param {Object} options
 * @param {Object} options.widgetConfig - Inbox widget settings configuration.
 * @param {string|null} options.urlLang - Language code passed via URL query parameter `lang`.
 * @param {string|null} options.browserLang - Language code detected from navigator.language.
 * @param {string[]} options.availableCodes - List of available language codes from the server.
 * @returns {string} The resolved language code to load.
 */
export function determineInitialLanguage({ widgetConfig, urlLang, browserLang, availableCodes }) {
  const fallbackLang = widgetConfig?.fallback_language || 'en-US'

  if (widgetConfig?.language === 'auto') {
    const hostLang = resolveLanguage(urlLang, availableCodes)
    if (hostLang) {
      return hostLang
    }

    const detectedBrowserLang = resolveLanguage(browserLang, availableCodes)
    return detectedBrowserLang || fallbackLang
  }

  return widgetConfig?.language || fallbackLang
}
