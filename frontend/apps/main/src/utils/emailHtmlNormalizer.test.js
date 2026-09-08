/** @vitest-environment jsdom */

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { cwd } from 'node:process'
import { describe, expect, test, vi } from 'vitest'
import { allowedCssProperties, sanitize } from 'lettersanitizer'
import { compositeCssColor, getContrastRatio, parseCssColor } from './emailColorNormalizer.js'
import { normalizeEmailHtml } from './emailHtmlNormalizer.js'

const loadFixture = (name) =>
  readFileSync(resolve(cwd(), `apps/main/src/utils/__fixtures__/${name}.html`), 'utf8')

const parseFragment = (html) => {
  const template = document.createElement('template')
  template.innerHTML = html
  return template.content
}

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
    const output = parseFragment(normalizeEmailHtml(loadFixture('outlook-desktop'), true))
    const container = output.querySelector('div')
    const paragraphs = output.querySelectorAll('p')

    expect(
      getContrastRatio(container.style.color, container.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
    expect(
      getContrastRatio(paragraphs[0].style.color, container.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
    const warning = parseCssColor(paragraphs[1].style.color)
    expect(warning.r).toBeGreaterThan(0.9)
    expect(warning.g).toBeLessThan(0.25)
    expect(warning.b).toBeLessThan(0.25)
  })

  test('normalizes Gmail style blocks and quoted content', () => {
    const output = parseFragment(normalizeEmailHtml(loadFixture('gmail-reply'), true))
    const stylesheet = output.querySelector('style').textContent
    const quotedColor = output.querySelector('.gmail_quote div').style.color

    expect(stylesheet).toContain('color: hsl(')
    expect(stylesheet).toContain('background-color: hsl(')
    expect(getContrastRatio(quotedColor, 'hsl(120 2.6% 7.6%)')).toBeGreaterThanOrEqual(4.5)
  })

  test('enforces contrast across separate stylesheet rules', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<style>.surface { background-color: #777777; } .copy { color: #888888; }</style><div class="surface"><span class="copy">Copy</span></div>',
        true
      )
    )
    const background = output
      .querySelector('style')
      .textContent.match(/background-color:\s*([^;}]+)/)[1]
    const foreground = output.querySelector('.copy').style.color

    expect(getContrastRatio(foreground, background)).toBeGreaterThanOrEqual(4.5)
  })

  test('normalizes Apple Mail presentational attributes', () => {
    const output = parseFragment(normalizeEmailHtml(loadFixture('apple-mail'), true))
    const container = output.querySelector('[bgcolor]')
    const font = output.querySelector('font')
    const previousMessage = output.querySelector('div div')

    expect(container.getAttribute('bgcolor')).toMatch(/^#[\da-f]{6}$/)
    expect(font.getAttribute('color')).toMatch(/^#[\da-f]{6}$/)
    expect(
      getContrastRatio(font.getAttribute('color'), container.getAttribute('bgcolor'))
    ).toBeGreaterThanOrEqual(4.5)
    expect(
      getContrastRatio(previousMessage.style.color, previousMessage.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
  })

  test('checks inline text against an element bgcolor', () => {
    const output = parseFragment(
      normalizeEmailHtml('<td bgcolor="#000000" style="color: #ffffff">Copy</td>', true)
    )
    const cell = output.querySelector('td')

    expect(getContrastRatio(cell.style.color, cell.getAttribute('bgcolor'))).toBeGreaterThanOrEqual(
      4.5
    )
  })

  test('checks nested text against the composited background', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<div style="background-color: rgba(255, 255, 255, 0.5)"><span style="color: #777777">Copy</span></div>',
        true
      )
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')
    const background = compositeCssColor(container.style.backgroundColor, 'hsl(120 2.6% 7.6%)')

    expect(getContrastRatio(text.style.color, background)).toBeGreaterThanOrEqual(4.5)
  })

  test('normalizes a color-only background shorthand with its text color', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<div style="background: #ffffff; color: #000000"><span style="color: #000000">Copy</span></div>',
        true
      )
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')

    expect(
      getContrastRatio(container.style.color, container.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
    expect(
      getContrastRatio(text.style.color, container.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
  })

  test('normalizes a color-only background shorthand in a style block', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<style>.surface { background: #ffffff; color: #000000; }</style><div class="surface">Copy</div>',
        true
      )
    )
    const stylesheet = output.querySelector('style').textContent
    const background = stylesheet.match(/background-color:\s*([^;}]+)/)[1]
    const foreground = stylesheet.match(/(?:^|;)\s*color:\s*([^;}]+)/)[1]

    expect(getContrastRatio(foreground, background)).toBeGreaterThanOrEqual(4.5)
  })

  test('uses the important inline background when checking descendant contrast', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<div style="background-color: #ffffff !important; background-color: #000000"><span style="color: #000000">Copy</span></div>',
        true
      )
    )
    const container = output.querySelector('div')
    const text = output.querySelector('span')

    expect(
      getContrastRatio(text.style.color, container.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
  })

  test('leaves legacy attributes untouched when CSS overrides them', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<div bgcolor="#ffffff" style="background-color: #000000"><font color="#000000" style="color: #ffffff">Copy</font></div>',
        true
      )
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
    const output = parseFragment(normalizeEmailHtml(loadFixture('marketing-banner'), true))
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
    const output = parseFragment(
      normalizeEmailHtml(
        '<style>.card { color: #000000; background-color: #ffffff; }</style>' +
          '<div style="background-image: url(https://cdn.example.com/hero.jpg)">' +
          '<div id="banner-card" class="card">Banner</div></div>' +
          '<div id="regular-card" class="card">Regular</div>',
        true
      )
    )
    const stylesheet = output.querySelector('style').textContent
    const banner = output.querySelector('#banner-card')
    const regular = output.querySelector('#regular-card')

    expect(stylesheet).toContain('color: #000000')
    expect(stylesheet).toContain('background-color: #ffffff')
    expect(banner.getAttribute('style')).toBeNull()
    expect(regular.style.backgroundColor).not.toBe('rgb(255, 255, 255)')
    expect(
      getContrastRatio(regular.style.color, regular.style.backgroundColor)
    ).toBeGreaterThanOrEqual(4.5)
  })

  test('keeps legacy table background images through sanitization', () => {
    const normalized = normalizeEmailHtml(
      '<table><tr><td background="https://cdn.example.com/banner.jpg"><font color="#ffffff">Copy</font></td></tr></table>',
      true
    )
    const sanitized = sanitize(normalized, null, {
      noWrapper: true,
      allowedSchemas: ['cid', 'https', 'http', 'mailto'],
      allowedCssProperties
    })
    const output = parseFragment(sanitized)

    expect(output.querySelector('td').style.backgroundImage).toContain('banner.jpg')
  })

  test('handles deeply nested reply chains without recursive traversal', () => {
    let input = '<span style="color: #000000">Latest reply</span>'
    for (let depth = 0; depth < 1500; depth += 1) {
      input = `<blockquote style="color: #111111; background-color: #ffffff">${input}</blockquote>`
    }

    expect(() => normalizeEmailHtml(input, true)).not.toThrow()
  })

  test('keeps a nested table-cell background image through sanitization', () => {
    const normalized = normalizeEmailHtml(
      '<div style="background-image: url(https://cdn.example.com/hero.jpg)">' +
        '<table><tr><td background="https://cdn.example.com/cell.jpg">' +
        '<font color="#ffffff">Copy</font></td></tr></table></div>',
      true
    )
    const sanitized = sanitize(normalized, null, {
      noWrapper: true,
      allowedSchemas: ['cid', 'https', 'http', 'mailto'],
      allowedCssProperties
    })
    const output = parseFragment(sanitized)

    expect(output.querySelector('td').style.backgroundImage).toContain('cell.jpg')
  })

  test('does not protect a table cell with a blank legacy background attribute', () => {
    const output = parseFragment(
      normalizeEmailHtml(
        '<table><tr><td background="  " style="color: #000000">Copy</td></tr></table>',
        true
      )
    )

    expect(output.querySelector('td').style.color).not.toBe('rgb(0, 0, 0)')
  })

  test('leaves prefers-color-scheme media blocks untouched', () => {
    const input =
      '<style>@media (prefers-color-scheme: dark) { .a { color: #eeeeee; background-color: #111111; } } .b { color: #222222; }</style><div class="a">Dark</div><div class="b">Light</div>'
    const output = parseFragment(normalizeEmailHtml(input, true))
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
    const output = parseFragment(
      normalizeEmailHtml('<style>.a { color: #222222; }</style><div class="a">Text</div>', true)
    )
    const styleEl = output.querySelector('style')

    expect(styleEl.getAttribute('style')).toBeNull()
  })
})
