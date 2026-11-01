// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { printTranscript } from './print-transcript'
import api from '@main/api'
import transcriptTemplate from '@public-static/transcript.html?raw'

vi.mock('@main/api', () => ({ default: { getTranscriptTemplate: vi.fn() } }))

const customTemplate = `<!doctype html><html>
  <head><link rel="stylesheet" href="pdf.css"></head>
  <body><img src="logo.png" alt="Company logo"><h2 data-transcript-title></h2><div data-transcript-body></div></body>
</html>`

describe('printTranscript', () => {
  beforeEach(() => {
    window.history.replaceState(null, '', '/inboxes/all/conversation/123')
  })

  afterEach(() => {
    document.body.replaceChildren()
    window.history.replaceState(null, '', '/')
    vi.restoreAllMocks()
  })

  it.each([
    ['default', transcriptTemplate, null, '/static/public/static/'],
    ['custom', customTemplate, null, '/static/public/static/'],
    [
      'custom base',
      customTemplate.replace('<head>', '<head><base href="brand/">'),
      null,
      '/static/public/static/brand/'
    ],
    ['missing', null, { response: { status: 404 } }, '/static/public/static/']
  ])('prints multilingual text with %s', async (_name, template, error, basePath) => {
    if (error) api.getTranscriptTemplate.mockRejectedValue(error)
    else api.getTranscriptTemplate.mockResolvedValue({ data: template })
    const heading = 'Conversation #123 <script>alert("heading")</script>'
    const body = 'Hello 世界\nनमस्कार\nمرحبا\n👋\n<script>alert("message")</script>'
    const text = `${heading}\n${body}`
    const printing = printTranscript(text, 'transcript-123')
    await Promise.resolve()
    const frame = document.querySelector('iframe')
    const parsed = new DOMParser().parseFromString(frame.srcdoc, 'text/html')
    frame.contentDocument.documentElement.innerHTML = parsed.documentElement.innerHTML
    const print = vi.spyOn(frame.contentWindow, 'print').mockImplementation(() => {})
    Object.defineProperty(frame.contentDocument, 'fonts', {
      value: { ready: Promise.resolve() }
    })
    frame.dispatchEvent(new Event('load'))

    await printing

    expect(print).toHaveBeenCalledOnce()
    expect(frame.contentDocument.title).toBe('transcript-123')
    expect(frame.contentDocument.querySelector('[data-transcript-title]').textContent).toBe(heading)
    expect(frame.contentDocument.querySelector('[data-transcript-body]').textContent).toBe(body)
    const expectedBase = new URL(basePath, window.location.origin).href
    expect(frame.contentDocument.baseURI).toBe(expectedBase)
    const logo = frame.contentDocument.querySelector('img')
    if (logo) {
      expect(logo.src).toBe(`${expectedBase}logo.png`)
      expect(frame.contentDocument.querySelector('link').href).toBe(`${expectedBase}pdf.css`)
    }
    expect(frame.contentDocument.querySelector('script')).toBeNull()
    expect(frame.getAttribute('sandbox').split(' ')).not.toContain('allow-scripts')
    frame.contentWindow.dispatchEvent(new Event('afterprint'))
    expect(document.querySelector('iframe')).toBeNull()
  })
})
