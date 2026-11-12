import { transformAsync } from '@babel/core'
import { parse, compileTemplate } from 'vue/compiler-sfc'
import babel from '@lingui/cli/api/extractors/babel'
import { generateMessageId } from '@lingui/message-utils/generateMessageId'
import { pluralMessage } from '../shared-ui/i18n/plural.js'

// Supplement the official extractors with the two Gettext-style Vue helpers.
// Parse JavaScript syntax rather than matching calls with regular expressions.
export async function extractGettextHelpers(filename, code, onMessage, ctx) {
  if (!code.includes('ngettext') && !code.includes('pgettext')) return
  let parts = [{ source: code }]
  if (filename.endsWith('.vue')) {
    const { descriptor, errors } = parse(code, { filename })
    if (errors.length) throw errors[0]
    const isTs = descriptor.script?.lang === 'ts' || descriptor.scriptSetup?.lang === 'ts'
    const template = descriptor.template && compileTemplate({
      source: descriptor.template.content,
      filename,
      id: filename,
      inMap: descriptor.template.map,
      compilerOptions: { isTS: isTs },
    })
    if (template?.errors.length) throw template.errors[0]
    parts = [
      descriptor.script && { source: descriptor.script.content, map: descriptor.script.map },
      descriptor.scriptSetup && { source: descriptor.scriptSetup.content, map: descriptor.scriptSetup.map },
      template && { source: template.code, map: template.map },
    ].filter(Boolean).map(part => ({ ...part, isTs }))
  }

  for (const part of parts) {
    const ids = new Set()
    const result = await transformAsync(part.source, {
      filename: filename + (part.isTs ? '.ts' : '.js'),
      parserOpts: { plugins: part.isTs ? ['typescript'] : [] },
      babelrc: false,
      configFile: false,
      inputSourceMap: part.map,
      sourceMaps: true,
      plugins: [({ types: t }) => ({
        visitor: {
          CallExpression(path) {
            const { callee, arguments: args } = path.node
            if (!t.isMemberExpression(callee) || callee.computed) return
            const object = callee.object
            const isI18n = t.isIdentifier(object, { name: 'i18n' }) ||
              (t.isMemberExpression(object) && t.isIdentifier(object.property, { name: 'i18n' }))
            if (!isI18n || !['ngettext', 'pgettext'].includes(callee.property.name)) return
            const literal = (index, fallback) => {
              if (args[index] === undefined && fallback !== undefined) return fallback
              if (!t.isStringLiteral(args[index])) {
                throw path.buildCodeFrameError('Gettext messages and contexts must be string literals')
              }
              return args[index].value
            }
            const plural = callee.property.name === 'ngettext'
            const message = plural ? pluralMessage(literal(0), literal(1)) : literal(1)
            const context = plural ? literal(4, '') : literal(0)
            const id = generateMessageId(message, context)
            ids.add(id)
            // Let Lingui's extractor handle descriptors and source-map positions.
            const descriptor = t.objectExpression(Object.entries({ id, message, context }).map(
              ([key, value]) => t.objectProperty(t.identifier(key), t.stringLiteral(value))
            ))
            t.inherits(descriptor, path.node)
            path.replaceWith(t.callExpression(t.memberExpression(t.identifier('i18n'), t.identifier('_')), [descriptor]))
          },
        },
      })],
    })
    if (!ids.size) continue
    await babel.extract(filename + (part.isTs ? '.ts' : '.js'), result.code, message => {
      if (ids.has(message.id)) onMessage(message)
    }, { ...ctx, sourceMaps: result.map })
  }
}

export function withGettextHelpers(extractor) {
  return {
    ...extractor,
    async extract(filename, code, onMessage, ctx) {
      await extractor.extract(filename, code, onMessage, ctx)
      await extractGettextHelpers(filename, code, onMessage, ctx)
    },
  }
}
