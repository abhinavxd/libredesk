const TURNSTILE_SRC = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

export const TURNSTILE_FAILED = 'Could not verify that you are a person. Please try again.'

let scriptPromise = null

export function needsTurnstileToken(siteKey, isVisitor) {
  return typeof siteKey === 'string' && siteKey.trim() !== '' && isVisitor !== false
}

export function loadTurnstileScript(doc = document) {
  if (typeof window !== 'undefined' && window.turnstile) {
    return Promise.resolve(window.turnstile)
  }
  if (scriptPromise) return scriptPromise
  scriptPromise = new Promise((resolve, reject) => {
    const script = doc.createElement('script')
    script.src = TURNSTILE_SRC
    script.async = true
    script.onload = () => {
      if (window.turnstile) resolve(window.turnstile)
      else {
        scriptPromise = null
        reject(new Error(TURNSTILE_FAILED))
      }
    }
    script.onerror = () => {
      scriptPromise = null
      reject(new Error(TURNSTILE_FAILED))
    }
    doc.head.appendChild(script)
  })
  return scriptPromise
}

// requestTurnstileToken renders a fresh widget at submit time. The Cloudflare dashboard
// chooses invisible or managed. The token is single use, so this does not keep a widget.
export function requestTurnstileToken(siteKey, options = {}) {
  const doc = options.document || document
  const load = options.load || (() => loadTurnstileScript(doc))
  return load(doc).then(
    (turnstile) =>
      new Promise((resolve, reject) => {
        const host = doc.createElement('div')
        host.style.position = 'fixed'
        host.style.left = '0'
        host.style.right = '0'
        host.style.bottom = '0'
        host.style.zIndex = '2147483646'
        host.style.display = 'flex'
        host.style.justifyContent = 'center'
        ;(doc.body || doc.documentElement).appendChild(host)

        let widgetId
        let settled = false
        const removeWidget = () => {
          if (widgetId == null) return
          try {
            turnstile.remove(widgetId)
          } catch {
            /* Widget was already removed. */
          }
        }
        const finish = (fn, value) => {
          if (settled) return
          settled = true
          removeWidget()
          host.remove()
          fn(value)
        }

        try {
          // The callback can run before render returns, so removal is repeated once the id exists.
          widgetId = turnstile.render(host, {
            sitekey: siteKey,
            callback: (token) => finish(resolve, token),
            'error-callback': () => finish(reject, new Error(TURNSTILE_FAILED)),
            'expired-callback': () => finish(reject, new Error(TURNSTILE_FAILED)),
            'timeout-callback': () => finish(reject, new Error(TURNSTILE_FAILED))
          })
        } catch {
          finish(reject, new Error(TURNSTILE_FAILED))
          return
        }
        if (settled) removeWidget()
        else if (widgetId == null) finish(reject, new Error(TURNSTILE_FAILED))
      })
  )
}

export async function attachTurnstileToken(data, { siteKey, isVisitor } = {}) {
  if (!needsTurnstileToken(siteKey, isVisitor)) return data
  const token = await requestTurnstileToken(siteKey)
  return { ...data, turnstile_token: token }
}
