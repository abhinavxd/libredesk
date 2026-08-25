/* eslint-env node */
// Stand-in for Meta's Graph API, so the WhatsApp channel can be driven without a real Business Account.
import http from 'node:http'
import crypto from 'node:crypto'

const DEFAULT_PORT = 9099

const state = {
  requests: [],
  templates: new Map(),
  media: new Map(),
  failSend: 0,
  failValidate: false,
  counter: 0
}

const json = (res, code, body) => {
  res.writeHead(code, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(body))
}

const metaError = (res, code, message, errorCode = 100) =>
  json(res, code, {
    error: { message, type: 'OAuthException', code: errorCode, error_user_msg: message, fbtrace_id: 'MOCK' }
  })

const nextID = () => ++state.counter

const readBody = (req) =>
  new Promise((resolve) => {
    const chunks = []
    req.on('data', (chunk) => chunks.push(chunk))
    req.on('end', () => resolve(Buffer.concat(chunks)))
  })

const parseUpload = (raw, contentType) => {
  const boundary = (contentType.match(/boundary=(.+)$/) || [])[1]
  if (!boundary) return {}
  const text = raw.toString('latin1')
  const part = text.split('--' + boundary).find((p) => p.includes('name="file"')) || ''
  return {
    filename: (part.match(/filename="([^"]*)"/) || [])[1] || '',
    contentType: (part.match(/Content-Type:\s*([^\r\n]+)/) || [])[1] || ''
  }
}

const handle = async (req, res) => {
  const url = new URL(req.url, 'http://mock')
  const path = url.pathname
  const raw = await readBody(req)
  const contentType = req.headers['content-type'] || ''
  const isMultipart = contentType.startsWith('multipart/form-data')

  let body = null
  if (raw.length && !isMultipart) {
    try {
      body = JSON.parse(raw.toString())
    } catch {
      body = raw.toString()
    }
  }

  if (path.startsWith('/__ctl/')) {
    const action = path.replace('/__ctl/', '')
    if (action === 'reset') {
      state.requests = []
      state.templates.clear()
      state.media.clear()
      state.failSend = 0
      state.failValidate = false
      return json(res, 200, { ok: true })
    }
    if (action === 'requests') {
      return json(res, 200, state.requests)
    }
    if (action === 'fail') {
      if (url.searchParams.has('send')) state.failSend = Number(url.searchParams.get('send'))
      if (url.searchParams.has('validate')) state.failValidate = url.searchParams.get('validate') === '1'
      return json(res, 200, { ok: true })
    }
    if (action === 'media') {
      const id = url.searchParams.get('id') || 'MEDIA' + nextID()
      state.media.set(id, { body: raw, mime: url.searchParams.get('mime') || 'application/octet-stream' })
      return json(res, 200, { id })
    }
    return json(res, 404, { error: 'unknown control endpoint' })
  }

  state.requests.push({
    method: req.method,
    path,
    query: url.search.replace(/^\?/, ''),
    auth: req.headers.authorization || '',
    body: isMultipart ? parseUpload(raw, contentType) : body
  })

  if (path.startsWith('/media/')) {
    const entry = state.media.get(path.replace('/media/', ''))
    if (!entry) return metaError(res, 404, 'media not found')
    res.writeHead(200, { 'Content-Type': entry.mime })
    return res.end(entry.body)
  }

  const parts = path.split('/').filter(Boolean).slice(1)

  if (parts.length === 2 && parts[1] === 'messages' && req.method === 'POST') {
    if (body?.status === 'read') return json(res, 200, { success: true })
    if (state.failSend > 0) {
      state.failSend--
      return metaError(res, 400, 'Message failed to send because more than 24 hours have passed since the customer last replied to this number.', 131047)
    }
    const messageID = 'wamid.MOCK' + nextID()
    // The app never exposes the Meta id over its API, so the recorded call is how a test learns it.
    state.requests[state.requests.length - 1].messageID = messageID
    return json(res, 200, {
      messaging_product: 'whatsapp',
      contacts: [{ input: body?.to, wa_id: body?.to }],
      messages: [{ id: messageID, message_status: 'accepted' }]
    })
  }

  if (parts.length === 2 && parts[1] === 'media' && req.method === 'POST') {
    const id = 'MEDIAUP' + nextID()
    state.media.set(id, { body: raw, mime: 'application/octet-stream' })
    return json(res, 200, { id })
  }

  if (parts.length === 2 && parts[1] === 'phone_numbers') {
    if (state.failValidate) return metaError(res, 400, 'Object with ID does not exist', 803)
    return json(res, 200, { data: [{ id: 'PHONE1', display_phone_number: '+1 555 000 1111' }] })
  }

  if (parts.length === 2 && parts[1] === 'subscribed_apps' && req.method === 'POST') {
    return json(res, 200, { success: true })
  }

  if (parts.length === 2 && parts[1] === 'message_templates') {
    if (req.method === 'GET') {
      return json(res, 200, { data: [...state.templates.values()], paging: {} })
    }
    if (req.method === 'POST') {
      const id = String(1000000000000000 + nextID())
      state.templates.set(id, { ...body, id, status: 'PENDING' })
      return json(res, 200, { id, status: 'PENDING', category: body?.category })
    }
    if (req.method === 'DELETE') {
      const name = url.searchParams.get('name')
      for (const [id, tmpl] of state.templates) {
        if (tmpl.name === name) state.templates.delete(id)
      }
      return json(res, 200, { success: true })
    }
  }

  // A single segment is either media info, a template edit, or the phone number check.
  if (parts.length === 1) {
    const id = parts[0]
    if (req.method === 'POST') {
      const tmpl = state.templates.get(id)
      if (tmpl) state.templates.set(id, { ...tmpl, ...body, status: 'PENDING' })
      return json(res, 200, { success: true })
    }
    const entry = state.media.get(id)
    if (entry) {
      return json(res, 200, {
        url: `http://127.0.0.1:${server.address().port}/media/${id}`,
        mime_type: entry.mime,
        file_size: entry.body.length,
        id,
        messaging_product: 'whatsapp'
      })
    }
    if (state.failValidate) return metaError(res, 400, 'Object with ID does not exist', 803)
    return json(res, 200, { id, display_phone_number: '+1 555 000 1111', verified_name: 'Mock Co' })
  }

  return metaError(res, 404, 'unsupported request ' + path)
}

const server = http.createServer((req, res) => {
  handle(req, res).catch(() => metaError(res, 500, 'mock failure'))
})

export const start = (port = DEFAULT_PORT) =>
  new Promise((resolve, reject) => {
    server.once('error', reject)
    server.listen(port, '127.0.0.1', () => resolve(server))
  })

export const stop = () => new Promise((resolve) => server.close(resolve))

export const control = {
  reset() {
    state.requests = []
    state.templates.clear()
    state.media.clear()
    state.failSend = 0
    state.failValidate = false
    return null
  },
  requests: () => state.requests,
  failSend(n) {
    state.failSend = n
    return null
  },
  failValidate(on) {
    state.failValidate = !!on
    return null
  },
  putMedia({ id, body, mime }) {
    state.media.set(id, { body: Buffer.from(body), mime })
    return null
  },
  sign({ body, secret }) {
    return 'sha256=' + crypto.createHmac('sha256', secret).update(body).digest('hex')
  }
}

// Allow running standalone: `node cypress/support/metaMock.mjs`
if (process.argv[1] && process.argv[1].endsWith('metaMock.mjs')) {
  start(Number(process.env.META_MOCK_PORT) || DEFAULT_PORT).then((s) =>
    console.log('meta mock listening on', s.address())
  )
}
