// The steps run in order and share the record created by the first one.

// Random, not a truncated timestamp: the last six digits of Date.now() repeat
// every ~17 minutes and a leaked status would then collide by name.
const stamp = Math.random().toString(36).slice(2, 8)
// The name is capped at 25 characters, so keep the stamp short.
const statusName = `CyStatus ${stamp}`
const renamedStatus = `CyStatus ${stamp} v2`
const listPath = '/admin/conversations/statuses'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const openNewDialog = () => cy.contains('button', 'New status').click()

const openEditDialog = (name) => {
  filterList(name)
  cy.contains('tr', name).contains('span', name).click()
}

const pickCategory = (optionText) => {
  cy.get('select[name="category"]').siblings('button[role="combobox"]').click()
  cy.get('[role="option"]').contains(optionText).click()
}

describe('Status form', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a status', () => {
    cy.intercept('POST', '**/api/v1/statuses').as('createStatus')

    cy.visit(listPath)
    openNewDialog()

    cy.get('input[name="name"]').type(statusName)
    pickCategory('Waiting')

    cy.get('button[type="submit"]').click()
    cy.wait('@createStatus').its('response.statusCode').should('eq', 200)

    cy.get('[role="dialog"]').should('not.exist')
    filterList(statusName)
    cy.contains(statusName).should('exist')
  })

  it('loads the saved values back into the edit dialog', () => {
    cy.visit(listPath)
    openEditDialog(statusName)

    cy.get('input[name="name"]').should('have.value', statusName)
    cy.get('select[name="category"]').should('have.value', 'waiting')
  })

  it('persists a changed name and category', () => {
    cy.intercept('PUT', '**/api/v1/statuses/*').as('updateStatus')

    cy.visit(listPath)
    openEditDialog(statusName)
    cy.get('input[name="name"]').clear().type(renamedStatus)
    pickCategory('Resolved')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateStatus').its('response.statusCode').should('eq', 200)

    cy.visit(listPath)
    openEditDialog(renamedStatus)
    cy.get('input[name="name"]').should('have.value', renamedStatus)
    cy.get('select[name="category"]').should('have.value', 'resolved')
  })

  it('rejects a submit with no name or category', () => {
    cy.intercept('POST', '**/api/v1/statuses').as('createStatus')

    cy.visit(listPath)
    openNewDialog()
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createStatus.all').should('have.length', 0)
    cy.get('[role="dialog"]').should('be.visible')
  })

  it('deletes the status', () => {
    cy.intercept('DELETE', '**/api/v1/statuses/*').as('deleteStatus')

    cy.visit(listPath)
    filterList(renamedStatus)
    cy.contains('tr', renamedStatus).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteStatus').its('response.statusCode').should('eq', 200)
    cy.contains(renamedStatus).should('not.exist')
  })
})
