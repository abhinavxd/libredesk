// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { resolveLanguage, determineInitialLanguage } from './language.js'
import { createI18n, useI18n } from 'vue-i18n'
import { createApp, defineComponent, nextTick } from 'vue'

describe('resolveLanguage', () => {
  const available = [
    'da-DK',
    'de-DE',
    'en-US',
    'es-ES',
    'fr-FR',
    'it-IT',
    'ja-JP',
    'pt-BR',
    'pt-PT',
    'zh-CN'
  ]

  it('matches exact language code', () => {
    expect(resolveLanguage('zh-CN', available)).toBe('zh-CN')
    expect(resolveLanguage('de-DE', available)).toBe('de-DE')
    expect(resolveLanguage('en-US', available)).toBe('en-US')
  })

  it('matches case-insensitively', () => {
    expect(resolveLanguage('zh-cn', available)).toBe('zh-CN')
    expect(resolveLanguage('DE-de', available)).toBe('de-DE')
    expect(resolveLanguage('EN-US', available)).toBe('en-US')
  })

  it('matches bare primary tags', () => {
    expect(resolveLanguage('zh', available)).toBe('zh-CN')
    expect(resolveLanguage('de', available)).toBe('de-DE')
    expect(resolveLanguage('fr', available)).toBe('fr-FR')
    expect(resolveLanguage('it', available)).toBe('it-IT')
    expect(resolveLanguage('ja', available)).toBe('ja-JP')
    expect(resolveLanguage('es', available)).toBe('es-ES')
    expect(resolveLanguage('da', available)).toBe('da-DK')
  })

  it('matches primary tag when subtag is different', () => {
    expect(resolveLanguage('zh-TW', available)).toBe('zh-CN')
    expect(resolveLanguage('de-AT', available)).toBe('de-DE')
    expect(resolveLanguage('fr-CA', available)).toBe('fr-FR')
    expect(resolveLanguage('en-GB', available)).toBe('en-US')
  })

  it('returns the first match for multiple variants', () => {
    expect(resolveLanguage('pt', available)).toBe('pt-BR')
    expect(resolveLanguage('pt-PT', available)).toBe('pt-PT')
    expect(resolveLanguage('pt-BR', available)).toBe('pt-BR')
  })

  it('returns null for unknown codes', () => {
    expect(resolveLanguage('xx', available)).toBeNull()
    expect(resolveLanguage('klingon', available)).toBeNull()
    expect(resolveLanguage('ru-RU', available)).toBeNull()
  })

  it('returns null for invalid inputs', () => {
    expect(resolveLanguage('', available)).toBeNull()
    expect(resolveLanguage('   ', available)).toBeNull()
    expect(resolveLanguage(null, available)).toBeNull()
    expect(resolveLanguage(undefined, available)).toBeNull()
    expect(resolveLanguage(123, available)).toBeNull()
    expect(resolveLanguage('en', null)).toBeNull()
    expect(resolveLanguage('en', [])).toBeNull()
  })
})

describe('determineInitialLanguage', () => {
  const available = [
    'da-DK',
    'de-DE',
    'en-US',
    'es-ES',
    'fr-FR',
    'it-IT',
    'ja-JP',
    'pt-BR',
    'zh-CN'
  ]

  it('uses host-provided urlLang when inbox is auto and lang is valid exact code', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'en-US' },
      urlLang: 'zh-CN',
      browserLang: 'en-US',
      availableCodes: available
    })
    expect(lang).toBe('zh-CN')
  })

  it('resolves host-provided bare tag when inbox is auto', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'en-US' },
      urlLang: 'de',
      browserLang: 'en-US',
      availableCodes: available
    })
    expect(lang).toBe('de-DE')
  })

  it('falls back to browserLang when host urlLang is not provided', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'en-US' },
      urlLang: null,
      browserLang: 'fr-FR',
      availableCodes: available
    })
    expect(lang).toBe('fr-FR')
  })

  it('resolves browser bare tag when host urlLang is not provided', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'en-US' },
      urlLang: null,
      browserLang: 'it',
      availableCodes: available
    })
    expect(lang).toBe('it-IT')
  })

  it('falls back to browserLang when host urlLang is invalid or not available', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'en-US' },
      urlLang: 'xx-YY',
      browserLang: 'de-DE',
      availableCodes: available
    })
    expect(lang).toBe('de-DE')
  })

  it('falls back to configured fallback_language when neither urlLang nor browserLang matches', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto', fallback_language: 'es-ES' },
      urlLang: 'unknown',
      browserLang: 'unknown',
      availableCodes: available
    })
    expect(lang).toBe('es-ES')
  })

  it('defaults to en-US when fallback_language is not specified and nothing matches', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'auto' },
      urlLang: null,
      browserLang: 'unknown',
      availableCodes: available
    })
    expect(lang).toBe('en-US')
  })

  it('respects pinned fixed language configuration over host urlLang', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'es-ES', fallback_language: 'en-US' },
      urlLang: 'zh-CN',
      browserLang: 'en-US',
      availableCodes: available
    })
    expect(lang).toBe('es-ES')
  })

  it('respects pinned fixed language configuration when urlLang is omitted', () => {
    const lang = determineInitialLanguage({
      widgetConfig: { language: 'ja-JP' },
      urlLang: null,
      browserLang: 'en-US',
      availableCodes: available
    })
    expect(lang).toBe('ja-JP')
  })
})

describe('vue-i18n dynamic locale switching', () => {
  it('updates messages and switches locale in place without remounting', async () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en-US',
      fallbackLocale: 'en-US',
      messages: {
        'en-US': { greeting: 'Hello' }
      }
    })

    const root = document.createElement('div')
    document.body.appendChild(root)

    let exposedApi
    const AppComp = defineComponent({
      template: '<span>{{ t("greeting") }}</span>',
      setup() {
        const { t, locale, setLocaleMessage, getLocaleMessage } = useI18n()
        exposedApi = { locale, setLocaleMessage, getLocaleMessage }
        return { t }
      }
    })

    const app = createApp(AppComp)
    app.use(i18n)
    app.mount(root)

    expect(root.textContent).toBe('Hello')
    expect(exposedApi.getLocaleMessage('zh-CN')).toEqual({})

    // Switch locale dynamically
    exposedApi.setLocaleMessage('zh-CN', { greeting: '你好' })
    expect(exposedApi.getLocaleMessage('zh-CN')).toEqual({ greeting: '你好' })
    exposedApi.locale.value = 'zh-CN'
    await nextTick()

    expect(root.textContent).toBe('你好')

    app.unmount()
    root.remove()
  })
})
