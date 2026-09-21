const imagePath = /^\/(?:uploads\/[a-f0-9-]{36}|api\/v1\/conversations\/[a-f0-9-]{36}\/messages\/[a-f0-9-]{36}\/images\/[a-f0-9]{64})$/i

export function editorImageURL(source) {
  if (typeof source !== 'string' || /[\s\\]/.test(source)) return ''
  try {
    const url = new URL(source, window.location.origin)
    if (url.origin !== window.location.origin || url.username || url.password || !imagePath.test(url.pathname)) return ''
    if (url.pathname.startsWith('/uploads/')) url.searchParams.set('proxy', '1')
    return url.pathname + url.search
  } catch {
    return ''
  }
}

export function prepareEditorContent(content) {
  if (typeof content !== 'string') return ''
  const template = document.createElement('template')
  template.innerHTML = content
  template.content.querySelectorAll('script,style,link,iframe,object,embed,video,audio,source,track,svg,math,base,meta,template').forEach(el => el.remove())
  template.content.querySelectorAll('*').forEach(el => {
    const classes = [...el.classList].filter(name => ['ld-mention', 'ld-conversation-reference', 'inline-image'].includes(name))
    const source = el.tagName === 'IMG' ? el.getAttribute('src') || el.getAttribute('data-libredesk-image-src') : null
    for (const attr of [...el.attributes]) {
      if (!['href', 'alt', 'title', 'width', 'height', 'colspan', 'rowspan', 'data-type', 'data-id', 'data-label', 'data-conversation-uuid', 'data-reference-number'].includes(attr.name)) {
        el.removeAttribute(attr.name)
      }
    }
    if (classes.length) el.setAttribute('class', classes.join(' '))
    if (source) el.setAttribute('data-libredesk-image-src', source)
  })
  return template.innerHTML
}

export function serializeEditorContent(content) {
  const template = document.createElement('template')
  template.innerHTML = content
  template.content.querySelectorAll('img[data-libredesk-image-src]').forEach(el => {
    el.setAttribute('src', el.getAttribute('data-libredesk-image-src'))
    el.removeAttribute('data-libredesk-image-src')
  })
  return template.innerHTML
}
