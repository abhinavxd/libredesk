// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const attributeName = `Cypress Attr ${stamp}`
const attributeKey = attributeName.toLowerCase().replace(/ /g, '_')
const listPath = '/admin/custom-attributes'

const openNewDialog = () => {
  cy.visit(listPath)
  cy.contains('button', 'New custom attribute').click()
  cy.get('[role="dialog"]').should('be.visible')
}

const openEditDialog = () => {
  cy.visit(listPath)
  filterList(attributeName)
  cy.contains('tr', attributeName).find('button[aria-haspopup="menu"]').click()
  cy.get('[role="menuitem"]').contains('Edit').click()
  cy.get('[role="dialog"]').should('be.visible')
}

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// A radix Select renders a hidden native select next to its trigger, options go in a portal.
const pickOption = (field, optionText) => {
  cy.get(`select[name="${field}"]`).siblings('button[role="combobox"]').click()
  cy.get('[role="option"]').contains(optionText).click()
}

describe('Custom attribute form', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a custom attribute', () => {
    cy.intercept('POST', '**/api/v1/custom-attributes').as('createAttribute')

    openNewDialog()
    cy.get('input[name="name"]').type(attributeName)
    cy.get('input[name="description"]').type('Created by the custom attribute form spec')
    pickOption('data_type', 'Text')
    cy.get('input[name="regex"]').type('^[a-z]+$')
    cy.get('input[name="regex_hint"]').type('lowercase letters only')

    cy.get('button[type="submit"]').click()
    cy.wait('@createAttribute').its('response.statusCode').should('eq', 200)

    filterList(attributeName)
    cy.contains(attributeName).should('exist')
  })

  it('loads the saved values back into the edit dialog', () => {
    openEditDialog()

    cy.get('input[name="name"]').should('have.value', attributeName)
    cy.get('input[name="key"]').should('have.value', attributeKey)
    cy.get('input[name="description"]').should(
      'have.value',
      'Created by the custom attribute form spec'
    )
    cy.get('select[name="data_type"]').should('have.value', 'text')
    cy.get('input[name="regex"]').should('have.value', '^[a-z]+$')
    cy.get('input[name="regex_hint"]').should('have.value', 'lowercase letters only')
  })

  it('persists a changed description', () => {
    cy.intercept('PUT', '**/api/v1/custom-attributes/*').as('updateAttribute')

    openEditDialog()
    cy.get('input[name="description"]').clear().type('Edited by the spec')
    cy.get('button[type="submit"]').click()
    cy.wait('@updateAttribute').its('response.statusCode').should('eq', 200)

    openEditDialog()
    cy.get('input[name="description"]').should('have.value', 'Edited by the spec')
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/custom-attributes').as('createAttribute')

    openNewDialog()
    cy.get('input[name="description"]').type('No name given')
    cy.get('button[type="submit"]').click()

    cy.get('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.contains('Must be between 3 and 140 characters').should('be.visible')
    cy.get('@createAttribute.all').should('have.length', 0)
    cy.get('[role="dialog"]').should('be.visible')
  })

  it('deletes the custom attribute', () => {
    cy.intercept('DELETE', '**/api/v1/custom-attributes/*').as('deleteAttribute')

    cy.visit(listPath)
    filterList(attributeName)
    cy.contains('tr', attributeName).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteAttribute').its('response.statusCode').should('eq', 200)
    cy.contains(attributeName).should('not.exist')
  })
})
