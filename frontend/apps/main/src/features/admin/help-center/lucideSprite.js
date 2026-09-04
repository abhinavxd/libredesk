const SPRITE_URL = '/static/public/static/lucide-sprite.svg'

let spritePromise = null

// Injects the lucide sprite into the document once so `<use href="#name">` resolves, and
// returns the icon names it defines.
export function loadLucideSprite() {
  if (!spritePromise) {
    spritePromise = fetch(SPRITE_URL)
      .then((res) => {
        if (!res.ok) throw new Error(`sprite fetch failed: ${res.status}`)
        return res.text()
      })
      .then((text) => {
        const doc = new DOMParser().parseFromString(text, 'image/svg+xml')
        const symbols = [...doc.querySelectorAll('symbol[id]')]
        const holder = document.createElement('div')
        // Not display:none - Safari won't resolve <use> refs into hidden sprites.
        holder.style.cssText = 'position:absolute;width:0;height:0;overflow:hidden'
        holder.setAttribute('aria-hidden', 'true')
        holder.appendChild(document.importNode(doc.documentElement, true))
        document.body.appendChild(holder)
        return symbols.map((s) => s.id)
      })
      .catch((err) => {
        spritePromise = null
        throw err
      })
  }
  return spritePromise
}
