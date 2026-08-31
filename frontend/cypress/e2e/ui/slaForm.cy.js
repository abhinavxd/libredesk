// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const policyName = `Cypress SLA ${stamp}`
const renamedPolicy = `Cypress SLA ${stamp} edited`
const newPath = '/admin/sla/new'
const listPath = '/admin/sla'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

describe('SLA form', () => {
  let policyId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates an SLA policy', () => {
    cy.intercept('POST', '**/api/v1/sla').as('createSla')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(policyName)
    cy.get('input[name="description"]').type('Created by the SLA form spec')
    cy.get('input[name="first_response_time"]').type('2h')
    cy.get('input[name="resolution_time"]').type('8h')
    cy.get('input[name="next_response_time"]').type('45m')

    cy.get('button[type="submit"]').click()
    cy.wait('@createSla').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      policyId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(policyName)
    cy.contains(policyName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(policyId, 'policy from the create step').to.be.a('number')

    cy.visit(`${listPath}/${policyId}/edit`)

    cy.get('input[name="name"]').should('have.value', policyName)
    cy.get('input[name="description"]').should('have.value', 'Created by the SLA form spec')
    cy.get('input[name="first_response_time"]').should('have.value', '2h')
    cy.get('input[name="resolution_time"]').should('have.value', '8h')
    cy.get('input[name="next_response_time"]').should('have.value', '45m')
  })

  it('persists a changed field', () => {
    cy.intercept('PUT', `**/api/v1/sla/${policyId}`).as('updateSla')

    cy.visit(`${listPath}/${policyId}/edit`)
    cy.get('input[name="name"]').should('have.value', policyName).clear().type(renamedPolicy)
    cy.get('input[name="resolution_time"]').clear().type('12h')
    cy.get('button[type="submit"]').click()
    cy.wait('@updateSla').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${policyId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedPolicy)
    cy.get('input[name="resolution_time"]').should('have.value', '12h')
  })

  it('rejects a submit with no name and no SLA time', () => {
    cy.intercept('POST', '**/api/v1/sla').as('createSla')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains('SLA Policy name should be between 1 and 255 characters').should('be.visible')
    cy.contains(/at least one of first response time/i).should('be.visible')
    cy.get('@createSla.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the SLA policy', () => {
    cy.intercept('DELETE', `**/api/v1/sla/${policyId}`).as('deleteSla')

    cy.visit(listPath)
    filterList(renamedPolicy)
    cy.contains('tr', renamedPolicy).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteSla').its('response.statusCode').should('eq', 200)
    cy.contains(renamedPolicy).should('not.exist')
  })
})
