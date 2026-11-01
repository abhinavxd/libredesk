import api from '@main/api'

export async function printTranscript(text, title) {
  const { data: transcriptTemplate } = await api.getTranscriptTemplate()
  const frame = document.createElement('iframe')
  frame.title = title
  frame.setAttribute('sandbox', 'allow-same-origin allow-modals')
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText = 'position: fixed; width: 0; height: 0; border: 0'

  const loaded = new Promise((resolve) => frame.addEventListener('load', resolve, { once: true }))
  frame.srcdoc = transcriptTemplate
  document.body.appendChild(frame)

  try {
    await loaded
    const doc = frame.contentDocument
    doc.title = title
    doc.documentElement.lang = document.documentElement.lang

    const headingEnd = text.indexOf('\n')
    doc.querySelector('[data-transcript-title]').textContent =
      headingEnd === -1 ? text : text.slice(0, headingEnd)
    doc.querySelector('[data-transcript-body]').textContent =
      headingEnd === -1 ? '' : text.slice(headingEnd + 1)
    await doc.fonts.ready

    frame.contentWindow.addEventListener('afterprint', () => frame.remove(), { once: true })
    frame.contentWindow.print()
  } catch (error) {
    frame.remove()
    throw error
  }
}
