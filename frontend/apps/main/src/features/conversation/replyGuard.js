/**
 * Outbound reply guard.
 *
 * A reply leaves the desk with one keystroke, so text meant for the team - an
 * agent mention, a heading from a drafted handoff, an unfilled placeholder, an
 * internal system name - is easy to send to the customer by mistake. This module
 * is the pure half of the guard: it scans the composer's plain text and reports
 * what a human should look at before the message goes out. It decides nothing;
 * the composer shows the dialog and only an explicit confirmation sends.
 *
 * Every rule has the shape `{ id, label, test(text, context) -> string[] }`,
 * where `label` is an i18n key and `test` returns the matched fragments. Rules
 * live in `REPLY_GUARD_RULES` so an admin-configurable list can replace or
 * extend this built-in set later without touching the composer.
 */

// Message type that is never guarded: a private note is internal by definition.
const PRIVATE_NOTE_MESSAGE_TYPE = 'private_note'

const MAX_EXCERPT_LENGTH = 120

/**
 * Words and names that belong to the team, not to the customer. Matched
 * whole-word and case-insensitively, tolerating a plural "s". Kept as one flat
 * list so it stays cheap to edit.
 */
export const INTERNAL_VOCABULARY = [
  // Agent and people names.
  'Claude',
  'Codex',
  'Juno',
  'Slava',
  'Maria',
  'Mariia',
  // Systems and infrastructure.
  'Libredesk',
  'desk.pmslava.com',
  'pmslava',
  'ntfy',
  'Joplin',
  'VPS',
  'Quasar',
  'jim',
  'PowerSync',
  'Paddle',
  'RevenueCat',
  'Brevo',
  'Migadu',
  'Authelia',
  'Vaultwarden',
  'GitHub',
  'pull request',
  'PR #',
  'commit',
  'branch',
  'deploy',
  'migration',
  'psql',
  'SQL',
  'sudo',
  'docker',
  'systemd',
  'cron',
  'webhook',
  'endpoint',
  'localhost',
  '127.0.0.1',
  'api.drifttt.com',
  'sync.drifttt.com',
  'stats.drifttt.com',
  // Secrets.
  'API key',
  'secret',
  'token',
  'password',
  'credential',
  // Meta phrases about the message itself.
  'do not send',
  "don't send",
  'don’t send',
  'not for the customer',
  'internal only',
  'inner',
  'kitchen'
]

/** Headings an AI or an agent writes when handing work to the team. */
const HANDOFF_HEADINGS = [
  'Draft reply',
  'Proposed action',
  'Question for',
  'Why:',
  'Private note',
  'Internal'
]

/** Placeholder markers that mean the draft was never finished. */
const PLACEHOLDER_WORDS = ['TODO', 'FIXME', 'TBD', 'XXX', 'lorem ipsum']

const escapeRegExp = (value) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const HANDOFF_HEADING_PATTERN = new RegExp(
  `^(?:${HANDOFF_HEADINGS.map(escapeRegExp).join('|')})(?![\\w-])`,
  'i'
)

const isWordCharacter = (character) => /[A-Za-z0-9_]/.test(character)

/**
 * A whole-word matcher for an arbitrary term, including terms carrying dots,
 * spaces or a trailing "#". Word boundaries are only required on the sides that
 * actually start or end on a word character, so "PR #" and "127.0.0.1" behave.
 * A trailing plural "s" is accepted so "tokens" trips the "token" term.
 * The leading boundary is consumed rather than looked behind, so the term is
 * captured in group 1.
 */
const termPattern = (term) => {
  const prefix = isWordCharacter(term[0]) ? '(?:^|[^A-Za-z0-9_])' : ''
  const suffix = isWordCharacter(term[term.length - 1]) ? 's?(?![A-Za-z0-9_])' : ''
  return new RegExp(`${prefix}(${escapeRegExp(term)}${suffix})`, 'gi')
}

const truncate = (value) =>
  value.length > MAX_EXCERPT_LENGTH ? `${value.slice(0, MAX_EXCERPT_LENGTH - 1)}…` : value

/** Collects every match of `pattern`, preferring capture group 1 when present. */
const collectMatches = (text, pattern, into) => {
  let match
  while ((match = pattern.exec(text)) !== null) {
    if (match[0] === '') {
      pattern.lastIndex += 1
      continue
    }
    into.push({ index: match.index + match[0].indexOf(match[1] ?? match[0]), value: match[1] ?? match[0] })
  }
  return into
}

