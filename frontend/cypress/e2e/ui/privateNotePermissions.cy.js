describe('Private note permissions', () => {
  const stamp = Date.now()
  const password = 'StrongPass!123'
  const contactEmail = `private-note.customer.${stamp}@example.com`
  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)
  const commonPermissions = [
    'conversations:read_all',
    'conversations:read',
    'messages:read',
    'view:manage'
  ]
  const combinations = [
    {
      key: 'both',
      label: 'both permissions',
      permissions: ['messages:write', 'messages:write_private'],
      canReply: true,
      canPrivateNote: true
    },
    {
      key: 'public',
      label: 'public-message permission only',
      permissions: ['messages:write'],
      canReply: true,
      canPrivateNote: false
    },
    {
      key: 'private',
      label: 'private-note permission only',
      permissions: ['messages:write_private'],
      canReply: false,
      canPrivateNote: true
    },
    {
      key: 'neither',
      label: 'neither permission',
      permissions: [],
      canReply: false,
      canPrivateNote: false
    }
  ].map((combination) => ({
    ...combination,
    roleName: `Private Note ${combination.key} ${stamp}`,
    email: `private-note.${combination.key}.${stamp}@example.com`
  }))

  let conversationUUID
  let adminNoteUUID
  let inboxID
  const roleIDs = []
  const agentIDs = []

  const loginAsAgent = (combination) => {
    cy.session(
      ['private-note-permissions', combination.email],
      () => {
        cy.visit('/')
        cy.get('#email').clear()
        cy.get('#email').type(combination.email)
        cy.get('#password').clear()
        cy.get('#password').type(password, { log: false })
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

  const sendMessage = (combination, privateNote) =>
    cy.api(
      'POST',
      `/api/v1/conversations/${conversationUUID}/messages`,
      {
        sender_type: 'agent',
        private: privateNote,
        message: `<p>${combination.key}-${privateNote ? 'note' : 'reply'}-${stamp}</p>`,
        attachments: [],
        mentions: [],
        to: privateNote ? [] : [contactEmail],
        cc: [],
        bcc: []
      },
      { failOnStatusCode: false }
    )

  const deleteMessage = (messageUUID) =>
    cy.api('DELETE', `/api/v1/conversations/${conversationUUID}/messages/${messageUUID}`, null, {
      failOnStatusCode: false
    })

  const summarize = () =>
    cy.api(
      'POST',
      '/api/v1/ai/summarize',
      { conversation_uuid: conversationUUID },
      { failOnStatusCode: false }
    )

  const expectPermissionResult = (response, allowed) => {
    expect(response.status).to.eq(allowed ? 200 : 403)
    if (!allowed) expect(response.body.error_type).to.eq('PermissionException')
  }

  const openConversation = () => {
    cy.intercept('GET', '**/messages?page=*').as('loadMessages')
    // Stubbed so the copilot answer renders without a configured AI provider.
    cy.intercept('GET', '**/ai/copilot/messages*', {
      body: { data: [{ role: 'assistant', content: '<p>copilot answer</p>' }] }
    }).as('loadCopilot')
    cy.visit(`/inboxes/all/conversation/${conversationUUID}`)
    cy.wait('@loadMessages')
  }

  before(() => {
    cy.login()

    cy.api('POST', '/api/v1/inboxes', {
      name: `Private Note Permissions ${stamp}`,
      channel: 'email',
      enabled: true,
      from: `Private Note Permissions <private-note+${stamp}@cypress.test>`,
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
      inboxID = body.data.id
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: contactEmail,
        first_name: 'Private',
        last_name: `Note${stamp}`,
        subject: `Private note permissions ${stamp}`,
        content: '<p>Permission test conversation.</p>',
        initiator: 'contact'
      }).then((response) => {
        conversationUUID = response.body.data.uuid
        cy.api('POST', `/api/v1/conversations/${conversationUUID}/messages`, {
          sender_type: 'agent',
          private: true,
          message: `<p>admin-note-${stamp}</p>`,
          attachments: [],
          mentions: [],
          to: [],
          cc: [],
          bcc: []
        }).then(({ body }) => {
          adminNoteUUID = body.data.uuid
        })
      })
    })

    combinations.forEach((combination) => {
      cy.api('POST', '/api/v1/roles', {
        name: combination.roleName,
        description: `Tests ${combination.label}`,
        permissions: [...commonPermissions, ...combination.permissions]
      }).then(({ body }) => {
        roleIDs.push(body.data.id)
      })
      cy.api('POST', '/api/v1/agents', {
        first_name: 'Permission',
        last_name: combination.key,
        email: combination.email,
        roles: [combination.roleName],
        enabled: true,
        send_welcome_email: false
      }).then(({ body }) => {
        agentIDs.push(body.data.id)
        cy.api('PUT', `/api/v1/agents/${body.data.id}`, {
          first_name: 'Permission',
          last_name: combination.key,
          email: combination.email,
          roles: [combination.roleName],
          enabled: true,
          new_password: password
        })
      })
    })
  })

  after(() => {
    cy.login()
    const drop = (path) => cy.api('DELETE', path, null, { failOnStatusCode: false })
    agentIDs.forEach((id) => drop(`/api/v1/agents/${id}`))
    roleIDs.forEach((id) => drop(`/api/v1/roles/${id}`))
    if (inboxID) drop(`/api/v1/inboxes/${inboxID}`)
  })

  combinations.forEach((combination) => {
    it(`enforces and renders ${combination.label}`, () => {
      cy.viewport(1440, 900)
      loginAsAgent(combination)

      sendMessage(combination, false).then((response) => {
        expectPermissionResult(response, combination.canReply)
      })
      sendMessage(combination, true).then((response) => {
        expectPermissionResult(response, combination.canPrivateNote)
        if (combination.canPrivateNote) {
          deleteMessage(response.body.data.uuid).its('status').should('eq', 200)
        } else {
          deleteMessage(adminNoteUUID).then((deleteResponse) => {
            expectPermissionResult(deleteResponse, false)
          })
        }
      })

      // AI is not configured in CI, so an allowed agent fails later than the permission gate.
      summarize().then((response) => {
        if (combination.canPrivateNote) expect(response.status).to.not.eq(403)
        else expectPermissionResult(response, false)
      })

      openConversation()
      cy.contains('button', /^Reply$/).should(combination.canReply ? 'exist' : 'not.exist')
      cy.contains('button', /^Private note$/).should(
        combination.canPrivateNote ? 'exist' : 'not.exist'
      )
      cy.get('.tiptap.ProseMirror').should(
        combination.canReply || combination.canPrivateNote ? 'exist' : 'not.exist'
      )

      if (!combination.canReply && combination.canPrivateNote) {
        cy.contains('button', /^Private note$/).should('have.attr', 'data-state', 'active')
      }

      cy.get('button.w-11.h-11.p-0').click()
      cy.contains('[role="menuitem"]', 'Download transcript').should('exist')
      cy.contains('[role="menuitem"]', 'Summarize with AI').should(
        combination.canPrivateNote ? 'exist' : 'not.exist'
      )

      cy.get('body').type('{esc}')
      cy.contains('button', 'Juno').click()
      cy.wait('@loadCopilot')
      cy.contains('copilot answer')
        .closest('.flex.flex-col.gap-1')
        .find('button')
        .should(
          'have.length',
          1 + Number(combination.canReply) + Number(combination.canPrivateNote)
        )
    })
  })
})
