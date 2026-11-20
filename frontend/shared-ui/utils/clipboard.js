// Copy text to the clipboard.
// Uses the async clipboard API when it is available and falls back to the legacy
// execCommand API, which also works in insecure contexts (http on a non-localhost
// origin) where navigator.clipboard is not exposed.
// Returns true if the text was copied.
export async function copyToClipboard(text) {
  const value = text == null ? '' : String(text)

  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // The write was blocked, try the legacy API before giving up.
    }
  }

  return legacyCopy(value)
}

// Copy using a hidden textarea and the deprecated execCommand API.
function legacyCopy(value) {
  if (typeof document === 'undefined' || !document.body) return false

  // Selecting the textarea steals focus, put it back where it was.
  const previouslyFocused = document.activeElement

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '-9999px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  textarea.setSelectionRange(0, value.length)

  let copied = false
  try {
    copied = document.execCommand('copy')
  } catch {
    copied = false
  }
  document.body.removeChild(textarea)
  previouslyFocused?.focus?.()
  return copied
}
