const openSockets = []
const createdInboxes = []

const baseConfig = () => ({
  brand_name: 'Cypress',
  colors: { primary: '#112233' },
  launcher: { position: 'right', spacing: { side: 20, bottom: 20 } },
  session_duration: '4h',
  visitors: { allow_start_conversation: true },
  users: { allow_start_conversation: true }
})

const merge = (base, extra) => {
  const out = { ...base }
  Object.entries(extra || {}).forEach(([key, value]) => {
    out[key] =
      value && typeof value === 'object' && !Array.isArray(value)
        ? merge(base[key] || {}, value)
        : value
  })
  return out
}

Cypress.Commands.add('createLivechatInbox', (configOverrides = {}, inboxOverrides = {}) => {
  const stamp = `${Date.now()}${Cypress._.random(1000, 9999)}`
  const payload = {
    name: `Livechat ${stamp}`,
    channel: 'livechat',
    enabled: true,
    from: `Livechat <livechat+${stamp}@cypress.test>`,
    config: merge(baseConfig(), configOverrides),
    ...inboxOverrides
  }
  cy.login()
  return cy.api('POST', '/api/v1/inboxes', payload).then(({ body }) => {
    const inbox = { id: body.data.id, uuid: body.data.uuid, payload }
    expect(inbox.uuid, 'inbox uuid').to.be.a('string').and.not.be.empty
    createdInboxes.push(inbox.id)
    return inbox
  })
})

Cypress.Commands.add('saveLivechatInbox', (inbox, configOverrides = {}) => {
  const payload = { ...inbox.payload, config: merge(inbox.payload.config, configOverrides) }
  inbox.payload = payload
  return cy.api('PUT', `/api/v1/inboxes/${inbox.id}`, payload).its('status').should('eq', 200)
})

Cypress.Commands.add('widgetInit', (inboxUuid, body = {}, options = {}) =>
  cy
    .request({
      method: 'POST',
      url: '/api/v1/widget/chat/conversations/init',
      headers: { 'X-Libredesk-Inbox-ID': inboxUuid, ...(options.headers || {}) },
      body: { message: body.message || `Visitor hello ${Date.now()}`, ...body },
      failOnStatusCode: options.failOnStatusCode !== false
    })
    .then((res) => {
      if (res.status !== 200) return res
      return {
        status: res.status,
        body: res.body,
        message: res.requestBody?.message,
        sessionToken: res.body.data.session_token,
        conversationUuid: res.body.data.conversation.uuid
      }
    })
)

Cypress.Commands.add('widgetApi', (method, path, sessionToken, inboxUuid, body, options = {}) =>
  cy.request({
    method,
    url: path,
    body,
    failOnStatusCode: options.failOnStatusCode !== false,
    headers: {
      'X-Libredesk-Inbox-ID': inboxUuid,
      ...(sessionToken ? { Authorization: `Bearer ${sessionToken}` } : {}),
      ...(options.headers || {})
    }
  })
)

// keepAlive false lets the server's 20s read deadline expire.
Cypress.Commands.add('openWidgetSocket', (sessionToken, inboxUuid, { keepAlive = true } = {}) =>
  cy.window().then((win) => {
    const scheme = win.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new win.WebSocket(`${scheme}//${win.location.host}/widget/ws`)
    const socket = { ws, received: [], closed: false, pinger: null }
    ws.addEventListener('message', (event) => socket.received.push(JSON.parse(event.data)))
    ws.addEventListener('close', () => {
      socket.closed = true
      win.clearInterval(socket.pinger)
    })
    ws.addEventListener('open', () => {
      ws.send(JSON.stringify({ type: 'join', token: sessionToken, data: { inbox_id: inboxUuid } }))
      if (keepAlive) {
        socket.pinger = win.setInterval(() => ws.send(JSON.stringify({ type: 'ping' })), 5000)
      }
    })
    openSockets.push(socket)
    return socket
  })
)

// cy.wrap(null).should(fn) retries the callback until it passes or times out.
Cypress.Commands.add('waitForFrame', (predicate, errorMsg, timeout = 20000) =>
  cy.wrap(null, { timeout, log: false }).should(() => {
    expect(predicate(), errorMsg).to.be.true
  })
)

Cypress.Commands.add('frameOfType', (socket, type) => {
  const frame = socket.received.find((m) => m.type === type)
  return cy.wrap(frame, { log: false })
})

