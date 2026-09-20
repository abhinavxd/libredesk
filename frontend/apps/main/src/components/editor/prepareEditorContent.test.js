// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest'
import { Editor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { ResizableImage } from './extensions/ResizableImage'
import { editorImageURL, prepareEditorContent, serializeEditorContent } from './prepareEditorContent'

vi.mock('@main/i18n', () => ({ getI18n: () => ({ global: { t: key => key } }) }))

describe('conversation editor resources', () => {
  it('removes active resources before the editor parses HTML', () => {
    const html = prepareEditorContent('<style>@import "https://tracker.example/style"</style><iframe src="https://tracker.example/frame"></iframe><table style="background:url(https://tracker.example/css)"><tr><td>Hello</td></tr></table><img src="https://tracker.example/image" srcset="https://tracker.example/2x 2x" onload="alert(1)">')
    const template = document.createElement('template')
    template.innerHTML = html
    expect(template.content.querySelector('iframe,style,[src],[srcset],[style],[onload]')).toBeNull()
    expect(template.content.querySelector('img').dataset.libredeskImageSrc).toBe('https://tracker.example/image')
    expect(template.content.textContent).toContain('Hello')
  })

  it('keeps remote sources out of the live editor and its DOM serializer', () => {
    const content = '<p>Hello</p><img src="https://tracker.example/image" alt="Logo">'
    const editor = new Editor({
      extensions: [StarterKit, ResizableImage.configure({ restrictResources: true })],
      content: prepareEditorContent(content)
    })
    expect(editor.view.dom.querySelector('img').hasAttribute('src')).toBe(false)
    const serialized = editor.getHTML()
    expect(serialized).not.toMatch(/\ssrc=/)
    expect(serializeEditorContent(serialized)).toContain('src="https://tracker.example/image"')
    editor.destroy()
  })

  it('only displays LibreDesk media or message gateway URLs', () => {
    const id = '12345678-1234-1234-1234-123456789abc'
    expect(editorImageURL(`/uploads/${id}?sig=abc`)).toBe(`/uploads/${id}?sig=abc&proxy=1`)
    for (const source of ['https://tracker.example/pixel', '//tracker.example/pixel', '/api/v1/settings', 'data:image/svg+xml,test', '/uploads/../redirect', 'https://user:pass@localhost/uploads/' + id]) {
      expect(editorImageURL(source)).toBe('')
    }
  })

  it('preserves conversation references without arbitrary classes', () => {
    expect(prepareEditorContent('<a class="ld-conversation-reference arbitrary" data-id="abc" href="/inboxes/all/conversation/abc">#1</a>')).toContain('class="ld-conversation-reference"')
  })
})
