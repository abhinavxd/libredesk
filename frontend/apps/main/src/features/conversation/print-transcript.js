import api from '@main/api'
import defaultTemplate from '@public-static/transcript.html?raw'

export async function printTranscript(text, title) {
  let transcriptTemplate = defaultTemplate
  try {
    const response = await api.getTranscriptTemplate()
    transcriptTemplate = response.data
  } catch (error) {
    if (error.response?.status !== 404) throw error
  }
  const template = new DOMParser().parseFromString(transcriptTemplate, 'text/html')
  const base = template.querySelector('base') || template.createElement('base')
  const templateDirectory = new URL('/static/public/static/', window.location.href)
  base.href = new URL(base.getAttribute('href') || '.', templateDirectory).href
  template.head.prepend(base)

  const frame = document.createElement('iframe')
  frame.title = title
  frame.setAttribute('sandbox', 'allow-same-origin allow-modals')
  frame.setAttribute('aria-hidden', 'true')
  frame.style.cssText = 'position: fixed; width: 0; height: 0; border: 0'

  const loaded = new Promise((resolve) => frame.addEventListener('load', resolve, { once: true }))
  frame.srcdoc = `<!doctype html>${template.documentElement.outerHTML}`
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
