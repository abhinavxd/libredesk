import {
  compositeCssColor,
  ensureColorContrast,
  parseCssColor,
  remapColorForDarkMode,
  remapColorPairForDarkMode
} from './emailColorNormalizer.js'

const DARK_MESSAGE_BACKGROUND = 'hsl(120 2.6% 7.6%)'
const DARK_MESSAGE_FOREGROUND = 'hsl(150 6% 93%)'
const COLOR_DECLARATION = /(^|;)(\s*)(color|background-color|background)(\s*:\s*)([^;}]+)/gi
const CSS_RULE = /([^{}]+)\{([^{}]*)\}/g
const PREFERS_COLOR_SCHEME_BLOCK =
  /(@media[^{}]*prefers-color-scheme[^{}]*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\})/gi
const NON_CONTENT_TAGS = new Set(['style', 'script', 'title'])

const parseDeclarationValue = (value) => {
  const important = value.match(/\s*!important\s*$/i)?.[0] || ''
  return {
    color: value.slice(0, value.length - important.length).trim(),
    important
  }
}

const collectColorDeclarations = (cssText) => {
  const declarations = []
  for (const match of cssText.matchAll(COLOR_DECLARATION)) {
    const parsed = parseDeclarationValue(match[5])
    const property = match[3].toLowerCase()
    if (property === 'background' && !parseCssColor(parsed.color)) continue

    declarations.push({
      important: Boolean(parsed.important),
      index: match.index,
      property: property === 'background' ? 'background-color' : property,
      value: parsed.color
    })
  }
  return declarations
}

const findWinningDeclaration = (declarations, property) =>
  declarations.reduce((winner, declaration) => {
    if (declaration.property !== property) return winner
    if (!winner || Number(declaration.important) > Number(winner.important)) return declaration
    if (declaration.important === winner.important && declaration.index > winner.index) {
      return declaration
    }
    return winner
  }, null)

const normalizeDeclarations = (cssText, inheritedBackground) => {
  const declarations = collectColorDeclarations(cssText)
  const foreground = findWinningDeclaration(declarations, 'color')
  const background = findWinningDeclaration(declarations, 'background-color')
  const replacements = new Map()

  if (foreground && background) {
    const pair = remapColorPairForDarkMode(foreground.value, background.value, inheritedBackground)
    replacements.set(foreground.index, pair.color)
    replacements.set(background.index, pair.backgroundColor)
  }

  declarations.forEach((declaration) => {
    if (replacements.has(declaration.index)) return
    const remapped = remapColorForDarkMode(declaration.value)
    replacements.set(
      declaration.index,
      declaration.property === 'color'
        ? ensureColorContrast(remapped, inheritedBackground)
        : remapped
    )
  })

  const normalizedCss = cssText.replace(
    COLOR_DECLARATION,
    (match, separator, whitespace, property, colon, value, offset) => {
      const parsed = parseDeclarationValue(value)
      const replacement = replacements.get(offset)
      if (!replacement || replacement === parsed.color) return match
      const normalizedProperty =
        property.toLowerCase() === 'background' ? 'background-color' : property
      return `${separator}${whitespace}${normalizedProperty}${colon}${replacement}${parsed.important}`
    }
  )

  return {
    cssText: normalizedCss,
    backgroundColor: background ? replacements.get(background.index) : null,
    color: foreground ? replacements.get(foreground.index) : null
  }
}

const normalizeStyleRuleText = (cssText) =>
  cssText.replace(/\{([^{}]*)\}/g, (rule, declarations) => {
    return `{${normalizeDeclarations(declarations, DARK_MESSAGE_BACKGROUND).cssText}}`
  })

const normalizeStyleBlocks = (fragment) => {
  fragment.querySelectorAll('style').forEach((style) => {
    style.textContent = style.textContent
      .split(PREFERS_COLOR_SCHEME_BLOCK)
      .map((segment, index) => (index % 2 === 1 ? segment : normalizeStyleRuleText(segment)))
      .join('')
  })
}

const splitSelectors = (selectorText) => {
  const selectors = []
  let start = 0
  let depth = 0

  for (let index = 0; index < selectorText.length; index += 1) {
    const character = selectorText[index]
    if (character === '(' || character === '[') depth += 1
    else if (character === ')' || character === ']') depth -= 1
    else if (character === ',' && depth === 0) {
      selectors.push(selectorText.slice(start, index).trim())
      start = index + 1
    }
  }
  selectors.push(selectorText.slice(start).trim())
  return selectors.filter(Boolean)
}

