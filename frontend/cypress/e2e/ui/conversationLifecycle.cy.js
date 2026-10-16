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
    cy.get('input[aria-label="Select tags"]').click()
    cy.get('input[aria-label="Select tags"]').type(tagName)
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
    recipientRow(/^CC$/).find('input').should('have.value', ccAddress)
    sendReply('Reply with the prefilled CC')
    expectSentRecipients(ccConversationUuid, { cc: [ccAddress], bcc: [] })
  })

  it('preserves sender and recipient edits across window and divider resizing', () => {
    const sender = 'billing@example.com'
    const to = 'edited@example.com'
    cy.intercept('GET', `**/api/v1/conversations/${ccConversationUuid}`, (req) => {
      req.continue((res) => {
        res.body.data.inbox_aliases = [{ email: sender, verification_status: 'verified' }]
      })
    })
    openConversation(ccConversationUuid)
    recipientRow(/^CC$/).find('input').should('have.value', ccAddress)
    recipientRow(/^From$/).find('[role="combobox"]').click()
    cy.contains('[role="option"]', sender).trigger('keydown', { key: 'Enter' })
    recipientRow(/^To$/).find('input').clear()
    recipientRow(/^To$/).find('input').type(to)
    recipientRow(/^CC$/).find('input').clear()
    recipientRow(/^CC$/).find('input').type('edited.cc@example.com')
    cy.contains('button', /^BCC$/).click()
    recipientRow(/^BCC$/).find('input').type(bccAddress)
    cy.get('.tiptap.ProseMirror').first().then(($editor) => {
      const originalEditor = $editor[0]
      const checkEdits = () => {
        for (const name of ['From', 'To', 'CC', 'BCC']) {
          cy.contains('label', new RegExp(`^${name}$`)).should(($label) => {
            expect($label[0].control, `${name} label control`).not.to.equal(null)
          })
        }
        recipientRow(/^From$/).find('[role="combobox"]').should('contain', sender)
        recipientRow(/^To$/).find('input').should('have.value', to)
        recipientRow(/^CC$/).find('input').should('have.value', 'edited.cc@example.com')
        recipientRow(/^BCC$/).find('input').should('have.value', bccAddress)
        cy.get('.tiptap.ProseMirror').first().should(($current) => {
          expect($current[0]).to.equal(originalEditor)
        })
      }
      for (const width of [1200, 900, 390, 1440, 900, 1440]) {
        cy.viewport(width, 900)
        checkEdits()
      }
      cy.get('[role="separator"][aria-label="Conversations"]').focus()
      cy.get('[role="separator"][aria-label="Conversations"]').trigger('keydown', { key: 'End' })
      cy.get('button[aria-label="Details"]').should('have.attr', 'aria-expanded', 'false')
      cy.get('[role="separator"][aria-label="Details"]').should('not.exist')
      checkEdits()
      cy.get('[role="separator"][aria-label="Conversations"]').trigger('keydown', { key: 'Home' })
      cy.get('[role="separator"][aria-label="Details"]').should('exist')
      checkEdits()
    })
  })

  it('drops the CC once its row is closed', () => {
    captureSend()
    openConversation(ccConversationUuid)
    recipientRow(/^CC$/).find('button[aria-label="Remove CC"]').click()
    cy.contains('label', /^CC$/).should('not.exist')
    sendReply('Reply after closing CC')
    expectSentRecipients(ccConversationUuid, { cc: [], bcc: [] })
  })

  it('sends only what each conversation shows after switching between them', () => {
    captureSend()
    openConversation(ccConversationUuid)
    recipientRow(/^CC$/).find('button[aria-label="Remove CC"]').click()
    cy.contains('button', /^BCC$/).click()
    recipientRow(/^BCC$/).find('input').type(bccAddress)

    cy.contains(contactName).click()
    cy.location('pathname').should('include', conversationUuid)
    // A filled TO box means the new thread has loaded.
    recipientRow(/^To$/).find('input').should('have.value', customerEmail)
    cy.contains('label', /^CC$/).should('not.exist')
    cy.contains('label', /^BCC$/).should('not.exist')
    sendReply('Reply after switching away')
    expectSentRecipients(conversationUuid, { cc: [], bcc: [] })

    cy.contains(ccContactName).click()
    cy.location('pathname').should('include', ccConversationUuid)
    recipientRow(/^To$/).find('input').should('have.value', ccCustomerEmail)
    recipientRow(/^CC$/).find('input').should('have.value', ccAddress)
    cy.contains('label', /^BCC$/).should('not.exist')
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

  it('keeps the unread badge hidden while sending private notes', () => {
    let authorID
    const events = []
    const unreadCounts = []
    const badgeSelector = 'div.bg-primary.rounded-full'
    cy.api('GET', '/api/v1/agents/me').then(({ body }) => { authorID = body.data.id })
    cy.api('GET', `/api/v1/conversations/${ccConversationUuid}/messages?page=1`).then(({ body }) => {
      cy.api('PUT', `/api/v1/conversations/${ccConversationUuid}/last-seen`, {
        message_uuid: body.data.results[0].uuid
      })
    })
    cy.visit(`/inboxes/all/conversation/${ccConversationUuid}`, {
      onBeforeLoad(win) {
        const OriginalWebSocket = win.WebSocket
        win.WebSocket = class extends OriginalWebSocket {
          constructor(...args) {
            super(...args)
            this.addEventListener('message', ({ data }) => {
              if (data === 'pong') return
              const event = JSON.parse(data)
              if (event.type === 'new_message' && event.data.conversation_uuid === ccConversationUuid) {
                events.push(event.data)
              }
            })
          }
        }
      }
    })
    cy.get(`a[href="/inboxes/all/conversation/${ccConversationUuid}"]`).as('noteRow')
    cy.get('@noteRow').find(badgeSelector).should('not.exist')
    cy.get('@noteRow').then(($row) => {
      const win = $row[0].ownerDocument.defaultView
      const observer = new win.MutationObserver((records) => {
        for (const record of records) {
          for (const node of record.addedNodes) {
            if (node.nodeType !== 1) continue
            if (node.matches(badgeSelector)) unreadCounts.push(node.textContent)
            for (const badge of node.querySelectorAll(badgeSelector)) unreadCounts.push(badge.textContent)
          }
        }
      })
      observer.observe($row[0], { childList: true, subtree: true })
      cy.wrap(observer, { log: false }).as('unreadObserver')
    })
    cy.contains('button', 'Private note').click()
    for (let index = 0; index < 3; index++) {
      const content = `Unread regression ${stamp} ${index}`
      sendReply(content)
      cy.get('[data-message-uuid]').contains(content).should('be.visible')
      cy.wrap(events).should('have.length', index + 1)
    }
    cy.then(() => {
      expect(events.every((event) => Number(event.sender_id) === Number(authorID))).to.eq(true)
      expect(unreadCounts).to.deep.eq([])
    })
    cy.get('@noteRow').find(badgeSelector).should('not.exist')
    cy.get('@unreadObserver').invoke('disconnect')
  })

  it('loads older messages on scroll without moving the visible message or repeating pages', () => {
    const requestedPages = []
    const settleScroll = () => cy.window().then((win) =>
      new Cypress.Promise((resolve) => win.requestAnimationFrame(() => win.requestAnimationFrame(resolve)))
    )
    let anchor
    let anchorTop
    cy.api('GET', `/api/v1/conversations/${conversationUuid}/messages?page=1`).then(({ body }) => {
      const original = body.data.results.find((message) => message.type === 'incoming')
      const messages = Array.from({ length: 90 }, (_, index) => ({
        ...original,
        id: original.id + index,
        uuid: `${conversationUuid.slice(0, -12)}${String(index + 1).padStart(12, '0')}`,
        created_at: new Date(Date.now() - (90 - index) * 60000).toISOString(),
        content_type: 'text',
        content: `History message ${index + 1}`,
        attachments: []
      }))
      cy.intercept('GET', `**/conversations/${conversationUuid}/messages?page=*`, (request) => {
        const page = Number(request.query.page)
        requestedPages.push(page)
        request.reply({
          delay: page > 1 ? 250 : 0,
          body: {
            data: {
              results: messages.slice(90 - page * 30, 120 - page * 30).reverse(),
              page,
              per_page: 30,
              total: 90,
              total_pages: 3
            }
          }
        })
      }).as('historyPage')
      cy.intercept('PUT', `**/conversations/${conversationUuid}/last-seen`, {
        body: { data: { conversation_uuid: conversationUuid, read_version: 1, is_unread: false } }
      })
    })
    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.wait('@historyPage')
    cy.get('[data-message-uuid]').should('have.length', 30)
    cy.get('[class*="overflow-anchor"]').should(($thread) => {
      const thread = $thread[0]
      expect(thread.scrollHeight - thread.clientHeight - thread.scrollTop).to.be.lessThan(2)
      expect(requestedPages).to.deep.eq([1])
    })
    settleScroll()
    cy.get('[class*="overflow-anchor"]').then(($thread) => {
      $thread[0].scrollTop = 0
      anchor = $thread[0].querySelector('[data-message-uuid]')
      anchorTop = anchor.getBoundingClientRect().top
      $thread[0].dispatchEvent(new Event('scroll'))
      $thread[0].dispatchEvent(new Event('scroll'))
    })
    cy.wait('@historyPage').its('request.query.page').should('eq', '2')
    cy.get('[data-message-uuid]').should('have.length', 60)
    cy.get('[class*="overflow-anchor"]').find('[role="status"]').should('not.exist')
    cy.get('[class*="overflow-anchor"]').should(() => {
      expect(Math.abs(anchor.getBoundingClientRect().top - anchorTop)).to.be.lessThan(2)
      expect(requestedPages).to.deep.eq([1, 2])
    })
    settleScroll()
    cy.get('[class*="overflow-anchor"]').scrollTo('top')
    cy.wait('@historyPage').its('request.query.page').should('eq', '3')
    cy.get('[data-message-uuid]').should('have.length', 90)
    cy.get('[class*="overflow-anchor"]').find('[role="status"]').should('not.exist')
    settleScroll()
    cy.get('[class*="overflow-anchor"]').scrollTo('top')
    cy.get('[class*="overflow-anchor"]').then(($thread) => {
      $thread[0].dispatchEvent(new Event('scroll'))
      expect(requestedPages).to.deep.eq([1, 2, 3])
    })
  })

  for (const scenario of [
    { name: 'loads until the thread can scroll', height: 2400, total: 90 },
    { name: 'stops at the final page while the thread still fits', height: 4000, total: 60 },
    { name: 'stops after an older page fails', height: 4000, total: 90, fail: true }
  ]) {
    it(`fills a short first page and ${scenario.name}`, () => {
      const requestedPages = []
      cy.viewport(1440, scenario.height)
      cy.api('GET', `/api/v1/conversations/${conversationUuid}/messages?page=1`).then(({ body }) => {
        const original = body.data.results.find((message) => message.type === 'incoming')
        const messages = Array.from({ length: scenario.total }, (_, index) => ({
          ...original,
          id: original.id + index,
          uuid: `${conversationUuid.slice(0, -12)}${String(index + 1).padStart(12, '0')}`,
          created_at: new Date(Date.now() - (scenario.total - index) * 1000).toISOString(),
          content_type: 'text',
          content: `Short history ${index + 1}`,
          attachments: []
        }))
        cy.intercept('GET', `**/conversations/${conversationUuid}/messages?page=*`, (request) => {
          const page = Number(request.query.page)
          requestedPages.push(page)
          if (scenario.fail && page > 1) {
            request.reply({ statusCode: 503, body: { error: { type: 'GeneralException', message: 'History request failed' } } })
            return
          }
          request.reply({ body: { data: {
            results: messages.slice(Math.max(0, scenario.total - page * 30), scenario.total - (page - 1) * 30).reverse(),
            page, per_page: 30, total: scenario.total, total_pages: Math.ceil(scenario.total / 30)
          } } })
        }).as('shortHistoryPage')
        cy.intercept('PUT', `**/conversations/${conversationUuid}/last-seen`, {
          body: { data: { conversation_uuid: conversationUuid, read_version: 1, is_unread: false } }
        })
      })
      cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
      cy.wait('@shortHistoryPage').its('request.query.page').should('eq', '1')
      cy.wait('@shortHistoryPage').then(({ request, response }) => {
        expect(request.query.page).to.eq('2')
        expect(response.statusCode).to.eq(scenario.fail ? 503 : 200)
      })
      if (scenario.fail) {
        cy.get('[data-message-uuid]').should('have.length', 30)
      } else {
        cy.get('[data-message-uuid]').should('have.length', 60)
      }
      cy.get('[class*="overflow-anchor"]').find('[role="status"]').should('not.exist')
      cy.get('[class*="overflow-anchor"]').should(($thread) => {
        const thread = $thread[0]
        if (scenario.height === 2400) expect(thread.scrollHeight).to.be.greaterThan(thread.clientHeight)
        else expect(thread.scrollHeight).to.eq(thread.clientHeight)
        expect(requestedPages).to.deep.eq([1, 2])
      })
      if (scenario.height === 4000) cy.get('.absolute.bottom-5.right-6').should('not.be.visible')
    })
  }

  it('keeps file and image tiles equal and reveals controls only for the active attachment', () => {
    const finishTransitions = () => cy.get('.group\\/attachment').then(($items) => {
      for (const item of $items) {
        for (const animation of item.getAnimations({ subtree: true })) animation.finish()
      }
    })
    const attachments = [
      { uuid: 'receipt', name: 'receipt.pdf', content_type: 'application/pdf' },
      { uuid: 'export', name: 'export.csv', content_type: 'text/csv' },
      {
        uuid: 'delivery', name: 'delivery.png', content_type: 'image/png',
        thumbnail_url: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jBz0AAAAASUVORK5CYII='
      }
    ].map((attachment) => ({ ...attachment, size: 1024, url: `/uploads/${attachment.name}` }))
    cy.api('GET', `/api/v1/conversations/${conversationUuid}/messages?page=1`).then(({ body }) => {
      const original = body.data.results.find((message) => message.type === 'incoming')
      cy.intercept('GET', `**/conversations/${conversationUuid}/messages?page=*`, {
        body: { data: { results: [{ ...original, attachments }], page: 1, total_pages: 1 } }
      }).as('attachmentMessage')
    })
    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.wait('@attachmentMessage')
    cy.get('.group\\/attachment').should('have.length', 3).each(($attachment) => {
      const tile = $attachment.find('.h-28.w-28')[0].getBoundingClientRect()
      expect(tile.width).to.eq(112)
      expect(tile.height).to.eq(112)
    })
    cy.get('.group\\/attachment').first().find('a[aria-label="Download"]').focus()
    finishTransitions()
    cy.get('.group\\/attachment').first().find('a[aria-label="Download"]').should('have.css', 'opacity', '1')
    cy.get('.group\\/attachment').eq(1).find('a[aria-label="Download"]').should('have.css', 'opacity', '0')
    cy.get('.group\\/attachment').eq(2).find('a[aria-label="Download"]').parent().should('have.css', 'opacity', '0')
    cy.get('.group\\/attachment').first().find('a[aria-label="Download"]').blur()
    cy.get('.group\\/attachment').eq(1).then(($attachment) => {
      const win = $attachment[0].ownerDocument.defaultView
      const tile = $attachment[0].getBoundingClientRect()
      const frame = Cypress.$('iframe.aut-iframe', window.top.document)[0].getBoundingClientRect()
      const scale = frame.width / win.innerWidth
      return Cypress.automation('remote:debugger:protocol', {
        command: 'Input.dispatchMouseEvent',
        params: { type: 'mouseMoved', x: frame.left + (tile.left + 30) * scale, y: frame.top + (tile.top + 30) * scale }
      })
    })
    finishTransitions()
    cy.get('.group\\/attachment').eq(1).find('a[aria-label="Download"]').should('have.css', 'opacity', '1')
    cy.get('.group\\/attachment').first().find('a[aria-label="Download"]').should('have.css', 'opacity', '0')
    cy.get('.group\\/attachment').eq(2).find('a[aria-label="Download"]').parent().should('have.css', 'opacity', '0')
    for (const width of [1024, 390]) {
      cy.viewport(width, 900)
      cy.get('.group\\/attachment').each(($attachment) => {
        const tile = $attachment.find('.h-28.w-28')[0].getBoundingClientRect()
        expect(tile.width).to.eq(112)
        expect(tile.height).to.eq(112)
      })
    }
  })

  it('delays the private note spinner and cancels it when the note finishes', () => {
    let author
    cy.api('GET', '/api/v1/agents/me').then(({ body }) => { author = body.data })
    openConversation()
    cy.clock(Date.now(), ['setTimeout', 'clearTimeout'])
    cy.contains('button', 'Private note').click()
    for (const duration of [20, 100]) {
      let pendingRequest
      let finishRequest
      const content = `Spinner timing ${duration}ms`
      cy.intercept('POST', `**/conversations/${conversationUuid}/messages`, (request) => {
        pendingRequest = request
        return new Cypress.Promise((resolve) => { finishRequest = resolve })
      }).as('timedNote')
      cy.get('.tiptap.ProseMirror').first().type(content)
      cy.contains('button', /^Send$/).click()
      cy.wrap(null).should(() => { expect(pendingRequest).to.exist })
      cy.get('[data-message-uuid]').contains(content).closest('[data-message-uuid]').as('pendingNote')
      cy.tick(duration === 100 ? 99 : duration)
      cy.get('@pendingNote').find('[role="status"]').should('not.exist')
      if (duration === 100) {
        cy.tick(1)
        cy.get('@pendingNote').find('[role="status"]').should('exist')
      }
      cy.then(() => {
        pendingRequest.reply({ body: { data: {
        id: 100000 + duration,
        uuid: `${conversationUuid.slice(0, -12)}${String(duration).padStart(12, '0')}`,
        conversation_uuid: conversationUuid,
        type: 'outgoing', sender_type: 'agent', sender_id: author.id, author,
        private: true, status: 'sent', content_type: 'html',
        content: pendingRequest.body.message, created_at: new Date().toISOString(),
        attachments: [], meta: {}
        } } })
        finishRequest()
      })
      cy.wait('@timedNote')
      cy.get('[data-message-uuid]').contains(content).closest('[data-message-uuid]').find('[role="status"]').should('not.exist')
      cy.tick(100)
      cy.get('[data-message-uuid]').contains(content).closest('[data-message-uuid]').find('[role="status"]').should('not.exist')
    }
  })
})
