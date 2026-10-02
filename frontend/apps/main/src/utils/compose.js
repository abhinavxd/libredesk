import { validateEmail } from '@shared-ui/utils/string'

// Shared rules for the Compose New dialog, kept free of Vue for testing.

export const splitRecipients = (value) =>
  String(value || '')
    .split(/[,;\n]/)
    .map((entry) => entry.trim())
    .filter(Boolean)

export const joinRecipients = (list) => (Array.isArray(list) ? list.join(', ') : '')

// The address a new email starts from: the one being viewed, else the first usable one.
export function defaultFromAddress(addresses, preferredID) {
  const usable = addresses.filter((address) => address.enabled)
  const preferred = usable.find((address) => String(address.id) === String(preferredID))
  return (preferred || usable[0])?.id ?? null
}

// Returns i18n keys (with values) describing what stops the email from being sent.
export function composeProblems({ addressID, to, cc, bcc, subject, hasBody }) {
  const problems = []
  if (!addressID) problems.push({ key: 'compose.fromRequired' })
  const recipients = { to: splitRecipients(to), cc: splitRecipients(cc), bcc: splitRecipients(bcc) }
  if (!recipients.to.length) problems.push({ key: 'compose.toRequired' })
  for (const [field, list] of Object.entries(recipients)) {
    const invalid = list.filter((email) => !validateEmail(email))
    if (invalid.length) problems.push({ key: 'compose.invalidRecipients', values: { field: field.toUpperCase(), list: invalid.join(', ') } })
  }
  if (!String(subject || '').trim()) problems.push({ key: 'compose.subjectRequired' })
  if (!hasBody) problems.push({ key: 'compose.bodyRequired' })
  return problems
}
