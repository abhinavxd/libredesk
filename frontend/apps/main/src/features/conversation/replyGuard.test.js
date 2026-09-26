// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'
import { Editor } from '@tiptap/vue-3'
import { buildConversationExtensions } from '@main/components/editor/editorExtensions'
import { findReplyGuardMatches, groupReplyGuardMatches, parsePhrases } from './replyGuard'

const excerptsFor = (rule, input) =>
  findReplyGuardMatches(typeof input === 'string' ? { text: input } : input)
    .filter((match) => match.rule === rule)
    .map((match) => match.excerpt)

// Built by concatenation so the fake keys never appear whole in the source.
const fake = (prefix, body) => prefix + body

describe('reply guard', () => {
  it('reports nothing for empty content', () => {
    expect(findReplyGuardMatches()).toEqual([])
    expect(findReplyGuardMatches({ text: '  \n ' })).toEqual([])
    expect(findReplyGuardMatches({ text: undefined })).toEqual([])
  })

  describe('text that must not trigger', () => {
    it.each([
      ['a URL', 'See https://example.com/help/articles/reset-your-password-2024?ref=email#step-2'],
      [
        'a URL with a long id',
        'The file is at https://docs.example.com/document/d/1a2B3c4D5e6F7g8H9i0JkLmNoPqRsTuVwXyZ/edit'
      ],
      ['an e-mail address', 'Write to support@example.com or anna.k+desk@mail.example.co.uk.'],
      [
        'a Cyrillic sentence',
        'Здравствуйте, Мария! Ваш заказ отправлен и прибудет в течение трёх дней.'
      ],
      ['an order number', 'Your order #A-10234-7788 (invoice INV-2024-000123) has shipped.'],
      ['a UUID', 'Reference 123e4567-e89b-12d3-a456-426614174000 for your records.'],
      [
        'a normal English paragraph',
        'Hi Anna, thanks for your patience. The refund of $49.99 is on its way and should reach ' +
          'your card within 3-5 working days. Let us meet @ 5pm if you have questions.'
      ]
    ])('%s', (_, text) => {
      expect(findReplyGuardMatches({ text })).toEqual([])
    })

    it('ignores the HTML, so an inline image data URI cannot match', () => {
      const data =
        'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg'
      const html = `<p>Screenshot attached.</p><img src="data:image/png;base64,${data}" alt="@anna {{ x }}">`
      expect(findReplyGuardMatches({ text: 'Screenshot attached.', html })).toEqual([])
    })
  })

  describe('mentions', () => {
    it('flags a plain-text handle', () => {
      expect(excerptsFor('mentions', 'Sending this over, @maria please check.')).toEqual(['@maria'])
      expect(excerptsFor('mentions', '@Anna took a look already.')).toEqual(['@Anna'])
    })

    it('flags a handle in any script and drops trailing punctuation', () => {
      expect(excerptsFor('mentions', 'Передаю @Мария.')).toEqual(['@Мария'])
      expect(excerptsFor('mentions', '田中さん (@田中) に確認します')).toEqual(['@田中'])
    })

    it('does not flag e-mail addresses, including non-ASCII local parts', () => {
      expect(excerptsFor('mentions', 'Write to josé@example.com or 田中@example.jp.')).toEqual([])
      expect(excerptsFor('mentions', 'Write to jose\u0301@example.com.')).toEqual([])
    })

    it('reads mention nodes from the editor content', () => {
      const editor = new Editor({
        extensions: buildConversationExtensions({ getPlaceholder: () => '' }),
        content: {
          type: 'doc',
          content: [
            {
              type: 'paragraph',
              content: [
                { type: 'mention', attrs: { id: '7', label: 'Anna Smith', type: 'agent' } },
                { type: 'text', text: ' can you check this?' }
              ]
            }
          ]
        }
      })
      const input = { text: editor.getText(), html: editor.getHTML() }
      editor.destroy()
      expect(input.text).toBe('@Anna Smith can you check this?')
      expect(excerptsFor('mentions', input)).toEqual(['@Anna Smith'])
    })
  })

  describe('placeholders', () => {
    it('flags unfilled template expressions', () => {
      expect(
        excerptsFor('placeholders', 'Dear {{ .Contact.FirstName }}, your code is {{code}}.')
      ).toEqual(['{{ .Contact.FirstName }}', '{{code}}'])
    })

    it('does not flag single braces', () => {
      expect(excerptsFor('placeholders', 'Use {name} or { } in the form.')).toEqual([])
    })
  })

  describe('secrets', () => {
    it.each([
      [
        'a JSON web token',
        fake(
          'eyJ',
          'hbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U'
        )
      ],
      ['an sk- key', fake('sk-', 'proj-4f9aB2cD8eF1gH7iJ3kL5mN6')],
      ['an sk_live_ key', fake('sk_live_', '51HqLyjWDarjtT1zdp7dc')],
      ['an sk_test_ key', fake('sk_test_', '4eC39HqLyjWDarjtT1zdp7dc')],
      ['an AWS access key id', fake('AKIA', 'IOSFODNN7EXAMPLE')],
      ['a ghp_ token', fake('ghp_', 'a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8')],
      ['a gho_ token', fake('gho_', 'a1B2c3D4e5F6g7H8i9J0k1L2m3N4o5P6q7R8')],
      ['a github_pat_ token', fake('github_pat_', '11ABCDEFG0123456789_abcdefghijklmnop')],
      ['an xoxb- token', fake('xoxb-', '1234567890-abcdefABCDEF')],
      ['an xoxp- token', fake('xoxp-', '1234567890-abcdefABCDEF')],
      ['an xoxa- token', fake('xoxa-', '1234567890-abcdefABCDEF')],
      ['a Google API key', fake('AIza', 'SyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q')],
      ['a long hex run', fake('', '9f86d081884c7d659a2feaa0c55ad015a3bf4f1b')],
      ['a long base64 run', fake('', 'q8Zp3Rt+Wx/7bN2mLk5Hj9Gf4Ds1Aa0Qe6Yu=')]
    ])('flags %s', (_, secret) => {
      expect(excerptsFor('secrets', `Here you go: ${secret} thanks`)).toEqual([secret])
    })

    it('still flags a key inside a URL', () => {
      const key = fake('AIza', 'SyA1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q')
      expect(excerptsFor('secrets', `https://maps.example.com/api?key=${key}`)).toEqual([key])
    })

    it('does not flag a long word or a run without digits', () => {
      expect(excerptsFor('secrets', 'Donaudampfschifffahrtsgesellschaftskapitänsmütze')).toEqual([])
      expect(excerptsFor('secrets', 'abcdefghijklmnopqrstuvwxyzabcdefghij')).toEqual([])
    })
  })

  describe('admin phrases', () => {
    it('parses one phrase per line, trimmed and without blanks or duplicates', () => {
      expect(parsePhrases(' Juno \r\n\nwarehouse B\nJuno\n')).toEqual(['Juno', 'warehouse B'])
      expect(parsePhrases(undefined)).toEqual([])
    })

    it('matches whole words, ignoring case', () => {
      const phrases = 'juno\nwarehouse b'
      expect(excerptsFor('phrases', { text: 'Ask JUNO about Warehouse B.', phrases })).toEqual([
        'JUNO',
        'Warehouse B'
      ])
      expect(excerptsFor('phrases', { text: 'Junos and warehouse bay', phrases })).toEqual([])
    })

    it('matches non-Latin words as whole words', () => {
      const phrases = 'Мария'
      expect(excerptsFor('phrases', { text: 'Спросите у мария, пожалуйста.', phrases })).toEqual([
        'мария'
      ])
      expect(excerptsFor('phrases', { text: 'Марияна и ПраМария здесь.', phrases })).toEqual([])
    })

    it('finds a phrase inside a run of a script written without spaces', () => {
      expect(excerptsFor('phrases', { text: '担当は田中です', phrases: '田中' })).toEqual(['田中'])
      expect(excerptsFor('phrases', { text: 'ติดต่อสมชายได้เลย', phrases: 'สมชาย' })).toEqual([
        'สมชาย'
      ])
    })

    it('treats combining marks as part of a word', () => {
      expect(
        excerptsFor('phrases', { text: 'Ask Juno\u0332 to confirm.', phrases: 'Juno' })
      ).toEqual([])
    })

    it('matches composed and decomposed forms alike', () => {
      expect(excerptsFor('phrases', { text: 'Ask Jose\u0301.', phrases: 'Jos\u00e9' })).toEqual([
        'Jos\u00e9'
      ])
    })

    it('takes phrases literally, not as patterns', () => {
      expect(excerptsFor('phrases', { text: 'one plus one', phrases: 'o.e\n.*' })).toEqual([])
      expect(excerptsFor('phrases', { text: 'Tagged #internal.', phrases: '#internal' })).toEqual([
        '#internal'
      ])
    })

    it('runs only the built-in rules when the list is empty', () => {
      expect(findReplyGuardMatches({ text: 'Ask Juno.', phrases: '' })).toEqual([])
    })
  })

  describe('grouping', () => {
    it('folds matches into one entry per rule, in rule order', () => {
      const matches = findReplyGuardMatches({
        text: 'Juno says {{ name }} is ready, @anna',
        phrases: 'juno'
      })
      expect(groupReplyGuardMatches(matches)).toEqual([
        { rule: 'mentions', label: 'replyGuard.rule.mentions', excerpts: ['@anna'] },
        { rule: 'placeholders', label: 'replyGuard.rule.placeholders', excerpts: ['{{ name }}'] },
        { rule: 'phrases', label: 'replyGuard.rule.phrases', excerpts: ['Juno'] }
      ])
    })
  })
})
