// Decides whether an incoming email needs the light canvas in dark mode (#449).
// Emails without their own colors render fine in the theme. Emails that set colors
// get the canvas, unless all of their coloring sits inside a dark background of their own.

const NOT_A_COLOR = new Set([
  'inherit',
  'initial',
  'unset',
  'revert',
  'currentcolor',
  'transparent',
  'none'
])
const NAMED_COLORS = {
  black: [0, 0, 0],
  white: [255, 255, 255],
  gray: [128, 128, 128],
  grey: [128, 128, 128],
  silver: [192, 192, 192],
  navy: [0, 0, 128],
  maroon: [128, 0, 0]
}
const DARK_LUMINANCE = 0.35

function styleValue(el, properties) {
  const style = el.getAttribute('style')
  if (!style) return null
  let value = null
  for (const declaration of style.split(';')) {
    const index = declaration.indexOf(':')
    if (index === -1) continue
    const property = declaration.slice(0, index).trim().toLowerCase()
    if (properties.includes(property)) value = declaration.slice(index + 1).trim()
  }
  return value
}

// Returns a value only if it actually sets a color (or might).
function colorValue(value) {
  if (!value) return null
  const trimmed = value.replace(/!important/i, '').trim()
  return NOT_A_COLOR.has(trimmed.toLowerCase()) ? null : trimmed
}

function backgroundOf(el) {
  return colorValue(
    styleValue(el, ['background', 'background-color']) || el.getAttribute('bgcolor')
  )
}

function textColorOf(el) {
  const fontColor = el.tagName === 'FONT' ? el.getAttribute('color') : null
  const bodyText = el.tagName === 'BODY' ? el.getAttribute('text') : null
  return colorValue(styleValue(el, ['color']) || fontColor || bodyText)
}

// Parses the first color in a CSS value into [r, g, b, alpha], or null if it can't be read.
function parseColor(value) {
  const hex = value.match(/#([0-9a-f]{3,8})\b/i)
  if (hex) {
    let digits = hex[1]
    if (digits.length === 3 || digits.length === 4) {
      digits = digits
        .split('')
        .map((d) => d + d)
        .join('')
    }
    if (digits.length !== 6 && digits.length !== 8) return null
    const alpha = digits.length === 8 ? parseInt(digits.slice(6, 8), 16) / 255 : 1
    return [...[0, 2, 4].map((i) => parseInt(digits.slice(i, i + 2), 16)), alpha]
  }
  const rgb = value.match(/rgba?\(\s*(\d+)[\s,]+(\d+)[\s,]+(\d+)(?:[\s,/]+([\d.]+)(%?))?/i)
  if (rgb) {
    const alpha = rgb[4] === undefined ? 1 : Number(rgb[4]) / (rgb[5] ? 100 : 1)
    return [...rgb.slice(1, 4).map(Number), alpha]
  }
  const named = value.toLowerCase().match(/^[a-z]+/)
  return named && NAMED_COLORS[named[0]] ? [...NAMED_COLORS[named[0]], 1] : null
}

// A background only counts as dark when it is opaque; a see-through one shows the theme behind it.
function isDark(value) {
  const color = parseColor(value)
  if (!color || color[3] < 1) return false
  const rgb = color.slice(0, 3)
  const [r, g, b] = rgb.map((c) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b < DARK_LUMINANCE
}

export function needsLightCanvas(html) {
  if (!html) return false
  const doc = new DOMParser().parseFromString(html, 'text/html')

  // Colors from <style> rules can't be mapped to elements here, so they keep the canvas.
  for (const style of doc.querySelectorAll('style')) {
    if (/(^|[\s{;])(color|background(-color)?)\s*:/i.test(style.textContent)) return true
  }

  const elements = [...doc.querySelectorAll('body, body *')]
  const withBackground = elements.filter((el) => backgroundOf(el))
  const withTextColor = elements.filter((el) => textColorOf(el))
  if (withBackground.length === 0 && withTextColor.length === 0) return false

  // The outermost backgrounds. If they are all dark and every other color sits inside them,
  // the email is designed for a dark background and renders without the canvas.
  const roots = withBackground.filter(
    (el) => !withBackground.some((other) => other !== el && other.contains(el))
  )
  if (roots.length === 0 || !roots.every((el) => isDark(backgroundOf(el)))) return true
  const insideRoot = (el) => roots.some((root) => root.contains(el))
  return !withTextColor.every(insideRoot)
}
