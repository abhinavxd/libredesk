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
// Matches a top-level `@media (... prefers-color-scheme ...) { ... }` block (one level of rule
// nesting). These blocks are already conditioned on a colour scheme, so their declarations must
// be left byte-for-byte untouched: remapping a `dark` block would re-invert colours that are
// already dark-mode-safe, and remapping a `light` block would corrupt colours that should only
// ever apply when this normalizer's own dark mode is NOT active.
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

const normalizeStyleRuleText = (
  cssText,
  fragment,
  imageBackedElements,
  selectorMatches,
  mixedStyleTargets
) =>
  cssText.replace(CSS_RULE, (rule, selector, declarations) => {
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
  })

const normalizeStyleBlocks = (
  fragment,
  imageBackedElements,
  selectorMatches,
  mixedStyleTargets
) => {
  fragment.querySelectorAll('style').forEach((style) => {
    style.textContent = style.textContent
      .split(PREFERS_COLOR_SCHEME_BLOCK)
      .map((segment, index) =>
        // Odd indices are the captured `prefers-color-scheme` blocks from the split above -
        // leave them untouched. Even indices are the surrounding text, which normalizes as usual.
        index % 2 === 1
          ? segment
          : normalizeStyleRuleText(
              segment,
              fragment,
              imageBackedElements,
              selectorMatches,
              mixedStyleTargets
            )
      )
      .join('')
  })
}

const hasImageBackground = (cssText) => {
  const probe = document.createElement('div')
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
  const stack = Array.from(fragment.children)
    .reverse()
    .map((element) => ({ element, protectedByAncestor: false }))
  while (stack.length > 0) {
    const { element, protectedByAncestor } = stack.pop()
    const isProtected = protectedByAncestor || roots.has(element)
    if (isProtected) protectedElements.add(element)
    Array.from(element.children)
      .reverse()
      .forEach((child) => stack.push({ element: child, protectedByAncestor: isProtected }))
  }
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

const stylesheetDeclarationWins = (declaration, inlineValue, inlinePriority) =>
  Boolean(
    declaration && (!inlineValue || (declaration.important && inlinePriority !== 'important'))
  )

const normalizeElement = (
  element,
  inheritedBackground,
  inheritedForeground,
  stylesheetColors,
  rawStylesheetColors,
  mixedStyleTargets
) => {
  if (NON_CONTENT_TAGS.has(element.tagName.toLowerCase())) {
    return { background: inheritedBackground, foreground: inheritedForeground }
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
    // Composited against the ancestor background, not `effectiveBackground`: a stylesheet
    // background-color always wins the cascade over a `bgcolor` attribute on the same element
    // (bgcolor never stacks under it), so a translucent stylesheet colour must be resolved
    // against what's truly behind the element, not against the bgcolor value it overrides.
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
      // Same reasoning as above: an inline background-color that wins the cascade replaces
      // any bgcolor/stylesheet background on this element rather than stacking on top of it,
      // so it must be composited against the ancestor background, not the overridden value.
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
      element.style.getPropertyValue('background-color'),
      element.style.getPropertyPriority('background-color')
    )
    const foregroundWins = stylesheetDeclarationWins(
      rawForeground,
      element.style.getPropertyValue('color'),
      element.style.getPropertyPriority('color')
    )
    const pair =
      backgroundWins && foregroundWins
        ? remapColorPairForDarkMode(rawForeground.value, rawBackground.value, inheritedBackground)
        : null

    if (backgroundWins) {
      const normalizedBackground =
        pair?.backgroundColor || remapColorForDarkMode(rawBackground.value)
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
      const normalizedForeground =
        pair?.color ||
        ensureColorContrast(remapColorForDarkMode(rawForeground.value), effectiveBackground)
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
      stylesheet?.color?.important ? 'important' : ''
    )
    effectiveForeground = adjustedForeground
  }

  return { background: effectiveBackground, foreground: effectiveForeground }
}

const normalizeElements = (
  fragment,
  imageBackedElements,
  stylesheetColors,
  rawStylesheetColors,
  mixedStyleTargets
) => {
  const stack = Array.from(fragment.children)
    .reverse()
    .map((element) => ({
      element,
      inheritedBackground: DARK_MESSAGE_BACKGROUND,
      inheritedForeground: DARK_MESSAGE_FOREGROUND
    }))

  while (stack.length > 0) {
    const { element, inheritedBackground, inheritedForeground } = stack.pop()
    if (imageBackedElements.has(element)) {
      preserveLegacyBackgroundImage(element)
      // Descend without normalizing: a protected element can itself contain a nested
      // legacy `background`-attribute element (e.g. a per-cell table banner inside an
      // outer image-backed wrapper) that also needs its raw attribute converted to an
      // inline style before sanitization, since `collectImageBackedElements` already
      // marked the whole subtree as protected.
      Array.from(element.children)
        .reverse()
        .forEach((child) =>
          stack.push({ element: child, inheritedBackground, inheritedForeground })
        )
      continue
    }

    const effective = normalizeElement(
      element,
      inheritedBackground,
      inheritedForeground,
      stylesheetColors,
      rawStylesheetColors,
      mixedStyleTargets
    )
    Array.from(element.children)
      .reverse()
      .forEach((child) =>
        stack.push({
          element: child,
          inheritedBackground: effective.background,
          inheritedForeground: effective.foreground
        })
      )
  }
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
    const selectorMatches = new Map()
    const imageBackedElements = collectImageBackedElements(template.content, selectorMatches)
    const rawStylesheetColors = collectStylesheetColors(template.content, selectorMatches)
    const mixedStyleTargets = new WeakSet()
    normalizeStyleBlocks(template.content, imageBackedElements, selectorMatches, mixedStyleTargets)
    const stylesheetColors = collectStylesheetColors(template.content, selectorMatches)
    normalizeElements(
      template.content,
      imageBackedElements,
      stylesheetColors,
      rawStylesheetColors,
      mixedStyleTargets
    )
    return template.innerHTML
  } catch (error) {
    console.warn('Could not normalize email colours for dark mode.', error)
    return html
  }
}