/**
 * Matches a list of terms, longest first, dropping any match that falls inside
 * an already matched span so "desk.pmslava.com" is not also reported as
 * "pmslava". Results come back in the order they appear in the text.
 */
const matchTerms = (text, terms) => {
  const claimed = []
  const found = []
  const byLengthDesc = [...terms].sort((a, b) => b.length - a.length)

  for (const term of byLengthDesc) {
    for (const match of collectMatches(text, termPattern(term), [])) {
      const start = match.index
      const end = start + match.value.length
      if (claimed.some((span) => start < span.end && end > span.start)) continue
      claimed.push({ start, end })
      found.push(match)
    }
  }

  return found.sort((a, b) => a.index - b.index).map((match) => match.value)
}

const mentionLabel = (mention) => {
  const label = mention?.label ?? mention?.name ?? mention?.id
  if (label === undefined || label === null) return null
  const text = String(label).trim()
  return text ? `@${text.replace(/^@+/, '')}` : null
}

export const REPLY_GUARD_RULES = [
  {
    id: 'mentions',
    label: 'replyGuard.rule.mentions',
    // The editor's mention nodes render as "@Label" in the plain text, so the
    // text scan already covers them; the structured list is folded in as well
    // so a mention whose label is not "@word" shaped is still reported. The "@"
    // must not be preceded by a word character, which is what keeps an email
    // address from reading as a mention.
    test: (text, { mentions = [] } = {}) => {
      const found = collectMatches(
        text,
        /(?:^|[^A-Za-z0-9_.@+-])(@[A-Za-z0-9](?:[\w.'’-]*[A-Za-z0-9])?)/g,
        []
      ).map((match) => match.value)
      for (const mention of mentions) {
        const label = mentionLabel(mention)
        if (label) found.push(label)
      }
      return found
    }
  },
  {
    id: 'handoffHeadings',
    label: 'replyGuard.rule.handoffHeadings',
    test: (text) =>
      text
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => HANDOFF_HEADING_PATTERN.test(line))
        .map(truncate)
  },
  {
    id: 'placeholders',
    label: 'replyGuard.rule.placeholders',
    test: (text) => {
      const found = []
      // "[name]" style slots: at least one letter, so footnotes like "[1]" pass.
      collectMatches(text, /\[[^[\]\n]*[A-Za-z][^[\]\n]*\]/g, found)
      // Unrendered template expressions.
      collectMatches(text, /\{\{[\s\S]{0,200}?\}\}/g, found)
      collectMatches(text, /<insert\b[^>\n]{0,60}>?/gi, found)
      for (const word of PLACEHOLDER_WORDS) collectMatches(text, termPattern(word), found)
      return found.sort((a, b) => a.index - b.index).map((match) => truncate(match.value))
    }
  },
  {
    id: 'internalVocabulary',
    label: 'replyGuard.rule.internalVocabulary',
    test: (text) => matchTerms(text, INTERNAL_VOCABULARY)
  }
]

/**
 * True when a composer message type is customer facing and therefore guarded.
 * Private notes never are.
 */
export const isGuardedMessageType = (messageType) => messageType !== PRIVATE_NOTE_MESSAGE_TYPE

/**
 * Scans plain composer text and returns `[{ rule, excerpt }]`, where `rule` is a
 * rule id from `REPLY_GUARD_RULES`. Excerpts are deduplicated case-insensitively
 * within a rule and ordered rule by rule.
 */
export function findReplyGuardMatches(text, { mentions = [] } = {}) {
  const value = typeof text === 'string' ? text : ''
  const matches = []
  if (!value.trim() && !mentions.length) return matches

  for (const rule of REPLY_GUARD_RULES) {
    const seen = new Set()
    for (const excerpt of rule.test(value, { mentions }) || []) {
      const trimmed = String(excerpt).trim()
      if (!trimmed) continue
      const key = trimmed.toLowerCase()
      if (seen.has(key)) continue
      seen.add(key)
      matches.push({ rule: rule.id, excerpt: trimmed })
    }
  }

  return matches
}

/** Folds matches into one entry per rule for display. */
export function groupReplyGuardMatches(matches = []) {
  return REPLY_GUARD_RULES.map((rule) => ({
    rule: rule.id,
    label: rule.label,
    excerpts: matches.filter((match) => match.rule === rule.id).map((match) => match.excerpt)
  })).filter((group) => group.excerpts.length > 0)
}
