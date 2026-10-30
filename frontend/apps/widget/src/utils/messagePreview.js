
import { defineMessage as msg } from '@lingui/core/macro'
const attachmentTerm = (contentType = '') => {
  if (contentType.startsWith('image/')) return msg({ message: '{count, plural, one {Image} other {Images}}' })
  if (contentType.startsWith('video/')) return msg({ message: '{count, plural, one {Video} other {Videos}}' })
  if (contentType.startsWith('audio/')) return msg({ message: 'Audio' })
  return msg({ message: '{count, plural, one {File} other {Files}}' })
}

export const lastMessagePreview = (lastMessage, i18n) => {
  const content = lastMessage?.content?.trim()
  if (content) return content
  const attachment = lastMessage?.attachments?.[0]
  if (!attachment) return ''
  return i18n._(attachmentTerm(attachment.content_type), 1)
}
