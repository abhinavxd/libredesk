// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const linkName = `Cypress Context Link ${stamp}`
const renamedLink = `Cypress Context Link ${stamp} edited`
const urlTemplate = `https://tools.cypress.test/lookup/${stamp}`
const updatedUrlTemplate = `https://tools.cypress.test/lookup/${stamp}/v2`
// The backend rejects a signing secret that is not exactly 32 characters.
const secret = `${stamp}`.padEnd(32, 'a').slice(0, 32)
const maskedSecret = '••••••••••'
const newPath = '/admin/context-links/new'
const listPath = '/admin/context-links'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

describe('Context link form', () => {
  let linkId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a context link', () => {
    cy.intercept('POST', '**/api/v1/context-links').as('createLink')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(linkName)
    cy.get('input[name="url_template"]').type(urlTemplate)
    cy.get('input[name="secret"]').type(secret)
    cy.get('input[name="token_expiry_seconds"]').clear().type('900')

    cy.get('button[type="submit"]').click()
    cy.wait('@createLink').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      linkId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(linkName)
    cy.contains(linkName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(linkId, 'link from the create step').to.be.a('number')

    cy.visit(`${listPath}/${linkId}/edit`)

    cy.get('input[name="name"]').should('have.value', linkName)
    cy.get('input[name="url_template"]').should('have.value', urlTemplate)
    cy.get('input[name="token_expiry_seconds"]').should('have.value', '900')
    // The signing secret comes back masked, never as the real value.
    cy.get('input[name="secret"]').should('have.value', maskedSecret)
    cy.get('button[role="checkbox"]').should('have.attr', 'data-state', 'checked')
  })

  it('persists a changed name, url template and active flag', () => {
    cy.intercept('PUT', `**/api/v1/context-links/${linkId}`).as('updateLink')

    cy.visit(`${listPath}/${linkId}/edit`)
    cy.get('input[name="name"]').should('have.value', linkName).clear().type(renamedLink)
    cy.get('input[name="url_template"]').clear().type(updatedUrlTemplate)
    cy.get('button[role="checkbox"]').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateLink').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${linkId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedLink)
    cy.get('input[name="url_template"]').should('have.value', updatedUrlTemplate)
    cy.get('button[role="checkbox"]').should('have.attr', 'data-state', 'unchecked')
  })

  it('rejects a submit with no name or url template', () => {
    cy.intercept('POST', '**/api/v1/context-links').as('createLink')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createLink.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the context link', () => {
    cy.intercept('DELETE', `**/api/v1/context-links/${linkId}`).as('deleteLink')

    cy.visit(listPath)
    filterList(renamedLink)
    cy.contains('tr', renamedLink).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteLink').its('response.statusCode').should('eq', 200)
    cy.contains(renamedLink).should('not.exist')
  })
})
