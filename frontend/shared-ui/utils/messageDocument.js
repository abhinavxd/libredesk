import { localMediaURL } from './resourceURL.js'

export function messageDocument(html) {
  const template = document.createElement('template')
  template.innerHTML = html
  const sources = new Set()
  for (const image of template.content.querySelectorAll('img')) {
    const source = localMediaURL(image.getAttribute('src'))
    image.removeAttribute('srcset')
    if (!source) {
      image.replaceWith(document.createTextNode(image.getAttribute('alt') || ''))
      continue
    }
    image.setAttribute('src', source)
    image.setAttribute('referrerpolicy', 'no-referrer')
    const url = new URL(source, window.location.origin)
    sources.add(url.origin + url.pathname)
  }
  const nonce = Array.from(crypto.getRandomValues(new Uint8Array(18)), byte => byte.toString(16).padStart(2, '0')).join('')
  const policy = `default-src 'none'; script-src 'none'; img-src ${[...sources].join(' ') || "'none'"}; style-src 'unsafe-inline'; style-src-elem 'nonce-${nonce}'; style-src-attr 'unsafe-inline'; base-uri 'none'; form-action 'none'`
  return `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="${policy}"><meta name="referrer" content="no-referrer"><meta http-equiv="x-dns-prefetch-control" content="off"><style nonce="${nonce}">
html { overflow: hidden; }
body { display: flow-root; margin: 0; overflow-wrap: anywhere; font: 14px/1.5 system-ui, sans-serif; }
* { box-sizing: border-box; max-width: 100%; }
p, ul, ol { margin: 0 0 8px; }
p:last-child, ul:last-child, ol:last-child { margin-bottom: 0; }
ul, ol { padding-left: 24px; }
li { padding-left: 4px; }
h1, h2, h3, h4, h5, h6 { margin: 0; font-size: 20px; font-weight: 700; }
blockquote { margin: 0; }
body.hide-quotes blockquote { display: none; }
pre, code { white-space: pre-wrap; }
table { border-collapse: collapse; table-layout: fixed; }
img { height: auto; cursor: zoom-in; }
a { color: var(--link-color, #0066cc); text-decoration: none; }
a:hover { text-decoration: underline; }
</style></head><body>${template.innerHTML}</body></html>`
}
