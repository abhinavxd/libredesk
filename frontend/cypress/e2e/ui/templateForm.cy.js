// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const templateName = `Cypress Template ${stamp}`
const renamedTemplate = `Cypress Template ${stamp} edited`
const templateBody = `Cypress template body ${stamp}`
const updatedBody = `Cypress template body ${stamp} edited`
const listPath = '/admin/templates'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// CodeMirror ignores synthetic key events, so text is written into its line element instead.
const typeBody = (text) => {
  cy.get('.cm-content').click()
  cy.get('.cm-content .cm-line').first().then(($line) => {
    $line[0].textContent = text
  })
  cy.get('.cm-content').should('have.text', text)
}

describe('Template form', () => {
  let templateId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates an outgoing email template', () => {
    cy.intercept('POST', '**/api/v1/templates').as('createTemplate')

    cy.visit(listPath)
    cy.contains('button', 'New template').click()

    cy.get('input[name="name"]').type(templateName)
    typeBody(templateBody)

    cy.get('button[type="submit"]').click()
    cy.wait('@createTemplate').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.body).to.eq(templateBody)
      templateId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(templateName)
    cy.contains(templateName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(templateId, 'template from the create step').to.be.a('number')

    cy.visit(`${listPath}/${templateId}/edit`)

    cy.get('input[name="name"]').should('have.value', templateName)
    cy.get('.cm-content').should('have.text', templateBody)
    cy.get('button[role="checkbox"]').should('have.attr', 'data-state', 'unchecked')
  })

  it('persists a changed name and body', () => {
    cy.intercept('PUT', `**/api/v1/templates/${templateId}`).as('updateTemplate')

    cy.visit(`${listPath}/${templateId}/edit`)
    cy.get('input[name="name"]').should('have.value', templateName).clear().type(renamedTemplate)
    typeBody(updatedBody)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateTemplate').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${templateId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedTemplate)
    cy.get('.cm-content').should('have.text', updatedBody)
  })

  it('rejects a submit with no name and no body', () => {
    cy.intercept('POST', '**/api/v1/templates').as('createTemplate')

    cy.visit(`${listPath}/new?type=email_outgoing`)
    cy.get('button[type="submit"]').click()

    // Only `exist`: an overflow container clips this message.
    cy.contains('Name Required').should('exist')
    cy.get('@createTemplate.all').should('have.length', 0)
    cy.location('pathname').should('eq', `${listPath}/new`)
  })

  it('deletes the template', () => {
    cy.intercept('DELETE', `**/api/v1/templates/${templateId}`).as('deleteTemplate')

    cy.visit(listPath)
    filterList(renamedTemplate)
    // Wait for the filter to settle, else the click hits a row the re-render detaches.
    cy.get('tbody tr').should('have.length', 1)
    cy.contains('tr', renamedTemplate).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteTemplate').its('response.statusCode').should('eq', 200)
    cy.contains(renamedTemplate).should('not.exist')
  })
})
