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

// sessionForWidgetRequest returns the session a widget request must send.
// A bound session is the one captured before dispatch. If the live session has
// moved, the request is dropped instead of borrowing the new bearer token.
export function sessionForWidgetRequest(boundSession, liveSession) {
  if (boundSession !== undefined && boundSession !== liveSession) {
    throw sessionChangedError()
  }
  return boundSession !== undefined ? boundSession : liveSession
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

  // tokenSiteKey is the key the posted token was issued for, not the key at error time.
  let tokenSiteKey = ''

  const withToken = async (payload) => {
    if (changed()) throw sessionChangedError()
    const siteKey = getSiteKey()
    if (!needsTurnstileToken(siteKey, visitorAtStart)) return payload
    let token
    try {
      token = await requestToken(siteKey)
    } catch (error) {
      if (changed()) throw sessionChangedError()
      throw error
    }
    if (changed()) throw sessionChangedError()
    tokenSiteKey = siteKey
    return { ...payload, turnstile_token: token }
  }

  const guardedPost = async (payload) => {
    if (changed()) throw sessionChangedError()
    const response = await post(payload, sessionAtStart)
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
      try {
        await refreshSiteKey()
      } catch (refreshError) {
        if (changed()) throw sessionChangedError()
        throw refreshError
      }
      // A rejected token for the same site key is final. The visitor can submit again.
      // A campaign create has no token yet, and a changed site key needs a new one.
      if (payload?.turnstile_token && getSiteKey() === tokenSiteKey) throw error
      payload = await withToken(data)
      if (!payload?.turnstile_token) throw error
    }
  }
}
