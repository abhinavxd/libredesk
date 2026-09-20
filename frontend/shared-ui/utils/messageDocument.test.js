// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { messageDocument } from './messageDocument.js'

const upload = '/uploads/00000000-0000-0000-0000-000000000001'

function parse(html) {
  return new DOMParser().parseFromString(messageDocument(html), 'text/html')
}

describe('message document boundary', () => {
  it('starts with a restrictive policy before email content', () => {
    const doc = parse('<p style="color: red">Hello</p>')
    const policy = doc.querySelector('meta[http-equiv="Content-Security-Policy"]')
    expect(policy.content).toContain("default-src 'none'")
    expect(policy.content).toContain("script-src 'none'")
    expect(policy.content).toContain("img-src 'none'")
    expect(policy.content).toContain("base-uri 'none'; form-action 'none'")
    expect(policy.content).toContain(`style-src-elem 'nonce-${doc.querySelector('style').nonce}'`)
    expect(policy.compareDocumentPosition(doc.querySelector('p')) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(doc.querySelector('p').style.color).toBe('red')
  })

  it('restricts images to the exact local media paths in the prepared content', () => {
    const doc = parse(`<img src="${upload}?signature=secret" srcset="https://tracker.example/pixel 2x"><img src="https://tracker.example/pixel" alt="Blocked"><img src="/api/v1/other"><img src="data:image/svg+xml,bad">`)
    const policy = doc.querySelector('meta[http-equiv="Content-Security-Policy"]').content
    expect(doc.querySelectorAll('img')).toHaveLength(1)
    expect(doc.querySelector('img').getAttribute('src')).toBe(upload + '?signature=secret')
    expect(doc.querySelector('img').hasAttribute('srcset')).toBe(false)
    expect(policy).toContain(`img-src ${window.location.origin}${upload};`)
    expect(policy).not.toMatch(/tracker|signature|data:|'self'/)
    expect(doc.body.textContent).toBe('Blocked')
  })

  it('uses a fresh style nonce for each document', () => {
    expect(parse('').querySelector('style').nonce).not.toBe(parse('').querySelector('style').nonce)
  })
})
