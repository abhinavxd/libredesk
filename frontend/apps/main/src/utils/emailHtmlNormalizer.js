import { serializeHex } from 'culori/fn'
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
const BACKGROUND_IMAGE_DECLARATION = /(?:^|;)\s*background-image\s*:\s*([^;}]+)/gi
const BACKGROUND_SHORTHAND_DECLARATION = /(?:^|;)\s*background\s*:\s*([^;}]+)/gi
// Sender-provided theme rules must not be inverted again.
const PREFERS_COLOR_SCHEME_BLOCK =
  /(@media[^{}]*prefers-color-scheme[^{}]*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\})/gi
const NON_CONTENT_TAGS = new Set(['style', 'script', 'title'])
let imageBackgroundProbe

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
      order: match.index,
      property: property === 'background' ? 'background-color' : property,
      value: parsed.color
    })
  }
  return declarations
}

const findWinningDeclaration = (declarations, property) =>
  declarations.reduce(
    (winner, declaration) =>
      declaration.property === property && declarationWins(declaration, winner)
        ? declaration
        : winner,
    null
  )

const remapBackgroundForDarkMode = (value) => {
  const parsed = parseCssColor(value)
  if (parsed?.a === 1) {
    const lightest = Math.max(parsed.r, parsed.g, parsed.b)
    const darkest = Math.min(parsed.r, parsed.g, parsed.b)
    if (darkest >= 0.9 && lightest - darkest <= 0.04) return 'transparent'
  }
  return remapColorForDarkMode(value)
}

const normalizeDeclarations = (cssText, inheritedBackground) => {
  const declarations = collectColorDeclarations(cssText)
  const foreground = findWinningDeclaration(declarations, 'color')
  const background = findWinningDeclaration(declarations, 'background-color')
  const backgroundColor = background && remapBackgroundForDarkMode(background.value)
  const pair =
    foreground &&
    background &&
    remapColorPairForDarkMode(foreground.value, background.value, inheritedBackground)
  const replacements = new Map(
    declarations.map((declaration) => [
      declaration.order,
      declaration === foreground && pair
        ? ensureColorContrast(pair.color, resolvedBackground(backgroundColor, inheritedBackground))
        : declaration.property === 'color'
          ? ensureColorContrast(remapColorForDarkMode(declaration.value), inheritedBackground)
          : remapBackgroundForDarkMode(declaration.value)
    ])
  )

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
    backgroundColor: background ? replacements.get(background.order) : null,
    color: foreground ? replacements.get(foreground.order) : null
  }
}

const normalizeStyleBlocks = ({
  fragment,
  imageBackedElements,
  selectorMatches,
  mixedStyleTargets
}) => {
  const normalizeRule = (rule, selector, declarations) => {
    const targets = querySelectorAll(fragment, selector, selectorMatches)
    if (hasImageBackground(declarations)) return rule

    const protectedTargets = targets.filter((element) => imageBackedElements.has(element))
    if (protectedTargets.length > 0) {
      targets
        .filter((element) => !imageBackedElements.has(element))
        .forEach((element) => mixedStyleTargets.add(element))
      return rule
    }
    return `${selector}{${normalizeDeclarations(declarations, DARK_MESSAGE_BACKGROUND).cssText}}`
  }
  fragment.querySelectorAll('style').forEach((style) => {
    style.textContent = style.textContent
      .split(PREFERS_COLOR_SCHEME_BLOCK)
      .map((segment, index) =>
        index % 2 === 1 ? segment : segment.replace(CSS_RULE, normalizeRule)
      )
      .join('')
  })
}

// Reply chains can be too deep for recursive traversal.
const walkElements = (fragment, initialContext, visit) => {
  const stack = Array.from(fragment.children)
    .reverse()
    .map((element) => [element, initialContext])
  while (stack.length) {
    const [element, inherited] = stack.pop()
    const context = visit(element, inherited)
    Array.from(element.children)
      .reverse()
      .forEach((child) => stack.push([child, context]))
  }
}

