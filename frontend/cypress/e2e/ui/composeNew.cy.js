describe('Compose New', () => {
  const address = `compose.${Date.now()}@example.test`
  let inboxID

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: 'Compose mailbox', channel: 'email', from: address, enabled: false,
      config: { auth_type: 'password', imap: [], smtp: [] }
    }).then(({ body }) => { inboxID = body.data.id })
  })

  after(() => {
    if (inboxID) cy.api('DELETE', `/api/v1/inboxes/${inboxID}`)
  })

  beforeEach(() => {
    cy.login()
    cy.visit('/addresses')
  })

  it('replaces New conversation with Compose New', () => {
    cy.contains('New conversation').should('not.exist')
    cy.get('button[aria-label="Compose New"]').should('be.visible')
  })

  it('explains what is missing before sending and closes cleanly', () => {
    cy.get('button[aria-label="Compose New"]').click()
    cy.get('[role="dialog"]').within(() => {
      cy.contains('New email').should('be.visible')
      cy.get('select').should('exist')
      cy.contains('button', /^Send$/).click()
      cy.contains('Add at least one recipient in To.').should('be.visible')
      cy.contains('Add a subject.').should('be.visible')
      cy.contains('button', 'Cancel').click()
    })
    cy.get('[role="dialog"]').should('not.exist')
  })

  it('asks before discarding a started email', () => {
    cy.get('button[aria-label="Compose New"]').click()
    cy.get('[role="dialog"] input[maxlength="998"]').type('Draft subject')
    cy.get('[role="dialog"]').contains('button', 'Cancel').click()
    cy.contains('Discard this email?').should('be.visible')
    cy.contains('button', 'Keep editing').click()
    cy.get('[role="dialog"] input[maxlength="998"]').should('have.value', 'Draft subject')
  })

  it('rejects a disabled address on the server', () => {
    cy.api('GET', '/api/v1/admin/addresses').then(({ body }) => {
      const entry = body.data.find((item) => item.address === address)
      cy.api('POST', '/api/v1/compose', {
        address_id: entry.id, subject: 'Hello', content: '<p>Hi</p>', to: ['someone@example.test']
      }, { failOnStatusCode: false }).its('status').should('eq', 400)
    })
  })
})
