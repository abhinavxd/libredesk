import { joined, gotMessage } from '../../../support/livechat'

describe('Live chat widget messaging', () => {
  let inbox
  let session
  let socket
  const openingMessage = `Visitor opening ${Date.now()}`

  beforeEach(() => {
    cy.createLivechatInbox().then((created) => {
      inbox = created
    })
    cy.then(() => cy.widgetInit(inbox.uuid, { message: openingMessage })).then((res) => {
      session = res
    })
    cy.visit('/inboxes/all')
    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      socket = s
    })
    cy.then(() => cy.waitForFrame(joined(socket), 'widget never joined the inbox'))
  })

  it('lands the visitor opening message on the conversation', () => {
    cy.then(() =>
      cy
        .api('GET', `/api/v1/conversations/${session.conversationUuid}/messages`)
        .then(({ status, body }) => {
          expect(status).to.eq(200)
          expect(JSON.stringify(body.data), 'opening message missing').to.include(openingMessage)
        })
    )
  })

  it('delivers an agent reply over the socket', () => {
    const reply = `Agent reply ${Date.now()}`
    cy.agentReply(() => session.conversationUuid, reply)
    cy.then(() => cy.waitForFrame(gotMessage(socket, reply), 'agent reply never reached the widget'))
  })

  it('accepts a visitor reply and echoes it back to the widget', () => {
    const message = `Visitor reply ${Date.now()}`
    cy.then(() =>
      cy
        .widgetApi(
          'POST',
          `/api/v1/widget/chat/conversations/${session.conversationUuid}/message`,
          session.sessionToken,
          inbox.uuid,
          { message }
        )
        .its('status')
        .should('eq', 200)
    )
    cy.then(() =>
      cy
        .api('GET', `/api/v1/conversations/${session.conversationUuid}/messages`)
        .then(({ body }) => {
          expect(JSON.stringify(body.data), 'visitor reply missing').to.include(message)
        })
    )
  })

  it('rejects an empty and an oversized visitor message', () => {
    const send = (message) =>
      cy.widgetApi(
        'POST',
        `/api/v1/widget/chat/conversations/${session.conversationUuid}/message`,
        session.sessionToken,
        inbox.uuid,
        { message },
        { failOnStatusCode: false }
      )

    cy.then(() => send('').its('status').should('eq', 400))
    cy.then(() => send('x'.repeat(10001)).its('status').should('eq', 400))
  })

  it('pushes a conversation update when the agent changes status', () => {
    cy.then(() =>
      cy
        .api('PUT', `/api/v1/conversations/${session.conversationUuid}/status`, { status: 'Resolved' })
        .its('status')
        .should('eq', 200)
    )
    cy.then(() =>
      cy.waitForFrame(
        () => socket.received.some((m) => m.type === 'conversation_update'),
        'status change never reached the widget'
      )
    )
  })

  it('accepts typing and page_visit frames without dropping the socket', () => {
    cy.then(() => {
      socket.ws.send(
        JSON.stringify({
          type: 'typing',
          data: { conversation_uuid: session.conversationUuid, is_typing: true }
        })
      )
      socket.ws.send(
        JSON.stringify({ type: 'page_visit', data: { url: 'https://example.test/pricing', title: 'Pricing' } })
      )
    })
    cy.then(() =>
      cy.waitForFrame(() => socket.received.some((m) => m.type === 'pong'), 'socket stopped answering pings', 15000)
    )
    cy.then(() => expect(socket.closed, 'socket dropped after client frames').to.be.false)
  })

  it('marks the conversation seen for the visitor', () => {
    cy.then(() =>
      cy
        .widgetApi(
          'POST',
          `/api/v1/widget/chat/conversations/${session.conversationUuid}/update-last-seen`,
          session.sessionToken,
          inbox.uuid,
          {}
        )
        .its('status')
        .should('eq', 200)
    )
  })
})
