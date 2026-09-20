// @vitest-environment jsdom
import { expect, it, vi } from 'vitest'
import { createApp, h } from 'vue'
import BubbleAttachmentItem from './BubbleAttachmentItem.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
it('shows unavailable attachments without download or preview controls', () => {
  const root = document.createElement('div')
  const app = createApp({
    render: () =>
      h(BubbleAttachmentItem, {
        attachment: {
          name: 'missing.png',
          size: 42,
          content_type: 'image/png',
          unavailable: true,
          unavailable_reason: 'storage_full',
          url: '',
          thumbnail_url: ''
        }
      })
  })
  try {
    app.mount(root)
    expect(root.textContent).toContain('missing.png')
    expect(root.textContent).toContain('media.attachmentUnavailable')
    expect(root.querySelectorAll('a, button, img, audio')).toHaveLength(0)
  } finally {
    app.unmount()
  }
})
