// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { buildTelegramReplyParts, groupTelegramAlbums, telegramButtonError } from './telegramReply'

const file = (id, content_type = 'image/jpeg') => ({ id, content_type })
const message = (uuid, overrides = {}) => ({
  uuid,
  conversation_id: 1,
  sender_id: 1,
  type: 'incoming',
  content: '',
  text_content: '',
  content_type: 'text',
  attachments: [file(uuid)],
  meta: { telegram_media_group_id: 'album' },
  ...overrides
})

describe('Telegram replies', () => {
  it('groups compatible photos and videos and preserves file order', () => {
    const files = [
      file(1),
      file(2, 'video/mp4'),
      file(3, 'audio/mpeg'),
      file(4, 'audio/mp4'),
      file(5, 'application/pdf')
    ]
    const parts = buildTelegramReplyParts('Caption', files)
    expect(parts.map((p) => p.attachments.map((f) => f.id))).toEqual([[], [1, 2], [3, 4], [5]])
    expect(parts.map((p) => p.content)).toEqual(['Caption', '', '', ''])
  })
  it('splits albums at ten attachments', () => {
    const files = Array.from({ length: 21 }, (_, i) => file(i))
    expect(buildTelegramReplyParts('', files).map((p) => p.attachments.length)).toEqual([10, 10, 1])
  })
  it('sends GIFs, voice messages and animated stickers separately', () => {
    const files = [
      file(1, 'image/gif'),
      file(2, 'image/gif'),
      file(3, 'audio/ogg'),
      { ...file(4, 'application/x-gzip'), name: 'sticker.tgs' }
    ]
    expect(buildTelegramReplyParts('', files).map((p) => p.attachments.length)).toEqual([
      1, 1, 1, 1
    ])
  })
  it('attaches buttons to text or a single file and sends them before albums', () => {
    const buttons = [
      { text: ' Yes ', url: '' },
      { text: 'Docs', url: 'https://example.com' }
    ]
    const keyboard = [
      { text: 'Yes', callback_data: 'Yes' },
      { text: 'Docs', url: 'https://example.com' }
    ]
    expect(buildTelegramReplyParts('Choose', [], buttons)).toEqual([
      { content: 'Choose', attachments: [], buttons: keyboard }
    ])
    expect(buildTelegramReplyParts('Choose', [file(1)], buttons)[0].buttons).toEqual(keyboard)
    expect(buildTelegramReplyParts('Choose', [file(1), file(2)], buttons)).toEqual([
      { content: 'Choose', attachments: [], buttons: keyboard },
      { content: '', attachments: [file(1), file(2)], buttons: [] }
    ])
  })
  it('validates callback byte limits and links', () => {
    expect(telegramButtonError([])).toBe(false)
    expect(telegramButtonError([{ text: '😀'.repeat(16), url: '' }])).toBe(false)
    expect(telegramButtonError([{ text: '😀'.repeat(17), url: '' }])).toBe(true)
    for (const url of ['https://example.com', 'http://example.com', 'tg://user?id=1']) {
      expect(telegramButtonError([{ text: 'Link', url }])).toBe(false)
    }
    for (const url of ['broken', '/relative', 'javascript:alert(1)']) {
      expect(telegramButtonError([{ text: 'Link', url }])).toBe(true)
    }
    for (const buttons of [
      Array.from({ length: 11 }, () => ({ text: 'Yes' })),
      [{ text: '' }],
      [{ text: ' ' }],
      [{ text: 'a'.repeat(65) }]
    ]) {
      expect(telegramButtonError(buttons)).toBe(true)
    }
  })
})

describe('Telegram album timeline', () => {
  it('groups adjacent members without mutating store messages and preserves jump targets', () => {
    const input = [
      message('1', { content: 'First', text_content: 'First' }),
      message('2', { content: 'Second', text_content: 'Second' })
    ]
    const before = structuredClone(input)
    const rows = groupTelegramAlbums(input)
    expect(input).toEqual(before)
    expect(rows).toHaveLength(1)
    expect(rows[0].attachments.map((f) => f.id)).toEqual(['1', '2'])
    expect(rows[0].albumMessageUUIDs).toEqual(['1', '2'])
    expect(rows[0].content).toBe('First\nSecond')
    expect(rows[0].text_content).toBe('First\nSecond')
  })
  it.each([
    { private: true },
    { sender_id: 2 },
    { conversation_id: 2 },
    { type: 'outgoing' },
    { meta: {} },
    { meta: { telegram_media_group_id: 'other' } }
  ])('keeps incompatible messages separate: %j', (overrides) => {
    expect(groupTelegramAlbums([message('1'), message('2', overrides)])).toHaveLength(2)
    expect(groupTelegramAlbums([message('1', overrides), message('2')])).toHaveLength(2)
  })
  it('groups formatted captions with uncaptioned and plain members safely', () => {
    const rows = groupTelegramAlbums([
      message('1', { content: '<b>First</b>', content_type: 'html' }),
      message('2'),
      message('3', { content: '<script>\n&', text_content: 'Text' })
    ])
    expect(rows).toHaveLength(1)
    expect(rows[0].content).toBe('<b>First</b><br>&lt;script&gt;<br>&amp;')
    expect(rows[0].content_type).toBe('html')
  })
  it('puts buttons on the first photo when there is no caption', () => {
    const parts = buildTelegramReplyParts(
      '<p><br></p>',
      [file(1), file(2), file(3)],
      [{ text: 'Yes' }]
    )
    expect(parts.map((p) => p.attachments.map((f) => f.id))).toEqual([[1], [2, 3]])
    expect(parts[0].buttons[0].callback_data).toBe('Yes')
  })
  it('returns messages that are not merged unchanged', () => {
    const text = { uuid: '1', content: 'Text' }
    const lone = message('2')
    const rows = groupTelegramAlbums([text, lone, message('3', { meta: {} })])
    expect(rows[0]).toBe(text)
    expect(rows[1]).toBe(lone)
    expect(rows[0].albumMessageUUIDs).toBeUndefined()
  })
  it('keeps unrelated messages and missing attachments intact', () => {
    expect(groupTelegramAlbums([message('1'), { uuid: 'text' }, message('2')])).toHaveLength(3)
    expect(
      groupTelegramAlbums([
        message('1', { attachments: undefined }),
        message('2', { attachments: undefined, content: 'Caption', text_content: 'Caption' })
      ])[0]
    ).toMatchObject({ content: 'Caption', attachments: [], albumMessageUUIDs: ['1', '2'] })
  })
})

describe('Long Telegram attachment replies', () => {
  it('sends long text separately and preserves all files and buttons', () => {
    const content = `<b>${'😀'.repeat(1025)}</b>`
    const files = [file(1), file(2)]
    const parts = buildTelegramReplyParts(content, files, [{ text: 'Yes' }])
    expect(parts).toEqual([
      { content, attachments: [], buttons: [{ text: 'Yes', callback_data: 'Yes' }] },
      { content: '', attachments: files, buttons: [] }
    ])
  })
  it('keeps formatted text intact when sending it before media', () => {
    const content = `<b>${'&amp;'.repeat(1024)}</b>`
    expect(buildTelegramReplyParts(content, [file(1)])[0]).toEqual({
      content,
      attachments: [],
      buttons: []
    })
    expect(buildTelegramReplyParts('x'.repeat(4096), [file(1)])).toHaveLength(2)
  })
})