const hasImageBackground = (cssText) => {
  const probe = (imageBackgroundProbe ??= document.createElement('div'))
  probe.style.cssText = cssText
  if (probe.style.backgroundImage && probe.style.backgroundImage !== 'none') return true

  for (const match of cssText.matchAll(BACKGROUND_IMAGE_DECLARATION)) {
    if (
      match[1]
        .replace(/\s*!important\s*$/i, '')
        .trim()
        .toLowerCase() !== 'none'
    )
      return true
  }
  for (const match of cssText.matchAll(BACKGROUND_SHORTHAND_DECLARATION)) {
    if (/(?:url|gradient|image-set|cross-fade|var)\s*\(/i.test(match[1])) return true
  }
  return false
}

const collectImageBackedElements = (fragment, selectorMatches) => {
  const roots = new WeakSet()

  fragment.querySelectorAll('[style], td[background], th[background]').forEach((element) => {
    const hasLegacyBackground =
      ['td', 'th'].includes(element.tagName.toLowerCase()) &&
      Boolean(element.getAttribute('background')?.trim())
    const hasInlineBackground = hasImageBackground(element.getAttribute('style') || '')
    if (hasLegacyBackground || hasInlineBackground) {
      roots.add(element)
    }
  })

  fragment.querySelectorAll('style').forEach((style) => {
    for (const match of style.textContent.matchAll(CSS_RULE)) {
      if (!hasImageBackground(match[2])) continue
      querySelectorAll(fragment, match[1], selectorMatches).forEach((element) => roots.add(element))
    }
  })

  const protectedElements = new WeakSet()
  walkElements(fragment, false, (element, protectedByAncestor) => {
    const isProtected = protectedByAncestor || roots.has(element)
    if (isProtected) protectedElements.add(element)
    return isProtected
  })
  return protectedElements
}

const preserveLegacyBackgroundImage = (element) => {
  const background = element.getAttribute('background')
  if (!background?.trim() || element.style.backgroundImage) return

  const escaped = background
    .replace(/\\/g, '\\\\')
    .replace(/"/g, '\\"')
    .replace(/[\n\r\f]/g, '')
  element.style.setProperty('background-image', `url("${escaped}")`)
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

const declarationWins = (candidate, current) =>
  !current ||
  (Number(candidate.important) - Number(current.important) ||
    compareSpecificity(candidate.specificity ?? [], current.specificity ?? []) ||
    candidate.order - current.order) > 0

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
        const declarations = selectorStyles.get(selector) || {}
        for (const property of properties) {
          const candidate = {
            value: rule.style.getPropertyValue(property),
            important: rule.style.getPropertyPriority(property) === 'important',
            specificity,
            order
          }
          if (declarationWins(candidate, declarations[property])) declarations[property] = candidate
        }
        selectorStyles.set(selector, declarations)
      })
      order += 1
    })
  })

  selectorStyles.forEach((declarations, selector) => {
    querySelectorAll(fragment, selector, selectorMatches).forEach((element) => {
      const styles = elementStyles.get(element) || {}
      for (const [property, candidate] of Object.entries(declarations)) {
        if (declarationWins(candidate, styles[property])) styles[property] = candidate
      }
      elementStyles.set(element, styles)
    })
  })

  return elementStyles
}

const resolvedBackground = (background, inheritedBackground) =>
  compositeCssColor(background, inheritedBackground) || inheritedBackground

const formatLegacyColor = (color) => serializeHex(parseCssColor(color) ?? undefined) || color

const stylesheetDeclarationWins = (declaration, style, property) =>
  Boolean(
    declaration &&
    (!style.getPropertyValue(property) ||
      (declaration.important && style.getPropertyPriority(property) !== 'important'))
  )

