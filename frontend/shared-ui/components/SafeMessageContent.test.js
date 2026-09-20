// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, reactive } from 'vue'
import SafeMessageContent from './SafeMessageContent.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (_key, { count } = {}) => `Blocked images: ${count}` })
}))

let app
let root

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class {
    observe() {}
    disconnect() {}
  })
})

afterEach(() => {
  app?.unmount()
  root?.remove()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function preparedContent(el) {
  const template = document.createElement('template')
  template.innerHTML = el.querySelector('iframe')?.getAttribute('srcdoc') || ''
  return template.content
}

function mount(message, props = {}) {
  root = document.createElement('div')
  document.body.appendChild(root)
  app = createApp({ render: () => h(SafeMessageContent, { message, ...props }) })
  app.mount(root)
  return root
}

describe('safe message content', () => {
  it('renders only the prepared copy and keeps its security attributes', () => {
    const el = mount({
      content_type: 'html',
      content: '<iframe src="https://tracker.example/frame"></iframe><img src="https://tracker.example/pixel">',
      display: {
        html: '<p style="color: blue">Hello</p><img src="/uploads/00000000-0000-0000-0000-000000000001" referrerpolicy="no-referrer"><a href="https://example.com" rel="noopener noreferrer" target="_blank">Visit</a>',
        blocked_images: 1,
        blocked_domains: ['tracker.example']
      }
    })
    expect(el.querySelector('iframe').getAttribute('sandbox')).toBe('allow-same-origin')
    expect(el.querySelector('iframe').getAttribute('referrerpolicy')).toBe('no-referrer')
    const content = preparedContent(el)
    expect(content.querySelector('iframe')).toBeNull()
    expect(content.querySelectorAll('img')).toHaveLength(1)
    expect(content.querySelector('img').getAttribute('src')).toBe('/uploads/00000000-0000-0000-0000-000000000001')
    expect(content.querySelector('img').getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(content.querySelector('a').rel).toBe('noopener noreferrer')
    expect(el.textContent).toContain('Blocked images: 1')
    expect(el.textContent).toContain('tracker.example')
  })

  it.each([undefined, null, {}, { html: null }, { html: 12 }])('never parses raw HTML when display is %s', display => {
    const raw = '<img src="https://tracker.example/pixel"><iframe src="https://tracker.example/frame"></iframe>'
    const el = mount({ content_type: 'html', content: raw, display })
    expect(el.querySelector('img, iframe')).toBeNull()
    expect(el.textContent).toBe(raw)
  })

  it('uses escaped text for pending messages', () => {
    const el = mount({ content_type: 'html', content: '<img src="https://tracker.example/x">', text_content: 'Hello <friend>' })
    expect(el.textContent).toBe('Hello <friend>')
    expect(el.querySelector('img, friend')).toBeNull()
  })

  it('does not fall back to raw content for an empty prepared copy', () => {
    const el = mount({ content_type: 'html', content: '<img src="https://tracker.example/x">', display: { html: '' } })
    expect(el.textContent).toBe('')
    expect(el.querySelector('img')).toBeNull()
  })

  it('treats text messages as text even if a display copy exists', () => {
    const el = mount({ content_type: 'text', content: '<b>Literal</b>', display: { html: '<b>HTML</b>' } })
    expect(el.textContent).toBe('<b>Literal</b>')
    expect(el.querySelector('b')).toBeNull()
  })

  it('updates rendered content and removes images when display is invalidated', async () => {
    const message = reactive({ content_type: 'html', text_content: 'Updated text', display: { html: '<img src="/uploads/00000000-0000-0000-0000-000000000001">' } })
    const el = mount(message)
    expect(preparedContent(el).querySelector('img')).not.toBeNull()
    message.display = { html: '<strong>Changed</strong>' }
    await nextTick()
    expect(el.querySelector('img')).toBeNull()
    expect(preparedContent(el).querySelector('strong').textContent).toBe('Changed')
    delete message.display
    await nextTick()
    expect(el.querySelector('strong')).toBeNull()
    expect(el.textContent).toBe('Updated text')
  })

  it('escapes domain metadata', () => {
    const el = mount({ display: { html: '', blocked_images: 1, blocked_domains: ['<img src="https://tracker.example/x">'] } })
    expect(el.querySelector('img')).toBeNull()
    expect(el.textContent).toContain('<img src=')
  })
  it('keeps quote controls and image previews working across the frame boundary', async () => {
    const onImageClick = vi.fn()
    const props = reactive({ showQuotedText: false, onImageClick })
    const el = mount({ content_type: 'html', display: { html: '<blockquote>Quote</blockquote>' } }, props)
    const frame = el.querySelector('iframe')
    const body = frame.contentDocument.body
    body.innerHTML = '<blockquote>Quote</blockquote><a href="https://example.com"><img src="/uploads/00000000-0000-0000-0000-000000000001" alt="Chart"></a>'
    frame.dispatchEvent(new Event('load'))
    expect(body.classList.contains('hide-quotes')).toBe(true)
    props.showQuotedText = true
    await nextTick()
    expect(body.classList.contains('hide-quotes')).toBe(false)
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })
    body.querySelector('img').dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(onImageClick).toHaveBeenCalledWith({
      images: [{ url: '/uploads/00000000-0000-0000-0000-000000000001', name: 'Chart' }], index: 0
    })
  })

  it('opens ordinary links through the parent without permitting executable URLs', () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => {})
    const el = mount({ display: { html: '' } })
    const frame = el.querySelector('iframe')
    const body = frame.contentDocument.body
    body.innerHTML = '<a href="https://example.com">Visit</a><a href="javascript:alert(1)">Bad</a>'
    frame.dispatchEvent(new Event('load'))
    for (const anchor of body.querySelectorAll('a')) {
      const event = new MouseEvent('click', { bubbles: true, cancelable: true })
      anchor.dispatchEvent(event)
      expect(event.defaultPrevented).toBe(true)
    }
    expect(open).toHaveBeenCalledExactlyOnceWith('https://example.com/', '_blank', 'noopener,noreferrer')
  })

})
