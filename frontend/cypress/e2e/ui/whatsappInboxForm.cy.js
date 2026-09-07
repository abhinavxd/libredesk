// Saving a WhatsApp inbox makes the backend ask Meta to verify the credentials, so this spec never creates one.

const stamp = Date.now()
const inboxName = `Cypress WhatsApp ${stamp}`
const newPath = '/admin/inboxes/new'

const openNewForm = () => {
  cy.visit(newPath)
  cy.contains('Create a WhatsApp inbox').click()
}

describe('WhatsApp inbox form', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('shows every credential and webhook field', () => {
    openNewForm()

    cy.get('input[name="name"]').should('exist')
    cy.get('input[name="reopen_window_hours"]').should('exist')
    cy.get('input[name="config.phone_number_id"]').should('exist')
    cy.get('input[name="config.waba_id"]').should('exist')
    cy.get('input[name="config.access_token"]').should('exist')
    cy.get('input[name="config.app_secret"]').should('exist')
    cy.get('input[name="config.webhook_verify_token"]').should('exist')
    cy.get('input[name="config.api_version"]').should('have.value', 'v25.0')

    // The callback URL only exists once the inbox has an id.
    cy.contains('Save the inbox to generate the webhook URL.').scrollIntoView()
    cy.contains('Save the inbox to generate the webhook URL.').should('be.visible')
  })

  it('keeps the CSAT template fields hidden until CSAT surveys are on', () => {
    openNewForm()

    cy.get('textarea[name="config.csat_template_body"]').should('not.be.visible')
    cy.contains('CSAT Surveys').parent().find('button[role="switch"]').click()
    cy.get('textarea[name="config.csat_template_body"]')
      .should('be.visible')
      .and('not.have.value', '')
    cy.get('input[name="config.csat_template_button_text"]').should('have.value', 'Rate us')
  })

  it('rejects a submit with no name and no credentials', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.get('button[type="submit"]').click()

    cy.get('input[name="name"]').scrollIntoView()
    cy.contains(/required/i).should('be.visible')
    cy.get('@createInbox.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a submit that is missing only the app secret', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.phone_number_id"]').type(`pn-${stamp}`)
    cy.get('input[name="config.waba_id"]').type(`waba-${stamp}`)
    cy.get('input[name="config.access_token"]').type('dummy-access-token')
    cy.get('input[name="config.webhook_verify_token"]').type(`verify-${stamp}`)

    cy.get('button[type="submit"]').click()

    cy.get('input[name="config.app_secret"]').scrollIntoView()
    cy.contains(/required/i).should('be.visible')
    cy.get('@createInbox.all').should('have.length', 0)
  })

  it('surfaces the error when Meta rejects the credentials', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')
    // Deterministic either way: the stand-in Graph API is told to refuse, and real Meta refuses anyway.
    cy.task('metaMock:failValidate', true)

    openNewForm()
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.phone_number_id"]').type(`pn-${stamp}`)
    cy.get('input[name="config.waba_id"]').type(`waba-${stamp}`)
    cy.get('input[name="config.access_token"]').type('dummy-access-token')
    cy.get('input[name="config.app_secret"]').type('dummy-app-secret')
    cy.get('input[name="config.webhook_verify_token"]').type(`verify-${stamp}`)

    cy.get('button[type="submit"]').click()

    cy.wait('@createInbox', { timeout: 90000 }).its('response.statusCode').should('eq', 400)
    cy.location('pathname').should('eq', newPath)
    cy.api('GET', '/api/v1/inboxes').then(({ body }) => {
      expect(body.data.filter((inbox) => inbox.name === inboxName)).to.have.length(0)
    })
    cy.task('metaMock:failValidate', false)
  })
})
