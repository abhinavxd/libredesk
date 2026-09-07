// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const firstName = `Cyagent${stamp}`
const lastName = 'Tester'
const renamedLastName = 'Tester edited'
const agentEmail = `cypress.agent.${stamp}@example.com`
const teamName = `Cypress Agent Team ${stamp}`
const roleName = 'Agent'
const newPath = '/admin/teams/agents/new'
const listPath = '/admin/teams/agents'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// Typing narrows the SelectTag combobox list, which is capped at 200 entries.
const pickTag = (placeholder, optionText) => {
  cy.get(`input[placeholder="${placeholder}"]`).click().type(optionText)
  cy.get('[role="option"]').contains(optionText).click()
}

const fieldBlock = (label) => cy.contains('label', label).parent()

describe('Agent form', () => {
  let teamId
  let agentId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/teams', {
      name: teamName,
      emoji: '🚀',
      conversation_assignment_type: 'Manual',
      timezone: 'UTC'
    }).then(({ body }) => {
      teamId = body.data.id
    })
  })

  after(() => {
    if (teamId) cy.api('DELETE', `/api/v1/teams/${teamId}`, null, { failOnStatusCode: false })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates an agent', () => {
    cy.intercept('POST', '**/api/v1/agents').as('createAgent')

    cy.visit(newPath)

    cy.get('input[name="first_name"]').type(firstName)
    cy.get('input[name="last_name"]').type(lastName)
    cy.get('input[name="email"]').type(agentEmail)
    pickTag('Select teams', teamName)
    pickTag('Select roles', roleName)
    cy.get('input[name="first_name"]').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@createAgent').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      agentId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(agentEmail)
    cy.contains(agentEmail).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(agentId, 'agent from the create step').to.be.a('number')

    cy.visit(`${listPath}/${agentId}/edit`)

    cy.get('input[name="first_name"]').should('have.value', firstName)
    cy.get('input[name="last_name"]').should('have.value', lastName)
    cy.get('input[name="email"]').should('have.value', agentEmail)
    fieldBlock('Teams').should('contain', teamName)
    fieldBlock('Roles').should('contain', roleName)
    cy.get('button[role="checkbox"]').should('have.attr', 'data-state', 'checked')
    cy.get('input[name="new_password"]').should('have.value', '')
  })

  it('persists a changed name and availability status', () => {
    cy.intercept('PUT', `**/api/v1/agents/${agentId}`).as('updateAgent')

    cy.visit(`${listPath}/${agentId}/edit`)
    cy.get('input[name="last_name"]').should('have.value', lastName).clear().type(renamedLastName)
    cy.get('select[name="availability_status"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Away').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateAgent').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${agentId}/edit`)
    cy.get('input[name="last_name"]').should('have.value', renamedLastName)
    cy.get('select[name="availability_status"]').should('have.value', 'away_manual')
  })

  it('rejects a submit with no name, email or role', () => {
    cy.intercept('POST', '**/api/v1/agents').as('createAgent')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    fieldBlock('Roles').should('contain', 'Required')
    cy.get('@createAgent.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a malformed email', () => {
    cy.intercept('POST', '**/api/v1/agents').as('createAgent')

    cy.visit(newPath)
    cy.get('input[name="first_name"]').type(firstName)
    cy.get('input[name="email"]').type('not-an-email')
    pickTag('Select roles', roleName)
    cy.get('input[name="first_name"]').click()
    cy.get('button[type="submit"]').click()

    cy.contains('Invalid email address').should('be.visible')
    cy.get('@createAgent.all').should('have.length', 0)
  })

  it('deletes the agent', () => {
    cy.intercept('DELETE', `**/api/v1/agents/${agentId}`).as('deleteAgent')

    cy.visit(listPath)
    filterList(agentEmail)
    cy.contains('tr', agentEmail).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteAgent').its('response.statusCode').should('eq', 200)
    cy.contains(agentEmail).should('not.exist')
  })
})
