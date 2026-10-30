import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'
import { makeConfig } from '@lingui/conf'
import { createCompiledCatalog } from '@lingui/cli/api'
import { generateMessageId } from '@lingui/message-utils/generateMessageId'
import PO from 'pofile'
import settings from '../lingui.config.mjs'

const root = fileURLToPath(new URL('../..', import.meta.url))
const config = makeConfig(settings)
const format = settings.format
const mode = process.argv[2]
const stdout = process.argv.includes('--stdout')
const domain = 'frontend'
const localeRoot = path.join(root, 'locales')

export async function extract() {
  const catalog = {}
  const files = []
  async function visit(directory) {
    for (const entry of await fs.readdir(directory, { withFileTypes: true })) {
      const name = path.join(directory, entry.name)
      if (entry.isDirectory()) {
        if (!['node_modules', 'dist', '__tests__'].includes(entry.name)) await visit(name)
      } else if (/\.(js|vue)$/.test(name) && !/\.(test|spec|cy)\./.test(name)) files.push(name)
    }
  }
  await visit(path.join(root, 'frontend/apps'))
  await visit(path.join(root, 'frontend/shared-ui'))
  for (const filename of files.sort()) {
    const extractor = config.extractors.find(e => e.match(filename))
    await extractor.extract(filename, await fs.readFile(filename, 'utf8'), msg => {
      if (!msg.message) msg = { ...msg, message: msg.id, id: generateMessageId(msg.id, msg.context) }
      const origin = [path.relative(root, msg.origin[0]), ...msg.origin.slice(1)]
      const existing = catalog[msg.id]
      if (existing && (existing.message !== msg.message || (existing.context || '') !== (msg.context || ''))) throw new Error(`Message ID collision: ${msg.id}`)
      if (existing) {
        if (!existing.origin.some(value => JSON.stringify(value) === JSON.stringify(origin))) existing.origin.push(origin)
      }
      else catalog[msg.id] = { ...msg, origin: [origin], translation: '' }
    }, { linguiConfig: config })
  }
  return Object.fromEntries(Object.entries(catalog).sort(([, a], [, b]) => `${a.context}\0${a.message}`.localeCompare(`${b.context}\0${b.message}`, 'en')))
}

async function pot() {
  const catalog = await extract()
  const result = format.serialize(catalog, { locale: 'en-US', sourceLocale: 'en-US', filename: 'locales/frontend.pot' })
  // The POT is a source template, not an English translation catalog.
  const po = PO.parse(result)
  delete po.headers.Language
  delete po.headers['Plural-Forms']
  delete po.headers['POT-Creation-Date']
  delete po.headers['PO-Revision-Date']
  for (const item of po.items) item.msgstr = item.msgid_plural ? ['', ''] : ['']
  return po.toString()
}

export function compilePO(contents, locale) {
  const po = PO.parse(contents)
  // Like msgfmt, exclude translations awaiting review so the UI falls back to English.
  po.items = po.items.filter(item => !item.obsolete && !item.flags.fuzzy && item.msgstr.every(value => value !== ''))
  const parsed = format.parse(po.toString(), { locale, sourceLocale: 'en-US' })
  const messages = Object.fromEntries(Object.entries(parsed).filter(([, item]) => !item.obsolete && item.translation).map(([id, item]) => [id, item.translation]))
  return compile(messages, locale)
}

function compile(messages, locale) {
  const result = createCompiledCatalog(locale, messages, { namespace: 'json', strict: true })
  if (result.errors.length) throw new Error(JSON.stringify(result.errors))
  return JSON.parse(result.source).messages
}

async function compiled() {
  const source = await extract()
  const result = { 'en-US': { locale: 'en-US', name: 'English', messages: compile(Object.fromEntries(Object.entries(source).map(([id, item]) => [id, item.message || id])), 'en-US') } }
  for (const entry of await fs.readdir(localeRoot, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name === 'en-US') continue
    const filename = path.join(localeRoot, entry.name, 'LC_MESSAGES/frontend.po')
    const contents = await fs.readFile(filename, 'utf8')
    execFileSync('msgfmt', ['--check', '--check-format', '-o', '/dev/null', filename])
    const po = PO.parse(contents)
    if (po.headers.Language !== entry.name) throw new Error(`Language header mismatch: ${filename}`)
    result[entry.name] = { locale: entry.name, name: po.headers['X-Language-Name'] || entry.name, messages: compilePO(contents, entry.name) }
  }
  return result
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (mode === 'extract') {
    const result = await pot()
    if (stdout) process.stdout.write(result)
    else await fs.writeFile(path.join(localeRoot, 'frontend.pot'), result)
  } else if (mode === 'compile' || mode === 'check') {
    const catalogs = await compiled()
    if (stdout) process.stdout.write(JSON.stringify(catalogs))
    else {
      if (mode === 'check' && await fs.readFile(path.join(localeRoot, 'frontend.pot'), 'utf8') !== await pot()) throw new Error('Run make i18n-extract: frontend.pot is outdated')
      for (const [locale, catalog] of Object.entries(catalogs)) {
        const filename = path.join(localeRoot, locale, 'LC_MESSAGES/frontend.json')
        const result = JSON.stringify(catalog) + '\n'
        if (mode === 'check') {
          if (await fs.readFile(filename, 'utf8') !== result) throw new Error(`Run make i18n-compile: ${filename} is outdated`)
          if (locale !== 'en-US') {
            const po = filename.replace(/\.json$/, '.po')
            // Check source coverage even for translations still awaiting review.
            // This does not decide whether fuzzy translations are compiled.
            for (const files of [[po, path.join(localeRoot, 'frontend.pot')], [path.join(localeRoot, 'frontend.pot'), po]]) execFileSync('msgcmp', ['--use-untranslated', '--use-fuzzy', ...files])
          }
        } else {
          await fs.mkdir(path.dirname(filename), { recursive: true })
          await fs.writeFile(filename, result)
        }
      }
    }
  } else throw new Error('Usage: node scripts/gettext.mjs extract|compile|check [--stdout]')
}
