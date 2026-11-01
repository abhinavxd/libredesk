// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest'
import { printTranscript } from './print-transcript'
import api from '@main/api'
import transcriptTemplate from '@public-static/transcript.html?raw'

vi.mock('@main/api', () => ({ default: { getTranscriptTemplate: vi.fn() } }))

describe('printTranscript', () => {
  afterEach(() => {
    document.body.replaceChildren()
    vi.restoreAllMocks()
  })

  it.each([
    ['default', transcriptTemplate],
    [
      'custom',
      '<!doctype html><html><head></head><body><h2 data-transcript-title></h2><div data-transcript-body></div></body></html>'
    ]
  ])('prints multilingual text safely with the %s template', async (_name, template) => {
    api.getTranscriptTemplate.mockResolvedValue({ data: template })
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
    expect(frame.contentDocument.querySelector('script')).toBeNull()
    expect(frame.getAttribute('sandbox').split(' ')).not.toContain('allow-scripts')
    frame.contentWindow.dispatchEvent(new Event('afterprint'))
    expect(document.querySelector('iframe')).toBeNull()
  })
})
