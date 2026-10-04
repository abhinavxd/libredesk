// @vitest-environment jsdom
import { afterEach, describe, expect, it } from 'vitest'
import { Editor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import { ActivationKey } from './ActivationKey'
import { prepareEditorContent } from '../prepareEditorContent'
import { readKeyPlaceholders, stripKeyPlaceholders } from '@main/utils/activation-keys'

let editor
const makeEditor = (content = '<p>Hi</p>') => {
  editor = new Editor({ extensions: [StarterKit, ActivationKey], content })
  return editor
}

afterEach(() => editor?.destroy())

describe('activation key chip', () => {
  it('inserts a placeholder that serializes as the server expects', () => {
    makeEditor()
    editor.commands.focus('end')
    editor.commands.insertActivationKey(3)
    const [chip] = readKeyPlaceholders(editor.getHTML())
    expect(chip.appId).toBe(3)
    expect(chip.id).toMatch(/^\d{9}$/)
    const span = new DOMParser().parseFromString(editor.getHTML(), 'text/html').querySelector('span')
    expect([span.dataset.type, span.dataset.id, span.dataset.appId, span.children.length]).toEqual(['activation-key', chip.id, '3', 0])
    expect(editor.getHTML()).toContain(`&lt;KEY_ID_${chip.id}&gt;`)
    expect(editor.getText()).toContain(`<KEY_ID_${chip.id}>`)
  })

  it('is a single uneditable unit', () => {
    makeEditor()
    editor.commands.focus('end')
    editor.commands.insertActivationKey(3)
    let chip
    editor.state.doc.descendants((node) => {
      if (node.type.name === 'activationKey') chip = node
    })
    // An atom leaf: the cursor can't enter it and one deletion removes it whole.
    expect([chip.isAtom, chip.isLeaf, chip.nodeSize]).toEqual([true, true, 1])
    const end = editor.state.selection.from
    editor.commands.deleteRange({ from: end - 1, to: end })
    expect(readKeyPlaceholders(editor.getHTML())).toHaveLength(0)
    expect(editor.getText()).not.toContain('KEY_ID')
  })

  it('survives a draft round trip through the editor sanitizer', () => {
    makeEditor()
    editor.commands.focus('end')
    editor.commands.insertActivationKey(7)
    const saved = editor.getHTML()
    editor.commands.setContent(prepareEditorContent(saved))
    expect(readKeyPlaceholders(editor.getHTML())).toEqual(readKeyPlaceholders(saved))
  })

  it('drops a second chip with the same ID', () => {
    const chip = '<span data-type="activation-key" data-id="458219512" data-app-id="3">&lt;KEY_ID_458219512&gt;</span>'
    makeEditor('<p>a</p>')
    editor.commands.insertContent(`<p>${chip} ${chip}</p>`)
    expect(readKeyPlaceholders(editor.getHTML())).toEqual([{ id: '458219512', appId: 3 }])
  })

  it('moves every chip to a newly chosen game', () => {
    makeEditor()
    editor.commands.focus('end')
    editor.commands.insertActivationKey(3)
    editor.commands.insertActivationKey(3)
    editor.commands.setActivationKeyApp(5)
    expect(readKeyPlaceholders(editor.getHTML()).map((p) => p.appId)).toEqual([5, 5])
  })

  it('treats typed lookalike text as plain text', () => {
    makeEditor()
    editor.commands.focus('end')
    editor.commands.insertContent('&lt;KEY_ID_458219512&gt;')
    expect(readKeyPlaceholders(editor.getHTML())).toHaveLength(0)
  })
})

describe('pasting', () => {
  it('strips chips from pasted HTML', () => {
    const pasted = '<p>Key: <span data-type="activation-key" data-id="458219512" data-app-id="3">&lt;KEY_ID_458219512&gt;</span></p>'
    const cleaned = stripKeyPlaceholders(prepareEditorContent(pasted))
    expect(readKeyPlaceholders(cleaned)).toHaveLength(0)
    expect(cleaned).toContain('Key:')
  })
})
