// @vitest-environment jsdom
import { describe, expect, test } from 'vitest'
import { needsLightCanvas } from './email-colors'

describe('needsLightCanvas', () => {
  test('renders plain emails in the theme', () => {
    expect(
      needsLightCanvas('<p>Hello,</p><p>my invoice is missing.</p><blockquote>quoted</blockquote>')
    ).toBe(false)
    expect(needsLightCanvas('<table border="1"><tr><td>Widget</td></tr></table>')).toBe(false)
    expect(needsLightCanvas('')).toBe(false)
  })

  test('keeps the canvas for emails that set their own colors', () => {
    expect(
      needsLightCanvas(
        '<table style="background:#ffffff"><tr><td style="color:#000000">News</td></tr></table>'
      )
    ).toBe(true)
    expect(needsLightCanvas('<p><font color="#000">black font tag</font></p>')).toBe(true)
    expect(needsLightCanvas('<style>p { color: #000 }</style><p>styled</p>')).toBe(true)
    expect(needsLightCanvas('<table bgcolor="#FFFF00"><tr><td>highlight</td></tr></table>')).toBe(
      true
    )
    expect(needsLightCanvas('<p style="color:rgb(0, 0, 0)">rgb</p>')).toBe(true)
  })

  test('keeps the canvas when a color cannot be read', () => {
    expect(needsLightCanvas('<p style="color: var(--x)">custom</p>')).toBe(true)
    expect(needsLightCanvas('<div style="background: var(--x)"><p>custom</p></div>')).toBe(true)
  })

  test('ignores values that do not set a color', () => {
    expect(needsLightCanvas('<p style="color: inherit; background: transparent">plain</p>')).toBe(
      false
    )
  })

  test('renders emails that bring their own dark background without the canvas', () => {
    expect(
      needsLightCanvas(
        '<div style="background:#1e1e1e;color:#e8e8e8"><h2 style="color:#fff">Build failed</h2><pre style="background:#000;color:#0f0">FAIL</pre></div>'
      )
    ).toBe(false)
    expect(
      needsLightCanvas(
        '<table bgcolor="#111"><tr><td><font color="#fff">dark</font></td></tr></table>'
      )
    ).toBe(false)
    expect(needsLightCanvas('<div style="background-color:#0d1117">dark</div><p>footer</p>')).toBe(
      false
    )
  })

  test('keeps the canvas when the dark background is see-through', () => {
    expect(needsLightCanvas('<p style="background:rgba(0,0,0,0);color:#000">text</p>')).toBe(true)
    expect(needsLightCanvas('<p style="background:rgba(0, 0, 0, 50%);color:#000">text</p>')).toBe(
      true
    )
    expect(needsLightCanvas('<p style="background:#0000;color:#000">text</p>')).toBe(true)
    expect(needsLightCanvas('<p style="background:#00000080;color:#000">text</p>')).toBe(true)
    expect(needsLightCanvas('<div style="background:rgba(17,17,17,1);color:#eee">dark</div>')).toBe(
      false
    )
  })

  test('keeps the canvas when colored text sits outside the dark background', () => {
    expect(
      needsLightCanvas(
        '<div style="background:#111;color:#eee">dark</div><p style="color:#000">footer</p>'
      )
    ).toBe(true)
    expect(
      needsLightCanvas(
        '<div style="background:#111">dark</div><div style="background:#fff">light</div>'
      )
    ).toBe(true)
  })
})