const normalizeElement = (element, inherited, context) => {
  const { background: inheritedBackground, foreground: inheritedForeground } = inherited
  const { stylesheetColors, rawStylesheetColors, mixedStyleTargets } = context
  if (NON_CONTENT_TAGS.has(element.tagName.toLowerCase())) {
    return inherited
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
    const normalizedBackground = remapBackgroundForDarkMode(legacyBackground)
    if (normalizedBackground === 'transparent') {
      element.removeAttribute('bgcolor')
    } else if (parseCssColor(normalizedBackground)) {
      const renderedBackground = resolvedBackground(normalizedBackground, inheritedBackground)
      const legacyColor = formatLegacyColor(renderedBackground)
      element.setAttribute('bgcolor', legacyColor)
      effectiveBackground = legacyColor
    }
  }

  if (stylesheetBackground && parseCssColor(stylesheetBackground.value)?.a !== 0) {
    // CSS replaces bgcolor; translucent colors composite over the ancestor, not the overridden fill.
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

  if (mixedStyleTargets.has(element)) {
    const rawStylesheet = rawStylesheetColors.get(element)
    const rawBackground = rawStylesheet?.['background-color']
    const rawForeground = rawStylesheet?.color
    const backgroundWins = stylesheetDeclarationWins(
      rawBackground,
      element.style,
      'background-color'
    )
    const foregroundWins = stylesheetDeclarationWins(rawForeground, element.style, 'color')
    const pair =
      backgroundWins && foregroundWins
        ? remapColorPairForDarkMode(rawForeground.value, rawBackground.value, inheritedBackground)
        : null

    if (backgroundWins) {
      const normalizedBackground = remapBackgroundForDarkMode(rawBackground.value)
      element.style.setProperty(
        'background-color',
        normalizedBackground,
        rawBackground.important ? 'important' : ''
      )
      effectiveBackground =
        parseCssColor(normalizedBackground)?.a === 0
          ? inheritedBackground
          : resolvedBackground(normalizedBackground, inheritedBackground)
    }

    if (foregroundWins) {
      const normalizedForeground = ensureColorContrast(
        pair?.color || remapColorForDarkMode(rawForeground.value),
        effectiveBackground
      )
      element.style.setProperty(
        'color',
        normalizedForeground,
        rawForeground.important ? 'important' : ''
      )
      effectiveForeground = normalizedForeground
    }
  }

  const adjustedForeground = ensureColorContrast(effectiveForeground, effectiveBackground)
  if (adjustedForeground !== effectiveForeground) {
    element.style.setProperty(
      'color',
      adjustedForeground,
      element.style.getPropertyPriority('color') ||
        (stylesheet?.color?.important ? 'important' : '')
    )
    effectiveForeground = adjustedForeground
  }

  return { background: effectiveBackground, foreground: effectiveForeground }
}

export const normalizeEmailHtml = (html, darkMode) => {
  if (!darkMode || typeof html !== 'string' || html.length === 0) return html

  try {
    const template = document.createElement('template')
    template.innerHTML = html
    const context = {
      fragment: template.content,
      selectorMatches: new Map(),
      mixedStyleTargets: new WeakSet()
    }
    const { fragment, selectorMatches } = context
    context.imageBackedElements = collectImageBackedElements(fragment, selectorMatches)
    context.rawStylesheetColors = collectStylesheetColors(fragment, selectorMatches)
    normalizeStyleBlocks(context)
    context.stylesheetColors = collectStylesheetColors(fragment, selectorMatches)
    walkElements(
      fragment,
      {
        background: DARK_MESSAGE_BACKGROUND,
        foreground: DARK_MESSAGE_FOREGROUND
      },
      (element, inherited) => {
        if (!context.imageBackedElements.has(element))
          return normalizeElement(element, inherited, context)
        // Protected subtrees still need nested legacy banner URLs preserved before sanitization.
        preserveLegacyBackgroundImage(element)
        return inherited
      }
    )
    return template.innerHTML
  } catch (error) {
    console.warn('Could not normalize email colours for dark mode.', error)
    return html
  }
}
