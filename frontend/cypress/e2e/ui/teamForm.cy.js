// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const teamName = `Cypress Team ${stamp}`
const renamedTeam = `Cypress Team ${stamp} edited`
const businessHoursName = `Cypress BH ${stamp}`
const newPath = '/admin/teams/teams/new'
const listPath = '/admin/teams/teams'

// A radix Select renders a hidden native select, so the field name drives it without labels.
const pickOption = (field, optionText) => {
  cy.get(`select[name="${field}"]`).siblings('button[role="combobox"]').click()
  cy.get('[role="option"]').contains(optionText).click()
}

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

describe('Team form', () => {
  let businessHoursId
  let teamId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/business-hours', {
      name: businessHoursName,
      description: 'Business hours for the team form spec',
      is_always_open: true,
      hours: {},
      holidays: []
    }).then(({ body }) => {
      businessHoursId = body.data.id
    })
  })

  // Teardown never asserts: a failed cleanup must not mask the real failure.
  after(() => {
    if (!businessHoursId) return
    cy.login()
    cy.api('DELETE', `/api/v1/business-hours/${businessHoursId}`, null, {
      failOnStatusCode: false
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a team', () => {
    cy.intercept('POST', '**/api/v1/teams').as('createTeam')

    cy.visit(newPath)

    // Pick the required emoji first so the next click closes the picker overlay.
    cy.get('input[name="emoji"]').click()
    cy.get('.v3-emoji-picker .v3-emojis button').first().click()

    cy.get('input[name="name"]').type(teamName)
    pickOption('conversation_assignment_type', 'Round robin')
    cy.get('input[name="max_auto_assigned_conversations"]').clear().type('7')
    pickOption('timezone', 'UTC (UTC+00:00)')
    pickOption('business_hours_id', businessHoursName)

    cy.get('button[type="submit"]').click()
    cy.wait('@createTeam').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      teamId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(teamName)
    cy.contains(teamName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(teamId, 'team from the create step').to.be.a('number')

    cy.visit(`${listPath}/${teamId}/edit`)

    cy.get('input[name="name"]').should('have.value', teamName)
    cy.get('input[name="max_auto_assigned_conversations"]').should('have.value', '7')
    cy.get('input[name="emoji"]').should('not.have.value', '')
    cy.get('select[name="conversation_assignment_type"]').should('have.value', 'Round robin')
    cy.get('select[name="timezone"]').should('have.value', 'UTC')
    cy.get('select[name="business_hours_id"]').should('have.value', String(businessHoursId))
  })

  it('persists a changed field', () => {
    cy.intercept('PUT', `**/api/v1/teams/${teamId}`).as('updateTeam')

    cy.visit(`${listPath}/${teamId}/edit`)
    cy.get('input[name="name"]').should('have.value', teamName).clear().type(renamedTeam)
    cy.get('button[type="submit"]').click()
    cy.wait('@updateTeam').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${teamId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedTeam)
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/teams').as('createTeam')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createTeam.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the team', () => {
    cy.intercept('DELETE', `**/api/v1/teams/${teamId}`).as('deleteTeam')

    cy.visit(listPath)
    filterList(renamedTeam)
    cy.contains('tr', renamedTeam).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteTeam').its('response.statusCode').should('eq', 200)
    cy.contains(renamedTeam).should('not.exist')
  })
})
