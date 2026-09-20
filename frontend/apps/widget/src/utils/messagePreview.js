const attachmentTerm = (contentType = '') => {
  if (contentType.startsWith('image/')) return 'globals.terms.image'
  if (contentType.startsWith('video/')) return 'globals.terms.video'
  if (contentType.startsWith('audio/')) return 'globals.terms.audio'
  return 'globals.terms.file'
}

export const lastMessagePreview = (lastMessage, t) => {
  const content = lastMessage?.content?.trim()
  if (content) return content
  const attachment = lastMessage?.attachments?.[0]
  if (!attachment) return ''
  return t(attachmentTerm(attachment.content_type), 1)
}
