export function formatBytes (bytes) {
    if (bytes < 1024 * 1024) {
        return (bytes / 1024).toFixed(2) + ' KB'
    } else {
        return (bytes / (1024 * 1024)).toFixed(2) + ' MB'
    }
}

const UUID_V4_RE = /[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}/i

export function getThumbFilepath (url) {
    if (!url) return url
    const match = url.match(UUID_V4_RE)
    if (!match) return url
    const qIdx = url.indexOf('?')
    const query = qIdx >= 0 ? url.substring(qIdx) : ''
    return `/uploads/thumb_${match[0]}${query}`
}

export function downloadUrl (url) {
    if (!url) return url
    const match = url.match(UUID_V4_RE)
    if (!match) return url
    return `/uploads/${match[0]}?download=1`
}
