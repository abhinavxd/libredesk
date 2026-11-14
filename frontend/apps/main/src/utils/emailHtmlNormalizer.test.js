/** @vitest-environment jsdom */

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test, vi } from 'vitest'
import { allowedCssProperties, sanitize } from 'lettersanitizer'
import { compositeCssColor, getContrastRatio, parseCssColor } from './emailColorNormalizer.js'
import { normalizeEmailHtml } from './emailHtmlNormalizer.js'

const loadFixture = (name) =>
  readFileSync(resolve(import.meta.dirname, `__fixtures__/${name}.html`), 'utf8')

const parseFragment = (html) => {
  const template = document.createElement('template')
  template.innerHTML = html
  return template.content
}

const normalize = (html) => parseFragment(normalizeEmailHtml(html, true))
const expectContrast = (foreground, background = 'hsl(120 2.6% 7.6%)') =>
  expect(getContrastRatio(foreground, background)).toBeGreaterThanOrEqual(4.5)

describe('normalizeEmailHtml', () => {
  test.each([
    'outlook-desktop',
    'gmail-reply',
    'apple-mail',
    'marketing-banner',
    'plain-text-wrapped'
  ])('returns the %s fixture byte-for-byte in light mode', (fixture) => {
    const input = loadFixture(fixture)
    expect(normalizeEmailHtml(input, false)).toBe(input)
  })

  test('normalizes Outlook desktop inline colours', () => {
    const output = normalize(loadFixture('outlook-desktop'))
    const container = output.querySelector('div')
    const paragraphs = output.querySelectorAll('p')

    expectContrast(container.style.color, container.style.backgroundColor)
    expectContrast(paragraphs[0].style.color, container.style.backgroundColor)
    const warning = parseCssColor(paragraphs[1].style.color)
    expect(warning.r).toBeGreaterThan(0.9)
    expect(warning.g).toBeLessThan(0.25)
    expect(warning.b).toBeLessThan(0.25)
  })

  test.each(['CanvasText', 'rgb(calc(10 + 10), 0, 0)', 'hsl(calc(60 * 2) 100% 10%)'])(
    'normalizes browser-native %s text against the conversation surface',
    (color) => {
      const output = normalize(`<div style="color:${color};background-color:#fff">Copy</div>`)
      const element = output.querySelector('div')
      expect(element.style.backgroundColor).toBe('transparent')
      expectContrast(element.style.color)
    }
  )

  test('normalizes Gmail style blocks and quoted content', () => {
    const output = normalize(loadFixture('gmail-reply'))
    const stylesheet = output.querySelector('style').textContent
    const quotedColor = output.querySelector('.gmail_quote div').style.color

    expect(stylesheet).toContain('color: hsl(')
    expect(stylesheet).toContain('background-color: transparent')
    expectContrast(quotedColor)
  })

  test('enforces contrast across separate stylesheet rules', () => {
    const output = normalize(
      '<style>.surface { background-color: #777777; } .copy { color: #888888; }</style><div class="surface"><span class="copy">Copy</span></div>'
    )
    const background = output
      .querySelector('style')
      .textContent.match(/background-color:\s*([^;}]+)/)[1]
    const foreground = output.querySelector('.copy').style.color

    expectContrast(foreground, background)
  })

  test('aggregates repeated selectors before updating matching elements', () => {
    const rules = Array.from(
      { length: 40 },
      (_, index) => `.copy { color: rgb(${index}, 0, 0) }`
    ).join('')
    const set = vi.spyOn(WeakMap.prototype, 'set')
    try {
      normalize(`<style>${rules}</style>${'<p class="copy">Copy</p>'.repeat(20)}`)
      const updates = set.mock.calls.filter(
        ([element, styles]) => element?.tagName === 'P' && styles?.color?.specificity
      )
      expect(updates).toHaveLength(40)
    } finally {
      set.mockRestore()
    }
  })

  test.each([
    ['importance', '.copy{color:#000!important}.copy{color:#f00}', '.copy{color:#000!important}'],
    ['source order', '.copy{color:#000}.copy{color:#f00}', '.copy{color:#f00}'],
    ['specificity', 'div.copy{color:#000}.copy{color:#f00}', 'div.copy{color:#000}'],
    ['equal specificity', '.copy{color:#000}.alt{color:#f00}', '.alt{color:#f00}']
  ])('preserves %s when aggregating selectors', (_name, rules, winner) => {
    const color = (css) =>
      normalize(
        `<style>${css}</style><div style="background-image:url(hero.png)">` +
          '<div class="copy alt">Banner</div></div><div class="copy alt" data-test="regular">Copy</div>'
      ).querySelector('[data-test="regular"]').style.color
    expect(color(rules)).toBe(color(winner))
    expect(color(rules)).not.toBe('')
  })

  test('normalizes Apple Mail presentational attributes', () => {
    const output = normalize(loadFixture('apple-mail'))
    const container = output.querySelector('div')
    const font = output.querySelector('font')
    const previousMessage = output.querySelector('div div')

    expect(container.hasAttribute('bgcolor')).toBe(false)
    expect(font.getAttribute('color')).toMatch(/^#[\da-f]{6}$/)
    expectContrast(font.getAttribute('color'))
    expectContrast(previousMessage.style.color, previousMessage.style.backgroundColor)
  })

  test('checks inline text against an element bgcolor', () => {
    const output = normalize('<td bgcolor="#000000" style="color: #ffffff">Copy</td>')
    const cell = output.querySelector('td')

    expectContrast(cell.style.color, cell.getAttribute('bgcolor'))
  })

  test('checks nested text against the composited background', () => {
    const output = normalize(
      '<div style="background-color: rgba(255, 255, 255, 0.5)"><span style="color: #777777">Copy</span></div>'
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')
    const background = compositeCssColor(container.style.backgroundColor, 'hsl(120 2.6% 7.6%)')

    expectContrast(text.style.color, background)
  })

  test('normalizes a color-only background shorthand with its text color', () => {
    const output = normalize(
      '<div style="background: #ffffff; color: #000000"><span style="color: #000000">Copy</span></div>'
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')

    expectContrast(container.style.color, container.style.backgroundColor)
    expectContrast(text.style.color, container.style.backgroundColor)
  })

  test('normalizes a color-only background shorthand in a style block', () => {
    const output = normalize(
      '<style>.surface { background: #ffffff; color: #000000; }</style><div class="surface">Copy</div>'
    )
    const stylesheet = output.querySelector('style').textContent
    const background = stylesheet.match(/background-color:\s*([^;}]+)/)[1]
    const foreground = stylesheet.match(/(?:^|;)\s*color:\s*([^;}]+)/)[1]

    expectContrast(foreground, background)
  })

  test.each(['#fff', '#f5f5f5', '#f2f2f7', 'rgb(255, 255, 255)'])(
    'lets the conversation surface show through a plain %s email canvas',
    (background) => {
      const input = `<div style="background: ${background}; color: #000"><p>Copy</p></div>`
      const output = normalize(input)
      const container = output.querySelector('div')

      expect(container.style.backgroundColor).toBe('transparent')
      expectContrast(container.style.color)
      expect(normalizeEmailHtml(input, false)).toBe(input)
    }
  )

  test('retains colored sections while clearing nested neutral canvases', () => {
    const output = normalize(
      '<div style="background-color:#edf3ff"><div style="background-color:#fff;color:#000">Copy</div></div>'
    )
    const section = output.querySelector('div')
    const copy = section.querySelector('div')

    expect(parseCssColor(section.style.backgroundColor).a).toBe(1)
    expect(copy.style.backgroundColor).toBe('transparent')
    expectContrast(copy.style.color, section.style.backgroundColor)
  })

  test('uses the important inline background when checking descendant contrast', () => {
    const output = normalize(
      '<div style="background-color: #ffffff !important; background-color: #000000"><span style="color: #000000">Copy</span></div>'
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')

    expectContrast(text.style.color, container.style.backgroundColor)
  })

  test('leaves legacy attributes untouched when CSS overrides them', () => {
    const output = normalize(
      '<div bgcolor="#ffffff" style="background-color: #000000"><font color="#000000" style="color: #ffffff">Copy</font></div>'
    )

    expect(output.querySelector('div').getAttribute('bgcolor')).toBe('#ffffff')
    expect(output.querySelector('font').getAttribute('color')).toBe('#000000')
  })

  test('normalizes auto-wrapped plain-text HTML without adding a wrapper', () => {
    const input = loadFixture('plain-text-wrapped')
    const output = normalizeEmailHtml(input, true)

    expect(output.trimStart().startsWith('<div')).toBe(true)
    expect(output).toContain('Line one<br>Line two<br>Line three')
    expect(output).not.toContain('<html')
    expect(output).not.toContain('<body')
  })

  test('falls back to the original HTML when normalization fails', () => {
    const input = '<p style="color: #000000">Original copy</p>'
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const createElement = vi.spyOn(document, 'createElement').mockImplementationOnce(() => {
      throw new Error('DOM parser unavailable')
    })

    try {
      expect(normalizeEmailHtml(input, true)).toBe(input)
      expect(warning).toHaveBeenCalledOnce()
    } finally {
      createElement.mockRestore()
      warning.mockRestore()
    }
  })

  test('preserves colours inside image-backed marketing banners', () => {
    const output = normalize(loadFixture('marketing-banner'))
    const gradientBanner = output.querySelector('[style*="background-image"]')
    const shorthandBanner = output.querySelector('[style*="texture.png"]')
    const stylesheetBanner = output.querySelector('.hero')
    const tableCell = output.querySelector('td[background]')
    const noImage = output.querySelector('[style*="background-image: none"]')
    const footer = output.querySelector('p')

    expect(gradientBanner.style.color).toBe('rgb(255, 255, 255)')
    expect(gradientBanner.querySelector('span').style.color).toBe('rgb(255, 255, 255)')
    expect(shorthandBanner.getAttribute('style')).toContain('color: #ffffff')
    expect(shorthandBanner.querySelector('span').getAttribute('style')).toContain('color: #ffffff')
    expect(stylesheetBanner.querySelector('span').className).toBe('hero-copy')
    expect(output.querySelector('style').textContent).toContain('.hero-copy { color: #ffffff; }')
    expect(tableCell.getAttribute('bgcolor')).toBe('#ffffff')
    expect(tableCell.querySelector('font').getAttribute('color')).toBe('#ffffff')
    expect(tableCell.style.backgroundImage).toContain('banner.jpg')
    expect(noImage.style.color).not.toBe('rgb(0, 0, 0)')
    expect(footer.style.color).not.toBe('rgb(0, 0, 0)')
  })

  test('normalizes ordinary targets of a rule shared with an image-backed banner', () => {
    const output = normalize(
      '<style>.card { color: #000000; background-color: #ffffff; }</style>' +
        '<div style="background-image: url(https://cdn.example.com/hero.jpg)">' +
        '<div id="banner-card" class="card">Banner</div></div>' +
        '<div id="regular-card" class="card">Regular</div>'
    )
    const stylesheet = output.querySelector('style').textContent
    const banner = output.querySelector('#banner-card')
    const regular = output.querySelector('#regular-card')

    expect(stylesheet).toContain('color: #000000')
    expect(stylesheet).toContain('background-color: #ffffff')
    expect(banner.getAttribute('style')).toBeNull()
    expect(regular.style.backgroundColor).not.toBe('rgb(255, 255, 255)')
    expectContrast(regular.style.color, regular.style.backgroundColor)
  })

  test('reuses the image probe without leaking declarations between elements or calls', () => {
    const createElement = vi.spyOn(document, 'createElement')
    try {
      const output = normalize(
        '<div style="background-image:url(hero.png);color:#010203">Banner</div>' +
          '<p style="color:#000">Copy</p><p style="color:#000">More copy</p>'
      )
      expect(output.querySelector('div').style.color).toBe('rgb(1, 2, 3)')
      output.querySelectorAll('p').forEach((element) => expectContrast(element.style.color))
      expectContrast(normalize('<p style="color:#000">Next email</p>').querySelector('p').style.color)
      expect(createElement.mock.calls.filter(([tag]) => tag === 'div').length).toBeLessThanOrEqual(1)
    } finally {
      createElement.mockRestore()
    }
  })

  test('preserves inline color priority when a mixed selector changes the backdrop', () => {
    const output = normalize(
      '<style>.card { background-color: #fff; }</style>' +
        '<div style="background-image: url(hero.png)"><div class="card">Banner</div></div>' +
        '<div id="regular" class="card" style="color: #000 !important">Copy</div>'
    )
    const regular = output.querySelector('#regular')

    expect(regular.style.backgroundColor).toBe('transparent')
    expect(regular.style.getPropertyPriority('color')).toBe('important')
    expectContrast(regular.style.color)
  })

  test.each([
    [
      'legacy table',
      '<table><tr><td background="https://cdn.example.com/banner.jpg"><font color="#ffffff">Copy</font></td></tr></table>',
      'banner.jpg'
    ],
    [
      'nested table-cell',
      '<div style="background-image: url(https://cdn.example.com/hero.jpg)"><table><tr><td background="https://cdn.example.com/cell.jpg"><font color="#ffffff">Copy</font></td></tr></table></div>',
      'cell.jpg'
    ]
  ])('keeps %s background images through sanitization', (_name, html, image) => {
    const normalized = normalizeEmailHtml(html, true)
    const sanitized = sanitize(normalized, null, {
      noWrapper: true,
      allowedSchemas: ['cid', 'https', 'http', 'mailto'],
      allowedCssProperties
    })
    const output = parseFragment(sanitized)

    expect(output.querySelector('td').style.backgroundImage).toContain(image)
  })

  test('handles deeply nested reply chains without recursive traversal', () => {
    let input = '<span style="color: #000000">Latest reply</span>'
    for (let depth = 0; depth < 1500; depth += 1) {
      input = `<blockquote style="color: #111111; background-color: #ffffff">${input}</blockquote>`
    }

    const warning = vi.spyOn(console, 'warn')
    try {
      const output = normalizeEmailHtml(input, true)
      expect(warning).not.toHaveBeenCalled()
      expect(output).not.toBe(input)
      const reply = parseFragment(output).querySelector('span')
      expect(reply.textContent).toBe('Latest reply')
      expect(reply.style.color).not.toBe('rgb(0, 0, 0)')
      expectContrast(reply.style.color)
    } finally {
      warning.mockRestore()
    }
  })

  test('does not protect a table cell with a blank legacy background attribute', () => {
    const output = normalize(
      '<table><tr><td background="  " style="color: #000000">Copy</td></tr></table>'
    )

    expect(output.querySelector('td').style.color).not.toBe('rgb(0, 0, 0)')
  })

  test('leaves prefers-color-scheme media blocks untouched', () => {
    const input =
      '<style>@media (prefers-color-scheme: dark) { .a { color: #eeeeee; background-color: #111111; } } .b { color: #222222; }</style><div class="a">Dark</div><div class="b">Light</div>'
    const output = normalize(input)
    const stylesheet = output.querySelector('style').textContent

    expect(stylesheet).toContain('color: #eeeeee')
    expect(stylesheet).toContain('background-color: #111111')
    expect(stylesheet).toMatch(/\.b\s*\{\s*color:\s*hsl\(/)
  })

  test('ignores an overridden bgcolor attribute when resolving a translucent inline background', () => {
    const spanColor = (html) => parseFragment(html).querySelector('span').style.color

    const withBgcolorAttr = normalizeEmailHtml(
      '<div bgcolor="rgba(0, 0, 0, 0.5)" style="background-color: rgba(255, 255, 255, 0.3)"><span style="color: #777777">Copy</span></div>',
      true
    )
    const withoutBgcolorAttr = normalizeEmailHtml(
      '<div style="background-color: rgba(255, 255, 255, 0.3)"><span style="color: #777777">Copy</span></div>',
      true
    )

    expect(spanColor(withBgcolorAttr)).toBe(spanColor(withoutBgcolorAttr))
  })

  test('does not add a style attribute to non-content elements while walking the tree', () => {
    const output = normalize('<style>.a { color: #222222; }</style><div class="a">Text</div>')
    const styleEl = output.querySelector('style')

    expect(styleEl.getAttribute('style')).toBeNull()
  })
})
