const DARK_MIN_LIGHTNESS = 0.08
const DARK_MAX_LIGHTNESS = 0.92
const MIN_CONTRAST_RATIO = 4.5
const CONTRAST_ROUNDING_MARGIN = 0.05
const MAX_CACHE_ENTRIES = 512
const CSS_WIDE_KEYWORDS = new Set([
  'currentcolor',
  'inherit',
  'initial',
  'revert',
  'revert-layer',
  'unset'
])
const parsedColorCache = new Map()
const remappedColorCache = new Map()
const contrastColorCache = new Map()

const clamp = (value, min = 0, max = 1) => Math.min(max, Math.max(min, value))

const cacheValue = (cache, key, value) => {
  if (cache.size >= MAX_CACHE_ENTRIES) {
    cache.delete(cache.keys().next().value)
  }
  cache.set(key, value)
  return value
}

const parseAlpha = (value) => {
  const trimmed = value.trim()
  if (trimmed.endsWith('%')) {
    return clamp(Number.parseFloat(trimmed) / 100)
  }
  return clamp(Number.parseFloat(trimmed))
}

const parseRgbChannel = (value) => {
  const trimmed = value.trim()
  if (trimmed.endsWith('%')) {
    return clamp(Number.parseFloat(trimmed) / 100)
  }
  return clamp(Number.parseFloat(trimmed) / 255)
}

const parseHue = (value) => {
  const trimmed = value.trim().toLowerCase()
  let degrees = Number.parseFloat(trimmed)
  if (trimmed.endsWith('turn')) degrees *= 360
  else if (trimmed.endsWith('grad')) degrees *= 0.9
  else if (trimmed.endsWith('rad')) degrees = (degrees * 180) / Math.PI
  return ((degrees % 360) + 360) % 360
}

const splitFunctionalColor = (body) => {
  if (body.includes(',')) {
    const parts = body.split(',').map((part) => part.trim())
    if (parts.length !== 3 && parts.length !== 4) {
      return { channels: [], alpha: undefined }
    }
    return {
      channels: parts.slice(0, 3),
      alpha: parts[3]
    }
  }

  const [channels, alpha] = body.split('/').map((part) => part.trim())
  return {
    channels: channels.split(/\s+/),
    alpha
  }
}

const hslToRgb = ({ h, s, l, a = 1 }) => {
  const chroma = (1 - Math.abs(2 * l - 1)) * s
  const hue = h / 60
  const x = chroma * (1 - Math.abs((hue % 2) - 1))
  let red = 0
  let green = 0
  let blue = 0

  if (hue < 1) [red, green] = [chroma, x]
  else if (hue < 2) [red, green] = [x, chroma]
  else if (hue < 3) [green, blue] = [chroma, x]
  else if (hue < 4) [green, blue] = [x, chroma]
  else if (hue < 5) [red, blue] = [x, chroma]
  else [red, blue] = [chroma, x]

  const match = l - chroma / 2
  return { r: red + match, g: green + match, b: blue + match, a }
}

const rgbToHsl = ({ r, g, b, a = 1 }) => {
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  const delta = max - min
  const l = (max + min) / 2
  let h = 0

  if (delta !== 0) {
    if (max === r) h = 60 * (((g - b) / delta) % 6)
    else if (max === g) h = 60 * ((b - r) / delta + 2)
    else h = 60 * ((r - g) / delta + 4)
  }

  return {
    h: (h + 360) % 360,
    s: delta === 0 ? 0 : delta / (1 - Math.abs(2 * l - 1)),
    l,
    a
  }
}

