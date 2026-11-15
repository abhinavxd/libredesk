/** @vitest-environment jsdom */

import { describe, expect, test, vi } from 'vitest'
import {
  ensureColorContrast,
  getContrastRatio,
  parseCssColor,
  remapColorForDarkMode,
  remapColorPairForDarkMode
} from './emailColorNormalizer.js'

describe('remapColorForDarkMode', () => {
  test.each([
    ['black to near-white', '#000000', 'hsl(0 0% 92%)'],
    ['white to near-black', '#FFFFFF', 'hsl(0 0% 8%)'],
    ['saturated red without hue or saturation changes', '#ff0000', 'hsl(0 100% 50%)'],
    ['a named colour', 'navy', 'hsl(240 100% 70.92%)'],
    ['transparent unchanged', 'transparent', 'transparent']
  ])('maps %s', (_name, input, output) => {
    expect(remapColorForDarkMode(input)).toBe(output)
  })

  test('keeps a mid-grey near the middle of the range', () => {
    const output = parseCssColor(remapColorForDarkMode('#808080'))
    expect(output.r).toBeCloseTo(output.g, 5)
    expect(output.g).toBeCloseTo(output.b, 5)
    expect(output.r).toBeCloseTo(0.5, 1)
  })

  test('parses grad hues without also applying the radian conversion', () => {
    expect(parseCssColor('hsl(10grad 100% 50%)')).toEqual(parseCssColor('hsl(9deg 100% 50%)'))
  })

  test('caches named colours without touching the live document', () => {
    const appendChild = vi.spyOn(document.documentElement, 'appendChild')

    expect(parseCssColor('papayawhip')).toBe(parseCssColor('papayawhip'))

    expect(appendChild).not.toHaveBeenCalled()
    appendChild.mockRestore()
  })

  test.each([
    ['CanvasText', '#000'],
    ['Canvas', '#fff'],
    ['Highlight', 'rgb(0, 153, 255)'],
    ['rgb(calc(50 + 50), 0, 0)', 'rgb(100, 0, 0)'],
    ['hsl(calc(60 * 2) 100% 50%)', '#0f0']
  ])('resolves browser-native %s colors without leaving a DOM probe', (input, expected) => {
    const childCount = document.documentElement.childElementCount
    expect(parseCssColor(input)).toEqual(parseCssColor(expected))
    expect(document.documentElement.childElementCount).toBe(childCount)
  })

  test('resolves each browser-native expression only once', () => {
    const appendChild = vi.spyOn(document.documentElement, 'appendChild')
    try {
      const input = 'rgb(calc(20 + 22), 0, 0)'
      expect(parseCssColor(input)).toBe(parseCssColor(input))
      expect(appendChild).toHaveBeenCalledOnce()
    } finally {
      appendChild.mockRestore()
    }
  })

  test('removes the native probe when computed-style resolution fails', () => {
    const childCount = document.documentElement.childElementCount
    const getComputedStyle = vi.spyOn(window, 'getComputedStyle').mockImplementationOnce(() => {
      throw new Error('Computed style unavailable')
    })
    try {
      expect(() => parseCssColor('rgb(calc(21 + 22), 0, 0)')).toThrow('Computed style unavailable')
      expect(document.documentElement.childElementCount).toBe(childCount)
    } finally {
      getComputedStyle.mockRestore()
    }
  })

  test.each([
    'currentColor',
    'inherit',
    'initial',
    'unset',
    'revert',
    'revert-layer',
    'var(--email-color)',
    'rgb(calc(var(--red) + 20) 0 0)',
    'rgb(not a colour)',
    'rgb(1,2,3,4,5)'
  ])('leaves contextual or malformed %s unchanged', (value) => {
    expect(remapColorForDarkMode(value)).toBe(value)
  })

  test('preserves rgba alpha', () => {
    const output = parseCssColor(remapColorForDarkMode('rgba(12, 34, 56, 0.4)'))
    expect(output.a).toBeCloseTo(0.4, 3)
  })

  test('raises paired colour contrast to WCAG AA', () => {
    const output = remapColorPairForDarkMode('#777777', '#888888')
    expect(getContrastRatio(output.color, output.backgroundColor)).toBeGreaterThanOrEqual(4.5)
  })

  test('parses named colours even when DOM operations are unavailable', () => {
    const appendChild = vi.spyOn(document.documentElement, 'appendChild').mockImplementation(() => {
      throw new Error('boom')
    })

    try {
      expect(() => remapColorForDarkMode('deeppink')).not.toThrow()
      expect(appendChild).not.toHaveBeenCalled()
      expect(parseCssColor('deeppink')).toEqual({ r: 1, g: 20 / 255, b: 147 / 255, a: 1 })
    } finally {
      appendChild.mockRestore()
    }
  })
})

describe('ensureColorContrast', () => {
  test('boosts a near-transparent foreground to opaque to reach WCAG AA', () => {
    const output = ensureColorContrast('rgba(128, 128, 128, 0.05)', 'rgb(128, 128, 128)')
    expect(getContrastRatio(output, 'rgb(128, 128, 128)')).toBeGreaterThanOrEqual(4.5)
  })
})
