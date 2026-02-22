/**
 * Convert hex color to HSL format for CSS variables.
 * @param {string} hex - Hex color (e.g., "#2563eb" or "2563eb")
 * @returns {string} HSL values formatted for CSS (e.g., "217 91% 60%")
 */
export const hexToHSL = (hex) => {
  hex = hex.replace(/^#/, '')

  const r = parseInt(hex.substring(0, 2), 16) / 255
  const g = parseInt(hex.substring(2, 4), 16) / 255
  const b = parseInt(hex.substring(4, 6), 16) / 255

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
 * Apply a hex color to a CSS variable on the document root.
 * @param {string} variableName - CSS variable name (e.g., "--primary")
 * @param {string} hexColor - Hex color value
 */
export const applyCSSColor = (variableName, hexColor) => {
  if (hexColor) {
    const hslValue = hexToHSL(hexColor)
    document.documentElement.style.setProperty(variableName, hslValue)
  }
}
