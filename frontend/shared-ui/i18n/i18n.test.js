// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { computed, createApp, nextTick } from 'vue'
import { createCompiledCatalog } from '@lingui/cli/api'
import { generateMessageId } from '@lingui/message-utils/generateMessageId'
import { createI18n } from './index'
import Messages from './__tests__/Messages.vue'
import french from '../../../locales/fr-FR/LC_MESSAGES/frontend.json'
import english from '../../../locales/en-US/LC_MESSAGES/frontend.json'
import { reportsNavItems, contactNavItems, adminNavItems } from '../../apps/main/src/constants/navigation'

const files = '{count, plural, one {# file} other {# files}}'
const id = generateMessageId
function compile(locale, messages) {
  const result = createCompiledCatalog(locale, messages, { namespace: 'json', strict: true })
  expect(result.errors).toEqual([])
  return JSON.parse(result.source).messages
}
const source = compile('en-US', {
  [id('Hello {name}')]: 'Hello {name}',
  [id(files)]: files,
  [id('Open', 'button')]: 'Open',
})
let app, element
afterEach(() => { app?.unmount(); app = undefined; element?.remove() })

describe('Lingui Vue integration', () => {
  it('renders inline messages, variables, plurals and contexts, and reacts to locale changes', async () => {
    const translated = compile('fr-FR', {
      [id('Hello {name}')]: 'Bonjour {name}',
      [id(files)]: '{count, plural, one {# fichier} other {# fichiers}}',
      [id('Open', 'button')]: 'Ouvrir',
    })
    const plugin = createI18n({ locale: 'fr-FR', messages: translated, english: source })
    element = document.createElement('div')
    document.body.append(element)
    app = createApp(Messages, { name: '<img src=x onerror=alert(1)>', count: 2 })
    app.use(plugin).mount(element)
    expect(element.textContent).toContain('Bonjour <img src=x onerror=alert(1)>')
    expect(element.querySelector('img')).toBeNull()
    expect(element.textContent).toContain('2 fichiers')
    expect(element.querySelector('button').textContent).toBe('Ouvrir')
    plugin.global.setLocaleMessage('en', { locale: 'en-US', messages: source, english: source })
    plugin.global.locale.value = 'en'
    await nextTick()
    expect(element.textContent).toContain('Hello ')
    expect(element.textContent).toContain('2 files')
    expect(element.querySelector('button').textContent).toBe('Open')
  })

  it('uses English plural rules for a missing Russian translation', () => {
    const { ngettext } = createI18n({ locale: 'ru', messages: {}, english: source }).global
    expect(ngettext('# file', '# files', 21)).toBe('21 files')
  })

  it('selects translations by context as well as plural form', () => {
    const views = '{count, plural, one {View} other {Views}}'
    const messages = compile('fr-FR', {
      [id(views)]: '{count, plural, one {Vue} other {Vues}}',
      [id(views, 'Article page views')]: '{count, plural, one {Consultation} other {Consultations}}',
    })
    const { ngettext } = createI18n({ locale: 'fr-FR', messages, english: source }).global
    expect(ngettext('View', 'Views', 1)).toBe('Vue')
    expect(ngettext('View', 'Views', 2)).toBe('Vues')
    expect(ngettext('View', 'Views', 1, {}, 'Article page views')).toBe('Consultation')
    expect(ngettext('View', 'Views', 2, {}, 'Article page views')).toBe('Consultations')
  })

  it('uses the Persian singular for zero and one', () => {
    const messages = compile('fa-IR', { [id(files)]: '{count, plural, one {ONE} other {OTHER}}' })
    const { ngettext } = createI18n({ locale: 'fa-IR', messages, english: source }).global
    expect([0, 1, 2].map(count => ngettext('# file', '# files', count))).toEqual(['ONE', 'ONE', 'OTHER'])
  })

  it('translates direct source text using the real compiled French catalog', () => {
    const i18n = createI18n({ ...french, english: english.messages }).global
    expect(i18n._('Save')).toBe('Sauvegarder')
    expect(i18n._('Must be between {min} and {max}', { min: 2, max: 5 })).toContain('2')
    expect(i18n._('Must be between {min} and {max}', { min: 2, max: 5 })).toContain('5')
  })

  it('translates English navigation descriptors reactively, including plurals and contexts', () => {
    // Supply a reviewed context fixture independently of the real PO's fuzzy flags.
    const messages = {
      ...french.messages,
      ...compile('fr-FR', {
        [id('{count, plural, one {Email} other {Emails}}', 'Communication channel')]:
          '{count, plural, one {E-mail} other {E-mails}}'
      })
    }
    const i18n = createI18n({ ...french, messages, english: english.messages }).global
    const overview = computed(() => i18n._(reportsNavItems[0].titleMessage))
    expect(overview.value).toBe("Vue d'ensemble")
    expect(i18n._(contactNavItems[0].titleMessage, 1)).toBe('Contact')
    expect(i18n._(contactNavItems[0].titleMessage, 2)).toBe('Contacts')
    const email = adminNavItems.flatMap(group => group.children || [])
      .find(item => item.titleMessage.id === id('{count, plural, one {Email} other {Emails}}', 'Communication channel'))
    expect(email).toBeDefined()
    expect(i18n._(email.titleMessage)).toBe('E-mail')
    i18n.setLocaleMessage('en-US', { ...english, english: english.messages })
    i18n.locale.value = 'en-US'
    expect(overview.value).toBe('Overview')
    expect(i18n._(email.titleMessage)).toBe('Email')
  })
})
