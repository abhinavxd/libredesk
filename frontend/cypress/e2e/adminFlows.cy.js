// End-to-end journey against a real backend (no API mocking):
// login -> create email inbox -> create team -> create agent ->
// create an outgoing conversation -> reply (sent through MailHog).
//
// The steps run in order and share state, so the conversation reuses the
// inbox created earlier in the same spec.

describe('Admin setup and outgoing conversation', () => {
  const stamp = Date.now()
  const inboxName = `Cypress Inbox ${stamp}`
  const fromAddress = `Cypress Support <support+${stamp}@cypress.test>`
  const teamName = `Cypress Team ${stamp}`
  const agentFirstName = 'Cypress'
  const agentEmail = `cypress.agent.${stamp}@example.com`
  const contactEmail = `cypress.customer.${stamp}@example.com`
  const subject = `Cypress subject ${stamp}`
  const replyBody = `Automated reply from Cypress ${stamp}`

  // SMTP points at the MailHog sink running as a CI service; IMAP is dummy.
  const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
  const smtpPort = Cypress.env('SMTP_PORT') || '1025'
  const mailhogUrl = Cypress.env('MAILHOG_URL')

  let conversationUuid

  beforeEach(() => {
    cy.viewport(1280, 800) // desktop layout so the inbox sidebar (New conversation) is visible
    cy.login()
  })

  it('creates an email inbox that sends through MailHog', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    cy.visit('/admin/inboxes/new')
    cy.contains('Create an email inbox').click()
    cy.contains('Configure IMAP and SMTP manually').click()

    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="from"]').type(fromAddress)

    cy.get('input[name="imap.username"]').type('cypress')
    cy.get('input[name="imap.password"]').type('cypress')

    // smtp.host/port also exist in the hidden OAuth section, so target the visible ones.
    cy.get('input[name="smtp.host"]:visible').clear().type(smtpHost)
    cy.get('input[name="smtp.port"]:visible').clear().type(smtpPort)
    cy.get('input[name="smtp.username"]').type('cypress')
    cy.get('input[name="smtp.password"]').type('cypress')
    cy.selectOption('Login', 'None') // SMTP auth protocol -> None (MailHog needs none)

    cy.contains('button', 'Create').click()
    cy.wait('@createInbox').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/admin/inboxes')
  })

  it('creates a team', () => {
    cy.intercept('POST', '**/api/v1/teams').as('createTeam')

    cy.visit('/admin/teams/teams/new')

    // Emoji is required. Pick it first so the picker overlay is closed (by the
    // next click) before filling the rest of the form.
    cy.get('input[name="emoji"]').click()
    cy.get('.v3-emoji-picker .v3-emojis button').first().click()

    cy.get('input[name="name"]').type(teamName)
    cy.selectOption('Select an assignment type', 'Round robin')
    cy.contains('button[role="combobox"]', 'Select timezone').click()
    cy.get('[role="option"]').first().click()

    cy.contains('button', 'Create').click()
    cy.wait('@createTeam').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/admin/teams/teams')
  })

  it('creates an agent', () => {
    cy.intercept('POST', '**/api/v1/agents').as('createAgent')

    cy.visit('/admin/teams/agents/new')

    cy.get('input[name="first_name"]').type(agentFirstName)
    cy.get('input[name="email"]').type(agentEmail)

    cy.get('input[placeholder="Select roles"]').click()
    cy.get('[role="option"]').contains('Agent').click()

    cy.contains('button', 'Create').click()
    cy.wait('@createAgent').its('response.statusCode').should('eq', 200)
    cy.location('pathname').should('eq', '/admin/teams/agents')
  })

  it('creates an outgoing conversation from the new inbox', () => {
    cy.intercept('POST', '**/api/v1/conversations').as('createConversation')

    cy.visit('/inboxes/assigned')
    cy.contains('New conversation').click()

    cy.get('[role="dialog"]').within(() => {
      cy.get('input[type="email"]').type(contactEmail)
      cy.get('input[name="first_name"]').type('Cypress')
      cy.get('input[name="last_name"]').type('Customer')
      cy.get('input[name="subject"]').type(subject)
      cy.contains('button[role="combobox"]', 'Select inbox').click()
    })
    // Radix Select options render in a portal outside the dialog.
    cy.get('[role="option"]').contains(inboxName).click()

    cy.get('[role="dialog"]')
      .find('.tiptap.ProseMirror')
      .click()
      .type('Hello, this is an automated outgoing conversation.')
    cy.get('[role="dialog"]').contains('button', 'Submit').click()

    cy.wait('@createConversation').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      conversationUuid = response.body.data.uuid
      expect(conversationUuid, 'conversation uuid').to.be.a('string').and.not.be.empty
    })
  })

  it('replies to the conversation and the email lands in MailHog', () => {
    expect(conversationUuid, 'conversation from previous step').to.be.a('string')

    cy.intercept('POST', `**/api/v1/conversations/${conversationUuid}/messages`).as('sendReply')

    cy.visit(`/inboxes/all/conversation/${conversationUuid}`)
    cy.get('.tiptap.ProseMirror').first().click().type(replyBody)
    cy.contains('button', /^Send$/).click() // exact: avoid the adjacent "Send and set status" split button

    cy.wait('@sendReply').its('response.statusCode').should('eq', 200)
    cy.contains(replyBody).should('exist')

    // Outgoing mail is dispatched asynchronously; poll MailHog until it shows up.
    // Only runs in CI where MAILHOG_URL is set.
    if (mailhogUrl) {
      const findInMailHog = (attempt = 0) => {
        cy.request(`${mailhogUrl}/api/v2/messages`).then((res) => {
          const delivered = (res.body.items || []).some((m) =>
            JSON.stringify(m).includes(contactEmail)
          )
          if (delivered) return
          if (attempt >= 15) {
            throw new Error(`Email to ${contactEmail} not found in MailHog`)
          }
          cy.wait(1000)
          findInMailHog(attempt + 1)
        })
      }
      findInMailHog()
    }
  })
})
