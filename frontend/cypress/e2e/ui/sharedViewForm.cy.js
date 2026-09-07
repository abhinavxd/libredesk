// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const viewName = `Cypress View ${stamp}`
const renamedView = `Cypress View ${stamp} edited`
const teamName = `Cypress View Team ${stamp}`
const listPath = '/admin/conversations/shared-views'
const newPath = `${listPath}/new`

const filterList = (text) => {
  cy.get('[data-sonner-toast]', { timeout: 10000 }).should('not.exist')
  return cy.get('input[placeholder="Search"]').clear().type(text)
}

const row = (groupIndex, rowIndex) =>
  cy.get('[data-cy="filter-group"]').eq(groupIndex).find('[data-cy="filter-row"]').eq(rowIndex)

const pick = (groupIndex, rowIndex, part, optionText) => {
  row(groupIndex, rowIndex).find(`[data-cy="filter-${part}"] button[role="combobox"]`).click()
  cy.get('[role="option"]').contains(optionText).click()
}

const condition = (groupIndex, rowIndex, field, operator, value) => {
  pick(groupIndex, rowIndex, 'field', field)
  pick(groupIndex, rowIndex, 'operator', operator)
  if (value) pick(groupIndex, rowIndex, 'value', value)
}

// The picker holds one page, so a team outside it is only reachable by typing.
const searchAndPickTeam = (name) => {
  cy.contains('label', 'Team').parent().find('button[role="combobox"]').click()
  cy.get('[data-radix-popper-content-wrapper] input').first().type(name)
  cy.get('[role="option"]').contains(name).click()
}

