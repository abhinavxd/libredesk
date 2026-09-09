import { describe, expect, it } from 'vitest'
import {
  findReplyGuardMatches,
  groupReplyGuardMatches,
  isGuardedMessageType,
  REPLY_GUARD_RULES
} from './replyGuard'

const excerptsFor = (text, rule, options) =>
  findReplyGuardMatches(text, options)
    .filter((match) => match.rule === rule)
    .map((match) => match.excerpt)

describe('reply guard', () => {
  it('reports nothing for a clean customer reply', () => {
    expect(
      findReplyGuardMatches(
        'Hi Anna, the refund is on its way and should reach your card in three working days.'
      )
    ).toEqual([])
  })

  it('reports nothing for empty or non-string content', () => {
    expect(findReplyGuardMatches('')).toEqual([])
    expect(findReplyGuardMatches('   \n  ')).toEqual([])
    expect(findReplyGuardMatches(undefined)).toEqual([])
  })

  it('exposes rules as { id, label, test }', () => {
    for (const rule of REPLY_GUARD_RULES) {
      expect(typeof rule.id).toBe('string')
      expect(typeof rule.label).toBe('string')
      expect(typeof rule.test).toBe('function')
    }
  })

  describe('mentions', () => {
    it('flags an at-mention', () => {
      expect(excerptsFor('Sending this over, @maria please double check.', 'mentions')).toEqual([
        '@maria'
      ])
    })

    it('flags a mention at the very start of the text', () => {
      expect(excerptsFor('@Anna took a look already.', 'mentions')).toEqual(['@Anna'])
    })

    it('treats combining marks as part of a word', () => {
      // "Slava" followed by a combining accent is still one word, not the bare internal name.
      expect(excerptsFor('Ask Slavá to confirm.', 'internalVocabulary')).toEqual([])
    })

    it('flags a mention written with non-ASCII letters', () => {
      expect(excerptsFor('Передаю @Мария, посмотри пожалуйста.', 'mentions')).toEqual(['@Мария'])
    })

    it('does not flag an email address with a non-ASCII local part', () => {
      expect(excerptsFor('Write to josé@example.com or to 田中@example.jp.', 'mentions')).toEqual([])
    })

    it('does not flag the at sign inside an email address', () => {
      expect(
        excerptsFor('Write to support@libredesk.io or to anna.k+desk@example.co.uk.', 'mentions')
      ).toEqual([])
    })

    it('does not flag a bare at sign used as a word', () => {
      expect(excerptsFor('Let us meet @ 5pm.', 'mentions')).toEqual([])
    })

    it('drops trailing punctuation from the excerpt', () => {
      expect(excerptsFor('Handing to @anna.', 'mentions')).toEqual(['@anna'])
    })

    it('folds in the editor mention nodes', () => {
      expect(
        excerptsFor('Please check.', 'mentions', { mentions: [{ id: 7, label: 'Support team' }] })
      ).toEqual(['@Support team'])
    })

    it('does not report an editor mention twice when the text already carries it', () => {
      expect(
        excerptsFor('@Anna can you look?', 'mentions', { mentions: [{ id: 3, label: 'anna' }] })
      ).toEqual(['@Anna'])
    })
  })

  describe('handoff headings', () => {
    it('flags a heading at the start of a line', () => {
      expect(
        excerptsFor('Thanks for waiting.\nDraft reply below\nHello there', 'handoffHeadings')
      ).toEqual(['Draft reply below'])
    })

    it('is case insensitive and matches every listed heading', () => {
      const text = [
        'draft reply',
        'PROPOSED ACTION: refund',
        'Question for the team',
        'Why: the card expired',
        'Private note',
        'Internal'
      ].join('\n')
      expect(excerptsFor(text, 'handoffHeadings')).toEqual([
        'draft reply',
        'PROPOSED ACTION: refund',
        'Question for the team',
        'Why: the card expired',
        'Private note',
        'Internal'
      ])
    })

    it('does not flag a heading word in the middle of a line', () => {
      expect(excerptsFor('We will send you a draft reply tomorrow.', 'handoffHeadings')).toEqual([])
    })

    it('does not flag a longer word that merely starts with a heading', () => {
      expect(excerptsFor('Internally we call it batch two.', 'handoffHeadings')).toEqual([])
    })
  })

  describe('placeholders', () => {
    it('flags square bracket slots, template expressions and insert markers', () => {
      expect(
        excerptsFor('Dear [customer name], {{ .Contact.Email }} <insert order id>', 'placeholders')
      ).toEqual(['[customer name]', '{{ .Contact.Email }}', '<insert order id>'])
    })

    it('flags leftover work markers whole-word', () => {
      expect(excerptsFor('TODO check the invoice, FIXME later, TBD, XXX', 'placeholders')).toEqual([
        'TODO',
        'FIXME',
        'TBD',
        'XXX'
      ])
    })

    it('flags lorem ipsum', () => {
      expect(excerptsFor('Lorem ipsum dolor sit amet.', 'placeholders')).toEqual(['Lorem ipsum'])
    })

    it('does not flag a numeric footnote', () => {
      expect(excerptsFor('See the manual [1] and [2].', 'placeholders')).toEqual([])
    })

    it('does not flag a word that merely contains a marker', () => {
      expect(excerptsFor('The order was inserted into the queue.', 'placeholders')).toEqual([])
    })
  })

  describe('internal vocabulary', () => {
    it('flags agent names, systems, secrets and meta phrases', () => {
      expect(
        excerptsFor(
          'Claude opened a pull request on GitHub; the API key sits on the VPS. Do not send this.',
          'internalVocabulary'
        )
      ).toEqual(['Claude', 'pull request', 'GitHub', 'API key', 'VPS', 'Do not send'])
    })

    it('matches whole words only', () => {
      expect(excerptsFor('We use PostgreSQL and a tokenizer.', 'internalVocabulary')).toEqual([])
    })

    it('tolerates a plural', () => {
      expect(excerptsFor('Rotate the tokens and the credentials.', 'internalVocabulary')).toEqual([
        'tokens',
        'credentials'
      ])
    })

    it('reports the longest matching term once', () => {
      expect(excerptsFor('Open desk.pmslava.com now.', 'internalVocabulary')).toEqual([
        'desk.pmslava.com'
      ])
    })

    it('flags a host and a loopback address', () => {
      expect(excerptsFor('Point it at 127.0.0.1 or sync.drifttt.com.', 'internalVocabulary')).toEqual(
        ['127.0.0.1', 'sync.drifttt.com']
      )
    })

    it('deduplicates repeated terms case-insensitively', () => {
      expect(excerptsFor('Deploy, deploy and deploy again.', 'internalVocabulary')).toEqual([
        'Deploy'
      ])
    })
  })

  describe('grouping', () => {
    it('folds matches into one entry per rule, in rule order', () => {
      const matches = findReplyGuardMatches('Draft reply for @anna\nTODO: mention the migration')
      expect(groupReplyGuardMatches(matches)).toEqual([
        { rule: 'mentions', label: 'replyGuard.rule.mentions', excerpts: ['@anna'] },
        {
          rule: 'handoffHeadings',
          label: 'replyGuard.rule.handoffHeadings',
          excerpts: ['Draft reply for @anna']
        },
        { rule: 'placeholders', label: 'replyGuard.rule.placeholders', excerpts: ['TODO'] },
        {
          rule: 'internalVocabulary',
          label: 'replyGuard.rule.internalVocabulary',
          excerpts: ['migration']
        }
      ])
    })

    it('returns no groups for a clean reply', () => {
      expect(groupReplyGuardMatches(findReplyGuardMatches('All sorted, thanks!'))).toEqual([])
    })
  })

  describe('message types', () => {
    it('guards a reply and exempts a private note', () => {
      expect(isGuardedMessageType('reply')).toBe(true)
      expect(isGuardedMessageType('private_note')).toBe(false)
    })
  })
})
