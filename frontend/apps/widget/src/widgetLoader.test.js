// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import fs from 'fs'
import path from 'path'

describe('static/widget.js loader', () => {
  let widgetCode

  beforeEach(() => {
    // Reset global state
    delete window.__libredeskWidgetLoaded
    delete window.Libredesk
    delete window.initLibredesk
    delete window.LibredeskSettings
    document.body.innerHTML = ''

    // Mock matchMedia
    window.matchMedia = vi.fn().mockImplementation((query) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn()
    }))

    // Mock fetch for settings
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({
        status: 'success',
        data: {
          launcher: {
            position: 'right',
            spacing: { bottom: 20, side: 20 },
            icon_scale: 100
          },
          theme: 'light',
          branding: {
            light: {
              launcher: { color: '#000000', logo_url: '' },
              colors: { primary: '#000000' }
            }
          }
        }
      })
    })

    if (!widgetCode) {
      const widgetJsPath = path.resolve(__dirname, '../../../../static/widget.js')
      widgetCode = fs.readFileSync(widgetJsPath, 'utf-8')
    }
  })

  afterEach(() => {
    vi.restoreAllMocks()
    document.body.innerHTML = ''
    delete window.__libredeskWidgetLoaded
    delete window.Libredesk
    delete window.initLibredesk
  })

  it('passes language in iframe src when config.language is provided', async () => {
    // Evaluate widget script
    new Function(widgetCode)()

    const instance = window.initLibredesk({
      baseURL: 'https://help.example.com',
      inboxID: 'inbox-123',
      language: 'zh-CN'
    })

    // Wait for async init (fetchWidgetSettings)
    await vi.waitFor(() => {
      expect(instance.iframe).not.toBeNull()
    })

    const iframeSrc = instance.iframe.src
    expect(iframeSrc).toContain('https://help.example.com/widget?')
    expect(iframeSrc).toContain('inbox_id=inbox-123')
    expect(iframeSrc).toContain('lang=zh-CN')
  })

  it('omits lang in iframe src when config.language is not provided', async () => {
    new Function(widgetCode)()

    const instance = window.initLibredesk({
      baseURL: 'https://help.example.com',
      inboxID: 'inbox-123'
    })

    await vi.waitFor(() => {
      expect(instance.iframe).not.toBeNull()
    })

    const iframeSrc = instance.iframe.src
    expect(iframeSrc).toContain('inbox_id=inbox-123')
    expect(iframeSrc).not.toContain('lang=')
  })

  it('setLanguage updates config and posts SET_LANGUAGE to iframe', async () => {
    new Function(widgetCode)()

    const instance = window.initLibredesk({
      baseURL: 'https://help.example.com',
      inboxID: 'inbox-123',
      language: 'en-US'
    })

    await vi.waitFor(() => {
      expect(instance.iframe).not.toBeNull()
    })

    const postMessageSpy = vi.fn()
    instance.iframe.contentWindow.postMessage = postMessageSpy

    // Call setLanguage on instance / window.Libredesk
    window.Libredesk.setLanguage('de-DE')

    expect(instance.config.language).toBe('de-DE')
    expect(postMessageSpy).toHaveBeenCalledWith(
      { type: 'SET_LANGUAGE', language: 'de-DE' },
      'https://help.example.com'
    )
  })

  it('setLanguage ignores non-string or empty inputs', async () => {
    new Function(widgetCode)()

    const instance = window.initLibredesk({
      baseURL: 'https://help.example.com',
      inboxID: 'inbox-123',
      language: 'en-US'
    })

    await vi.waitFor(() => {
      expect(instance.iframe).not.toBeNull()
    })

    const postMessageSpy = vi.fn()
    instance.iframe.contentWindow.postMessage = postMessageSpy

    window.Libredesk.setLanguage('')
    window.Libredesk.setLanguage(null)
    window.Libredesk.setLanguage(123)

    expect(instance.config.language).toBe('en-US')
    expect(postMessageSpy).not.toHaveBeenCalled()
  })

  it('queues setLanguage and delivers it when VUE_APP_READY arrives if changed before ready', async () => {
    new Function(widgetCode)()

    const instance = window.initLibredesk({
      baseURL: 'https://help.example.com',
      inboxID: 'inbox-123',
      language: 'en-US'
    })

    await vi.waitFor(() => {
      expect(instance.iframe).not.toBeNull()
    })

    const postMessageSpy = vi.fn()
    instance.iframe.contentWindow.postMessage = postMessageSpy

    // Change language before Vue app reports ready
    instance.setLanguage('fr-FR')
    expect(postMessageSpy).toHaveBeenCalledWith(
      { type: 'SET_LANGUAGE', language: 'fr-FR' },
      'https://help.example.com'
    )

    postMessageSpy.mockClear()

    // Trigger VUE_APP_READY from iframe
    window.dispatchEvent(
      new MessageEvent('message', {
        data: { type: 'VUE_APP_READY' },
        origin: 'https://help.example.com',
        source: instance.iframe.contentWindow
      })
    )

    // Should have re-sent SET_LANGUAGE during VUE_APP_READY
    expect(postMessageSpy).toHaveBeenCalledWith(
      { type: 'SET_LANGUAGE', language: 'fr-FR' },
      'https://help.example.com'
    )
  })
})
