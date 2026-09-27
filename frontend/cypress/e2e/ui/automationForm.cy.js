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

const pickType = (label) => {
  cy.get('select[name="type"]').siblings('button[role="combobox"]').click()
  cy.contains('[role="option"]', label).click()
  cy.get('[role="option"]').should('not.exist')
}

const openFieldList = () => conditionRows().eq(0).find('button[role="combobox"]').eq(0).click()

const closeList = () => {
  cy.get('[role="listbox"]').type('{esc}')
  cy.get('[role="option"]').should('not.exist')
}

const switchField = (label) => {
  openFieldList()
  cy.contains('[role="option"]', label).click()
  cy.get('[role="option"]').should('not.exist')
}

const caseSensitiveBox = () => conditionRows().eq(0).parent().find('button[role="checkbox"]')

const addPrivateNoteAction = () => {
  cy.contains('button', 'Add action').click()
  pickOption('Select action', 'Add private note')
  cy.get('.tiptap.ProseMirror').click()
  cy.get('.tiptap.ProseMirror').type(noteBody)
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

describe('Automation form case sensitive match', () => {
  const caseRuleName = `Cypress case rule ${stamp}`
  const savedRuleName = `Cypress saved case rule ${stamp}`
  const created = []

  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
  })

  after(() => {
    cy.login()
    created.forEach((id) => {
      cy.api('DELETE', `/api/v1/automations/rules/${id}`, null, { failOnStatusCode: false })
    })
  })

  it('clears the tick when the field changes', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(caseRuleName)
    pickType('New conversation')

    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', 'Subject')
    caseSensitiveBox().click().should('have.attr', 'data-state', 'checked')

    switchField(/^Content$/)
    caseSensitiveBox().should('have.attr', 'data-state', 'unchecked')

    caseSensitiveBox().click().should('have.attr', 'data-state', 'checked')
    switchField(/^Email$/)
    caseSensitiveBox().should('not.exist')

    switchField(/^To email address$/)
    caseSensitiveBox().should('not.exist')

    switchField(/^Subject$/)
    caseSensitiveBox().should('have.attr', 'data-state', 'unchecked')

    caseSensitiveBox().click().should('have.attr', 'data-state', 'checked')
    switchField(/^Email$/)
    pickOption('Select operator', /^equals$/)
    conditionRows().eq(0).find('input[type="text"]').type('Customer@Example.com')
    addPrivateNoteAction()

    cy.get('button[type="submit"]').click()
    cy.wait('@createRule').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      created.push(response.body.data.id)
      const condition = request.body.rules[0].groups[0].rules[0]
      expect(condition.field).to.eq('contact_email')
      expect(condition.case_sensitive_match).to.eq(false)
    })
  })

  it('saves a ticked subject condition as case sensitive', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(`${caseRuleName} ticked`)
    pickType('New conversation')

    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', 'Subject')
    pickOption('Select operator', /^equals$/)
    conditionRows().eq(0).find('input[type="text"]').type('Exact Subject')
    caseSensitiveBox().click()
    caseSensitiveBox().should('have.attr', 'data-state', 'checked')
    addPrivateNoteAction()

    cy.get('button[type="submit"]').click()
    cy.wait('@createRule').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      created.push(response.body.data.id)
      const condition = request.body.rules[0].groups[0].rules[0]
      expect(condition.field).to.eq('subject')
      expect(condition.case_sensitive_match).to.eq(true)
    })
  })

  it('shows a saved tick when editing and clears it on a field change', () => {
    cy.api('POST', '/api/v1/automations/rules', {
      name: savedRuleName,
      description: 'created by the case sensitive spec',
      type: 'new_conversation',
      enabled: false,
      events: [],
      rules: [
        {
          group_operator: 'OR',
          groups: [
            {
              logical_op: 'OR',
              rules: [
                {
                  field: 'subject',
                  field_type: 'conversation',
                  operator: 'equals',
                  value: `Cypress subject ${stamp}`,
                  case_sensitive_match: true
                }
              ]
            },
            { logical_op: 'OR', rules: [] }
          ],
          actions: [{ type: 'send_private_note', value: [noteBody] }]
        }
      ]
    }).then(({ body }) => {
      const id = body.data.id
      created.push(id)
      cy.intercept('PUT', `**/api/v1/automations/rules/${id}`).as('updateRule')

      openRule(id)
      caseSensitiveBox().should('have.attr', 'data-state', 'checked')

      switchField(/^Email$/)
      caseSensitiveBox().should('not.exist')
      switchField(/^Subject$/)
      caseSensitiveBox().should('have.attr', 'data-state', 'unchecked')

      pickOption('Select operator', /^equals$/)
      conditionRows().eq(0).find('input[type="text"]').type(`Cypress subject ${stamp}`)
      cy.get('button[type="submit"]').click()
      cy.wait('@updateRule').then(({ request, response }) => {
        expect(response.statusCode).to.eq(200)
        const condition = request.body.rules[0].groups[0].rules[0]
        expect(condition.field).to.eq('subject')
        expect(condition.case_sensitive_match).to.eq(false)
      })
    })
  })
})

describe('Automation form To email address field', () => {
  const toRuleName = `Cypress to rule ${stamp}`
  const created = []

  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
  })

  after(() => {
    cy.login()
    created.forEach((id) => {
      cy.api('DELETE', `/api/v1/automations/rules/${id}`, null, { failOnStatusCode: false })
    })
  })

  it('is offered only on new conversation rules', () => {
    cy.visit(newPath)

    pickType('New conversation')
    cy.contains('button', 'Add condition').first().click()
    openFieldList()
    cy.contains('[role="option"]', /^To email address$/).should('exist')
    closeList()

    pickType('Conversation update')
    cy.contains('button', 'Add condition').first().click()
    openFieldList()
    cy.contains('[role="option"]', /^Status$/).should('exist')
    cy.contains('[role="option"]', /^To email address$/).should('not.exist')
    closeList()

    pickType('Time triggers')
    cy.contains('button', 'Add condition').first().click()
    openFieldList()
    cy.contains('[role="option"]', /^Status$/).should('exist')
    cy.contains('[role="option"]', /^To email address$/).should('not.exist')
    closeList()
  })

  it('saves a To email address condition and loads it back', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createRule')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(toRuleName)
    pickType('New conversation')

    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', 'To email address')
    caseSensitiveBox().should('not.exist')
    pickOption('Select operator', /^equals$/)
    conditionRows().eq(0).find('input[type="text"]').type('sales@example.com')
    addPrivateNoteAction()

    cy.get('button[type="submit"]').click()
    cy.wait('@createRule').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      created.push(response.body.data.id)
      const condition = request.body.rules[0].groups[0].rules[0]
      expect(condition.field).to.eq('to')
      expect(condition.field_type).to.eq('conversation')
      expect(condition.operator).to.eq('equals')
      expect(condition.value).to.eq('sales@example.com')

      openRule(response.body.data.id)
      conditionRows().eq(0).find('button[role="combobox"]').eq(0).should('contain.text', 'To email address')
      conditionRows().eq(0).find('button[role="combobox"]').eq(1).should('contain.text', 'equals')
      conditionRows().eq(0).find('input[type="text"]').should('have.value', 'sales@example.com')
      caseSensitiveBox().should('not.exist')
    })
  })
})
