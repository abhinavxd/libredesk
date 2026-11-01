export async function printTranscript(text, title) {
  const frame = document.createElement('iframe')
  frame.title = title
  frame.setAttribute('sandbox', 'allow-same-origin allow-modals')
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText = 'position: fixed; width: 0; height: 0; border: 0'

  const loaded = new Promise((resolve) => frame.addEventListener('load', resolve, { once: true }))
  frame.srcdoc = '<!doctype html><html><head><meta charset="utf-8"></head><body></body></html>'
  document.body.appendChild(frame)

  try {
    await loaded
    const doc = frame.contentDocument
    doc.title = title
    doc.documentElement.lang = document.documentElement.lang

    const style = doc.createElement('style')
    style.textContent = `
      @page { size: A4; margin: 12.7mm; }
      body { margin: 0; }
      pre {
        font: 10pt/1.4 system-ui, sans-serif;
        margin: 0;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
        unicode-bidi: plaintext;
        tab-size: 4;
      }
    `
    doc.head.appendChild(style)

    const content = doc.createElement('pre')
    content.textContent = text
    doc.body.appendChild(content)
    await doc.fonts.ready

    frame.contentWindow.addEventListener('afterprint', () => frame.remove(), { once: true })
    frame.contentWindow.print()
  } catch (error) {
    frame.remove()
    throw error
  }
}
