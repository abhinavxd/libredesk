import { describe, expect, it } from 'vitest'
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { makeConfig } from '@lingui/conf'
import { createVueExtractor } from '@lingui/extractor-vue'
import PO from 'pofile'
import settings from '../lingui.config.mjs'
import { withGettextHelpers } from './gettext-extractor.mjs'
import { pluralMessage } from '../shared-ui/i18n/plural.js'

const config = makeConfig(settings)
const filename = fileURLToPath(new URL('../shared-ui/i18n/__tests__/Messages.vue', import.meta.url))
const extractor = withGettextHelpers(createVueExtractor())
async function extract(code) {
  const messages = []
  await extractor.extract(filename, code, msg => messages.push(msg), { linguiConfig: config })
  return messages
}

describe('Gettext calls in Vue templates', () => {
  it('extracts inline plurals and contexts with their original source locations', async () => {
    const source = await fs.readFile(filename, 'utf8')
    const messages = await extract(source)
    const plural = messages.find(msg => msg.message === pluralMessage('# file', '# files'))
    const contextual = messages.find(msg => msg.message === 'Open')
    expect(contextual.context).toBe('button')
    expect(plural).toBeDefined()
    expect(path.basename(plural.origin[0])).toBe('Messages.vue')
    expect(plural.origin[1]).toBe(source.split('\n').findIndex(line => line.includes('i18n.ngettext')) + 1)
    const catalog = Object.fromEntries([plural, contextual].map(msg => [msg.id, { ...msg, origin: [msg.origin], translation: '' }]))
    const po = PO.parse(settings.format.serialize(catalog, { locale: 'en-US', sourceLocale: 'en-US', filename: 'frontend.pot' }))
    expect(po.items.find(item => item.msgid === '# file').msgid_plural).toBe('# files')
    expect(po.items.find(item => item.msgid === 'Open').msgctxt).toBe('button')
  })

  it('rejects dynamic source messages instead of silently omitting them', async () => {
    await expect(extract('<template>{{ i18n.ngettext(label, "Files", count) }}</template>'))
      .rejects.toThrow('must be string literals')
  })

  it('keeps identical plurals with different meanings separate in Gettext catalogs', async () => {
    const messages = await extract(`<template>
      {{ i18n.ngettext('View', 'Views', count) }}
      {{ i18n.ngettext('View', 'Views', count, {}, 'Article page views') }}
    </template>`)
    const plurals = messages.filter(msg => msg.message === pluralMessage('View', 'Views'))
    expect(plurals).toHaveLength(2)
    expect(new Set(plurals.map(msg => msg.id)).size).toBe(2)
    const catalog = Object.fromEntries(plurals.map(msg => [msg.id, {
      ...msg, origin: [msg.origin], translation: ''
    }]))
    const po = PO.parse(settings.format.serialize(catalog, {
      locale: 'en-US', sourceLocale: 'en-US', filename: 'frontend.pot'
    }))
    expect(po.items.map(item => [item.msgctxt || '', item.msgid, item.msgid_plural]))
      .toEqual(expect.arrayContaining([
        ['', 'View', 'Views'],
        ['Article page views', 'View', 'Views']
      ]))
  })

  it('ignores similarly named methods on unrelated objects', async () => {
    expect(await extract('<template>{{ other.ngettext("File", "Files", count) }}</template>')).toEqual([])
  })
})
