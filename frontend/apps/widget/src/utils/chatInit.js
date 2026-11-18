import { needsTurnstileToken } from './turnstile.js'

export const SESSION_CHANGED = 'session_changed'

// The server sends this code when a new visitor conversation needs a Turnstile token.
export function isTurnstileRequired(error) {
  return error?.response?.data?.data?.code === 'turnstile_required'
}

function sessionChangedError() {
  const error = new Error('session changed')
  error.code = SESSION_CHANGED
  return error
}

// runChatInit posts a new live chat. A visitor token is collected for a normal init.
// A campaign reply is posted once without a token, because a reply to an existing
// conversation does not need one. The server asks for a token only when that reply
// would create a conversation. A session change during the check drops the submit,
// so visitor form data is not sent with the new bearer token.
export async function runChatInit({ data, getSession, isVisitor, getSiteKey, requestToken, refreshSiteKey, post }) {
  const sessionAtStart = getSession()
  const visitorAtStart = isVisitor()
  const changed = () => getSession() !== sessionAtStart

  const withToken = async (payload) => {
    if (changed()) throw sessionChangedError()
    const siteKey = getSiteKey()
    if (!needsTurnstileToken(siteKey, visitorAtStart)) return payload
    const token = await requestToken(siteKey)
    if (changed()) throw sessionChangedError()
    return { ...payload, turnstile_token: token }
  }

  const guardedPost = async (payload) => {
    if (changed()) throw sessionChangedError()
    const response = await post(payload)
    if (changed()) throw sessionChangedError()
    return response
  }

  let payload = data?.delivery_id ? data : await withToken(data)
  let refreshed = false
  for (;;) {
    try {
      return await guardedPost(payload)
    } catch (error) {
      if (error?.code === SESSION_CHANGED) throw error
      if (changed()) throw sessionChangedError()
      if (!visitorAtStart || !isTurnstileRequired(error) || refreshed) throw error
      refreshed = true
      const previousKey = getSiteKey()
      await refreshSiteKey()
      // A rejected token for the same site key is final. The visitor can submit again.
      // A campaign create has no token yet, and a changed site key needs a new one.
      if (payload?.turnstile_token && getSiteKey() === previousKey) throw error
      payload = await withToken(data)
      if (!payload?.turnstile_token) throw error
    }
  }
}
