// @vitest-environment jsdom
import { afterEach, expect, it, vi } from 'vitest'
import { createApp, h } from 'vue'
import BubbleAttachmentPreview from './BubbleAttachmentPreview.vue'

vi.mock('@/features/conversation/message/attachment/BubbleAttachmentItem.vue', () => ({
  default: {
    props: ['attachment'],
    setup: (props) => () => h('div', { 'data-file-card': props.attachment.uuid })
  }
}))
vi.mock('./TelegramSticker.vue', () => ({ default: { setup: () => () => null } }))
vi.mock('@/components/ImageLightbox.vue', () => ({ default: { setup: () => () => null } }))
vi.mock('@main/components/DownloadLink.vue', () => ({
  default: { props: ['url'], setup: (props) => () => h('a', { href: props.url }, 'Download') }
}))

let app, root
const mount = (channel, attachments) => {
  root = document.createElement('div')
  document.body.append(root)
  app = createApp(BubbleAttachmentPreview, { channel, attachments })
  app.mount(root)
}
const voice = { uuid: 'voice', name: 'voice.ogg', content_type: 'audio/ogg', url: '/uploads/voice' }
afterEach(() => {
  app.unmount()
  root.remove()
})

it('plays Telegram voice messages inline without autoplay and keeps the download', () => {
  mount('telegram', [voice])
  const audio = root.querySelector('audio')
  expect(audio.controls).toBe(true)
  expect(audio.autoplay).toBe(false)
  expect(audio.preload).toBe('metadata')
  expect(audio.getAttribute('aria-label')).toBe('voice.ogg')
  expect(audio.getAttribute('src')).toBe('/uploads/voice')
  expect(root.querySelector('a').getAttribute('href')).toBe('/uploads/voice')
  expect(root.querySelector('[data-file-card]')).toBeNull()
})
it.each(['email', 'whatsapp', 'livechat', ''])(
  'keeps the existing audio card for %s',
  (channel) => {
    mount(channel, [voice])
    expect(root.querySelector('audio')).toBeNull()
    expect(root.querySelector('[data-file-card="voice"]')).not.toBeNull()
  }
)
it('keeps other Telegram files downloadable through the existing file card', () => {
  mount('telegram', [
    {
      uuid: 'document',
      name: 'invoice.pdf',
      content_type: 'application/pdf',
      url: '/uploads/document'
    }
  ])
  expect(root.querySelector('[data-file-card="document"]')).not.toBeNull()
  expect(root.querySelector('audio')).toBeNull()
})
