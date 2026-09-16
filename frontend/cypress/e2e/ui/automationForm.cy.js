// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const ruleName = `Cypress Rule ${stamp}`
const renamedRule = `Cypress Rule ${stamp} edited`
const ruleDescription = `Rule from the automation form spec ${stamp}`
const conditionValue = `cypress subject ${stamp}`
const noteBody = `Private note from the automation form spec ${stamp}`
const newPath = '/admin/automations/new'
const listPath = '/admin/automations'

// Condition and action rows are not vee-validate fields, so they have no name attribute.
const conditionRows = () => cy.get('div.flex.space-x-5.items-start')
const actionRows = () => cy.get('div.flex.items-start.justify-between.gap-5')

const pickOption = (trigger, optionText) => {
  cy.contains('button[role="combobox"]', trigger).click()
  cy.contains('[role="option"]', optionText).click()
}

const openRule = (id) => {
  cy.visit(`${listPath}/${id}/edit`)
  cy.get('input[name="name"]').should('not.have.value', '')
}

describe('Automation form', () => {
  let ruleId

  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
  })

  it('creates a rule with a condition and an action', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(ruleName)
    cy.get('input[name="description"]').type(ruleDescription)
    cy.get('select[name="type"]').siblings('button[role="combobox"]').click()
    cy.contains('[role="option"]', 'New conversation').click()

    cy.contains('button', 'Add condition').first().click()
    conditionRows().should('have.length', 1)
    pickOption('Select field', 'Subject')
    pickOption('Select operator', /^equals$/)
    conditionRows().eq(0).find('input[type="text"]').type(conditionValue)

    cy.contains('button', 'Add action').click()
    actionRows().should('have.length', 1)
    pickOption('Select action', 'Add private note')
    cy.get('.tiptap.ProseMirror').click().type(noteBody)

    cy.get('button[type="submit"]').click()
    cy.wait('@createRule').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      ruleId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    cy.contains(ruleName).should('exist')
  })

  it('loads the saved condition and action back into the edit form', () => {
    expect(ruleId, 'rule from the create step').to.be.a('number')

    openRule(ruleId)

    cy.get('input[name="name"]').should('have.value', ruleName)
    cy.get('input[name="description"]').should('have.value', ruleDescription)
    cy.get('select[name="type"]').should('have.value', 'new_conversation')

    conditionRows().should('have.length', 1)
    conditionRows().eq(0).find('button[role="combobox"]').eq(0).should('contain.text', 'Subject')
    conditionRows().eq(0).find('button[role="combobox"]').eq(1).should('contain.text', 'equals')
    conditionRows().eq(0).find('input[type="text"]').should('have.value', conditionValue)

    actionRows().should('have.length', 1)
    actionRows().eq(0).find('button[role="combobox"]').should('contain.text', 'Add private note')
    cy.get('.tiptap.ProseMirror').should('contain.text', noteBody)
  })

  it('adds and removes a condition row without submitting the form', () => {
    cy.intercept('PUT', `**/api/v1/automations/rules/${ruleId}`).as('updateRule')

    openRule(ruleId)
    conditionRows().should('have.length', 1)

    cy.contains('button', 'Add condition').first().click()
    conditionRows().should('have.length', 2)

    conditionRows().eq(1).find('button[aria-label="Close"]').click()
    cy.get('@updateRule.all').should('have.length', 0)

    conditionRows().should('have.length', 1)
    conditionRows().eq(0).find('input[type="text"]').should('have.value', conditionValue)
    cy.location('pathname').should('eq', `${listPath}/${ruleId}/edit`)
  })

  it('persists a changed name', () => {
    cy.intercept('PUT', `**/api/v1/automations/rules/${ruleId}`).as('updateRule')

    openRule(ruleId)
    cy.get('input[name="name"]').should('have.value', ruleName).clear().type(renamedRule)
    cy.get('button[type="submit"]').click()
    cy.wait('@updateRule').its('response.statusCode').should('eq', 200)

    openRule(ruleId)
    cy.get('input[name="name"]').should('have.value', renamedRule)
    conditionRows().eq(0).find('input[type="text"]').should('have.value', conditionValue)
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('exist')
    cy.get('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createRule.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a rule with no condition', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(`${ruleName} no condition`)
    cy.get('select[name="type"]').siblings('button[role="combobox"]').click()
    cy.contains('[role="option"]', 'New conversation').click()
    cy.get('button[type="submit"]').click()

    cy.contains('Please add at least one condition.').should('exist')
    cy.get('@createRule.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the rule', () => {
    cy.intercept('DELETE', `**/api/v1/automations/rules/${ruleId}`).as('deleteRule')

    cy.visit(listPath)
    cy.contains('div.box', renamedRule).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteRule').its('response.statusCode').should('eq', 200)
    cy.contains(renamedRule).should('not.exist')
  })
})
