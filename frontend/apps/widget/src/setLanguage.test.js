// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'

const { getAvailableLanguagesMock, getLanguageMock, getHelpMock, widgetConfigRef } = vi.hoisted(
  () => ({
    getAvailableLanguagesMock: vi.fn(),
    getLanguageMock: vi.fn(),
    getHelpMock: vi.fn(),
    widgetConfigRef: {
      current: {
        language: 'auto',
        fallback_language: 'en-US',
        help: { help_center_id: 'help-1' }
      }
    }
  })
)

vi.mock('@widget/api/index.js', () => ({
  default: {
    getAvailableLanguages: getAvailableLanguagesMock,
    getLanguage: getLanguageMock,
    getHelp: getHelpMock,
    exchangeJWTForSession: vi.fn(),
    getAuthMe: vi.fn(),
    getChatConversations: vi.fn().mockResolvedValue({ data: { data: [] } }),
    getWidgetSettings: vi.fn()
  },
  setApiSessionToken: vi.fn(),
  initVisitorToken: vi.fn(),
  saveSession: vi.fn(),
  registerStores: vi.fn()
}))

vi.mock('./websocket.js', () => ({
  initWidgetWS: vi.fn(),
  closeWidgetWebSocket: vi.fn(),
  sendPageVisit: vi.fn(),
  skipInitialWsSync: vi.fn()
}))

vi.mock('./composables/useUnreadCount.js', () => ({
  useUnreadCount: vi.fn()
}))

vi.mock('@shared-ui/composables/useNotificationSound.js', () => ({
  initAudioContext: vi.fn()
}))

vi.mock('@widget/layouts/MainLayout.vue', () => ({
  default: {
    setup: () => () => h('div', { class: 'mock-main-layout' })
  }
}))

