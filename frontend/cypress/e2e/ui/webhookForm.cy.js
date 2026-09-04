// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const webhookName = `Cypress Webhook ${stamp}`
const renamedWebhook = `Cypress Webhook ${stamp} edited`
const webhookUrl = `https://cypress.test/hooks/${stamp}`
const updatedUrl = `https://cypress.test/hooks/${stamp}/v2`
const newPath = '/admin/webhooks/new'
const listPath = '/admin/webhooks'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const checkEvent = (label) =>
  cy.contains('label', label).siblings('button[role="checkbox"]').click()

const eventCheckbox = (label) => cy.contains('label', label).siblings('button[role="checkbox"]')

describe('Webhook form', () => {
  let webhookId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a webhook', () => {
    cy.intercept('POST', '**/api/v1/webhooks').as('createWebhook')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(webhookName)
    cy.get('input[name="url"]').type(webhookUrl)
    checkEvent('Conversation created')
    checkEvent('Message created')
    cy.get('input[name="secret"]').type('cypress-secret')

    cy.get('button[type="submit"]').click()
    cy.wait('@createWebhook').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      webhookId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(webhookName)
    cy.contains(webhookName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(webhookId, 'webhook from the create step').to.be.a('number')

    cy.visit(`${listPath}/${webhookId}/edit`)

    cy.get('input[name="name"]').should('have.value', webhookName)
    cy.get('input[name="url"]').should('have.value', webhookUrl)
    eventCheckbox('Conversation created').should('have.attr', 'data-state', 'checked')
    eventCheckbox('Message created').should('have.attr', 'data-state', 'checked')
    eventCheckbox('Conversation assigned').should('have.attr', 'data-state', 'unchecked')
  })

  it('persists a changed url and event list', () => {
    cy.intercept('PUT', `**/api/v1/webhooks/${webhookId}`).as('updateWebhook')

    cy.visit(`${listPath}/${webhookId}/edit`)
    cy.get('input[name="name"]').should('have.value', webhookName).clear().type(renamedWebhook)
    cy.get('input[name="url"]').clear().type(updatedUrl)
    checkEvent('Conversation assigned')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateWebhook').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${webhookId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedWebhook)
    cy.get('input[name="url"]').should('have.value', updatedUrl)
    eventCheckbox('Conversation assigned').should('have.attr', 'data-state', 'checked')
  })

  it('rejects a submit with no name, url or event', () => {
    cy.intercept('POST', '**/api/v1/webhooks').as('createWebhook')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    // Only `exist`: an overflow container clips this message.
    cy.contains(/required/i).should('exist')
    cy.get('@createWebhook.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a malformed url', () => {
    cy.intercept('POST', '**/api/v1/webhooks').as('createWebhook')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(webhookName)
    cy.get('input[name="url"]').type('not-a-url')
    checkEvent('Conversation created')
    cy.get('button[type="submit"]').click()

    cy.contains('Invalid URL').should('be.visible')
    cy.get('@createWebhook.all').should('have.length', 0)
  })

  it('deletes the webhook', () => {
    cy.intercept('DELETE', `**/api/v1/webhooks/${webhookId}`).as('deleteWebhook')

    cy.visit(listPath)
    filterList(renamedWebhook)
    cy.contains('tr', renamedWebhook).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteWebhook').its('response.statusCode').should('eq', 200)
    cy.contains(renamedWebhook).should('not.exist')
  })
})
