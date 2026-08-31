describe('Conversation lifecycle', () => {
  const stamp = Date.now()
  const inboxName = `Lifecycle Inbox ${stamp}`
  const teamName = `Lifecycle Team ${stamp}`
  const agentFirstName = 'Lifecycle'
  const agentLastName = `Agent${stamp}`
  const agentName = `${agentFirstName} ${agentLastName}`
  const tagName = `lifecycle-${stamp}`
  const contactLastName = `Customer${stamp}`
  const contactName = `Lifecycle ${contactLastName}`
  const subject = `Lifecycle subject ${stamp}`
  const noteBody = `Internal note ${stamp}`
  const replyBody = `Public reply ${stamp}`

  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)

  let conversationUuid
  let inboxId
  let teamId
  let agentId
  let tagId

  const openConversation = () => {
    cy.intercept('GET', '**/messages?page=*').as('loadMessages')
    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.wait('@loadMessages')
  }

  // The open set of sidebar sections is remembered per browser, so blind toggling closes it later.
  const openActionsSection = () => {
    cy.contains('button', 'Actions').then(($trigger) => {
      if ($trigger.attr('aria-expanded') !== 'true') cy.wrap($trigger).click()
    })
  }

  // The list column also renders a status dropdown; this one is the header badge.
  const statusBadge = () => cy.get('div.bg-primary.rounded-md')

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: inboxName,
      channel: 'email',
      enabled: true,
      from: `Lifecycle <lifecycle+${stamp}@cypress.test>`,
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
      const inboxID = body.data.id
      inboxId = inboxID
      cy.api('POST', '/api/v1/teams', {
        name: teamName,
        emoji: '🐝',
        conversation_assignment_type: 'Round robin',
        timezone: 'Asia/Kolkata'
      }).then((res) => {
        teamId = res.body.data.id
      })
      cy.api('POST', '/api/v1/agents', {
        first_name: agentFirstName,
        last_name: agentLastName,
        email: `lifecycle.agent.${stamp}@example.com`,
        roles: ['Agent'],
        enabled: true,
        send_welcome_email: false
      }).then((res) => {
        agentId = res.body.data.id
      })
      cy.api('POST', '/api/v1/tags', { name: tagName }).then((res) => {
        tagId = res.body.data.id
      })
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: `lifecycle.customer.${stamp}@example.com`,
        first_name: 'Lifecycle',
        last_name: contactLastName,
        subject,
        content: '<p>Customer opened this conversation.</p>',
        initiator: 'contact'
      }).then((res) => {
        conversationUuid = res.body.data.uuid
        expect(conversationUuid, 'conversation uuid').to.be.a('string').and.not.be.empty
      })
    })
  })

  // Conversations have no delete endpoint, so the conversation itself stays.
  // Teardown never asserts: a failed cleanup must not mask the real failure.
  after(() => {
    cy.login()
    const drop = (path) => cy.api('DELETE', path, null, { failOnStatusCode: false })
    if (tagId) drop(`/api/v1/tags/${tagId}`)
    if (agentId) drop(`/api/v1/agents/${agentId}`)
    if (teamId) drop(`/api/v1/teams/${teamId}`)
    if (inboxId) drop(`/api/v1/inboxes/${inboxId}`)
  })

  beforeEach(() => {
    cy.viewport(1440, 900) // desktop layout: list, conversation and sidebar all on screen
    cy.login()
  })

  it('opens the conversation from the inbox list', () => {
    cy.visit('/inboxes/all')
    cy.contains(contactName).click()

    cy.location('pathname').should('include', conversationUuid)
    cy.contains('Customer opened this conversation.').should('be.visible')
    statusBadge().should('contain.text', 'Open')
  })

  it('assigns the conversation to an agent', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignee/user`).as('assignAgent')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select agent').click()
    cy.get('[role="option"]').contains(agentName).click()

    cy.wait('@assignAgent').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', agentName).should('exist')
  })

  it('assigns the conversation to a team', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignee/team`).as('assignTeam')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select team').click()
    cy.get('[role="option"]').contains(teamName).click()

    cy.wait('@assignTeam').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', teamName).should('exist')
  })

  it('changes the priority', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/priority`).as('setPriority')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select priority').click()
    cy.get('[role="option"]').contains('High').click()

    cy.wait('@setPriority').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', 'High').should('exist')

    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.priority')
      .should('eq', 'High')
  })

  it('adds and removes a tag', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/tags`).as('setTags')

    openConversation()
    openActionsSection()
    cy.get('input[placeholder="Select tags"]').click()
    cy.get('[role="option"]').contains(tagName).click()

    cy.wait('@setTags').its('response.statusCode').should('eq', 200)
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.tags')
      .should('include', tagName)

    cy.contains('[data-radix-vue-collection-item]', tagName).find('button').click()
    cy.wait('@setTags').its('response.statusCode').should('eq', 200)
    cy.api('GET', `/api/v1/conversations/${conversationUuid}`)
      .its('body.data.tags')
      .should('not.include', tagName)
  })

  it('posts a private note that renders as a note, not a reply', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendNote')

    openConversation()
    cy.contains('button', 'Private note').click()
    cy.get('.tiptap.ProseMirror').first().click().type(noteBody)
    cy.contains('button', /^Send$/).click()

    cy.wait('@sendNote').its('response.statusCode').should('eq', 200)
    // Scoped to the thread: the list column shows the same text as the preview.
    cy.get('[data-message-uuid]').contains(noteBody).closest('.bg-private').should('exist')
  })

  it('sends a reply that appears in the thread', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendReply')

    openConversation()
    // The recipient comes from the loaded thread, sending early fails with "recipient required".
    cy.get('input[placeholder="Email addresses separated by comma"]')
      .first()
      .should('not.have.value', '')
    cy.get('.tiptap.ProseMirror').first().click().type(replyBody)
    cy.contains('button', /^Send$/).click() // exact: the split button next to it is "send and set status"

    cy.wait('@sendReply').its('response.statusCode').should('eq', 200)
    cy.get('[data-message-uuid]').contains(replyBody).closest('.bg-private').should('not.exist')
  })

  it('resolves the conversation and it moves out of the open list', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/status`).as('setStatus')
    cy.intercept('GET', '**/conversations/all?*').as('loadList')

    openConversation()
    statusBadge().click()
    cy.contains('[role="menuitem"]', 'Resolved').click()
    cy.wait('@setStatus').its('response.statusCode').should('eq', 200)
    statusBadge().should('contain.text', 'Resolved')

    // Wait for the list to actually load, else "not.exist" passes on an empty list.
    cy.visit('/inboxes/all')
    cy.wait('@loadList')
    cy.contains(contactName).should('not.exist')

    cy.contains('button', /\d+\s*Open/).click()
    cy.contains('[role="menuitem"]', 'Resolved').click()
    cy.wait('@loadList')
    cy.contains(contactName).should('exist')
  })
})
