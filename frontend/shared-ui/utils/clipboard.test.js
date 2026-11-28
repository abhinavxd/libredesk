import { describe, it, expect, vi, afterEach } from 'vitest'
import { copyToClipboard } from './clipboard'

// The util only touches a handful of DOM APIs, stub just those instead of
// pulling in a full DOM environment.
function stubDocument(execCommandResult = true) {
  const textarea = {
    value: '',
    style: {},
    setAttribute: vi.fn(),
    select: vi.fn(),
    setSelectionRange: vi.fn()
  }
  const body = {
    appendChild: vi.fn(),
    removeChild: vi.fn()
  }
  const previouslyFocused = { focus: vi.fn() }
  const doc = {
    body,
    activeElement: previouslyFocused,
    createElement: vi.fn(() => textarea),
    execCommand: vi.fn(() => execCommandResult)
  }
  vi.stubGlobal('document', doc)
  return { doc, textarea, body, previouslyFocused }
}

describe('copyToClipboard', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('uses the async clipboard api when available', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    stubDocument()

    expect(await copyToClipboard('secret-key')).toBe(true)
    expect(writeText).toHaveBeenCalledWith('secret-key')
  })

  it('falls back to execCommand when the async api rejects', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'))
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const { doc, textarea, body, previouslyFocused } = stubDocument()

    expect(await copyToClipboard('secret-key')).toBe(true)
    expect(doc.execCommand).toHaveBeenCalledWith('copy')
    expect(textarea.value).toBe('secret-key')
    expect(body.appendChild).toHaveBeenCalledWith(textarea)
    expect(body.removeChild).toHaveBeenCalledWith(textarea)
    expect(previouslyFocused.focus).toHaveBeenCalled()
  })

  it('falls back to execCommand when the clipboard api is unavailable', async () => {
    vi.stubGlobal('navigator', {})
    const { doc } = stubDocument()

    expect(await copyToClipboard('secret-key')).toBe(true)
    expect(doc.execCommand).toHaveBeenCalledWith('copy')
  })

  it('returns false when the copy command fails', async () => {
    vi.stubGlobal('navigator', {})
    const { body, textarea } = stubDocument(false)

    expect(await copyToClipboard('secret-key')).toBe(false)
    expect(body.removeChild).toHaveBeenCalledWith(textarea)
  })

  it('returns false when the copy command throws', async () => {
    vi.stubGlobal('navigator', {})
    const { doc } = stubDocument()
    doc.execCommand = vi.fn(() => {
      throw new Error('denied')
    })

    expect(await copyToClipboard('secret-key')).toBe(false)
  })

  it('stringifies nullish values instead of throwing', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })

    expect(await copyToClipboard(undefined)).toBe(true)
    expect(writeText).toHaveBeenCalledWith('')
  })
})
