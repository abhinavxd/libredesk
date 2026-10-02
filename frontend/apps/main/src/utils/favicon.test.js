// @vitest-environment jsdom
import { beforeEach, describe, expect, it } from 'vitest'
import { setFavicon } from './favicon'

const icons = () =>
  [...document.querySelectorAll('link[rel~="icon"]')].map((l) => [
    l.getAttribute('href'),
    l.getAttribute('type')
  ])

describe('setFavicon', () => {
  beforeEach(() => {
    document.head.innerHTML = `
      <link rel="icon" type="image/x-icon" href="/favicon.ico?v=fern-stem" />
      <link id="app-favicon" rel="icon" type="image/svg+xml" href="/favicon.svg?v=fern-stem" />
      <link rel="apple-touch-icon" href="/images/pwa-icon-192.png" />`
  })

  it('points every icon at the logo and restores the bundled icons when cleared', () => {
    setFavicon('/uploads/550e8400-e29b-41d4-a716-446655440000')
    expect(icons()).toEqual([
      ['/uploads/550e8400-e29b-41d4-a716-446655440000', null],
      ['/uploads/550e8400-e29b-41d4-a716-446655440000', null]
    ])

    setFavicon('')
    expect(icons()).toEqual([
      ['/favicon.ico?v=fern-stem', 'image/x-icon'],
      ['/favicon.svg?v=fern-stem', 'image/svg+xml']
    ])
  })

  it('leaves the bundled icons alone when no logo is set', () => {
    setFavicon('')
    expect(icons()).toEqual([
      ['/favicon.ico?v=fern-stem', 'image/x-icon'],
      ['/favicon.svg?v=fern-stem', 'image/svg+xml']
    ])
    expect(document.querySelector('link[rel="apple-touch-icon"]').getAttribute('href')).toBe(
      '/images/pwa-icon-192.png'
    )
  })
})