const selectorSpecificity = (selector) => {
  const withoutStrings = selector.replace(/(["'])(?:\\.|(?!\1).)*\1/g, '')
  return [
    (withoutStrings.match(/#[\w-]+/g) || []).length,
    (withoutStrings.match(/\.[\w-]+|\[[^\]]+\]|:(?!:)[\w-]+(?:\([^)]*\))?/g) || []).length,
    (withoutStrings.match(/(^|[\s>+~])(?:[a-z][\w-]*|\*)/gi) || []).length +
      (withoutStrings.match(/::[\w-]+/g) || []).length
  ]
}

const compareSpecificity = (left, right) => {
  for (let index = 0; index < left.length; index += 1) {
    if (left[index] !== right[index]) return left[index] - right[index]
  }
  return 0
}

const collectCssRules = (cssText) => {
  if (typeof CSSStyleSheet !== 'undefined' && CSSStyleSheet.prototype.replaceSync) {
    try {
      const sheet = new CSSStyleSheet()
      sheet.replaceSync(cssText)
      const rules = []
      const walk = (cssRules) => {
        Array.from(cssRules).forEach((rule) => {
          if (rule.selectorText && rule.style) {
            rules.push({ selector: rule.selectorText, style: rule.style })
          } else if (rule.cssRules) {
            const matches =
              !rule.conditionText ||
              typeof window.matchMedia !== 'function' ||
              window.matchMedia(rule.conditionText).matches
            if (matches) walk(rule.cssRules)
          }
        })
      }
      walk(sheet.cssRules)
      return rules
    } catch {
      // Fall through to the fragment parser used by jsdom and older browsers.
    }
  }

  const rules = []
  for (const match of cssText.matchAll(CSS_RULE)) {
    const probe = document.createElement('div')
    probe.style.cssText = match[2]
    rules.push({ selector: match[1].trim(), style: probe.style })
  }
  return rules
}

const querySelectorAll = (fragment, selector, cache) => {
  const normalizedSelector = selector.trim()
  if (cache.has(normalizedSelector)) return cache.get(normalizedSelector)

  let matches
  try {
    matches = Array.from(fragment.querySelectorAll(normalizedSelector))
  } catch {
    matches = []
  }
  cache.set(normalizedSelector, matches)
  return matches
}

const collectStylesheetColors = (fragment, selectorMatches) => {
  const elementStyles = new WeakMap()
  const selectorStyles = new Map()
  let order = 0

  fragment.querySelectorAll('style').forEach((style) => {
    collectCssRules(style.textContent).forEach((rule) => {
      const properties = ['color', 'background-color'].filter((property) =>
        rule.style.getPropertyValue(property)
      )
      if (properties.length === 0) return

      splitSelectors(rule.selector).forEach((selector) => {
        const specificity = selectorSpecificity(selector)
        const styles = selectorStyles.get(selector) || {}
        for (const property of properties) {
          const candidate = {
            value: rule.style.getPropertyValue(property),
            important: rule.style.getPropertyPriority(property) === 'important',
            specificity,
            order
          }
          const current = styles[property]
          if (
            !current ||
            Number(candidate.important) > Number(current.important) ||
            (candidate.important === current.important && candidate.order > current.order)
          ) {
            styles[property] = candidate
          }
        }
        selectorStyles.set(selector, styles)
      })
      order += 1
    })
  })

  selectorStyles.forEach((declarations, selector) => {
    querySelectorAll(fragment, selector, selectorMatches).forEach((element) => {
      const styles = elementStyles.get(element) || {}
      Object.entries(declarations).forEach(([property, candidate]) => {
        const current = styles[property]
        const wins =
          !current ||
          Number(candidate.important) > Number(current.important) ||
          (candidate.important === current.important &&
            (compareSpecificity(candidate.specificity, current.specificity) > 0 ||
              (compareSpecificity(candidate.specificity, current.specificity) === 0 &&
                candidate.order > current.order)))
        if (wins) styles[property] = candidate
      })
      elementStyles.set(element, styles)
    })
  })

  return elementStyles
}

const resolvedBackground = (background, inheritedBackground) =>
  compositeCssColor(background, inheritedBackground) || inheritedBackground

const formatLegacyColor = (color) => {
  const parsed = parseCssColor(color)
  if (!parsed) return color

  const hex = [parsed.r, parsed.g, parsed.b]
    .map((channel) =>
      Math.round(channel * 255)
        .toString(16)
        .padStart(2, '0')
    )
    .join('')
  return `#${hex}`
}

const normalizeElement = (element, inheritedBackground, inheritedForeground, stylesheetColors) => {
  if (NON_CONTENT_TAGS.has(element.tagName.toLowerCase())) {
    return
  }

  let effectiveBackground = inheritedBackground
  let effectiveForeground = inheritedForeground
  const style = element.getAttribute('style')
  const legacyBackground = element.getAttribute('bgcolor')
  const legacyForeground =
    element.tagName.toLowerCase() === 'font' ? element.getAttribute('color') : null
  const stylesheet = stylesheetColors.get(element)
  const stylesheetBackground = stylesheet?.['background-color']
  const inlineBackground = element.style.getPropertyValue('background-color')

  if (legacyBackground && !stylesheetBackground && !inlineBackground) {
    const normalizedBackground = remapColorForDarkMode(legacyBackground)
    if (parseCssColor(normalizedBackground)) {
      const renderedBackground = resolvedBackground(normalizedBackground, inheritedBackground)
      const legacyColor = formatLegacyColor(renderedBackground)
      element.setAttribute('bgcolor', legacyColor)
      effectiveBackground = legacyColor
    }
  }

  if (stylesheetBackground && parseCssColor(stylesheetBackground.value)?.a !== 0) {
    effectiveBackground = resolvedBackground(stylesheetBackground.value, inheritedBackground)
  }
  if (parseCssColor(stylesheet?.color?.value)) {
    effectiveForeground = stylesheet.color.value
  }

  if (style !== null) {
    const normalized = normalizeDeclarations(style, effectiveBackground)
    if (normalized.cssText !== style) element.setAttribute('style', normalized.cssText)
    const inlineBackgroundWins =
      !stylesheetBackground?.important ||
      element.style.getPropertyPriority('background-color') === 'important'
    if (
      inlineBackgroundWins &&
      normalized.backgroundColor &&
      parseCssColor(normalized.backgroundColor)?.a !== 0
    ) {
      effectiveBackground = resolvedBackground(normalized.backgroundColor, inheritedBackground)
    }
    const inlineColorWins =
      !stylesheet?.color?.important || element.style.getPropertyPriority('color') === 'important'
    if (inlineColorWins && normalized.color) effectiveForeground = normalized.color
  }

  if (legacyForeground && !stylesheet?.color && !element.style.color) {
    const normalizedForeground = ensureColorContrast(
      remapColorForDarkMode(legacyForeground),
      effectiveBackground
    )
    if (parseCssColor(normalizedForeground)) {
      const renderedForeground =
        compositeCssColor(normalizedForeground, effectiveBackground) || normalizedForeground
      const legacyColor = formatLegacyColor(renderedForeground)
      element.setAttribute('color', legacyColor)
      effectiveForeground = legacyColor
    }
  }

  const adjustedForeground = ensureColorContrast(effectiveForeground, effectiveBackground)
  if (adjustedForeground !== effectiveForeground) {
    element.style.setProperty(
      'color',
      adjustedForeground,
      stylesheet?.color?.important ? 'important' : ''
    )
    effectiveForeground = adjustedForeground
  }

  Array.from(element.children).forEach((child) =>
    normalizeElement(child, effectiveBackground, effectiveForeground, stylesheetColors)
  )
}

/**
 * Remaps email colors for dark mode, or returns the original HTML unchanged.
 * @param {string} html
 * @param {boolean} darkMode
 * @returns {string}
 */
export const normalizeEmailHtml = (html, darkMode) => {
  if (!darkMode || typeof html !== 'string' || html.length === 0) return html

  try {
    const template = document.createElement('template')
    template.innerHTML = html
    normalizeStyleBlocks(template.content)
    const stylesheetColors = collectStylesheetColors(template.content, new Map())
    Array.from(template.content.children).forEach((element) =>
      normalizeElement(element, DARK_MESSAGE_BACKGROUND, DARK_MESSAGE_FOREGROUND, stylesheetColors)
    )
    return template.innerHTML
  } catch (error) {
    console.warn('Could not normalize email colours for dark mode.', error)
    return html
  }
}
