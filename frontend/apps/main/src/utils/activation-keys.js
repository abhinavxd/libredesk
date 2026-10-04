// Activation key placeholders in composer HTML. The chip stands for one key;
// the server swaps it for a real key only in the email it sends.
export const KEY_PLACEHOLDER_TYPE = 'activation-key'
const SELECTOR = `span[data-type="${KEY_PLACEHOLDER_TYPE}"]`

export const keyPlaceholderLabel = (id) => `<KEY_ID_${id}>`

// A 9-digit ID, unique enough within one email; the server rejects repeats.
export function newKeyPlaceholderID() {
  const values = new Uint32Array(1)
  crypto.getRandomValues(values)
  return String(100000000 + (values[0] % 900000000))
}

// readKeyPlaceholders lists the chips in an HTML string, in order.
export function readKeyPlaceholders(html) {
  if (typeof html !== 'string' || !html.includes(KEY_PLACEHOLDER_TYPE)) return []
  const template = document.createElement('template')
  template.innerHTML = html
  return [...template.content.querySelectorAll(SELECTOR)].map((el) => ({
    id: el.getAttribute('data-id') || '',
    appId: Number(el.getAttribute('data-app-id')) || 0
  }))
}

// stripKeyPlaceholders removes chips from pasted HTML. A chip only ever comes
// from the key card, so copying one never adds a key.
export function stripKeyPlaceholders(html) {
  if (typeof html !== 'string' || !html.includes(KEY_PLACEHOLDER_TYPE)) return html
  const template = document.createElement('template')
  template.innerHTML = html
  template.content.querySelectorAll(SELECTOR).forEach((el) => el.remove())
  return template.innerHTML
}