describe('App.vue SET_LANGUAGE handling', () => {
  let app
  let root
  let i18n
  let pinia
  let AppComp

  beforeEach(async () => {
    vi.clearAllMocks()
    root = document.createElement('div')
    document.body.appendChild(root)

    getAvailableLanguagesMock.mockResolvedValue({
      data: {
        status: 'success',
        data: [
          { code: 'en-US', name: 'English' },
          { code: 'zh-CN', name: 'Chinese' },
          { code: 'de-DE', name: 'German' }
        ]
      }
    })

    getLanguageMock.mockImplementation((lang) =>
      Promise.resolve({
        data: { greeting: `hello_${lang}` }
      })
    )

    getHelpMock.mockResolvedValue({
      data: { status: 'success', data: { tree: [], featured_ids: [] } }
    })

    widgetConfigRef.current = {
      language: 'auto',
      fallback_language: 'en-US',
      help: { help_center_id: 'help-1' }
    }

    i18n = createI18n({
      legacy: false,
      locale: 'en-US',
      fallbackLocale: 'en-US',
      messages: {
        'en-US': { greeting: 'hello_en-US' }
      }
    })

    pinia = createPinia()

    // Import App component dynamically
    AppComp = (await import('./App.vue')).default
  })

  afterEach(() => {
    app?.unmount()
    root?.remove()
    vi.restoreAllMocks()
  })

  const mountApp = async (config = widgetConfigRef.current) => {
    app = createApp(AppComp)
    app.use(pinia)
    app.use(i18n)
    app.config.globalProperties.$widgetConfig = config
    app.mount(root)
    for (let i = 0; i < 5; i++) await nextTick()
  }

  const dispatchParentMessage = (data, origin = window.location.origin) => {
    const event = new MessageEvent('message', {
      data,
      origin,
      source: window.parent
    })
    window.dispatchEvent(event)
  }

  it('switches locale and loads messages when SET_LANGUAGE is received', async () => {
    await mountApp()
    expect(i18n.global.locale.value).toBe('en-US')

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh-CN' })
    for (let i = 0; i < 10; i++) await nextTick()

    expect(getAvailableLanguagesMock).toHaveBeenCalled()
    expect(getLanguageMock).toHaveBeenCalledWith('zh-CN')
    expect(i18n.global.locale.value).toBe('zh-CN')
    expect(i18n.global.getLocaleMessage('zh-CN')).toEqual({ greeting: 'hello_zh-CN' })
    expect(getHelpMock).toHaveBeenCalledWith('zh-CN')
  })

  it('resolves bare primary tag like "de" to "de-DE"', async () => {
    await mountApp()

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'de' })
    for (let i = 0; i < 10; i++) await nextTick()

    expect(getLanguageMock).toHaveBeenCalledWith('de-DE')
    expect(i18n.global.locale.value).toBe('de-DE')
  })

  it('uses cached messages when switching back to a previously loaded locale', async () => {
    await mountApp()

    // Switch to zh-CN
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh-CN' })
    for (let i = 0; i < 10; i++) await nextTick()
    expect(getLanguageMock).toHaveBeenCalledTimes(1)
    expect(i18n.global.locale.value).toBe('zh-CN')

    // Switch back to en-US (already loaded at startup)
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'en-US' })
    for (let i = 0; i < 10; i++) await nextTick()
    expect(i18n.global.locale.value).toBe('en-US')
    // getLanguageMock should NOT have been called for en-US since it was cached
    expect(getLanguageMock).toHaveBeenCalledTimes(1)
  })

  it('does not change locale if inbox language is pinned to a fixed language', async () => {
    await mountApp({
      language: 'en-US', // Fixed language, not 'auto'
      fallback_language: 'en-US'
    })

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh-CN' })
    for (let i = 0; i < 10; i++) await nextTick()

    expect(i18n.global.locale.value).toBe('en-US')
    expect(getLanguageMock).not.toHaveBeenCalled()
  })

  it('ignores invalid, unknown, or whitespace-only language codes', async () => {
    await mountApp()

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'xx' })
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: '   ' })
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: '' })
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: null })
    for (let i = 0; i < 10; i++) await nextTick()

    expect(i18n.global.locale.value).toBe('en-US')
    expect(getLanguageMock).not.toHaveBeenCalled()
  })

  it('resolves underscore format like "zh_CN" to "zh-CN"', async () => {
    await mountApp()

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh_CN' })
    for (let i = 0; i < 10; i++) await nextTick()

    expect(getLanguageMock).toHaveBeenCalledWith('zh-CN')
    expect(i18n.global.locale.value).toBe('zh-CN')
  })

  it('caches available language codes across multiple SET_LANGUAGE calls', async () => {
    await mountApp()

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh-CN' })
    for (let i = 0; i < 10; i++) await nextTick()
    expect(getAvailableLanguagesMock).toHaveBeenCalledTimes(1)

    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'de-DE' })
    for (let i = 0; i < 10; i++) await nextTick()
    // getAvailableLanguagesMock should NOT be called again
    expect(getAvailableLanguagesMock).toHaveBeenCalledTimes(1)
  })

  it('prevents race conditions when requests resolve out of order', async () => {
    let resolveDe
    let resolveZh
    const dePromise = new Promise((res) => {
      resolveDe = res
    })
    const zhPromise = new Promise((res) => {
      resolveZh = res
    })

    getLanguageMock.mockImplementation((lang) => {
      if (lang === 'de-DE') return dePromise
      if (lang === 'zh-CN') return zhPromise
      return Promise.resolve({ data: { greeting: `hello_${lang}` } })
    })

    await mountApp()

    // 1. Trigger de-DE (slow)
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'de-DE' })
    await nextTick()

    // 2. Trigger zh-CN (fast)
    dispatchParentMessage({ type: 'SET_LANGUAGE', language: 'zh-CN' })
    await nextTick()

    // 3. zh-CN resolves first
    resolveZh({ data: { greeting: 'hello_zh-CN' } })
    for (let i = 0; i < 10; i++) await nextTick()
    expect(i18n.global.locale.value).toBe('zh-CN')

    // 4. de-DE resolves later (out of order)
    resolveDe({ data: { greeting: 'hello_de-DE' } })
    for (let i = 0; i < 10; i++) await nextTick()

    // The widget must NOT revert to de-DE; it must remain zh-CN
    expect(i18n.global.locale.value).toBe('zh-CN')
  })
})
