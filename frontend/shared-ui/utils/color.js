// Mirrors `--primary-foreground` values defined in main.scss so computed
// foregrounds stay visually consistent with the theme's hand-picked tokens.
const LIGHT_FOREGROUND_HSL = '0 0% 98%'
const DARK_FOREGROUND_HSL = '240 7% 11%'

const parseHexRGB = (hex) => {
  hex = hex.replace(/^#/, '')
  if (hex.length === 3) {
    hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2]
  }
  return [
    parseInt(hex.substring(0, 2), 16),
    parseInt(hex.substring(2, 4), 16),
    parseInt(hex.substring(4, 6), 16)
  ]
}

const relativeLuminance = (hex) => {
  const [r8, g8, b8] = parseHexRGB(hex)
  const toLinear = (c) => {
    const v = c / 255
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
  }
  return 0.2126 * toLinear(r8) + 0.7152 * toLinear(g8) + 0.0722 * toLinear(b8)
}

/**
 * Convert hex color to HSL format for CSS variables.
 * @param {string} hex - Hex color (e.g., "#2563eb" or "2563eb")
 * @returns {string} HSL values formatted for CSS (e.g., "217 91% 60%")
 */
export const hexToHSL = (hex) => {
  const [r8, g8, b8] = parseHexRGB(hex)
  const r = r8 / 255
  const g = g8 / 255
  const b = b8 / 255

  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  let h,
    s,
    l = (max + min) / 2

  if (max === min) {
    h = s = 0
  } else {
    const d = max - min
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min)
    switch (max) {
      case r:
        h = ((g - b) / d + (g < b ? 6 : 0)) / 6
        break
      case g:
        h = ((b - r) / d + 2) / 6
        break
      case b:
        h = ((r - g) / d + 4) / 6
        break
    }
  }

  return `${Math.round(h * 360)} ${Math.round(s * 100)}% ${Math.round(l * 100)}%`
}

/**
 * Return an HSL string for a foreground color that has sufficient contrast
 * against the given hex background color. Uses WCAG relative luminance.
 * @param {string} hex - Hex color (e.g., "#2563eb" or "2563eb")
 * @returns {string} HSL string ready for a CSS variable
 */
export const getContrastingHSL = (hex) => {
  return relativeLuminance(hex) > 0.179 ? DARK_FOREGROUND_HSL : LIGHT_FOREGROUND_HSL
}

// WCAG contrast ratio between two hex colors, from 1 (identical) to 21 (black/white).
export const contrastRatio = (hex1, hex2) => {
  const l1 = relativeLuminance(hex1)
  const l2 = relativeLuminance(hex2)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)
  return (lighter + 0.05) / (darker + 0.05)
}
