const uuid = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'
const uploadPath = new RegExp('^/uploads/(?:thumb_)?' + uuid + '$', 'i')
const imagePath = new RegExp('^/api/v1/conversations/' + uuid + '/messages/' + uuid + '/images/[0-9a-f]{64}$', 'i')

export function localMediaURL(value) {
  if (!value || /[\\\x00-\x20\x7f]/.test(value)) return ''
  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin || url.username || url.password) return ''
    if (!uploadPath.test(url.pathname) && !imagePath.test(url.pathname)) return ''
    return url.pathname + url.search
  } catch {
    return ''
  }
}

export function avatarURL(value) {
  const local = localMediaURL(value)
  if (local) return local
  if (!value || value.length > 8192) return ''
  if (value.startsWith('blob:' + window.location.origin + '/')) return value
  if (/[\\\x00-\x20\x7f]/.test(value)) return ''
  try {
    const url = new URL(value)
    if (url.protocol !== 'https:' || url.username || url.password || url.port) return ''
    return '/api/v1/resource-images/avatar?source=' + encodeURIComponent(value)
  } catch {
    return ''
  }
}

export function isPreviewImage(contentType) {
  return ['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/avif'].includes(contentType)
}

export function isPreviewAudio(contentType) {
  return ['audio/mpeg', 'audio/mp4', 'audio/ogg', 'audio/wav', 'audio/x-wav', 'audio/webm', 'audio/flac'].includes(contentType)
}
