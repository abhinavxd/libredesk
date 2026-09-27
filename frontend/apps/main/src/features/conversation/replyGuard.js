/**
 * Outbound reply guard: finds text in a customer reply that probably belongs in a private note.
 *
 * Text rules only ever see the editor's plain text, never its HTML, so image data URIs and
 * attributes cannot match. Mention nodes are read from the HTML structurally.
 * Each rule returns the matched fragments; `label` is an i18n key.
 */

const MAX_EXCERPT_LENGTH = 120

// Letters, numbers and combining marks, in any script.
const WORD_CHAR = '[\\p{L}\\p{N}\\p{M}]'
const NON_WORD_CHAR = '[^\\p{L}\\p{N}\\p{M}]'
// Scripts written without spaces between words, where a phrase can only be found inside a run.
const UNSPACED_SCRIPT =
  '[\\p{sc=Han}\\p{sc=Hiragana}\\p{sc=Katakana}\\p{sc=Thai}\\p{sc=Lao}\\p{sc=Khmer}\\p{sc=Myanmar}]'
const NEEDS_BOUNDARY = new RegExp(`(?!${UNSPACED_SCRIPT})${WORD_CHAR}`, 'u')

// An "@handle" not preceded by a word character or an e-mail local part character.
const HANDLE_PATTERN =
  /(?:^|[^\p{L}\p{N}\p{M}_.+-])(@\p{L}(?:[\p{L}\p{N}\p{M}_.-]*[\p{L}\p{N}\p{M}_])?)/gu

const PLACEHOLDER_PATTERN = /\{\{[^{}]*\}\}/g

const SECRET_PATTERNS = [
  // JSON web token: three base64url segments, the header always starts with eyJ.
  /\beyJ[\w-]{5,}\.[\w-]{5,}\.[\w-]{5,}/g,
  /\bsk_(?:live|test)_[A-Za-z0-9]{16,}/g,
  /\bsk-[\w-]{20,}/g,
  /\bAKIA[0-9A-Z]{16}/g,
  /\bgh[po]_[A-Za-z0-9]{36,}/g,
  /\bgithub_pat_\w{22,}/g,
  /\bxox[bpa]-[A-Za-z0-9-]{10,}/g,
  /\bAIza[\w-]{35}/g
]

// 32+ base64/hex characters. Hyphens and slashes also join slugs, paths and UUIDs, so a run
// only counts when it holds a long unbroken alphanumeric stretch as well as letters and digits.
const RANDOM_RUN_PATTERN = /[A-Za-z0-9+/=_-]{32,}/g
const isRandomRun = (run) => /[A-Za-z]/.test(run) && /\d/.test(run) && /[A-Za-z0-9]{16,}/.test(run)
const isURL = (token) => /(?:[a-z][a-z0-9+.-]*:\/\/|www\.)/i.test(token)

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const matchAll = (text, pattern) => [...text.matchAll(pattern)].map((match) => match[1] ?? match[0])

/** Splits the admin's setting into phrases, one per line. */
export const parsePhrases = (value) => [
  ...new Set(
    (value || '')
      .split(/\r?\n/)
      .map((line) => line.trim())
      .filter(Boolean)
  )
]

/**
 * Case-insensitive whole-word matcher for a plain phrase. A boundary is only required on a
 * side that starts or ends with a word character, so a phrase like "#internal" still works,
 * and not next to a script written without spaces.
 */
const phrasePattern = (phrase) => {
  const chars = [...phrase]
  const prefix = NEEDS_BOUNDARY.test(chars[0]) ? `(?:^|${NON_WORD_CHAR})` : ''
  const suffix = NEEDS_BOUNDARY.test(chars.at(-1)) ? `(?!${WORD_CHAR})` : ''
  return new RegExp(`${prefix}(${escapeRegExp(phrase)})${suffix}`, 'iu')
}

/** Labels of the editor's mention nodes, e.g. "@Anna Smith". */
const mentionLabels = (html) => {
  if (!html || typeof DOMParser === 'undefined') return []
  const doc = new DOMParser().parseFromString(html, 'text/html')
  return [...doc.querySelectorAll('span[data-mention-suggestion-char], span.ld-mention')]
    .map((node) => node.textContent.trim())
    .filter(Boolean)
}

export const REPLY_GUARD_RULES = [
  {
    id: 'mentions',
    label: 'replyGuard.rule.mentions',
    test: ({ text, html }) => {
      const labels = mentionLabels(html)
      // A mention node renders as its label in the plain text, drop it before looking for handles.
      const rest = labels.reduce((value, label) => value.split(label).join(' '), text)
      return [...labels, ...matchAll(rest, HANDLE_PATTERN)]
    }
  },
  {
    id: 'placeholders',
    label: 'replyGuard.rule.placeholders',
    test: ({ text }) => matchAll(text, PLACEHOLDER_PATTERN)
  },
  {
    id: 'secrets',
    label: 'replyGuard.rule.secrets',
    test: ({ text }) => {
      const found = []
      // Blank out what a prefixed shape matched so the run check below does not report it again.
      const rest = SECRET_PATTERNS.reduce(
        (value, pattern) =>
          value.replace(pattern, (match) => {
            found.push(match)
            return ' '
          }),
        text
      )
      // Links often carry long ids, the prefixed shapes above still apply to them.
      for (const token of rest.split(/\s+/)) {
        if (isURL(token)) continue
        found.push(...matchAll(token, RANDOM_RUN_PATTERN).filter(isRandomRun))
      }
      return found
    }
  },
  {
    id: 'phrases',
    label: 'replyGuard.rule.phrases',
    test: ({ text, phrases }) => {
      const normalized = text.normalize('NFC')
      return parsePhrases(phrases)
        .map((phrase) => phrasePattern(phrase.normalize('NFC')).exec(normalized)?.[1])
        .filter(Boolean)
    }
  }
]

const truncate = (value) =>
  value.length > MAX_EXCERPT_LENGTH ? `${value.slice(0, MAX_EXCERPT_LENGTH - 1)}…` : value

/**
 * Returns `[{ rule, excerpt }]` for a reply. `text` is the editor's plain text, `html` its
 * content (only read for mention nodes) and `phrases` the admin's list, one per line.
 * Excerpts are deduplicated case-insensitively within a rule.
 */
export function findReplyGuardMatches({ text = '', html = '', phrases = '' } = {}) {
  const input = { text: typeof text === 'string' ? text : '', html, phrases }
  const matches = []
  for (const rule of REPLY_GUARD_RULES) {
    const seen = new Set()
    for (const excerpt of rule.test(input)) {
      const key = excerpt.toLowerCase()
      if (seen.has(key)) continue
      seen.add(key)
      matches.push({ rule: rule.id, excerpt: truncate(excerpt) })
    }
  }
  return matches
}

/** Folds matches into one group per rule for display. */
export function groupReplyGuardMatches(matches = []) {
  return REPLY_GUARD_RULES.map((rule) => ({
    rule: rule.id,
    label: rule.label,
    excerpts: matches.filter((match) => match.rule === rule.id).map((match) => match.excerpt)
  })).filter((group) => group.excerpts.length > 0)
}