describe('Shared view form', () => {
  let viewId
  let teamId

  // radix-vue focuses the operator trigger it just unmounted, and Cypress fails on uncaught errors.
  Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/teams', {
      name: teamName,
      emoji: '🔭',
      conversation_assignment_type: 'Round robin',
      timezone: 'UTC',
      max_auto_assigned_conversations: 0
    }).then(({ body }) => {
      teamId = body.data.id
    })
  })

  beforeEach(() => {
    cy.viewport(1400, 1000)
    cy.login()
  })

  it('starts with one group holding one blank condition', () => {
    cy.visit(newPath)

    cy.get('[data-cy="filter-group"]').should('have.length', 1)
    cy.get('[data-cy="filter-row"]').should('have.length', 1)
    cy.contains('Match these rules').should('exist')
    cy.contains('button', 'ALL').should('exist')
    cy.contains('button', 'Add condition').should('exist')
    cy.contains('button', 'Add group').should('exist')
    cy.get('[aria-label="Remove group"]').should('not.exist')
  })

  it('shows the operator select only after a field is chosen', () => {
    cy.visit(newPath)

    row(0, 0).find('[data-cy="filter-operator"] button[role="combobox"]').should('not.exist')
    pick(0, 0, 'field', /^Status$/)
    row(0, 0).find('[data-cy="filter-operator"] button[role="combobox"]').should('exist')
  })

  it('rejects a submit with no name and no filter', () => {
    cy.intercept('POST', '**/api/v1/shared-views').as('createView')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.get('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createView.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('refuses to save a condition with a field but no operator', () => {
    cy.intercept('POST', '**/api/v1/shared-views').as('createView')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(viewName)
    pick(0, 0, 'field', /^Status$/)

    cy.get('button[type="submit"]').click()

    cy.get('@createView.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('refuses to save a condition with no value', () => {
    cy.intercept('POST', '**/api/v1/shared-views').as('createView')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(viewName)
    pick(0, 0, 'field', /^Status$/)
    pick(0, 0, 'operator', /^equals$/)

    cy.get('button[type="submit"]').click()

    cy.get('@createView.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('creates a view and sends a two-level filter tree', () => {
    cy.intercept('POST', '**/api/v1/shared-views').as('createView')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(viewName)
    condition(0, 0, /^Status$/, /^equals$/, 'Open')

    cy.get('button[type="submit"]').click()

    cy.wait('@createView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      viewId = response.body.data.id

      const filters = request.body.filters
      expect(filters, 'filters is an object, not a legacy array').to.be.an('object')
      expect(filters).to.not.be.an('array')
      expect(filters.logic).to.eq('AND')
      expect(filters.rules).to.have.length(1)

      const group = filters.rules[0]
      expect(group.logic).to.eq('AND')
      expect(group.rules).to.have.length(1)
      expect(group.rules[0]).to.deep.include({
        model: 'conversations',
        field: 'status_id',
        operator: 'equals'
      })
      expect(JSON.stringify(filters), 'no UI-only keys on the wire').to.not.contain('__id')
    })

    cy.location('pathname').should('eq', listPath)
    filterList(viewName)
    cy.contains(viewName).should('exist')
  })

  it('loads the saved filter back into the builder', () => {
    expect(viewId, 'view from the create step').to.be.a('number')

    cy.visit(`${listPath}/${viewId}/edit`)

    cy.get('input[name="name"]').should('have.value', viewName)
    cy.get('[data-cy="filter-group"]').should('have.length', 1)
    cy.get('[data-cy="filter-row"]').should('have.length', 1)
    row(0, 0).find('[data-cy="filter-field"]').should('contain.text', 'Status')
    row(0, 0).find('[data-cy="filter-operator"]').should('contain.text', 'equals')
    row(0, 0).find('[data-cy="filter-value"]').should('contain.text', 'Open')
  })

  it('saves an untouched edit byte for byte', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.api('GET', `/api/v1/shared-views/${viewId}`).then(({ body }) => {
      const loaded = body.data.filters

      cy.visit(`${listPath}/${viewId}/edit`)
      cy.get('[data-cy="filter-row"]').should('have.length', 1)
      cy.get('button[type="submit"]').click()

      cy.wait('@updateView').then(({ request }) => {
        expect(request.body.filters).to.deep.eq(loaded)
      })
    })
  })

  it('adds a second condition and keeps both in one group', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('[data-cy="filter-row"]').should('have.length', 1)
    cy.contains('button', 'Add condition').click()
    condition(0, 1, /^Priority$/, /^equals$/, 'High')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      const filters = request.body.filters
      expect(filters.rules, 'still one group').to.have.length(1)
      expect(filters.rules[0].rules).to.have.length(2)
      expect(filters.rules[0].rules.map((r) => r.field)).to.deep.eq(['status_id', 'priority_id'])
    })
  })

  it('switches a group to ANY and persists OR', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('[data-cy="filter-row"]').should('have.length', 2)
    cy.contains('button', 'ALL').click()
    cy.contains('button', 'ANY').should('exist')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request }) => {
      expect(request.body.filters.rules[0].logic).to.eq('OR')
    })

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.contains('button', 'ANY').should('exist')
  })

  it('adds a second group and keeps the groups separate', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('[data-cy="filter-group"]').should('have.length', 1)
    cy.contains('button', 'Add group').click()
    cy.get('[data-cy="filter-group"]').should('have.length', 2)
    condition(1, 0, /^Assign team$/, /^set$/)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      const filters = request.body.filters
      expect(filters.rules, 'two groups').to.have.length(2)
      expect(filters.rules[0].rules.map((r) => r.field)).to.deep.eq(['status_id', 'priority_id'])
      expect(filters.rules[0].logic, 'first group keeps its ANY').to.eq('OR')
      expect(filters.rules[1].rules).to.have.length(1)
      expect(filters.rules[1].rules[0]).to.deep.include({
        field: 'assigned_team_id',
        operator: 'set'
      })
    })

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('[data-cy="filter-group"]').should('have.length', 2)
    cy.get('[aria-label="Remove group"]').should('have.length', 2)
  })

  it('reloads two groups and re-saves them identically', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.api('GET', `/api/v1/shared-views/${viewId}`).then(({ body }) => {
      const loaded = body.data.filters

      cy.visit(`${listPath}/${viewId}/edit`)
      cy.get('[data-cy="filter-group"]').should('have.length', 2)
      cy.get('button[type="submit"]').click()

      cy.wait('@updateView').then(({ request }) => {
        expect(request.body.filters, 'round trip is stable').to.deep.eq(loaded)
      })
    })
  })

  it('removes a group and drops only that group', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('[data-cy="filter-group"]').should('have.length', 2)
    cy.get('[aria-label="Remove group"]').last().click()
    cy.get('[data-cy="filter-group"]').should('have.length', 1)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request }) => {
      const filters = request.body.filters
      expect(filters.rules).to.have.length(1)
      expect(filters.rules[0].rules.map((r) => r.field)).to.deep.eq(['status_id', 'priority_id'])
    })
  })

  it('persists a rename and a team-scoped visibility', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('input[name="name"]').should('have.value', viewName).clear().type(renamedView)

    cy.get('select[name="visibility"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Team').click()
    searchAndPickTeam(teamName)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.team_id, 'team id goes out as a number').to.be.a('number')
    })

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedView)
    cy.get('select[name="visibility"]').should('have.value', 'team')
    cy.contains('label', 'Team')
      .parent()
      .find('button[role="combobox"]')
      .should('contain.text', teamName)
  })

  it('resolves the saved team by id even though it sits outside the first page', () => {
    cy.intercept(
      { method: 'GET', url: '**/api/v1/teams/compact*', query: { page: '1' } },
      { body: { data: [] } }
    )
    cy.intercept('GET', '**/api/v1/teams/compact?*ids=*').as('teamById')

    cy.visit(`${listPath}/${viewId}/edit`)

    cy.wait('@teamById').its('response.statusCode').should('eq', 200)
    cy.contains('label', 'Team')
      .parent()
      .find('button[role="combobox"]')
      .should('contain.text', teamName)
  })

  it('searches the team picker on the server instead of holding every team', () => {
    cy.intercept('GET', '**/api/v1/teams/compact?*q=*').as('teamLookup')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.contains('label', 'Team').parent().find('button[role="combobox"]').click()
    cy.get('[data-radix-popper-content-wrapper] input').first().type(teamName)

    cy.wait('@teamLookup').its('request.url').should('include', 'q=')
    cy.get('[role="option"]').contains(teamName).should('exist')
  })

  it('clears team_id when visibility goes back to all agents', () => {
    cy.intercept('PUT', `**/api/v1/shared-views/${viewId}`).as('updateView')

    cy.visit(`${listPath}/${viewId}/edit`)
    cy.get('select[name="visibility"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('All agents').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateView').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.team_id).to.eq(null)
    })
  })

  it('deletes the view', () => {
    cy.intercept('DELETE', `**/api/v1/shared-views/${viewId}`).as('deleteView')

    cy.visit(listPath)
    filterList(renamedView)
    cy.contains('tr', renamedView).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteView').its('response.statusCode').should('eq', 200)
    cy.contains(renamedView).should('not.exist')
  })

  after(() => {
    cy.login()
    if (teamId) cy.api('DELETE', `/api/v1/teams/${teamId}`)
  })
})
