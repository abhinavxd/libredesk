// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest'
import { printTranscript } from './print-transcript'

describe('printTranscript', () => {
  afterEach(() => {
    document.body.replaceChildren()
    vi.restoreAllMocks()
  })

  it('prints multilingual text without interpreting message HTML', async () => {
    const text = 'Hello 世界\nनमस्कार\nمرحبا\n👋\n<script>alert("message")</script>'
    const printing = printTranscript(text, 'transcript-123')
    const frame = document.querySelector('iframe')
    const print = vi.spyOn(frame.contentWindow, 'print').mockImplementation(() => {})
    Object.defineProperty(frame.contentDocument, 'fonts', {
      value: { ready: Promise.resolve() }
    })
    frame.dispatchEvent(new Event('load'))

    await printing

    expect(print).toHaveBeenCalledOnce()
    expect(frame.contentDocument.title).toBe('transcript-123')
    expect(frame.contentDocument.querySelector('pre').textContent).toBe(text)
    expect(frame.contentDocument.querySelector('script')).toBeNull()
    expect(frame.getAttribute('sandbox').split(' ')).not.toContain('allow-scripts')
    frame.contentWindow.dispatchEvent(new Event('afterprint'))
    expect(document.querySelector('iframe')).toBeNull()
  })
})
