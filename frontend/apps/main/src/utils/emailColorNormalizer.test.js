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
  test('maps black to near-white', () => {
    expect(remapColorForDarkMode('#000000')).toBe('hsl(0 0% 92%)')
  })

  test('maps white to near-black', () => {
    expect(remapColorForDarkMode('#FFFFFF')).toBe('hsl(0 0% 8%)')
  })

  test('keeps a mid-grey near the middle of the range', () => {
    const output = parseCssColor(remapColorForDarkMode('#808080'))
    expect(output.r).toBeCloseTo(output.g, 5)
    expect(output.g).toBeCloseTo(output.b, 5)
    expect(output.r).toBeCloseTo(0.5, 1)
  })

  test('preserves the hue and saturation of saturated red', () => {
    expect(remapColorForDarkMode('#ff0000')).toBe('hsl(0 100% 50%)')
  })

  test('parses grad hues without also applying the radian conversion', () => {
    expect(parseCssColor('hsl(10grad 100% 50%)')).toEqual(parseCssColor('hsl(9deg 100% 50%)'))
  })

  test('leaves transparent unchanged', () => {
    expect(remapColorForDarkMode('transparent')).toBe('transparent')
  })

  test('remaps a named colour', () => {
    expect(remapColorForDarkMode('navy')).toBe('hsl(240 100% 70.92%)')
  })

  test('resolves each named colour against the live document once', () => {
    const appendChild = vi.spyOn(document.documentElement, 'appendChild')

    remapColorForDarkMode('papayawhip')
    remapColorForDarkMode('papayawhip')

    expect(appendChild).toHaveBeenCalledTimes(1)
    appendChild.mockRestore()
  })

  test.each(['currentColor', 'inherit'])('leaves %s unchanged', (value) => {
    expect(remapColorForDarkMode(value)).toBe(value)
  })

  test('preserves rgba alpha', () => {
    const output = parseCssColor(remapColorForDarkMode('rgba(12, 34, 56, 0.4)'))
    expect(output.a).toBeCloseTo(0.4, 3)
  })

  test('leaves malformed colours unchanged', () => {
    expect(remapColorForDarkMode('rgb(not a colour)')).toBe('rgb(not a colour)')
  })

  test('leaves rgb() values with too many components unchanged', () => {
    expect(remapColorForDarkMode('rgb(1,2,3,4,5)')).toBe('rgb(1,2,3,4,5)')
  })

  test('raises paired colour contrast to WCAG AA', () => {
    const output = remapColorPairForDarkMode('#777777', '#888888')
    expect(getContrastRatio(output.color, output.backgroundColor)).toBeGreaterThanOrEqual(4.5)
  })

  test('recovers if resolving a named colour throws', () => {
    const appendChild = vi.spyOn(document.documentElement, 'appendChild').mockImplementation(() => {
      throw new Error('boom')
    })

    try {
      expect(() => remapColorForDarkMode('deeppink')).not.toThrow()
      expect(appendChild).toHaveBeenCalled()
      expect(remapColorForDarkMode('deeppink')).toBe('deeppink')
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
