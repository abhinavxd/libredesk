import { describe, expect, it } from 'vitest'
import { generateMessageId } from '@lingui/message-utils/generateMessageId'
import PO from 'pofile'
import settings from '../lingui.config.mjs'
import { compilePO } from './gettext.mjs'
import { createI18n } from '../shared-ui/i18n/index.js'

const messages = [
  { message: 'Save', translation: 'Sauvegarder' },
  { message: 'Open', context: 'Action', translation: 'Ouvrir' },
  {
    message: '{count, plural, one {View} other {Views}}',
    context: 'Article page views',
    translation: '{count, plural, one {Consultation} other {Consultations}}'
  }
]
const ids = messages.map(({ message, context }) => generateMessageId(message, context))

function catalog(fuzzy, locale = 'fr-FR') {
  const entries = Object.fromEntries(messages.map((message, index) => [ids[index], {
    ...message, translation: locale === 'en-US' ? message.message : message.translation,
    origin: [], comments: []
  }]))
  const po = PO.parse(settings.format.serialize(entries, {
    locale, sourceLocale: 'en-US', filename: 'frontend.po'
  }))
  for (const item of po.items) {
    if (item.msgctxt) item.flags.fuzzy = fuzzy
  }
  return po.toString()
}

describe('Frontend catalog compilation', () => {
  it('excludes fuzzy singular and plural translations while keeping reviewed messages', () => {
    const compiled = compilePO(catalog(true), 'fr-FR')
    expect(compiled[ids[0]]).toEqual(['Sauvegarder'])
    expect(compiled).not.toHaveProperty(ids[1])
    expect(compiled).not.toHaveProperty(ids[2])
  })

  it('includes the translations once the fuzzy flag has been removed', () => {
    const compiled = compilePO(catalog(false), 'fr-FR')
    expect(compiled[ids[1]]).toEqual(['Ouvrir'])
    expect(compiled).toHaveProperty(ids[2])
  })

  it('falls back to English for excluded entries, including contextual plurals', () => {
    const i18n = createI18n({
      locale: 'fr-FR',
      messages: compilePO(catalog(true), 'fr-FR'),
      english: compilePO(catalog(false, 'en-US'), 'en-US')
    }).global
    expect(i18n._('Save')).toBe('Sauvegarder')
    expect(i18n.pgettext('Action', 'Open')).toBe('Open')
    expect(i18n.ngettext('View', 'Views', 1, {}, 'Article page views')).toBe('View')
    expect(i18n.ngettext('View', 'Views', 2, {}, 'Article page views')).toBe('Views')
  })
})
