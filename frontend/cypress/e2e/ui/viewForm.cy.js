// The steps run in order and share the view created by the first one.

const stamp = Date.now()
const viewName = `Cypress user view ${stamp}`
const renamedView = `Cypress user view ${stamp} edited`

const dialog = () => cy.get('[role="dialog"]')

const row = (groupIndex, rowIndex) =>
  dialog()
    .find('[data-cy="filter-group"]')
    .eq(groupIndex)
    .find('[data-cy="filter-row"]')
    .eq(rowIndex)

const pick = (groupIndex, rowIndex, part, optionText) => {
  row(groupIndex, rowIndex).find(`[data-cy="filter-${part}"] button[role="combobox"]`).click()
  cy.get('[role="option"]').contains(optionText).click()
}

const condition = (groupIndex, rowIndex, field, operator, value) => {
  pick(groupIndex, rowIndex, 'field', field)
  pick(groupIndex, rowIndex, 'operator', operator)
  if (value) pick(groupIndex, rowIndex, 'value', value)
}

const awaitFocusedName = () => {
  dialog().should('be.visible')
  dialog().find('input[name="name"]').should('be.focused')
}

const openCreateDialog = () => {
  cy.visit('/inboxes/assigned')
  cy.contains('My inbox').should('be.visible')
  // The add button is a bare icon that only appears on hover, so reach it through the section label.
  cy.contains('.sidebar-section-label', /^Views$/)
    .siblings('div')
    .find('svg')
    .click({ force: true })
  awaitFocusedName()
}

const openEditDialog = (name) => {
  cy.visit('/inboxes/assigned')
  cy.contains('button', name).parent().find('button[aria-haspopup="menu"]').click()
  cy.get('[role="menuitem"]').contains('Edit').click()
  awaitFocusedName()
}

// The dialog re-renders the name field just after opening and drops whatever was typed until then.
const setName = (value, attempt = 0) => {
  dialog().find('input[name="name"]').type(`{selectall}${value}`)
  dialog()
    .find('input[name="name"]')
    .then(($el) => {
      if ($el.val() !== value && attempt < 5) setName(value, attempt + 1)
    })
}

const save = () => dialog().find('button[type="submit"]').click()

