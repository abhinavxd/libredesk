// No IMAP here, the customer side is simulated over the API with messages:write_as_contact.

describe('Incoming conversation', () => {
  const stamp = Date.now()
  const inboxName = `Incoming Inbox ${stamp}`
  const roleName = `Incoming Role ${stamp}`
  const agentEmail = `incoming.agent.${stamp}@example.com`
  const agentPassword = 'StrongPass!123'
  const contactFirstName = 'Incoming'
  const contactLastName = `Customer${stamp}`
  const contactName = `${contactFirstName} ${contactLastName}`
  const subject = `Incoming subject ${stamp}`
  const firstInbound = `First inbound message ${stamp}`
  const secondInbound = `Second inbound message ${stamp}`
  const replyBody = `Agent reply ${stamp}`

  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)

  let conversationUuid
  let roleId
  let agentId
  let inboxId

  const loginAsAgent = () => {
    cy.session(
      agentEmail,
      () => {
        cy.visit('/')
        cy.get('#email').clear().type(agentEmail)
        cy.get('#password').clear().type(agentPassword, { log: false })
        cy.contains('button', 'Sign in').click()
        cy.url().should('include', '/inboxes')
      },
      {
        validate() {
          cy.request('/api/v1/agents/me').its('status').should('eq', 200)
        }
      }
    )
  }

  const openConversation = () => {
    cy.intercept('GET', '**/messages?page=*').as('loadMessages')
    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.wait('@loadMessages')
  }

  const bubbleFor = (text) =>
    cy.get('[data-message-uuid]').contains(text).closest('[data-message-uuid]')

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/roles', {
      name: roleName,
      description: 'Agent role that may post messages on behalf of a contact',
      permissions: [
        'conversations:read_all',
        'conversations:read_unassigned',
        'conversations:read_assigned',
        'conversations:read',
        'conversations:update_status',
        'messages:read',
        'messages:write',
        'messages:write_as_contact',
        'view:manage'
      ]
    }).then((res) => {
      roleId = res.body.data.id
    })
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Incoming',
      last_name: `Agent${stamp}`,
      email: agentEmail,
      roles: [roleName],
      enabled: true,
      send_welcome_email: false
    }).then(({ body }) => {
      agentId = body.data.id
      cy.api('PUT', `/api/v1/agents/${body.data.id}`, {
        first_name: 'Incoming',
        last_name: `Agent${stamp}`,
        email: agentEmail,
        roles: [roleName],
        enabled: true,
        new_password: agentPassword
      })
    })

    cy.api('POST', '/api/v1/inboxes', {
      name: inboxName,
      channel: 'email',
      enabled: true,
      from: `Incoming <incoming+${stamp}@cypress.test>`,
      config: {
        auth_type: 'password',
        imap: [],
        smtp: [
          {
            host: smtpHost,
            port: smtpPort,
            auth_protocol: 'none',
            max_conns: 2,
            idle_timeout: '5s',
            pool_wait_timeout: '5s',
            max_msg_retries: 1,
            tls_type: 'none'
          }
        ]
      }
    }).then(({ body }) => {
      inboxId = body.data.id
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: body.data.id,
        contact_email: `incoming.customer.${stamp}@example.com`,
        first_name: contactFirstName,
        last_name: contactLastName,
        subject,
        content: `<p>${firstInbound}</p>`,
        initiator: 'contact'
      }).then((res) => {
        conversationUuid = res.body.data.uuid
        expect(conversationUuid, 'conversation uuid').to.be.a('string').and.not.be.empty
      })
    })
  })

  // Conversations have no delete endpoint, so the conversation itself stays.
  // Teardown never asserts: a failed cleanup must not mask the real failure.
  // Logs back in as System because the throwaway agent cannot delete these.
  after(() => {
    cy.login()
    const drop = (path) => cy.api('DELETE', path, null, { failOnStatusCode: false })
    if (inboxId) drop(`/api/v1/inboxes/${inboxId}`)
    if (agentId) drop(`/api/v1/agents/${agentId}`)
    if (roleId) drop(`/api/v1/roles/${roleId}`)
  })

  beforeEach(() => {
    cy.viewport(1440, 900) // desktop layout: list, conversation and sidebar all on screen
    loginAsAgent()
  })

  it('shows the new conversation in the open list', () => {
    cy.intercept('GET', '**/conversations/all?*').as('loadList')

    cy.visit('/inboxes/all')
    cy.wait('@loadList')
    cy.contains(contactName).should('exist')
    cy.contains(subject).should('exist')
  })

  it('shows it in the unassigned list too', () => {
    cy.intercept('GET', '**/conversations/unassigned?*').as('loadUnassigned')

    cy.visit('/inboxes/unassigned')
    cy.wait('@loadUnassigned')
    cy.contains(contactName).should('exist')
  })

  it('renders the first message as an inbound bubble', () => {
    openConversation()

    cy.contains(firstInbound).should('be.visible')
    bubbleFor(firstInbound).find('div.text-left').should('have.class', 'items-start')
    bubbleFor(firstInbound)
      .find('.message-bubble')
      .should('not.have.class', 'bg-secondary')
      .and('not.have.class', 'bg-private')
  })

  it('renders the agent reply as outgoing', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendReply')

    openConversation()
    // The recipient comes from the loaded thread, sending early fails with "recipient required".
    cy.get('input[placeholder="Email addresses separated by comma"]')
      .first()
      .should('not.have.value', '')
    cy.get('.tiptap.ProseMirror').first().click().type(replyBody)
    cy.contains('button', /^Send$/).click() // exact: the split button next to it is "send and set status"

    cy.wait('@sendReply').its('response.statusCode').should('eq', 200)
    bubbleFor(replyBody).find('div.text-left').should('have.class', 'items-end')
    bubbleFor(replyBody)
      .find('.message-bubble')
      .should('have.class', 'bg-secondary')
      .and('not.have.class', 'bg-private')
  })

  it('shows a follow-up inbound message in the open thread', () => {
    openConversation()
    cy.contains(replyBody).should('exist')

    cy.api('POST', `/api/v1/conversations/${conversationUuid}/messages`, {
      message: `<p>${secondInbound}</p>`,
      sender_type: 'contact'
    })
      .its('status')
      .should('eq', 200)

    cy.contains(secondInbound, { timeout: 15000 }).should('be.visible')
    bubbleFor(secondInbound).find('div.text-left').should('have.class', 'items-start')
    bubbleFor(secondInbound)
      .find('.message-bubble')
      .should('not.have.class', 'bg-secondary')
      .and('not.have.class', 'bg-private')
  })
})
