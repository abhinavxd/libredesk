// Needs the app running with whatsapp.api_url pointed at the mock Graph API (LIBREDESK_WHATSAPP__API_URL), else the suite stops.

import { hoursAgo, inboundPayload, statusPayload, templateStatusPayload } from '../../../support/whatsapp'

const stamp = `${Date.now()}`
const inboxName = `WhatsApp E2E ${stamp}`
const appSecret = `secret-${stamp}`
const verifyToken = `verify-${stamp}`
const phoneNumberID = `PN-${stamp}`
const wabaID = `WABA-${stamp}`
const waID = `9199${stamp.slice(-8)}`
const staleWaID = `9188${stamp.slice(-8)}`
const templateName = `e2e_order_update_${stamp}`

describe('WhatsApp channel', () => {
  let inboxID
  let conversationUUID
  let templateID

  const inbound = (message) => inboundPayload({ wabaID, phoneNumberID, waID, message })
  const post = (payload, options = {}) =>
    cy.waPostWebhook(payload, { inboxID, secret: appSecret, ...options })
  // cy.then defers the URL until conversationUUID has actually been assigned.
  const messages = () => cy.then(() => cy.waMessages(conversationUUID))

  before(() => {
    cy.login()
    cy.task('metaMock:reset')
  })

  beforeEach(() => cy.login())

  it('creates the inbox once Meta accepts the credentials', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: inboxName,
      channel: 'whatsapp',
      enabled: true,
      csat_enabled: true,
      reopen_window_hours: 24,
      config: {
        phone_number_id: phoneNumberID,
        waba_id: wabaID,
        access_token: 'e2e-access-token',
        app_secret: appSecret,
        webhook_verify_token: verifyToken,
        api_version: 'v25.0',
        csat_template_language: 'en_US',
        csat_template_body: 'How did we do?',
        csat_template_button_text: 'Rate us'
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(
        status,
        'creating the inbox needs whatsapp.api_url pointed at the mock (LIBREDESK_WHATSAPP__API_URL)'
      ).to.eq(200)
      inboxID = body.data.id
      expect(body.data.webhook_url, 'callback url').to.contain(`/webhooks/whatsapp/${inboxID}`)
      // Secrets never come back in the clear.
      expect(body.data.config.access_token).to.not.eq('e2e-access-token')
    })
  })

  it("answers Meta's verification handshake", () => {
    cy.request({
      url: `/webhooks/whatsapp/${inboxID}?hub.mode=subscribe&hub.verify_token=${verifyToken}&hub.challenge=CHAL123`
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body).to.eq('CHAL123')
    })
    cy.request({
      url: `/webhooks/whatsapp/${inboxID}?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=CHAL123`,
      failOnStatusCode: false
    })
      .its('status')
      .should('eq', 403)
  })

  it('rejects a delivery signed with the wrong secret', () => {
    post(
      inbound({
        from: waID,
        id: `wamid.BADSIG.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'text',
        text: { body: 'should not arrive' }
      }),
      { secret: 'not-the-app-secret' }
    )
      .its('status')
      .should('eq', 403)
  })

  it('turns an inbound message into a conversation', () => {
    post(
      inbound({
        from: waID,
        id: `wamid.IN1.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'text',
        text: { body: 'my order is late' }
      })
    )
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the conversation to be created',
      () =>
        cy
          .api('GET', '/api/v1/conversations/all?order=desc&order_by=conversations.created_at&page=1&page_size=50')
          .then(({ body }) => body.data.results.find((c) => c.inbox_name === inboxName) ?? null),
      (found) => Boolean(found)
    ).then((conversation) => {
      conversationUUID = conversation.uuid
      expect(conversation.last_message).to.eq('my order is late')
      expect(conversation.contact.first_name).to.eq('E2E')
    })

    cy.waPoll(
      'the inbound message to be stored',
      () => messages().then((list) => list.find((m) => m.content?.includes('my order is late')) ?? null),
      (message) => Boolean(message)
    ).then((message) => {
      expect(message.type).to.eq('incoming')
    })
  })

  it('ignores a redelivery of the same message', () => {
    post(
      inbound({
        from: waID,
        id: `wamid.IN1.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'text',
        text: { body: 'my order is late' }
      })
    )
      .its('status')
      .should('eq', 200)

    // eslint-disable-next-line cypress/no-unnecessary-waiting -- asserting a duplicate never lands needs a settle window
    cy.wait(1500)
    messages().then((list) => {
      expect(list.filter((m) => m.content?.includes('my order is late'))).to.have.length(1)
    })
  })

  it('downloads inbound media and attaches it', () => {
    const mediaID = `MEDIA.${stamp}`
    cy.task('metaMock:putMedia', { id: mediaID, body: 'file-contents', mime: 'text/plain' })

    post(
      inbound({
        from: waID,
        id: `wamid.IN2.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'document',
        document: { id: mediaID, mime_type: 'text/plain', filename: 'notes.txt', caption: 'the receipt' }
      })
    )
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the media message to be stored',
      () => messages().then((list) => list.find((m) => m.content?.includes('the receipt')) ?? null),
      (found) => Boolean(found)
    ).then((message) => {
      expect(message.content).to.contain('the receipt')
      expect(message.attachments, 'attachments').to.have.length(1)
      expect(message.attachments[0].name).to.eq('notes.txt')
    })
  })

  it('sends a free-form reply inside the 24-hour window', () => {
    cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
      message: '<p>Checking on it now</p>',
      sender_type: 'agent'
    })
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the text send to reach Meta',
      () => cy.waMetaCalls((r) => r.method === 'POST' && r.path.endsWith('/messages') && r.body?.type === 'text'),
      (calls) => calls.length > 0
    ).then((calls) => {
      const sent = calls[calls.length - 1].body
      expect(sent.to).to.eq(waID)
      // HTML is flattened before it reaches WhatsApp.
      expect(sent.text.body).to.eq('Checking on it now')
    })

    cy.waPoll(
      'the reply to be marked sent',
      () => messages().then((list) => list.find((m) => m.content?.includes('Checking on it now')) ?? null),
      (message) => message?.status === 'sent'
    )
  })

  it('records delivery status from Meta and keeps the newest one', () => {
    cy.waMetaCalls((r) => r.messageID && r.body?.type === 'text')
      .then((calls) => calls[calls.length - 1].messageID)
      .then((wamid) => {
        const status = (value) => statusPayload({ wabaID, phoneNumberID, waID, id: wamid, status: value })
        const providerStatus = (list) =>
          list.find((m) => m.content?.includes('Checking on it now'))?.meta?.provider_status

        post(status('delivered')).its('status').should('eq', 200)
        cy.waPoll('delivered to be recorded', messages, (list) => providerStatus(list) === 'delivered')

        post(status('read')).its('status').should('eq', 200)
        cy.waPoll('read to be recorded', messages, (list) => providerStatus(list) === 'read')

        // An out-of-order delivered must not walk the status back.
        post(status('delivered')).its('status').should('eq', 200)
        // eslint-disable-next-line cypress/no-unnecessary-waiting -- asserting the status does not move needs a settle window
        cy.wait(1500)
        messages().then((list) => expect(providerStatus(list)).to.eq('read'))
      })
  })

  it('marks a send that Meta refuses as failed and retries it cleanly', () => {
    cy.task('metaMock:failSend', 1)
    cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
      message: '<p>this one fails</p>',
      sender_type: 'agent'
    }).then(({ body }) => {
      const uuid = body.data.uuid

      cy.waPoll(
        'the send to be marked failed',
        () => messages().then((list) => list.find((m) => m.uuid === uuid) ?? null),
        (message) => message?.status === 'failed'
      ).then((failed) => {
        expect(failed.meta.provider_failure_reason).to.contain('24 hours')
      })

      cy.api('PUT', `/api/v1/conversations/${conversationUUID}/messages/${uuid}/retry`)
        .its('status')
        .should('eq', 200)

      cy.waPoll(
        'the retry to go out',
        () => messages().then((list) => list.find((m) => m.uuid === uuid) ?? null),
        (message) => message?.status === 'sent'
      ).then((sent) => {
        // The previous attempt's failure must not linger, or later status webhooks are ignored.
        expect(sent.meta.provider_failure_reason, 'stale failure reason').to.be.undefined
        expect(sent.meta.provider_status, 'stale provider status').to.not.eq('failed')
      })

      cy.waMetaCalls((r) => r.messageID && r.body?.text?.body === 'this one fails').then((calls) => {
        expect(calls, 'the retry reached Meta').to.have.length(1)
      })
    })
  })

  it('refuses to send a template Meta has not approved', () => {
    cy.api('POST', '/api/v1/whatsapp/templates', {
      inbox_id: inboxID,
      name: templateName,
      language: 'en_US',
      category: 'UTILITY',
      body_content: 'Hi {{name}}, order {{order_id}} is on its way.',
      sample_values: { name: 'Ravi', order_id: 'A1' }
    }).then(({ body }) => {
      templateID = body.data.id
      expect(body.data.status).to.eq('PENDING')
      expect(body.data.meta_template_id, 'Meta accepted the submission').to.not.be.null
    })

    cy.then(() =>
      cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
        message: '',
        sender_type: 'agent',
        whatsapp_template_id: templateID,
        whatsapp_template_params: { 'body:name': 'Ravi', 'body:order_id': 'A1' }
      }, { failOnStatusCode: false }).then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.message).to.contain('not approved')
      })
    )
  })

  it('applies the approval webhook without storing a bogus reason', () => {
    post(templateStatusPayload({ wabaID, name: templateName, event: 'APPROVED' }))
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the template to be approved',
      () => cy.api('GET', `/api/v1/whatsapp/templates/${templateID}`).then(({ body }) => body.data),
      (template) => template.status === 'APPROVED'
    ).then((template) => {
      // Meta sends reason "NONE" on approval, which must not surface as a rejection reason.
      expect(template.rejection_reason).to.be.oneOf([null, ''])
    })
  })

  it('blocks free-form once the reply window closes but still sends templates', () => {
    // The window clock only moves forward, so a closed window needs a contact whose one message is old.
    let staleUUID
    cy.waPostWebhook(
      inboundPayload({
        wabaID,
        phoneNumberID,
        waID: staleWaID,
        contactName: 'Stale Contact',
        message: {
          from: staleWaID,
          id: `wamid.STALE.${stamp}`,
          timestamp: hoursAgo(30),
          type: 'text',
          text: { body: 'this came in yesterday' }
        }
      }),
      { inboxID, secret: appSecret }
    )
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the stale conversation to be created',
      () =>
        cy
          .api('GET', '/api/v1/conversations/all?order=desc&order_by=conversations.created_at&page=1&page_size=50')
          .then(
            ({ body }) =>
              body.data.results.find(
                (c) => c.inbox_name === inboxName && c.contact.first_name === 'Stale'
              ) ?? null
          ),
      (found) => Boolean(found)
    ).then((conversation) => {
      staleUUID = conversation.uuid
    })

    cy.then(() =>
      cy.api('POST', `/api/v1/conversations/${staleUUID}/messages`, {
        message: '<p>outside the window</p>',
        sender_type: 'agent'
      }, { failOnStatusCode: false }).then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.message).to.contain('24-hour reply window')
      })
    )

    cy.then(() =>
      cy.api('POST', `/api/v1/conversations/${staleUUID}/messages`, {
        message: '',
        sender_type: 'agent',
        whatsapp_template_id: templateID,
        whatsapp_template_params: { 'body:name': 'Stale', 'body:order_id': 'B2' }
      })
        .its('status')
        .should('eq', 200)
    )

    cy.waPoll(
      'the template send to reach Meta',
      () => cy.waMetaCalls((r) => r.body?.template?.name === templateName),
      (calls) => calls.length > 0
    ).then((calls) => {
      expect(JSON.stringify(calls[calls.length - 1].body.template.components)).to.contain('Stale')
    })

    // The timeline shows the filled-in copy, not the placeholders.
    cy.waPoll(
      'the rendered template in the timeline',
      () => cy.waMessages(staleUUID),
      (list) => list.some((m) => m.content?.includes('order B2 is on its way'))
    )
  })

  it('rejects a template that does not belong to the inbox', () => {
    cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
      message: '',
      sender_type: 'agent',
      whatsapp_template_id: 99999999
    }, { failOnStatusCode: false })
      .its('status')
      .should('be.oneOf', [400, 404])
  })

  it('provisions the CSAT template and sends it when the conversation is resolved', () => {
    let csatName
    cy.then(() => {
      csatName = `libredesk_csat_${inboxID}`
    })

    cy.waPoll(
      'the CSAT template to be provisioned',
      () =>
        cy
          .api('GET', `/api/v1/whatsapp/templates?inbox_id=${inboxID}`)
          .then(({ body }) => body.data.find((t) => t.name === csatName) ?? null),
      (template) => Boolean(template)
    ).then((csat) => {
      expect(csat.body_content).to.eq('How did we do?')
      expect(JSON.stringify(csat.buttons)).to.contain('Rate us')
    })

    cy.then(() => post(templateStatusPayload({ wabaID, name: csatName, event: 'APPROVED' })))
    cy.waPoll(
      'the CSAT template to be approved',
      () =>
        cy
          .api('GET', `/api/v1/whatsapp/templates?inbox_id=${inboxID}`)
          .then(({ body }) => body.data.find((t) => t.name === csatName) ?? null),
      (template) => template?.status === 'APPROVED'
    )

    cy.api('PUT', `/api/v1/conversations/${conversationUUID}/status`, { status: 'Resolved' })
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the survey to be sent',
      () => cy.waMetaCalls((r) => r.body?.template?.name === csatName),
      (calls) => calls.length > 0
    ).then((calls) => {
      // The survey link carries the CSAT response id as the button parameter.
      expect(JSON.stringify(calls[calls.length - 1].body.template.components)).to.contain('sub_type')
    })
  })

  it('reopens the resolved conversation when the contact writes back', () => {
    post(
      inbound({
        from: waID,
        id: `wamid.REOPEN.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'text',
        text: { body: 'still broken' }
      })
    )
      .its('status')
      .should('eq', 200)

    cy.waPoll(
      'the conversation to reopen',
      () => cy.api('GET', `/api/v1/conversations/${conversationUUID}`).then(({ body }) => body.data),
      (conversation) => conversation.status === 'Open'
    )
    messages().then((list) => {
      expect(list.find((m) => m.content?.includes('still broken')), 'reopening message').to.exist
    })
  })

  it('drops inbound messages while the inbox is disabled', () => {
    cy.api('PUT', `/api/v1/inboxes/${inboxID}/toggle`).its('status').should('eq', 200)

    post(
      inbound({
        from: waID,
        id: `wamid.OFF.${stamp}`,
        timestamp: hoursAgo(0),
        type: 'text',
        text: { body: 'nobody is listening' }
      })
    )
      .its('status')
      .should('eq', 200)

    // eslint-disable-next-line cypress/no-unnecessary-waiting -- asserting the message is dropped needs a settle window
    cy.wait(2000)
    messages().then((list) => {
      expect(list.find((m) => m.content?.includes('nobody is listening'))).to.be.undefined
    })

    cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
      message: '<p>while disabled</p>',
      sender_type: 'agent'
    }, { failOnStatusCode: false })
      .its('status')
      .should('eq', 400)

    cy.api('PUT', `/api/v1/inboxes/${inboxID}/toggle`).its('status').should('eq', 200)
  })

  after(() => {
    if (!inboxID) return
    cy.login()
    cy.api('DELETE', `/api/v1/inboxes/${inboxID}`, null, { failOnStatusCode: false })
  })
})
