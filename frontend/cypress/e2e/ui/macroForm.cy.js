// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const macroName = `Cypress Macro ${stamp}`
const renamedMacro = `Cypress Macro ${stamp} edited`
const teamName = `Cypress Macro Team ${stamp}`
const messageBody = `Canned reply from the macro form spec ${stamp}`
const newPath = '/admin/conversations/macros/new'
const listPath = '/admin/conversations/macros'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

describe('Macro form', () => {
  let macroId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/teams', {
      name: teamName,
      emoji: '🚀',
      conversation_assignment_type: 'Round robin',
      timezone: 'UTC',
      max_auto_assigned_conversations: 0
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a macro', () => {
    cy.intercept('POST', '**/api/v1/macros').as('createMacro')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(macroName)
    cy.get('.tiptap.ProseMirror').click().type(messageBody)

    cy.get('button[type="submit"]').click()
    cy.wait('@createMacro').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      macroId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(macroName)
    cy.contains(macroName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(macroId, 'macro from the create step').to.be.a('number')

    cy.visit(`${listPath}/${macroId}/edit`)

    cy.get('input[name="name"]').should('have.value', macroName)
    cy.get('.tiptap.ProseMirror').should('contain.text', messageBody)
    cy.get('select[name="visibility"]').should('have.value', 'all')
  })

  it('persists a changed name and visibility', () => {
    cy.intercept('PUT', `**/api/v1/macros/${macroId}`).as('updateMacro')

    cy.visit(`${listPath}/${macroId}/edit`)
    cy.get('input[name="name"]').should('have.value', macroName).clear().type(renamedMacro)

    cy.get('select[name="visibility"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Team').click()
    cy.contains('label', 'Team').parent().find('button[role="combobox"]').click()
    cy.get('[role="option"]').contains(teamName).click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateMacro').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${macroId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedMacro)
    cy.get('select[name="visibility"]').should('have.value', 'team')
    cy.contains('label', 'Team').parent().find('button[role="combobox"]').should('contain.text', teamName)
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/macros').as('createMacro')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.get('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createMacro.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a macro with no message content and no action', () => {
    cy.intercept('POST', '**/api/v1/macros').as('createMacro')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(macroName)
    cy.get('button[type="submit"]').click()

    // The fixed-height editor box clips the error, so assert it exists rather than is visible.
    cy.contains('Either message content or actions are required').should('exist')
    cy.get('@createMacro.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the macro', () => {
    cy.intercept('DELETE', `**/api/v1/macros/${macroId}`).as('deleteMacro')

    cy.visit(listPath)
    filterList(renamedMacro)
    cy.contains('tr', renamedMacro).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteMacro').its('response.statusCode').should('eq', 200)
    cy.contains(renamedMacro).should('not.exist')
  })
})
