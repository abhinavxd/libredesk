export const TELEGRAM_CHANNEL = 'telegram'
export const TELEGRAM_MAX_BUTTONS = 10
export const TELEGRAM_MAX_UPLOAD_BYTES = 50 * 1024 * 1024
const TELEGRAM_MAX_BUTTON_TEXT = 64

export function telegramButtonError(buttons) {
  if (buttons.length > TELEGRAM_MAX_BUTTONS) return true
  return buttons.some((button) => {
    const errors = telegramButtonFieldErrors(button)
    return Boolean(errors.text || errors.url)
  })
}

export function telegramButtonFieldErrors(button) {
  const errors = { text: '', url: false }
  if (!button.text.trim()) errors.text = 'required'
  else if (
    [...button.text].length > TELEGRAM_MAX_BUTTON_TEXT ||
    (!button.url && new TextEncoder().encode(button.text).length > TELEGRAM_MAX_BUTTON_TEXT)
  ) {
    errors.text = 'tooLong'
  }
  if (button.url) {
    try {
      const url = new URL(button.url)
      errors.url = !['https:', 'http:', 'tg:'].includes(url.protocol) || !url.hostname
    } catch {
      errors.url = true
    }
  }
  return errors
}

export function buildTelegramReplyParts(content, files, buttons = []) {
  const keyboard = buttons.map(({ text, url }) =>
    url ? { text: text.trim(), url: url.trim() } : { text: text.trim(), callback_data: text.trim() }
  )
  if (!files.length) return [{ content, attachments: [], buttons: keyboard }]
  const parts = []
  let family = ''
  for (const file of files) {
    const mime = file.content_type || ''
    const kind = ['image/jpeg', 'image/png', 'video/mp4'].includes(mime)
      ? 'visual'
      : ['audio/mpeg', 'audio/mp4'].includes(mime)
        ? 'audio'
        : ['image/gif', 'audio/ogg'].includes(mime) ||
            /\.tgs$/i.test(file.filename || file.name || '')
          ? 'single'
          : 'document'
    if (
      !parts.length ||
      kind === 'single' ||
      family !== kind ||
      parts.at(-1).attachments.length === 10
    ) {
      parts.push({ content: '', attachments: [], buttons: [] })
    }
    parts.at(-1).attachments.push(file)
    family = kind
  }
  if (content.replace(/<[^>]*>/g, '').trim()) {
    parts.unshift({ content, attachments: [], buttons: keyboard })
  } else if (keyboard.length && parts[0].attachments.length > 1) {
    const first = parts[0].attachments.shift()
    parts.unshift({ content: '', attachments: [first], buttons: keyboard })
  } else {
    parts[0].buttons = keyboard
  }
  return parts
}

export function groupTelegramAlbums(messages) {
  const result = []
  for (const message of messages) {
    const previous = result.at(-1)
    const id = message.meta?.telegram_media_group_id
    if (
      id &&
      !message.private &&
      !previous?.private &&
      previous?.meta?.telegram_media_group_id === id &&
      previous.conversation_id === message.conversation_id &&
      previous.sender_id === message.sender_id &&
      previous.type === message.type
    ) {
      previous.attachments.push(...(message.attachments || []))
      if (message.content) {
        const html = previous.content_type === 'html' || message.content_type === 'html'
        const first = html ? albumHTML(previous) : previous.content
        const next = html ? albumHTML(message) : message.content
        previous.content = first ? first + (html ? '<br>' : '\n') + next : next
        if (html) previous.content_type = 'html'
        previous.text_content = previous.text_content
          ? previous.text_content + '\n' + message.text_content
          : message.text_content
      }
      previous.albumMessageUUIDs.push(message.uuid)
    } else {
      result.push({
        ...message,
        attachments: [...(message.attachments || [])],
        albumMessageUUIDs: [message.uuid]
      })
    }
  }
  return result
}

function albumHTML(message) {
  if (message.content_type === 'html') return message.content || ''
  return (message.content || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>')
}
