// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const tagName = `Cypress Tag ${stamp}`
const renamedTag = `Cypress Tag ${stamp} edited`
const listPath = '/admin/conversations/tags'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const openNewDialog = () => cy.contains('button', 'New tag').click()

const openEditDialog = (name) => {
  filterList(name)
  cy.contains('tr', name).contains('span', name).click()
}

describe('Tag form', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a tag', () => {
    cy.intercept('POST', '**/api/v1/tags').as('createTag')

    cy.visit(listPath)
    openNewDialog()

    cy.get('input[name="name"]').type(tagName)

    cy.get('button[type="submit"]').click()
    cy.wait('@createTag').its('response.statusCode').should('eq', 200)

    cy.get('[role="dialog"]').should('not.exist')
    filterList(tagName)
    cy.contains(tagName).should('exist')
  })

  it('loads the saved value back into the edit dialog', () => {
    cy.visit(listPath)
    openEditDialog(tagName)

    cy.get('input[name="name"]').should('have.value', tagName)
  })

  it('persists a changed name', () => {
    cy.intercept('PUT', '**/api/v1/tags/*').as('updateTag')

    cy.visit(listPath)
    openEditDialog(tagName)
    cy.get('input[name="name"]').clear().type(renamedTag)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateTag').its('response.statusCode').should('eq', 200)

    cy.visit(listPath)
    openEditDialog(renamedTag)
    cy.get('input[name="name"]').should('have.value', renamedTag)
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/tags').as('createTag')

    cy.visit(listPath)
    openNewDialog()
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createTag.all').should('have.length', 0)
    cy.get('[role="dialog"]').should('be.visible')
  })

  it('rejects a name shorter than three characters', () => {
    cy.intercept('POST', '**/api/v1/tags').as('createTag')

    cy.visit(listPath)
    openNewDialog()
    cy.get('input[name="name"]').type('ab')
    cy.get('button[type="submit"]').click()

    cy.contains('Tag name should be at least 3 characters').should('be.visible')
    cy.get('@createTag.all').should('have.length', 0)
  })

  it('deletes the tag', () => {
    cy.intercept('DELETE', '**/api/v1/tags/*').as('deleteTag')

    cy.visit(listPath)
    filterList(renamedTag)
    cy.contains('tr', renamedTag).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteTag').its('response.statusCode').should('eq', 200)
    cy.contains(renamedTag).should('not.exist')
  })
})
