import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import api from '@widget/api/index.js'
import { useHelpStore } from '@widget/store/help.js'
import { matchLanguage, resolveLanguage } from './language.js'

const http = vi.hoisted(() => ({
  get: vi.fn(),
  interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } }
}))

vi.mock('axios', () => ({ default: { create: () => http } }))

const codes = ['de-DE', 'en-US', 'pt-BR', 'zh-CN']

describe('matchLanguage', () => {
  test('returns an exact code, ignoring case and underscores', () => {
    expect(matchLanguage('zh-CN', codes)).toBe('zh-CN')
    expect(matchLanguage('pt_br', codes)).toBe('pt-BR')
    expect(matchLanguage(' EN-us ', codes)).toBe('en-US')
  })

  test('falls back to a code with the same primary subtag', () => {
    expect(matchLanguage('zh', codes)).toBe('zh-CN')
    expect(matchLanguage('de-AT', codes)).toBe('de-DE')
    expect(matchLanguage('en-GB', codes)).toBe('en-US')
  })

  test('returns an empty string when nothing matches or nothing was asked for', () => {
    expect(matchLanguage('ko-KR', codes)).toBe('')
    expect(matchLanguage('', codes)).toBe('')
    expect(matchLanguage(null, codes)).toBe('')
    expect(matchLanguage('zh-CN', undefined)).toBe('')
  })

  test('keeps Chinese writing systems separate', () => {
    expect(matchLanguage('zh-TW', codes)).toBe('')
    expect(matchLanguage('zh-Hant', codes)).toBe('')
    expect(matchLanguage('zh-Hans', codes)).toBe('zh-CN')
    expect(matchLanguage('zh-Hant-TW', ['zh-CN', 'zh-Hant'])).toBe('zh-Hant')
  })

  test.each([
    { name: 'exact region overrides the first translation', requested: 'pt-PT', available: ['pt-BR', 'pt-PT'], want: 'pt-PT' },
    { name: 'UK alias selects the British translation', requested: 'en-UK', available: ['en-US', 'en-GB'], want: 'en-GB' },
    { name: 'explicit script overrides the region', requested: 'zh-Hans-TW', available: ['zh-Hant', 'zh-CN'], want: 'zh-CN' },
    { name: 'formatting options do not change the translation', requested: 'de-DE-u-ca-gregory', available: ['de-DE'], want: 'de-DE' },
    { name: 'legacy Hebrew code selects Hebrew', requested: 'iw-IL', available: ['en-US', 'he-IL'], want: 'he-IL' },
    { name: 'numeric region selects the same language', requested: 'es-419', available: ['en-US', 'es-ES'], want: 'es-ES' }
  ])('$name', ({ requested, available, want }) => {
    expect(matchLanguage(requested, available)).toBe(want)
  })
})

describe('resolveLanguage', () => {
  const config = { language: 'auto', fallback_language: 'en-US' }

  test('uses the configured fallback when the browser region has no exact translation', () => {
    expect(resolveLanguage(config, '', ['de-AT'], codes)).toEqual({
      locale: 'en-US', requestedLocale: 'en-US'
    })
  })

  test('uses an exact browser translation without a website preference', () => {
    expect(resolveLanguage(config, '', ['de-DE'], codes)).toEqual({
      locale: 'de-DE', requestedLocale: 'de-DE'
    })
  })

  test('matches website regions while keeping the browser fallback exact', () => {
    expect(resolveLanguage(config, 'de-AT', ['ko-KR'], codes)).toEqual({
      locale: 'de-DE', requestedLocale: 'de-AT'
    })
  })

  test('preserves the website region for help center articles', () => {
    expect(resolveLanguage(config, 'en-GB', ['de-DE'], codes)).toEqual({
      locale: 'en-US', requestedLocale: 'en-GB'
    })
  })

  test('keeps an unsupported writing system as the article preference', () => {
    expect(resolveLanguage(config, 'zh-TW', ['en-US'], codes)).toEqual({
      locale: 'en-US', requestedLocale: 'zh-TW'
    })
  })

  test('uses the configured fallback instead of another browser preference', () => {
    expect(resolveLanguage(config, '', ['ko-KR', 'de-DE'], codes)).toEqual({
      locale: 'en-US', requestedLocale: 'en-US'
    })
  })

  test('clearing a website preference returns to browser selection', () => {
    for (const requested of ['', null, undefined]) {
      expect(resolveLanguage(config, requested, ['de-AT'], codes)).toEqual({
        locale: 'en-US', requestedLocale: 'en-US'
      })
    }
  })

  test('fixed inbox language overrides website and browser preferences', () => {
    const fixed = { language: 'pt-BR', fallback_language: 'en-US' }
    expect(resolveLanguage(fixed, 'de-DE', ['en-US'], codes)).toEqual({
      locale: 'pt-BR', requestedLocale: 'pt-BR'
    })
  })

  test('uses the configured fallback when no preference is available', () => {
    const fallback = { language: 'auto', fallback_language: 'pt-BR' }
    expect(resolveLanguage(fallback, '', [], codes)).toEqual({
      locale: 'pt-BR', requestedLocale: 'pt-BR'
    })
  })

  test('defaults to English when no language is configured', () => {
    expect(resolveLanguage({ language: 'auto' }, '', [], codes)).toEqual({
      locale: 'en-US', requestedLocale: 'en-US'
    })
  })

  test.each([
    {
      name: 'an unmatched browser region keeps the custom fallback',
      config: { language: 'auto', fallback_language: 'pt-BR' },
      host: '', browsers: ['de-AT', 'de-DE'],
      want: { locale: 'pt-BR', requestedLocale: 'pt-BR' }
    },
    {
      name: 'untranslated browser language uses the configured label fallback',
      config: { language: 'auto', fallback_language: 'pt-BR' },
      host: '', browsers: ['ko-KR'],
      want: { locale: 'pt-BR', requestedLocale: 'pt-BR' }
    },
    {
      name: 'formatting options retain the website region for HC',
      config, host: 'en-GB-u-nu-latn', browsers: ['de-DE'],
      want: { locale: 'en-US', requestedLocale: 'en-GB' }
    },
    {
      name: 'a malformed website tag falls back to the browser',
      config, host: 'en--GB', browsers: ['de-AT'],
      want: { locale: 'en-US', requestedLocale: 'en-US' }
    },
    {
      name: 'a missing fixed language uses the configured fallback',
      config: { fallback_language: 'pt-BR' },
      host: 'de-DE', browsers: ['en-US'],
      want: { locale: 'pt-BR', requestedLocale: 'pt-BR' }
    }
  ])('$name', ({ config, host, browsers, want }) => {
    expect(resolveLanguage(config, host, browsers, codes)).toEqual(want)
  })
})

