// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Email Inbox ${stamp}`
const renamedInbox = `Cypress Email Inbox ${stamp} edited`
const fromAddress = `Cypress Support <support+${stamp}@cypress.test>`
const smtpHost = Cypress.env('SMTP_HOST') || '127.0.0.1'
const smtpPort = Cypress.env('SMTP_PORT') || '1025'
const maskedSecret = '••••••••••'
const newPath = '/admin/inboxes/new'
const listPath = '/admin/inboxes'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// The hidden OAuth section repeats imap/smtp field names first, so the manual copy is last.
const field = (name) => cy.get(`input[name="${name}"]`).last()

const openManualForm = () => {
  cy.visit(newPath)
  cy.contains('Create an email inbox').click()
  cy.contains('Configure IMAP and SMTP manually').click()
}

describe('Email inbox form', () => {
  let inboxId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates an email inbox', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openManualForm()

    field('name').type(inboxName)
    field('from').type(fromAddress)

    field('imap.username').type('cypress')
    field('imap.password').type('cypress')

    field('smtp.host').clear().type(smtpHost)
    field('smtp.port').clear().type(smtpPort)
    field('smtp.username').type('cypress')
    field('smtp.password').type('cypress')

    // MailHog accepts no SMTP auth, and the schema default is "login".
    cy.get('select[name="smtp.auth_protocol"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('None').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(inboxName)
    cy.contains(inboxName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(inboxId, 'inbox from the create step').to.be.a('number')

    cy.visit(`${listPath}/${inboxId}/edit`)

    field('name').should('have.value', inboxName)
    field('from').should('have.value', fromAddress)

    field('imap.host').should('have.value', 'imap.gmail.com')
    field('imap.port').should('have.value', '993')
    field('imap.mailbox').should('have.value', 'INBOX')
    field('imap.username').should('have.value', 'cypress')
    field('imap.read_interval').should('have.value', '5m')
    field('imap.scan_inbox_since').should('have.value', '48h')

    field('smtp.host').should('have.value', smtpHost)
    field('smtp.port').should('have.value', smtpPort)
    field('smtp.username').should('have.value', 'cypress')
    field('smtp.max_conns').should('have.value', '10')
    field('smtp.max_msg_retries').should('have.value', '3')
    field('smtp.idle_timeout').should('have.value', '25s')

    // Passwords come back masked, never as the real value.
    field('imap.password').should('have.value', maskedSecret)
    field('smtp.password').should('have.value', maskedSecret)

    cy.get('select[name="imap.tls_type"]').should('have.value', 'none')
    cy.get('select[name="smtp.tls_type"]').should('have.value', 'none')
    cy.get('select[name="smtp.auth_protocol"]').should('have.value', 'none')
  })

  it('persists a changed name and mailbox', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    field('name').should('have.value', inboxName).clear().type(renamedInbox)
    field('imap.mailbox').clear().type('Archive')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${inboxId}/edit`)
    field('name').should('have.value', renamedInbox)
    field('imap.mailbox').should('have.value', 'Archive')
  })

  it('rejects a submit with no name, from address or credentials', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openManualForm()
    cy.get('button[type="submit"]').click()

    // Submitting scrolls to the bottom of a long form, so bring the first error back into view.
    cy.contains(/required/i).first().scrollIntoView().should('be.visible')
    cy.get('@createInbox.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the inbox', () => {
    cy.intercept('DELETE', `**/api/v1/inboxes/${inboxId}`).as('deleteInbox')

    cy.visit(listPath)
    filterList(renamedInbox)
    cy.contains('tr', renamedInbox).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteInbox').its('response.statusCode').should('eq', 200)
    cy.contains(renamedInbox).should('not.exist')
  })
})