const parseHexColor = (value) => {
  const match = value.match(/^#([\da-f]{3,4}|[\da-f]{6}|[\da-f]{8})$/i)
  if (!match) return null

  let hex = match[1]
  if (hex.length <= 4) {
    hex = [...hex].map((character) => character.repeat(2)).join('')
  }

  const channels = [
    Number.parseInt(hex.slice(0, 2), 16),
    Number.parseInt(hex.slice(2, 4), 16),
    Number.parseInt(hex.slice(4, 6), 16)
  ]

  return {
    r: channels[0] / 255,
    g: channels[1] / 255,
    b: channels[2] / 255,
    a: hex.length === 8 ? Number.parseInt(hex.slice(6, 8), 16) / 255 : 1
  }
}

const parseRgbColor = (value) => {
  const match = value.match(/^rgba?\((.*)\)$/i)
  if (!match) return null

  const { channels, alpha } = splitFunctionalColor(match[1])
  if (channels.length !== 3) return null

  const parsedChannels = channels.map(parseRgbChannel)
  const parsedAlpha = alpha === undefined ? 1 : parseAlpha(alpha)
  if ([...parsedChannels, parsedAlpha].some((channel) => Number.isNaN(channel))) return null

  return {
    r: parsedChannels[0],
    g: parsedChannels[1],
    b: parsedChannels[2],
    a: parsedAlpha
  }
}

const parseHslColor = (value) => {
  const match = value.match(/^hsla?\((.*)\)$/i)
  if (!match) return null

  const { channels, alpha } = splitFunctionalColor(match[1])
  if (
    channels.length !== 3 ||
    !channels[1].trim().endsWith('%') ||
    !channels[2].trim().endsWith('%')
  ) {
    return null
  }

  const hsl = {
    h: parseHue(channels[0]),
    s: clamp(Number.parseFloat(channels[1]) / 100),
    l: clamp(Number.parseFloat(channels[2]) / 100),
    a: alpha === undefined ? 1 : parseAlpha(alpha)
  }

  if (Object.values(hsl).some((channel) => Number.isNaN(channel))) return null
  return hslToRgb(hsl)
}

const parseNamedColor = (value) => {
  if (typeof document === 'undefined' || typeof window === 'undefined') return null

  const probe = document.createElement('span')
  try {
    probe.style.color = value
    if (!probe.style.color) return null

    probe.hidden = true
    document.documentElement.appendChild(probe)
    const computed = window.getComputedStyle(probe).color
    return parseRgbColor(computed)
  } catch {
    return null
  } finally {
    probe.remove()
  }
}

/**
 * Parses a CSS color into RGBA channels scaled from 0 to 1.
 * @param {string} value
 * @returns {{r: number, g: number, b: number, a: number} | null}
 */
export const parseCssColor = (value) => {
  if (typeof value !== 'string') return null

  const trimmed = value.trim()
  const lower = trimmed.toLowerCase()
  if (parsedColorCache.has(lower)) return parsedColorCache.get(lower)
  if (!trimmed || CSS_WIDE_KEYWORDS.has(lower)) {
    return cacheValue(parsedColorCache, lower, null)
  }
  if (lower === 'transparent') {
    return cacheValue(parsedColorCache, lower, Object.freeze({ r: 0, g: 0, b: 0, a: 0 }))
  }

  const parsed =
    parseHexColor(trimmed) ||
    parseRgbColor(trimmed) ||
    parseHslColor(trimmed) ||
    parseNamedColor(trimmed)
  return cacheValue(parsedColorCache, lower, parsed ? Object.freeze(parsed) : null)
}

const formatNumber = (value, precision = 2) => Number(value.toFixed(precision)).toString()

const formatHslColor = ({ h, s, l, a = 1 }) => {
  const channels = `${formatNumber(h)} ${formatNumber(s * 100)}% ${formatNumber(l * 100)}%`
  if (a === 1) return `hsl(${channels})`
  return `hsl(${channels} / ${formatNumber(a, 3)})`
}

const composite = (foreground, background) => {
  const alpha = foreground.a + background.a * (1 - foreground.a)
  if (alpha === 0) return { r: 0, g: 0, b: 0, a: 0 }

  return {
    r: (foreground.r * foreground.a + background.r * background.a * (1 - foreground.a)) / alpha,
    g: (foreground.g * foreground.a + background.g * background.a * (1 - foreground.a)) / alpha,
    b: (foreground.b * foreground.a + background.b * background.a * (1 - foreground.a)) / alpha,
    a: alpha
  }
}

/**
 * Blends a foreground color over a background color.
 * @param {string} foregroundValue
 * @param {string} backgroundValue
 * @returns {string | null}
 */
export const compositeCssColor = (foregroundValue, backgroundValue) => {
  const foreground = parseCssColor(foregroundValue)
  const background = parseCssColor(backgroundValue)
  if (!foreground || !background) return null
  if (foreground.a === 1) return foregroundValue
  return formatHslColor(rgbToHsl(composite(foreground, background)))
}

const linearize = (channel) =>
  channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4

const relativeLuminance = ({ r, g, b }) =>
  0.2126 * linearize(r) + 0.7152 * linearize(g) + 0.0722 * linearize(b)

const contrastRatio = (foreground, background) => {
  const opaqueBackground =
    background.a === 1 ? background : composite(background, { r: 0, g: 0, b: 0, a: 1 })
  const renderedForeground = composite(foreground, opaqueBackground)
  const lighter = Math.max(
    relativeLuminance(renderedForeground),
    relativeLuminance(opaqueBackground)
  )
  const darker = Math.min(
    relativeLuminance(renderedForeground),
    relativeLuminance(opaqueBackground)
  )
  return (lighter + 0.05) / (darker + 0.05)
}

/**
 * Returns the WCAG contrast ratio between two CSS colors.
 * @param {string} foregroundValue
 * @param {string} backgroundValue
 * @returns {number | null}
 */
export const getContrastRatio = (foregroundValue, backgroundValue) => {
  const foreground = parseCssColor(foregroundValue)
  const background = parseCssColor(backgroundValue)
  if (!foreground || !background) return null
  return contrastRatio(foreground, background)
}

/**
 * Remaps a CSS color for dark mode without changing its hue or saturation.
 * @param {string} value
 * @returns {string}
 */
export const remapColorForDarkMode = (value) => {
  if (remappedColorCache.has(value)) return remappedColorCache.get(value)

  const parsed = parseCssColor(value)
  if (!parsed || parsed.a === 0) return cacheValue(remappedColorCache, value, value)

  const hsl = rgbToHsl(parsed)
  hsl.l = DARK_MIN_LIGHTNESS + (1 - hsl.l) * (DARK_MAX_LIGHTNESS - DARK_MIN_LIGHTNESS)
  return cacheValue(remappedColorCache, value, formatHslColor(hsl))
}

const findPassingLightness = (hsl, background, direction, minimumRatio) => {
  let failing = hsl.l
  let passing = direction > 0 ? 1 : 0
  if (contrastRatio(hslToRgb({ ...hsl, l: passing }), background) < minimumRatio) {
    return null
  }

  for (let iteration = 0; iteration < 14; iteration += 1) {
    const midpoint = (failing + passing) / 2
    if (contrastRatio(hslToRgb({ ...hsl, l: midpoint }), background) >= minimumRatio) {
      passing = midpoint
    } else {
      failing = midpoint
    }
  }
  return clamp(passing + direction * 0.0001)
}

const findPassingLightnessCandidates = (hsl, background, targetRatio, preferLighter) =>
  [
    findPassingLightness(hsl, background, -1, targetRatio),
    findPassingLightness(hsl, background, 1, targetRatio)
  ]
    .filter((lightness) => lightness !== null)
    .sort((left, right) => {
      const distance = Math.abs(left - hsl.l) - Math.abs(right - hsl.l)
      if (distance !== 0 || preferLighter === undefined) return distance
      return preferLighter ? right - left : left - right
    })

/**
 * Adjusts a foreground color until it meets the requested contrast ratio.
 * @param {string} foregroundValue
 * @param {string} backgroundValue
 * @param {number} [minimumRatio=4.5]
 * @param {boolean} [preferLighter]
 * @returns {string}
 */
export const ensureColorContrast = (
  foregroundValue,
  backgroundValue,
  minimumRatio = MIN_CONTRAST_RATIO,
  preferLighter
) => {
  const cacheKey = `${foregroundValue}\0${backgroundValue}\0${minimumRatio}\0${preferLighter}`
  if (contrastColorCache.has(cacheKey)) return contrastColorCache.get(cacheKey)

  const foreground = parseCssColor(foregroundValue)
  const background = parseCssColor(backgroundValue)
  if (!foreground || foreground.a === 0 || !background) {
    return cacheValue(contrastColorCache, cacheKey, foregroundValue)
  }
  if (contrastRatio(foreground, background) >= minimumRatio) {
    return cacheValue(contrastColorCache, cacheKey, foregroundValue)
  }

  const hsl = rgbToHsl(foreground)
  const targetRatio = minimumRatio + CONTRAST_ROUNDING_MARGIN
  let searchHsl = hsl
  let candidates = findPassingLightnessCandidates(searchHsl, background, targetRatio, preferLighter)

  if (candidates.length === 0 && hsl.a !== 1) {
    // Alpha blending caps the contrast achievable by adjusting lightness alone (a near-fully
    // transparent foreground is dominated by the background regardless of its own lightness).
    // Fall back to a fully opaque foreground: black or white against any background reaches a
    // ratio of up to ~21:1, so this recovers WCAG AA (4.5:1) requests that the lightness-only
    // search above could not. It is not guaranteed to satisfy every possible `minimumRatio` -
    // e.g. some backgrounds cap opaque contrast below AAA's 7:1 - in which case the final
    // fallback below still applies.
    searchHsl = { ...hsl, a: 1 }
    candidates = findPassingLightnessCandidates(searchHsl, background, targetRatio, preferLighter)
  }

  if (candidates.length === 0) {
    return cacheValue(contrastColorCache, cacheKey, foregroundValue)
  }
  return cacheValue(
    contrastColorCache,
    cacheKey,
    formatHslColor({ ...searchHsl, l: candidates[0] })
  )
}

/**
 * Remaps a foreground and background together, keeping their contrast direction.
 * @param {string} foregroundValue
 * @param {string} backgroundValue
 * @param {string} [backdropValue='rgb(0, 0, 0)']
 * @returns {{color: string, backgroundColor: string}}
 */
export const remapColorPairForDarkMode = (
  foregroundValue,
  backgroundValue,
  backdropValue = 'rgb(0, 0, 0)'
) => {
  const foreground = parseCssColor(foregroundValue)
  const background = parseCssColor(backgroundValue)
  const remappedForeground = remapColorForDarkMode(foregroundValue)
  const remappedBackground = remapColorForDarkMode(backgroundValue)
  const contrastBackground =
    compositeCssColor(remappedBackground, backdropValue) || remappedBackground

  if (!foreground || !background) {
    return {
      color: remappedForeground,
      backgroundColor: remappedBackground
    }
  }

  return {
    color: ensureColorContrast(
      remappedForeground,
      contrastBackground,
      MIN_CONTRAST_RATIO,
      relativeLuminance(foreground) <= relativeLuminance(background)
    ),
    backgroundColor: remappedBackground
  }
}