describe('help center language loading', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    http.get.mockReset()
  })

  test('preserves both language preferences when help resets and reloads', async () => {
    const help = useHelpStore()
    help.requestedLocale = 'fr-CA'
    help.fallbackLocale = 'fr-FR'
    http.get.mockResolvedValue({ data: { data: { locale: 'fr', tree: [] } } })

    await help.load()
    help.reset()
    await help.load()

    expect(http.get).toHaveBeenCalledTimes(2)
    for (const [path, options] of http.get.mock.calls) {
      expect(path).toBe('/api/v1/widget/chat/help')
      expect(options.params).toEqual({ locale: 'fr-CA', fallback_locale: 'fr-FR' })
    }
    expect(help.data.locale).toBe('fr')
    expect(help.requestedLocale).toBe('fr-CA')
    expect(help.fallbackLocale).toBe('fr-FR')
  })

  test('omits the fallback parameter when both preferences match', async () => {
    http.get.mockResolvedValue({ data: { data: null } })

    await api.getHelp('de-DE', 'de-DE')

    expect(http.get.mock.calls[0][1].params.locale).toBe('de-DE')
    expect(http.get.mock.calls[0][1].params.fallback_locale).toBeUndefined()
  })

  test('retains both preferences after a failed load and successful retry', async () => {
    const help = useHelpStore()
    help.requestedLocale = 'ko-KR'
    help.fallbackLocale = 'de-DE'
    http.get.mockRejectedValueOnce(new Error('Test request failed'))

    await help.load()
    expect(help.failed).toBe(true)
    expect(help.loading).toBe(false)

    http.get.mockResolvedValueOnce({ data: { data: { locale: 'de', tree: [] } } })
    await help.load()

    expect(help.failed).toBe(false)
    expect(help.data.locale).toBe('de')
    expect(http.get.mock.calls[1][1].params).toEqual({ locale: 'ko-KR', fallback_locale: 'de-DE' })
  })

  test.each(['resolve', 'reject'])('ignores a late %s from the previous language', async (outcome) => {
    const help = useHelpStore()
    help.requestedLocale = 'fr-CA'
    help.fallbackLocale = 'fr-FR'
    let resolvePrevious, rejectPrevious
    http.get.mockImplementationOnce(() => new Promise((resolve, reject) => {
      resolvePrevious = resolve
      rejectPrevious = reject
    }))
    const previousLoad = help.load()

    help.requestedLocale = 'de-AT'
    help.fallbackLocale = 'de-DE'
    help.reset()
    http.get.mockResolvedValueOnce({ data: { data: { locale: 'de', tree: [] } } })
    await help.load()

    if (outcome === 'resolve') resolvePrevious({ data: { data: { locale: 'fr', tree: [] } } })
    else rejectPrevious(new Error('Previous language failed'))
    await previousLoad

    expect(help.data.locale).toBe('de')
    expect(help.requestedLocale).toBe('de-AT')
    expect(help.fallbackLocale).toBe('de-DE')
    expect(help.failed).toBe(false)
    expect(help.loading).toBe(false)
  })
})
