describe('API: reply-only workspace', () => {
  beforeEach(() => cy.login())

  it('does not expose an agent-created conversation endpoint', () => {
    cy.api(
      'POST',
      '/api/v1/conversations',
      {
        inbox_id: 1,
        contact_email: 'not-supported@example.test',
        first_name: 'Not supported',
        subject: 'No outbound conversation',
        content: '<p>This must not be accepted.</p>'
      },
      { failOnStatusCode: false }
    ).its('status').should('eq', 404)
  })
})
