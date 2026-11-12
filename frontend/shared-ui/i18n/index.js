import { computed, inject, ref } from 'vue'
import { setupI18n } from '@lingui/core'
import { generateMessageId } from '@lingui/message-utils/generateMessageId'
import TranslatedText from './TranslatedText'
import { pluralMessage } from './plural'

const injectionKey = Symbol('libredesk-i18n')
const messageIds = new Map()

function sourceMessage(message, context = '') {
  const key = JSON.stringify([message, context])
  if (!messageIds.has(key)) messageIds.set(key, { id: generateMessageId(message, context), message })
  return messageIds.get(key)
}

// Each app/widget owns its engines. Missing messages must use English plural
// rules, not the active language's rules applied to an English translation.
export function createI18n(catalog) {
  const english = setupI18n({ locale: 'en-US', messages: { 'en-US': catalog.english } })
  const engine = setupI18n({ locale: catalog.locale, messages: { [catalog.locale]: catalog.messages } })
  const revision = ref(0)
  const resolvedLocales = new Map([[catalog.locale, catalog.locale]])
  engine.on('change', () => revision.value++)
  const locale = computed({
    get: () => { revision.value; return engine.locale },
    set: value => engine.activate(resolvedLocales.get(value) || value),
  })

  function translate(message, countOrValues, valuesOrCount) {
    revision.value
    const descriptor = typeof message === 'string' ? sourceMessage(message) : message
    if (!descriptor) return message ?? ''
    const values = { ...(typeof countOrValues === 'object' ? countOrValues : valuesOrCount && typeof valuesOrCount === 'object' ? valuesOrCount : {}) }
    const count = typeof countOrValues === 'number' ? countOrValues
      : typeof valuesOrCount === 'number' ? valuesOrCount : values.count ?? values.n ?? 1
    values.count ??= count
    values.n ??= count
    const selected = Number(count) === 0 && descriptor.zero ? descriptor.zero : descriptor
    const translator = Object.hasOwn(engine.messages, selected.id) ? engine : english
    return translator._(selected, values)
  }

  function setLocaleMessage(code, next) {
    resolvedLocales.set(code, next.locale)
    engine.load(code, next.messages)
    // Keep the requested code as an alias when the server resolved a regional
    // variant, but use the resolved locale to select the right plural rules.
    if (next.locale !== code) engine.load(next.locale, next.messages)
    english.load('en-US', next.english)
  }

  function ngettext(singular, plural, count = 1, values = {}, context = '') {
    return translate(sourceMessage(pluralMessage(singular, plural), context), { ...values, count })
  }

  function pgettext(context, message, values = {}) {
    return translate(sourceMessage(message, context), values)
  }

  const global = { _: translate, ngettext, pgettext, locale, setLocaleMessage }
  return {
    global,
    install(app) {
      app.provide(injectionKey, global)
      app.config.globalProperties.i18n = global
      app.component('TranslatedText', TranslatedText)
    },
  }
}

export function useI18n() {
  const i18n = inject(injectionKey)
  if (!i18n) throw new Error('The application translation plugin has not been installed')
  return i18n
}
