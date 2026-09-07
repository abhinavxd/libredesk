import { joined, gotMessage } from '../../../support/livechat'

describe('Live chat widget connection resilience', () => {
  let inbox
  let session

  beforeEach(() => {
    cy.createLivechatInbox().then((created) => {
      inbox = created
    })
    cy.then(() => cy.widgetInit(inbox.uuid)).then((res) => {
      session = res
    })
    cy.visit('/inboxes/all') // same-origin window to open the widget socket from
  })

  it('keeps delivering agent replies after the inbox is saved', () => {
    const before = `Reply before save ${Date.now()}`
    const after = `Reply after save ${Date.now()}`
    let socket

    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      socket = s
    })
    cy.then(() => cy.waitForFrame(joined(socket), 'widget never joined the inbox'))

    cy.agentReply(() => session.conversationUuid, before)
    cy.then(() => cy.waitForFrame(gotMessage(socket, before), 'agent reply never reached the widget'))

    cy.then(() => cy.saveLivechatInbox(inbox, { brand_name: 'Cypress saved' }))
    cy.then(() => cy.waitForFrame(() => socket.closed, 'inbox save left the widget socket open'))

    let reconnected
    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      reconnected = s
    })
    cy.then(() => cy.waitForFrame(joined(reconnected), 'widget never rejoined after the inbox save'))

    cy.agentReply(() => session.conversationUuid, after)
    cy.then(() =>
      cy.waitForFrame(gotMessage(reconnected, after), 'reply after the inbox save never reached the widget')
    )
  })

  it('drops a socket that stops sending pings', () => {
    let socket

    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid, { keepAlive: false })).then((s) => {
      socket = s
    })
    cy.then(() => cy.waitForFrame(joined(socket), 'widget never joined the inbox'))
    cy.then(() =>
      cy.waitForFrame(() => socket.closed, 'silent socket outlived the read deadline', 30000)
    )
  })

  it('refuses a join past the per-user connection cap', () => {
    const cap = 10
    const sockets = []

    Cypress._.times(cap, () => {
      cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => sockets.push(s))
    })
    cy.then(() =>
      cy.waitForFrame(
        () => sockets.every((s) => s.received.some((m) => m.type === 'joined')),
        'not every socket under the cap joined'
      )
    )

    let overflow
    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      overflow = s
    })
    cy.then(() =>
      cy.waitForFrame(
        () => overflow.received.some((m) => m.type === 'error'),
        'join past the connection cap was not refused'
      )
    )
    cy.then(() => expect(joined(overflow)(), 'capped socket still joined').to.be.false)
  })

  it('replays messages missed while disconnected', () => {
    const missed = `Reply while offline ${Date.now()}`
    let socket

    cy.then(() => cy.openWidgetSocket(session.sessionToken, inbox.uuid)).then((s) => {
      socket = s
    })
    cy.then(() => cy.waitForFrame(joined(socket), 'widget never joined the inbox'))
    cy.then(() => socket.ws.close())
    cy.then(() => cy.waitForFrame(() => socket.closed, 'socket never closed'))

    cy.agentReply(() => session.conversationUuid, missed)

    cy.then(() =>
      cy
        .widgetApi(
          'GET',
          `/api/v1/widget/chat/conversations/${session.conversationUuid}`,
          session.sessionToken,
          inbox.uuid
        )
        .then(({ status, body }) => {
          expect(status).to.eq(200)
          const messages = JSON.stringify(body.data)
          expect(messages, 'missed reply absent from the conversation').to.include(missed)
        })
    )
  })
})