export const joined = (socket) => () => socket.received.some((m) => m.type === 'joined')

export const gotMessage = (socket, bodyText) => () =>
  socket.received.some(
    (m) =>
      m.type === 'new_message' &&
      `${m.data?.content || ''}${m.data?.text_content || ''}`.includes(bodyText)
  )

// cy.then defers the URL until conversationUuid has actually been assigned.
Cypress.Commands.add('agentReply', (conversationUuidRef, body) =>
  cy.then(() => {
    const uuid = typeof conversationUuidRef === 'function' ? conversationUuidRef() : conversationUuidRef
    return cy
      .api('POST', `/api/v1/conversations/${uuid}/messages`, {
        message: `<p>${body}</p>`,
        private: false,
        sender_type: 'agent'
      })
      .its('status')
      .should('eq', 200)
  })
)

afterEach(() => {
  openSockets.splice(0).forEach((socket) => {
    if (socket.ws) socket.ws.close()
  })
})

after(() => {
  if (!createdInboxes.length) return
  cy.login()
  createdInboxes.splice(0).forEach((id) => {
    cy.api('DELETE', `/api/v1/inboxes/${id}`, null, { failOnStatusCode: false })
  })
})

const embedHostPath = '/__widget-embed-test'

// Served from the app's own origin, else the iframe is cross-origin and its DOM is unreachable.
Cypress.Commands.add('visitWidgetHost', (inboxUuid, { secret = null, jwtPayload = null } = {}) => {
  const html = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>widget embed host</title></head>
<body>
<h1 id="host-title">Customer site</h1>
<script>
window.__widgetMessages = []
window.addEventListener('message', function (e) {
  if (!e.source || e.source === window) return
  var lc = window.Libredesk
  var fromIframe = !!(lc && lc.iframe && e.source === lc.iframe.contentWindow)
  window.__widgetMessages.push(((e.data || {}).type || '?') + ':' + fromIframe)
})
</script>
<script>
function b64url (bytes) {
  let s = ''
  bytes.forEach(b => { s += String.fromCharCode(b) })
  return btoa(s).replace(/\\+/g, '-').replace(/\\//g, '_').replace(/=+$/, '')
}
async function signJWT (payload, secret) {
  const enc = new TextEncoder()
  const h = b64url(enc.encode(JSON.stringify({ alg: 'HS256', typ: 'JWT' })))
  const p = b64url(enc.encode(JSON.stringify(payload)))
  const key = await crypto.subtle.importKey('raw', enc.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign'])
  const sig = await crypto.subtle.sign('HMAC', key, enc.encode(h + '.' + p))
  return h + '.' + p + '.' + b64url(new Uint8Array(sig))
}
;(async function () {
  const cfg = { baseURL: window.location.origin, inboxID: ${JSON.stringify(inboxUuid)} }
  const secret = ${JSON.stringify(secret)}
  const payload = ${JSON.stringify(jwtPayload)}
  if (secret && payload) {
    payload.exp = Math.floor(Date.now() / 1000) + 3600
    cfg.userJWT = await signJWT(payload, secret)
    window.__widgetJWT = cfg.userJWT
  }
  window.LibredeskSettings = cfg
  const s = document.createElement('script')
  s.src = '/widget.js'
  document.body.appendChild(s)
})()
</script>
</body></html>`
  cy.intercept('GET', `${embedHostPath}*`, {
    statusCode: 200,
    headers: { 'content-type': 'text/html; charset=utf-8' },
    body: html
  }).as('embedHost')
  cy.visit(embedHostPath)
  return cy.window({ timeout: 20000 }).its('Libredesk.toggleButton', { timeout: 20000 })
})

Cypress.Commands.add('widgetLauncher', () =>
  cy.window().its('Libredesk.toggleButton').then((el) => cy.wrap(el, { log: false }))
)

Cypress.Commands.add('widgetWrapperSide', () =>
  cy.window().its('Libredesk.widgetButtonWrapper').then((el) => {
    const s = el.ownerDocument.defaultView.getComputedStyle(el)
    return { left: s.left, right: s.right, bottom: s.bottom }
  })
)

Cypress.Commands.add('widgetBody', () =>
  cy
    .get('iframe[src*="/widget?inbox_id="]', { timeout: 20000 })
    .its('0.contentDocument.body', { timeout: 20000 })
    .should('not.be.empty')
    .then((body) => cy.wrap(body, { log: false }))
)

// No cy.login() here: cy.session() blanks the page, which would tear down an embedded widget mid-test.
Cypress.Commands.add('latestConversation', (inbox) => {
  return cy
    .api(
      'GET',
      '/api/v1/conversations/all?order=desc&order_by=conversations.created_at&page=1&page_size=50'
    )
    .then(({ body }) => {
      const match = body.data.results.find((c) => c.inbox_name === inbox.payload.name)
      expect(match, `no conversation found for inbox ${inbox.payload.name}`).to.exist
      return match
    })
})

Cypress.Commands.add('agentReplyToLatestConversation', (inbox, body) =>
  cy.latestConversation(inbox).then((conversation) =>
    cy
      .api('POST', `/api/v1/conversations/${conversation.uuid}/messages`, {
        message: `<p>${body}</p>`,
        private: false,
        sender_type: 'agent'
      })
      .its('status')
      .should('eq', 200)
  )
)

Cypress.Commands.add('openWidget', (inbox, opts = {}) => {
  cy.visitWidgetHost(inbox.uuid, opts)
  cy.widgetLauncher().click()
  return cy.widgetBody()
})

Cypress.Commands.add('widgetSend', (text) => {
  cy.widgetBody().find('textarea').should('be.visible').type(text)
  cy.widgetBody().find('button[aria-label="Send"]').click()
  return cy.widgetBody().contains(text, { timeout: 20000 }).should('be.visible')
})

Cypress.Commands.add('closeConversation', (inbox) =>
  cy.latestConversation(inbox).then((conversation) =>
    cy
      .api('PUT', `/api/v1/conversations/${conversation.uuid}/status`, { status: 'Closed' })
      .its('status')
      .should('eq', 200)
  )
)

Cypress.Commands.add('conversationForInbox', (inbox) =>
  cy
    .api('GET', '/api/v1/conversations/all?order=desc&order_by=conversations.created_at&page=1&page_size=50')
    .then(({ body }) => body.data.results.find((c) => c.inbox_name === inbox.payload.name) || null)
)

// Retries because the visitor-to-contact merge lands on a later widget request, not on the exchange itself.
Cypress.Commands.add('waitForConversationContact', (inbox, email, timeout = 20000) =>
  cy.wrap(null, { timeout, log: false }).should(() => {
    const found = Cypress.$.ajax({
      url: '/api/v1/conversations/all?order=desc&order_by=conversations.created_at&page=1&page_size=50',
      async: false
    })
    const results = JSON.parse(found.responseText).data.results
    const match = results.find((c) => c.inbox_name === inbox.payload.name)
    expect(match && match.contact && match.contact.email, 'contact email on the conversation').to.eq(email)
  })
)

Cypress.Commands.add('setDefaultBusinessHours', (businessHoursId) =>
  cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
    const settings = { ...body.data, 'app.business_hours_id': businessHoursId }
    delete settings['app.version']
    delete settings['app.update']
    delete settings['app.restart_required']
    return cy.api('PUT', '/api/v1/settings/general', settings).its('status').should('eq', 200)
  })
)

// The widget renders a message optimistically, so the session cookie can lag behind it.
Cypress.Commands.add('waitForWidgetCookie', (inbox, type = 'session', timeout = 20000) =>
  cy.wrap(null, { timeout, log: false }).should(() => {
    const doc = cy.state('window').document
    expect(doc.cookie, `libredesk-${type} cookie`).to.include(`libredesk-${type}-${inbox.uuid}=`)
  })
)

const b64url = (bytes) => {
  let s = ''
  bytes.forEach((b) => {
    s += String.fromCharCode(b)
  })
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

Cypress.Commands.add('signWidgetJWT', (payload, secret) => {
  const enc = new TextEncoder()
  const body = { exp: Math.floor(Date.now() / 1000) + 3600, ...payload }
  const header = b64url(enc.encode(JSON.stringify({ alg: 'HS256', typ: 'JWT' })))
  const claims = b64url(enc.encode(JSON.stringify(body)))
  return cy.wrap(
    window.crypto.subtle
      .importKey('raw', enc.encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign'])
      .then((key) => window.crypto.subtle.sign('HMAC', key, enc.encode(`${header}.${claims}`)))
      .then((sig) => `${header}.${claims}.${b64url(new Uint8Array(sig))}`),
    { log: false }
  )
})
