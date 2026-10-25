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
const conditionRows = () => cy.get('[data-cy="automation-condition"]')
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

describe('Additional automation conditions and actions', () => {
  const createdRules = []
  let tagId
  const tagName = `Billing, refunds ${stamp}`
  const attributes = {}
  const attributeIds = []
  const newConditions = [
    'Channel', 'Tags', 'Status category', 'Country', 'Contact external ID',
    'Phone number', 'Contact type', 'Within business hours'
  ]
  const updateConditions = [
    ...newConditions, 'Subject', 'Content', 'Email', 'Hours waiting for an agent reply',
    'Hours since last public message', 'Hours since last customer message',
    'Hours since last activity', 'Last public message by', 'SLA policy',
    'SLA breached', 'Hours until next SLA deadline', 'CSAT rating', 'CSAT feedback'
  ]

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/tags', { name: tagName }).then(({ body }) => { tagId = body.data.id })
    cy.api('POST', '/api/v1/custom-attributes', {
      name: 'Automation amount', description: 'Numeric automation test field',
      applies_to: 'conversation', key: `automation_amount_${stamp}`, data_type: 'number', values: []
    }).then(({ body }) => {
      attributes.amount = body.data
      attributeIds.push(body.data.id)
    })
    for (const scope of ['contact', 'conversation']) {
      cy.api('POST', '/api/v1/custom-attributes', {
        name: `Automation ${scope} flag`, description: 'Automation test field',
        applies_to: scope, key: `automation_${scope}_${stamp}`, data_type: 'checkbox', values: []
      }).then(({ body }) => {
        attributes[scope] = body.data
        attributeIds.push(body.data.id)
      })
    }
  })

  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
  })

  after(() => {
    cy.login()
    createdRules.forEach((id) => cy.api('DELETE', `/api/v1/automations/rules/${id}`))
    cy.api('DELETE', `/api/v1/tags/${tagId}`)
    attributeIds.forEach((id) => cy.api('DELETE', `/api/v1/custom-attributes/${id}`))
  })

  for (const type of ['New conversation', 'Conversation update', 'Time triggers']) {
    it(`offers the supported conditions for ${type}`, () => {
      cy.visit(newPath)
      pickType(type)
      cy.contains('button', 'Add condition').first().click()
      openFieldList()
      for (const label of type === 'New conversation' ? newConditions : updateConditions) {
        cy.contains('[role="option"]', new RegExp(`^${label}$`)).should('exist')
      }
      cy.contains('[role="option"]', 'Automation contact flag').should('exist')
      cy.contains('[role="option"]', 'Automation conversation flag').should('exist')
      closeList()
    })
  }

  it('uses the actual status categories', () => {
    cy.visit(newPath)
    pickType('New conversation')
    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', 'Status category')
    pickOption('Select operator', /^equals$/)
    conditionRows().first().find('button[role="combobox"]').eq(2).click()
    for (const label of ['Open', 'Waiting', 'Resolved']) {
      cy.contains('[role="option"]', new RegExp(`^${label}$`)).should('exist')
    }
    cy.contains('[role="option"]', /^Snoozed$/).should('not.exist')
    closeList()
  })

  it('preserves tag names containing commas when saving a condition', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createTagRule')
    cy.visit(newPath)
    cy.get('input[name="name"]').type(`Tag condition ${stamp}`)
    pickType('New conversation')
    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', /^Tags$/)
    pickOption('Select operator', /^contains any of$/)
    conditionRows().first().find('input[role="combobox"]').click()
    cy.contains('[role="option"]', tagName).click()
    cy.get('body').type('{esc}')
    cy.contains('button', 'Add action').click()
    pickOption('Select action', 'Clear tags')
    cy.get('button[type="submit"]').click()
    cy.wait('@createTagRule').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      createdRules.push(response.body.data.id)
      expect(JSON.parse(request.body.rules[0].groups[0].rules[0].value)).to.deep.eq([tagName])
      openRule(response.body.data.id)
      conditionRows().first().should('contain.text', tagName)
    })
  })

  it('saves a conversation attribute condition separately from contact attributes', () => {
    cy.intercept('POST', '**/api/v1/automations/rules').as('createAdditionalRule')
    cy.visit(newPath)
    cy.get('input[name="name"]').type(`Conversation attribute rule ${stamp}`)
    pickType('New conversation')
    cy.contains('button', 'Add condition').first().click()
    pickOption('Select field', 'Automation conversation flag')
    pickOption('Select operator', /^equals$/)
    conditionRows().first().find('button[role="combobox"]').eq(2).click()
    cy.contains('[role="option"]', /^False$/).click()
    cy.contains('button', 'Add action').click()
    pickOption('Select action', 'Clear tags')
    cy.get('button[type="submit"]').click()
    cy.wait('@createAdditionalRule').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      createdRules.push(response.body.data.id)
      expect(request.body.rules[0].groups[0].rules[0]).to.include({
        field: attributes.conversation.key, field_type: 'conversation_custom_attribute', value: 'false'
      })
      expect(request.body.rules[0].actions[0]).to.include({ type: 'clear_tags' })
      openRule(response.body.data.id)
      conditionRows().first().should('contain.text', 'Automation conversation flag')
      conditionRows().first().should('contain.text', 'False')
    })
  })

  for (const value of ['1.5', '0']) {
    it(`submits decimal time conditions and numeric action value ${value}`, () => {
      cy.intercept('POST', '**/api/v1/automations/rules').as('createNumericRule')
      cy.visit(newPath)
      cy.get('input[name="name"]').type(`Numeric automation ${value} ${stamp}`)
      pickType('Time triggers')
      cy.contains('button', 'Add condition').first().click()
      pickOption('Select field', 'Hours waiting for an agent reply')
      pickOption('Select operator', /^greater than$/)
      conditionRows().first().find('input[type="number"]').type('0.5')
      cy.contains('button', 'Add action').click()
      pickOption('Select action', /^Set conversation custom attribute$/)
      actionRows().first().find('button[role="combobox"]').eq(1).click()
      cy.contains('[role="option"]', /^Automation amount$/).click()
      actionRows().first().find('input[type="number"]').type(value)
      cy.get('button[type="submit"]').click()
      cy.wait('@createNumericRule').then(({ request, response }) => {
        expect(response.statusCode).to.eq(200)
        createdRules.push(response.body.data.id)
        expect(request.body.rules[0].groups[0].rules[0].value).to.eq('0.5')
        expect(request.body.rules[0].actions[0].value).to.deep.eq([String(attributes.amount.id), value])
        openRule(response.body.data.id)
        actionRows().first().find('input[type="number"]').should('have.value', value)
      })
    })
  }

  for (const action of [
    { label: 'Unassign agent', type: 'unassign_user', values: [] },
    { label: 'Unassign team', type: 'unassign_team', values: [] },
    { label: 'Clear tags', type: 'clear_tags', values: [] },
    { label: 'Set subject', type: 'set_subject', values: ['Updated subject'] },
    { label: 'Set contact custom attribute', type: 'set_contact_custom_attribute', scope: 'contact' },
    { label: 'Remove contact custom attribute', type: 'remove_contact_custom_attribute', scope: 'contact', remove: true },
    { label: 'Set conversation custom attribute', type: 'set_conversation_custom_attribute', scope: 'conversation' },
    { label: 'Remove conversation custom attribute', type: 'remove_conversation_custom_attribute', scope: 'conversation', remove: true }
  ]) {
    it(`saves and reloads ${action.label}`, () => {
      cy.intercept('POST', '**/api/v1/automations/rules').as('createActionRule')
      cy.visit(newPath)
      cy.get('input[name="name"]').type(`${action.label} ${stamp}`)
      pickType('New conversation')
      cy.contains('button', 'Add condition').first().click()
      pickOption('Select field', 'Subject')
      pickOption('Select operator', /^equals$/)
      conditionRows().first().find('input').type(`never-match-${stamp}`)
      cy.contains('button', 'Add action').click()
      pickOption('Select action', new RegExp(`^${action.label}$`))
      if (action.scope) {
        actionRows().first().find('button[role="combobox"]').eq(1).click()
        cy.contains('[role="option"]', `Automation ${action.scope} flag`).click()
        if (!action.remove) {
          actionRows().first().find('button[role="combobox"]').eq(2).click()
          cy.contains('[role="option"]', /^False$/).click()
        }
      } else if (action.type === 'set_subject') {
        actionRows().first().find('input').type('Updated subject')
      }
      cy.get('button[type="submit"]').click()
      cy.wait('@createActionRule').then(({ request, response }) => {
        expect(response.statusCode).to.eq(200)
        createdRules.push(response.body.data.id)
        const values = action.scope
          ? [String(attributes[action.scope].id), ...(action.remove ? [] : ['false'])]
          : action.values
        expect(request.body.rules[0].actions[0]).to.deep.eq({ type: action.type, value: values })
        openRule(response.body.data.id)
        actionRows().first().should('contain.text', action.label)
        if (action.scope) actionRows().first().should('contain.text', `Automation ${action.scope} flag`)
        if (action.scope && !action.remove) actionRows().first().should('contain.text', 'False')
      })
    })
  }
})
