describe('Retired administration stays retired', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  for (const path of ['/admin/teams/agents', '/admin/teams/agents/new', '/admin/teams', '/admin/views', '/admin/contacts']) {
    it(`redirects the obsolete bookmark ${path} to Addresses`, () => {
      cy.visit(path)
      cy.location('pathname').should('match', /^\/addresses(?:\/|$)/)
      cy.get('input[name="first_name"]').should('not.exist')
      cy.contains('button', 'New view').should('not.exist')
      cy.contains('button', 'New team').should('not.exist')
    })
  }

  it('retains account identity and user API access', () => {
    cy.api('GET', '/api/v1/agents/me').its('body.data.email').should('eq', 'System')
    cy.api('GET', '/api/v1/agents/compact').its('body.data').should('be.an', 'array')
  })
})
