// File extensions WhatsApp accepts as media, mirroring the backend allowlist in internal/inbox/channel/whatsapp/whatsapp.go.
const WHATSAPP_MEDIA_EXTENSIONS = [
  'jpg',
  'jpeg',
  'png',
  'webp',
  'mp4',
  '3gp',
  '3gpp',
  'aac',
  'm4a',
  'mp3',
  'amr',
  'ogg',
  'opus',
  'pdf',
  'txt',
  'doc',
  'docx',
  'xls',
  'xlsx',
  'ppt',
  'pptx'
]

export const WHATSAPP_MEDIA_ACCEPT = WHATSAPP_MEDIA_EXTENSIONS.map((ext) => '.' + ext).join(',')
