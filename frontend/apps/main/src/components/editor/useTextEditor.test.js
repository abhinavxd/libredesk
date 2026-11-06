// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { createApp, defineComponent, h, nextTick, ref } from 'vue'
import { EditorContent } from '@tiptap/vue-3'
import { buildConversationExtensions } from './editorExtensions'
import { useTextEditor } from './useTextEditor'

const mountedApps = []

async function mountEditor(initialHtml = '') {
  const htmlContent = ref(initialHtml)
  const textContent = ref('')
  let result
  const app = createApp(
    defineComponent({
      setup() {
        result = useTextEditor({
          extensions: buildConversationExtensions({ getPlaceholder: () => '' }),
          htmlContent,
          textContent,
          autoFocus: false
        })
        return () => h(EditorContent, { editor: result.editor.value })
      }
    })
  )
  app.mount(document.createElement('div'))
  mountedApps.push(app)
  await nextTick()
  return { ...result, htmlContent, textContent }
}

// Places the cursor or a selection the way a click does, marking the editor as focused.
function select(editor, position) {
  editor.value.commands.setTextSelection(position)
  editor.value.emit('focus', { editor: editor.value, event: new FocusEvent('focus') })
}

afterEach(() => {
  mountedApps.splice(0).forEach((app) => app.unmount())
})

describe('useTextEditor insertHtml', () => {
  it('fills an empty editor without leaving a stray paragraph', async () => {
    const { insertHtml, htmlContent, textContent } = await mountEditor()

    insertHtml('<p>Canned reply</p>')

    expect(htmlContent.value).toBe('<p>Canned reply</p>')
    expect(textContent.value).toBe('Canned reply')
  })

  it('inserts at the cursor, keeping the text around it', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>Hello world</p>')
    select(editor, 7)

    insertHtml('<p>Canned reply</p>')

    expect(htmlContent.value).toBe('<p>Hello </p><p>Canned reply</p><p>world</p>')
  })

  it('adds a new paragraph when the cursor is at the end of a line', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>Hello</p>')
    select(editor, 6)

    insertHtml('<p>one</p><p>two</p>')

    expect(htmlContent.value).toBe('<p>Hello</p><p>one</p><p>two</p>')
  })

  it('leaves no empty paragraph at the edges of a line with neighbours', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>first</p><p>second</p>')
    select(editor, 6)

    insertHtml('<p>after first</p>')

    expect(htmlContent.value).toBe('<p>first</p><p>after first</p><p>second</p>')

    select(editor, 1)
    insertHtml('<p>before first</p>')

    expect(htmlContent.value).toBe('<p>before first</p><p>first</p><p>after first</p><p>second</p>')
  })

  it('stays inside the list item holding the cursor', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor(
      '<ul><li><p>one</p></li><li><p>two</p></li></ul>'
    )
    select(editor, 6)

    insertHtml('<p>note</p>')

    expect(htmlContent.value).toBe('<ul><li><p>one</p><p>note</p></li><li><p>two</p></li></ul>')
  })

  it('keeps block content such as lists intact', async () => {
    const { insertHtml, htmlContent } = await mountEditor()

    insertHtml('<ul><li><p>first</p></li><li><p>second</p></li></ul>')

    expect(htmlContent.value).toContain('<ul><li><p>first</p></li><li><p>second</p></li></ul>')
  })

  it('replaces selected text', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>Hello world</p>')
    select(editor, { from: 7, to: 12 })

    insertHtml('<p>there</p>')

    expect(htmlContent.value).toBe('<p>Hello </p><p>there</p>')
  })

  it('replaces a fully selected line without leaving an empty paragraph', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>first</p><p>second</p>')
    select(editor, { from: 1, to: 6 })

    insertHtml('<p>replaced</p>')

    expect(htmlContent.value).toBe('<p>replaced</p><p>second</p>')
  })

  it('appends to existing content when the editor was never focused', async () => {
    const { insertHtml, htmlContent } = await mountEditor('<p>Draft</p>')

    insertHtml('<p>Canned reply</p>')

    expect(htmlContent.value).toBe('<p>Draft</p><p>Canned reply</p>')
  })

  it('leaves the cursor after the inserted content', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor()

    insertHtml('<p>Canned</p>')
    editor.value.commands.insertContent('!')

    expect(htmlContent.value).toBe('<p>Canned!</p>')
  })

  it('can insert the same content twice', async () => {
    const { editor, insertHtml, htmlContent } = await mountEditor('<p>Hello world</p>')
    select(editor, 7)

    insertHtml('<p>Canned</p>')
    insertHtml('<p>Canned</p>')

    expect(htmlContent.value).toBe('<p>Hello </p><p>Canned</p><p>Canned</p><p>world</p>')
  })
})
