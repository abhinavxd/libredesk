import { describe, expect, it } from 'vitest'
import MessageCache from './conversation-message-cache'

function cachedMessage() {
  const cache = new MessageCache()
  const message = { uuid: 'message', content: 'Original', text_content: 'Original', content_type: 'html', display: { html: '<p>Original</p>' } }
  cache.addMessages('conversation', [message], 1, 1)
  return { cache, message }
}

describe('message display cache', () => {
  it.each(['content', 'text_content', 'content_type'])('invalidates display when %s changes', field => {
    const { cache, message } = cachedMessage()
    cache.updateMessage('conversation', 'message', { [field]: 'changed' })
    expect(message.display).toBeUndefined()
  })

  it('invalidates display for single-field updates', () => {
    const { cache, message } = cachedMessage()
    cache.updateMessageField('conversation', 'message', 'content', 'Deleted')
    expect(message.display).toBeUndefined()
  })

  it('keeps display for status-only and unchanged-content updates', () => {
    const { cache, message } = cachedMessage()
    cache.updateMessage('conversation', 'message', { status: 'sent', content: 'Original' })
    expect(message.display.html).toBe('<p>Original</p>')
  })

  it('accepts a replacement prepared copy with updated content', () => {
    const { cache, message } = cachedMessage()
    cache.updateMessage('conversation', 'message', { content: 'Changed', display: { html: '<p>Changed</p>' } })
    expect(message.display.html).toBe('<p>Changed</p>')
  })
})
