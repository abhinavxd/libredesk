// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const snippetTitle = `Cypress Snippet ${stamp}`
const renamedSnippet = `Cypress Snippet ${stamp} edited`
const snippetContent = `Refunds are processed within ${stamp} days.`
const editedContent = `Refunds are processed within ${stamp} hours.`
const listPath = '/admin/ai/snippets'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const openCreateDialog = () => {
  cy.visit(listPath)
  cy.contains('button', 'New snippet').click()
  cy.get('[role="dialog"]').contains('New snippet').should('be.visible')
}

const openEditDialog = (title) => {
  cy.visit(listPath)
  filterList(title)
  cy.contains('td span', title).click()
  cy.get('[role="dialog"]').contains('Edit snippet').should('be.visible')
}

describe('AI snippet form', () => {
  let snippetId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a snippet', () => {
    cy.intercept('POST', '**/api/v1/ai/snippets').as('createSnippet')

    openCreateDialog()

    cy.get('input[name="title"]').type(snippetTitle)
    cy.get('textarea[name="content"]').type(snippetContent)

    cy.get('[role="dialog"]').find('button[type="submit"]').click()
    cy.wait('@createSnippet').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      snippetId = response.body.data.id
    })

    filterList(snippetTitle)
    cy.contains(snippetTitle).should('exist')
  })

  it('loads the saved values back into the edit dialog', () => {
    expect(snippetId, 'snippet from the create step').to.be.a('number')

    openEditDialog(snippetTitle)

    cy.get('input[name="title"]').should('have.value', snippetTitle)
    cy.get('textarea[name="content"]').should('have.value', snippetContent)
    cy.get('[role="dialog"]').find('button[role="switch"]').should('have.attr', 'data-state', 'checked')
  })

  it('persists a changed title and content', () => {
    cy.intercept('PUT', `**/api/v1/ai/snippets/${snippetId}`).as('updateSnippet')

    openEditDialog(snippetTitle)
    cy.get('input[name="title"]').clear().type(renamedSnippet)
    cy.get('textarea[name="content"]').clear().type(editedContent)
    cy.get('[role="dialog"]').find('button[type="submit"]').click()
    cy.wait('@updateSnippet').its('response.statusCode').should('eq', 200)

    openEditDialog(renamedSnippet)
    cy.get('input[name="title"]').should('have.value', renamedSnippet)
    cy.get('textarea[name="content"]').should('have.value', editedContent)
  })

  it('rejects a submit with no title', () => {
    cy.intercept('POST', '**/api/v1/ai/snippets').as('createSnippet')

    openCreateDialog()
    cy.get('textarea[name="content"]').type(snippetContent)
    cy.get('[role="dialog"]').find('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('input[name="title"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createSnippet.all').should('have.length', 0)
  })

  it('deletes the snippet', () => {
    cy.intercept('DELETE', `**/api/v1/ai/snippets/${snippetId}`).as('deleteSnippet')

    cy.visit(listPath)
    filterList(renamedSnippet)
    cy.contains('tr', renamedSnippet).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteSnippet').its('response.statusCode').should('eq', 200)
    cy.contains(renamedSnippet).should('not.exist')
  })
})
