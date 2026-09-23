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
  const ccContactLastName = `CcCustomer${stamp}`
  const ccContactName = `Lifecycle ${ccContactLastName}`
  const ccAddress = `lifecycle.cc.${stamp}@example.com`
  const customerEmail = `lifecycle.customer.${stamp}@example.com`
  const ccCustomerEmail = `lifecycle.cc-customer.${stamp}@example.com`
  const bccAddress = `lifecycle.bcc.${stamp}@example.com`

  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)

  let conversationUuid
  let ccConversationUuid
  let inboxId
  let teamId
  let agentId
  let tagId

  const openConversation = (uuid = conversationUuid) => {
    cy.intercept('GET', '**/messages?page=*').as('loadMessages')
    cy.visit(`/inboxes/all/conversation/${uuid}`)
    cy.wait('@loadMessages')
  }

  // A stubbed reply keeps the thread, and the recipients it prefills, the same for later tests.
  const captureSend = () =>
    cy
      .intercept('POST', '**/conversations/*/messages', { statusCode: 200, body: { data: {} } })
      .as('captureSend')

  const recipientRow = (label) => cy.contains('label', label).parent()

  const sendReply = (text) => {
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(text)
    cy.contains('button', /^Send$/).click()
  }

  const expectSentRecipients = (uuid, { cc, bcc }) =>
    cy.wait('@captureSend').then(({ request }) => {
      expect(request.url, 'conversation').to.include(uuid)
      expect(request.body.cc, 'cc').to.deep.eq(cc)
      expect(request.body.bcc, 'bcc').to.deep.eq(bcc)
    })

  // The open set of sidebar sections is remembered per browser, so blind toggling closes it later.
  const openActionsSection = () => {
    cy.contains('button', 'Actions').then(($trigger) => {
      if ($trigger.attr('aria-expanded') !== 'true') cy.wrap($trigger).click()
    })
  }

  // Pickers list one server page, so a fresh row is only reachable through the search input.
  const searchInPicker = (text) =>
    cy
      .get('[data-radix-popper-content-wrapper]:visible input')
      .last()
      .invoke('val', text)
      .trigger('input')

  // The list column also renders a status dropdown; this one is the header badge.
  const statusBadge = () => cy.get('span.bg-primary.rounded-md')

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
        contact_email: customerEmail,
        first_name: 'Lifecycle',
        last_name: contactLastName,
        subject,
        content: '<p>Customer opened this conversation.</p>',
        initiator: 'contact'
      }).then((res) => {
        conversationUuid = res.body.data.uuid
        expect(conversationUuid, 'conversation uuid').to.be.a('string').and.not.be.empty
      })
      // The reply box prefills CC from the last email, so this thread ends on an agent reply that CC'd someone.
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: ccCustomerEmail,
        first_name: 'Lifecycle',
        last_name: ccContactLastName,
        subject: `${subject} with CC`,
        content: '<p>Customer opened the CC conversation.</p>',
        initiator: 'contact'
      }).then((res) => {
        ccConversationUuid = res.body.data.uuid
        cy.api('POST', `/api/v1/conversations/${ccConversationUuid}/messages`, {
          sender_type: 'agent',
          private: false,
          message: '<p>Looping in a colleague.</p>',
          to: [ccCustomerEmail],
          cc: [ccAddress],
          bcc: []
        })
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

  it('focuses the date button when the custom snooze picker opens', () => {
    openConversation()
    cy.get('body').trigger('keydown', { key: 'z', code: 'KeyZ', altKey: true })
    cy.get('[cmdk-input-wrapper] input').type('pick{enter}')
    cy.contains('button', 'Pick a date').should('be.focused')
  })

  it('suggests tags from the command palette for an open conversation', () => {
    cy.intercept('POST', '**/api/v1/ai/suggest-tags', { body: { data: [] } }).as('suggestTags')

    openConversation()
    cy.get('body').trigger('keydown', { key: 'k', code: 'KeyK', ctrlKey: true })
    cy.get('[cmdk-input-wrapper] input').type('suggest tags')
    cy.contains('[role="option"]', 'Suggest tags').click()

    cy.wait('@suggestTags').its('request.body.conversation_uuid').should('eq', conversationUuid)
  })

  it('assigns the conversation to an agent', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignee/user`).as('assignAgent')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select agent').click()
    searchInPicker(agentLastName)
    cy.contains('[role="option"]', agentName).click()

    cy.wait('@assignAgent').its('response.statusCode').should('eq', 200)
    cy.contains('button[role="combobox"]', agentName).should('exist')
  })

  it('assigns the conversation to a team', () => {
    cy.intercept('PUT', `**/conversations/${conversationUuid}/assignee/team`).as('assignTeam')

    openConversation()
    openActionsSection()
    cy.contains('button[role="combobox"]', 'Select team').click()
    searchInPicker(teamName)
    cy.contains('[role="option"]', teamName).click()

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
    cy.get('input[placeholder="Select tags"]').click().type(tagName)
    cy.contains('[role="option"]', tagName).click()

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
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(noteBody)
    cy.contains('button', /^Send$/).click()

    cy.wait('@sendNote').then(({ request, response }) => {
      expect(request.body.private).to.eq(true)
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.private).to.eq(true)
      cy.get(`[data-message-uuid="${response.body.data.uuid}"]`)
        .find('.message-bubble')
        .should('have.class', 'bg-private')
    })
  })

  it('sends the CC prefilled from the last email', () => {
    captureSend()
    openConversation(ccConversationUuid)
    recipientRow('CC:').find('input').should('have.value', ccAddress)
    sendReply('Reply with the prefilled CC')
    expectSentRecipients(ccConversationUuid, { cc: [ccAddress], bcc: [] })
  })

  it('drops the CC once its row is closed', () => {
    captureSend()
    openConversation(ccConversationUuid)
    recipientRow('CC:').find('button[aria-label="Remove CC"]').click()
    cy.contains('label', 'CC:').should('not.exist')
    sendReply('Reply after closing CC')
    expectSentRecipients(ccConversationUuid, { cc: [], bcc: [] })
  })

  it('sends only what each conversation shows after switching between them', () => {
    captureSend()
    openConversation(ccConversationUuid)
    recipientRow('CC:').find('button[aria-label="Remove CC"]').click()
    cy.contains('button', /^BCC$/).click()
    recipientRow('BCC:').find('input').type(bccAddress)

    cy.contains(contactName).click()
    cy.location('pathname').should('include', conversationUuid)
    // A filled TO box means the new thread has loaded.
    recipientRow('TO:').find('input').should('have.value', customerEmail)
    cy.contains('label', 'CC:').should('not.exist')
    cy.contains('label', 'BCC:').should('not.exist')
    sendReply('Reply after switching away')
    expectSentRecipients(conversationUuid, { cc: [], bcc: [] })

    cy.contains(ccContactName).click()
    cy.location('pathname').should('include', ccConversationUuid)
    recipientRow('TO:').find('input').should('have.value', ccCustomerEmail)
    recipientRow('CC:').find('input').should('have.value', ccAddress)
    cy.contains('label', 'BCC:').should('not.exist')
    sendReply('Reply after switching back')
    expectSentRecipients(ccConversationUuid, { cc: [ccAddress], bcc: [] })
  })

  it('sends a reply that appears in the thread', () => {
    cy.intercept('POST', `**/conversations/${conversationUuid}/messages`).as('sendReply')

    openConversation()
    // The recipient comes from the loaded thread, sending early fails with "recipient required".
    cy.get('input[placeholder="Email addresses separated by comma"]')
      .first()
      .should('not.have.value', '')
    cy.get('.tiptap.ProseMirror').first().click()
    cy.get('.tiptap.ProseMirror').first().type(replyBody)
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
