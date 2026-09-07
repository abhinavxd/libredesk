describe('Create conversation dialog', () => {
  const stamp = Date.now()
  const inboxName = `Create Conv Inbox ${stamp}`
  const teamName = `Create Conv Team ${stamp}`
  const agentFirstName = 'CreateConv'
  const agentLastName = `Agent${stamp}`
  const agentName = `${agentFirstName} ${agentLastName}`

  const existingEmail = `existing.contact.${stamp}@example.com`
  const existingFirstName = 'Existing'
  const existingLastName = `Contact${stamp}`

  const newEmail = `brand.new.${stamp}@example.com`
  const subject = `Create conv subject ${stamp}`
  const body = `Create conv body ${stamp}`

  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Number(Cypress.env('SMTP_PORT') || 1025)

  let inboxID

  const openDialog = () => {
    cy.visit('/inboxes/assigned')
    cy.contains('New conversation').click()
    cy.get('[role="dialog"]').should('be.visible')
  }

  // Radix Select and the combobox render options in a portal outside the dialog.
  const pickFromPortal = (triggerLabel, optionText) => {
    cy.get('[role="dialog"]').contains('button[role="combobox"]', triggerLabel).click()
    cy.get('[role="option"]').contains(optionText).click()
  }

  const typeBody = (text) =>
    cy.get('[role="dialog"]').find('.tiptap.ProseMirror').click().type(text)

  const submit = () => cy.get('[role="dialog"]').find('button[type="submit"]').click()

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: inboxName,
      channel: 'email',
      enabled: true,
      from: `Create Conv <createconv+${stamp}@cypress.test>`,
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
    }).then(({ body: inboxBody }) => {
      inboxID = inboxBody.data.id
      cy.api('POST', '/api/v1/teams', {
        name: teamName,
        emoji: '🐞',
        conversation_assignment_type: 'Round robin',
        timezone: 'Asia/Kolkata'
      })
      cy.api('POST', '/api/v1/agents', {
        first_name: agentFirstName,
        last_name: agentLastName,
        email: `createconv.agent.${stamp}@example.com`,
        roles: ['Agent'],
        enabled: true,
        send_welcome_email: false
      })
      // The only way to seed a contact: there is no POST /contacts endpoint.
      cy.api('POST', '/api/v1/conversations', {
        inbox_id: inboxID,
        contact_email: existingEmail,
        first_name: existingFirstName,
        last_name: existingLastName,
        subject: `Seed conversation ${stamp}`,
        content: '<p>Seeded so the contact exists for the search box.</p>',
        initiator: 'contact'
      })
    })
  })

  beforeEach(() => {
    cy.viewport(1440, 900) // desktop layout: the sidebar "New conversation" button is visible
    cy.login()
    cy.intercept('POST', '**/api/v1/conversations').as('createConversation')
  })

  it('shows every field of the dialog', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.contains('New conversation').should('be.visible')
      cy.get('input[type="email"]').should('have.value', '')
      cy.get('input[name="first_name"]').should('have.value', '')
      cy.get('input[name="last_name"]').should('have.value', '')
      cy.get('input[name="subject"]').should('have.value', '')
      cy.contains('button[role="combobox"]', 'Select inbox').should('exist')
      cy.contains('button[role="combobox"]', 'Select team').should('exist')
      cy.contains('button[role="combobox"]', 'Select agent').should('exist')
      cy.get('.tiptap.ProseMirror').should('exist')
      cy.get('input[type="file"]').should('exist')
      cy.get('button[type="submit"]').should('be.enabled')
    })
  })

  it('rejects a submit with every field empty', () => {
    openDialog()
    submit()

    cy.get('[role="dialog"]').within(() => {
      cy.contains('Invalid email address').should('exist')
      cy.contains('Subject cannot be empty').should('exist')
      cy.contains('Message cannot be empty').should('exist')
      cy.contains(/required/i).should('exist')
    })
    cy.get('@createConversation.all').should('have.length', 0)
    cy.get('[role="dialog"]').should('be.visible')
  })

  it('rejects a malformed contact email', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type('not-an-email')
      cy.get('input[name="first_name"]').type('New')
      cy.get('input[name="subject"]').type(subject)
    })
    pickFromPortal('Select inbox', inboxName)
    typeBody(body)
    submit()

    cy.get('[role="dialog"]').contains('Invalid email address').should('exist')
    cy.get('@createConversation.all').should('have.length', 0)
  })

  it('rejects a submit with no first name', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="subject"]').type(subject)
    })
    pickFromPortal('Select inbox', inboxName)
    typeBody(body)
    submit()

    cy.get('[role="dialog"]').contains(/required/i).should('exist')
    cy.get('@createConversation.all').should('have.length', 0)
  })

  it('rejects a submit with no subject', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="first_name"]').type('New')
    })
    pickFromPortal('Select inbox', inboxName)
    typeBody(body)
    submit()

    cy.get('[role="dialog"]').contains('Subject cannot be empty').should('exist')
    cy.get('@createConversation.all').should('have.length', 0)
  })

  it('rejects a submit with no inbox', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="first_name"]').type('New')
      cy.get('input[name="subject"]').type(subject)
    })
    typeBody(body)
    submit()

    cy.get('[role="dialog"]').contains(/required/i).should('exist')
    cy.get('@createConversation.all').should('have.length', 0)
  })

  it('rejects a submit with an empty message body', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="first_name"]').type('New')
      cy.get('input[name="subject"]').type(subject)
    })
    pickFromPortal('Select inbox', inboxName)
    submit()

    cy.get('[role="dialog"]').contains('Message cannot be empty').should('exist')
    cy.get('@createConversation.all').should('have.length', 0)
  })

  it('reopens clean after a failed submit', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="subject"]').type(subject)
      cy.get('button[type="submit"]').click()
      cy.contains(/required/i).should('exist')
    })
    cy.get('@createConversation.all').should('have.length', 0)

    // The dialog is v-if'd, so closing it destroys the form and its state.
    cy.get('[role="dialog"] > button.absolute.right-4.top-4').click()
    cy.get('[role="dialog"]').should('not.exist')

    cy.contains('New conversation').click()
    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').should('have.value', '')
      cy.get('input[name="subject"]').should('have.value', '')
      cy.contains('button[role="combobox"]', 'Select inbox').should('exist')
      cy.contains('Invalid email address').should('not.exist')
    })
  })

  it('fills the name fields from an existing contact and locks them', () => {
    openDialog()

    cy.get('[role="dialog"]').find('input[type="email"]').type(existingEmail)
    cy.get('[role="dialog"]').find('[role="option"]').contains(existingEmail).click()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[name="first_name"]').should('have.value', existingFirstName).and('be.disabled')
      cy.get('input[name="last_name"]').should('have.value', existingLastName).and('be.disabled')
    })
  })

  it('unlocks the name fields again when the email is edited away', () => {
    openDialog()

    cy.get('[role="dialog"]').find('input[type="email"]').type(existingEmail)
    cy.get('[role="dialog"]').find('[role="option"]').contains(existingEmail).click()
    cy.get('[role="dialog"]').find('input[name="first_name"]').should('be.disabled')

    cy.get('[role="dialog"]').find('input[type="email"]').type('x')
    cy.get('[role="dialog"]').within(() => {
      cy.get('input[name="first_name"]').should('have.value', '').and('be.enabled')
      cy.get('input[name="last_name"]').should('have.value', '').and('be.enabled')
    })
  })

  it('creates a conversation for a new contact and opens it', () => {
    openDialog()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="first_name"]').type('Brand')
      cy.get('input[name="last_name"]').type(`New${stamp}`)
      cy.get('input[name="subject"]').type(subject)
    })
    pickFromPortal('Select inbox', inboxName)
    pickFromPortal('Select team', teamName)
    pickFromPortal('Select agent', agentName)
    typeBody(body)
    submit()

    cy.wait('@createConversation').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.contact_email).to.eq(newEmail)
      expect(request.body.subject).to.eq(subject)
      expect(request.body.inbox_id).to.eq(inboxID)
      expect(request.body.initiator).to.eq('agent')
      expect(request.body.content).to.include(body)

      const uuid = response.body.data.uuid
      cy.get('[role="dialog"]').should('not.exist')
      cy.visit(`/inboxes/all/conversation/${uuid}`)
      cy.contains(subject).should('exist')
      cy.contains(body).should('exist')
    })
  })

  it('creates a conversation reusing the selected existing contact', () => {
    const reuseSubject = `Reuse subject ${stamp}`

    openDialog()
    cy.get('[role="dialog"]').find('input[type="email"]').type(existingEmail)
    cy.get('[role="dialog"]').find('[role="option"]').contains(existingEmail).click()
    cy.get('[role="dialog"]').find('input[name="subject"]').type(reuseSubject)
    pickFromPortal('Select inbox', inboxName)
    typeBody(body)
    submit()

    cy.wait('@createConversation').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.contact_email).to.eq(existingEmail)
      expect(request.body.reuse_contact).to.eq(true)
      cy.api('GET', `/api/v1/conversations/${response.body.data.uuid}`)
        .its('body.data.contact.email')
        .should('eq', existingEmail)
    })
  })

  it('attaches a file and sends its id with the conversation', () => {
    const attachSubject = `Attachment subject ${stamp}`
    cy.intercept('POST', '**/api/v1/media').as('uploadMedia')

    openDialog()
    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(newEmail)
      cy.get('input[name="first_name"]').type('Brand')
      cy.get('input[name="subject"]').type(attachSubject)
      // The file input is visually hidden and driven by the paperclip button.
      cy.get('input[type="file"]').selectFile(
        {
          contents: Cypress.Buffer.from('cypress attachment'),
          fileName: 'note.txt',
          mimeType: 'text/plain'
        },
        { force: true }
      )
    })
    cy.wait('@uploadMedia').its('response.statusCode').should('eq', 200)
    cy.get('[role="dialog"]').contains('note.txt').should('exist')

    pickFromPortal('Select inbox', inboxName)
    typeBody(body)
    submit()

    cy.wait('@createConversation').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.attachments).to.have.length(1)
    })
  })
})
