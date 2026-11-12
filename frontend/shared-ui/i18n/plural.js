// Shared by extraction and runtime so both address the same Lingui message.
export function pluralMessage(singular, plural) {
  return `{count, plural, one {${singular}} other {${plural}}}`
}