describe('User view form', () => {
  let viewId

  // radix-vue focuses the operator trigger it just unmounted, and Cypress fails on uncaught errors.
  Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))

  beforeEach(() => {
    cy.viewport(1400, 1000)
    cy.login()
  })

  after(() => {
    cy.login()
    cy.then(() => {
      if (viewId) cy.api('DELETE', `/api/v1/views/me/${viewId}`, null, { failOnStatusCode: false })
    })
  })

  it('starts with one group holding one blank condition', () => {
    openCreateDialog()

    dialog().find('[data-cy="filter-group"]').should('have.length', 1)
    dialog().find('[data-cy="filter-row"]').should('have.length', 1)
    dialog().contains('Match these rules').should('exist')
    dialog().contains('button', 'ALL').should('exist')
    dialog().contains('button', 'Add condition').should('exist')
    dialog().contains('button', 'Add group').should('exist')
    dialog().find('[aria-label="Remove group"]').should('not.exist')
  })

  it('rejects a submit with no name and no filter', () => {
    cy.intercept('POST', '**/api/v1/views/me').as('createView')

    openCreateDialog()
    save()

    dialog().find('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createView.all').should('have.length', 0)
    dialog().should('be.visible')
  })

  it('refuses to save a condition with a field but no operator', () => {
    cy.intercept('POST', '**/api/v1/views/me').as('createView')

    openCreateDialog()
    setName(viewName)
    pick(0, 0, 'field', /^Status$/)
    save()

    cy.get('@createView.all').should('have.length', 0)
    dialog().should('be.visible')
  })

  it('creates a view from one group holding one condition', () => {
    cy.intercept('POST', '**/api/v1/views/me').as('createView')

    openCreateDialog()
    setName(viewName)
    condition(0, 0, /^Status$/, /^equals$/, 'Open')
    save()

    cy.wait('@createView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      viewId = response.body.data.id

      const filters = request.body.filters
      expect(filters, 'filters is an object, not a legacy array').to.be.an('object')
      expect(filters.logic).to.eq('AND')
      expect(filters.rules).to.have.length(1)
      expect(filters.rules[0].logic).to.eq('AND')
      expect(filters.rules[0].rules).to.have.length(1)
      expect(filters.rules[0].rules[0]).to.deep.include({
        model: 'conversations',
        field: 'status_id',
        operator: 'equals'
      })
      expect(JSON.stringify(filters), 'no UI-only keys on the wire').to.not.contain('__id')
    })

    dialog().should('not.exist')
    cy.contains('button', viewName).should('exist')
  })

  it('loads the saved filter back into the builder', () => {
    openEditDialog(viewName)

    dialog().find('input[name="name"]').should('have.value', viewName)
    dialog().find('[data-cy="filter-group"]').should('have.length', 1)
    dialog().find('[data-cy="filter-row"]').should('have.length', 1)
    row(0, 0).find('[data-cy="filter-field"]').should('contain.text', 'Status')
    row(0, 0).find('[data-cy="filter-operator"]').should('contain.text', 'equals')
    row(0, 0).find('[data-cy="filter-value"]').should('contain.text', 'Open')
  })

  it('holds three conditions in one group joined by ALL', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    dialog().contains('button', 'Add condition').click()
    condition(0, 1, /^Priority$/, /^equals$/, 'High')
    dialog().contains('button', 'Add condition').click()
    condition(0, 2, /^Assign team$/, /^set$/)
    save()

    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      const filters = request.body.filters
      expect(filters.rules, 'still one group').to.have.length(1)
      expect(filters.rules[0].logic).to.eq('AND')
      expect(filters.rules[0].rules.map((r) => r.field)).to.deep.eq([
        'status_id',
        'priority_id',
        'assigned_team_id'
      ])
    })
  })

  it('switches that group to ANY and persists OR', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    dialog().find('[data-cy="filter-row"]').should('have.length', 3)
    dialog().contains('button', 'ALL').click()
    dialog().contains('button', 'ANY').should('exist')
    save()

    cy.wait('@updateView').then(({ request }) => {
      expect(request.body.filters.rules[0].logic).to.eq('OR')
    })

    openEditDialog(viewName)
    dialog().contains('button', 'ANY').should('exist')
  })

  it('adds a second group and keeps each group its own logic', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    dialog().contains('button', 'Add group').click()
    dialog().find('[data-cy="filter-group"]').should('have.length', 2)
    condition(1, 0, /^Priority$/, /^equals$/, 'Low')
    dialog().find('[data-cy="filter-group"]').eq(1).contains('button', 'Add condition').click()
    condition(1, 1, /^Status$/, /^equals$/, 'Closed')
    save()

    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      const filters = request.body.filters
      expect(filters.rules, 'two groups').to.have.length(2)
      expect(filters.logic, 'groups joined by AND until toggled').to.eq('AND')
      expect(filters.rules[0].logic, 'first group keeps its ANY').to.eq('OR')
      expect(filters.rules[0].rules).to.have.length(3)
      expect(filters.rules[1].logic).to.eq('AND')
      expect(filters.rules[1].rules.map((r) => r.field)).to.deep.eq(['priority_id', 'status_id'])
    })
  })

  it('switches the connector between the two groups to OR', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    dialog().find('[data-cy="filter-group"]').should('have.length', 2)
    dialog().contains('button', 'AND').click()
    dialog().contains('button', 'OR').should('exist')
    save()

    cy.wait('@updateView').then(({ request }) => {
      const filters = request.body.filters
      expect(filters.logic, 'groups now joined by OR').to.eq('OR')
      expect(filters.rules[0].logic, 'inner logic untouched').to.eq('OR')
      expect(filters.rules[1].logic).to.eq('AND')
    })

    openEditDialog(viewName)
    dialog().contains('button', 'OR').should('exist')
  })

  it('re-saves a two-group tree byte for byte', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    cy.api('GET', '/api/v1/views/me').then(({ body }) => {
      const loaded = body.data.find((v) => v.id === viewId).filters

      openEditDialog(viewName)
      dialog().find('[data-cy="filter-group"]').should('have.length', 2)
      save()

      cy.wait('@updateView').then(({ request }) => {
        expect(request.body.filters, 'round trip is stable').to.deep.eq(loaded)
      })
    })
  })

  it('removes a group and drops only that group', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    dialog().find('[aria-label="Remove group"]').should('have.length', 2).last().click()
    dialog().find('[data-cy="filter-group"]').should('have.length', 1)
    save()

    cy.wait('@updateView').then(({ request }) => {
      const filters = request.body.filters
      expect(filters.rules).to.have.length(1)
      expect(filters.rules[0].rules.map((r) => r.field)).to.deep.eq([
        'status_id',
        'priority_id',
        'assigned_team_id'
      ])
    })
  })

  it('persists a rename', () => {
    cy.intercept('PUT', `**/api/v1/views/me/${viewId}`).as('updateView')

    openEditDialog(viewName)
    setName(renamedView)
    dialog().find('input[name="name"]').should('have.value', renamedView)
    save()

    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.name, 'name on the wire').to.eq(renamedView)
    })
    cy.api('GET', '/api/v1/views/me').then(({ body }) => {
      expect(body.data.find((v) => v.id === viewId).name, 'stored name').to.eq(renamedView)
    })
    cy.contains('button', renamedView).should('exist')
  })

  it('deletes the view from the sidebar', () => {
    cy.intercept('DELETE', `**/api/v1/views/me/${viewId}`).as('deleteView')

    cy.visit('/inboxes/assigned')
    cy.contains('button', renamedView).parent().find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteView').its('response.statusCode').should('eq', 200)
    cy.contains('button', renamedView).should('not.exist')
    cy.then(() => {
      viewId = null
    })
  })
})
