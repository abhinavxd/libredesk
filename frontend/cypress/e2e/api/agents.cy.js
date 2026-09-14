describe('API: agents', () => {
  const stamp = Date.now()
  const email = `api.agent.${stamp}@example.com`
  let agentId

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a create with no email', () => {
    cy.api('POST', '/api/v1/agents', { first_name: 'NoEmail', roles: ['Agent'] }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/email/i)
    })
  })

  it('rejects a create with a malformed email', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Bad', email: 'not-an-email', roles: ['Agent']
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a create with no first name', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: '', email: `blank.${stamp}@example.com`, roles: ['Agent']
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a create with an unknown role', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Ghost', email: `ghost.${stamp}@example.com`, roles: ['NoSuchRole']
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an update with an unknown availability status', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Avail', email: `avail.${stamp}@example.com`, roles: ['Agent'], send_welcome_email: false
    }).then(({ body }) => {
      cy.api('PUT', `/api/v1/agents/${body.data.id}`, {
        first_name: 'Avail',
        email: `avail.${stamp}@example.com`,
        roles: ['Agent'],
        enabled: true,
        availability_status: 'not_a_status'
      }, { failOnStatusCode: false }).then((res) => {
        expect(res.status).to.eq(400)
        expect(res.body.error_type).to.eq('InputException')
      })
      cy.api('DELETE', `/api/v1/agents/${body.data.id}`)
    })
  })

  it('preserves availability status when the update omits it', () => {
    const availEmail = `keep.${stamp}@example.com`
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Keep', email: availEmail, roles: ['Agent'], send_welcome_email: false
    }).then(({ body }) => {
      const id = body.data.id
      const before = body.data.availability_status
      cy.api('PUT', `/api/v1/agents/${id}`, {
        first_name: 'Keep', email: availEmail, roles: ['Agent'], enabled: true
      }).its('status').should('eq', 200)
      cy.api('GET', `/api/v1/agents/${id}`)
        .its('body.data.availability_status')
        .should('eq', before)
      cy.api('DELETE', `/api/v1/agents/${id}`)
    })
  })

  it('creates an agent and persists every field', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Api',
      last_name: 'Agent',
      email,
      roles: ['Agent'],
      enabled: true,
      send_welcome_email: false
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      agentId = body.data.id
      expect(agentId).to.be.a('number')
      expect(body.data.email).to.eq(email)
      expect(body.data.first_name).to.eq('Api')
      expect(body.data.last_name).to.eq('Agent')
      expect(body.data.type).to.eq('agent')
      expect(body.data.enabled).to.eq(true)
      expect(body.data.roles).to.deep.eq(['Agent'])
    })
  })

  it('reads the agent back by id', () => {
    cy.api('GET', `/api/v1/agents/${agentId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.email).to.eq(email)
      expect(body.data.first_name).to.eq('Api')
    })
  })

  it('lists the agent', () => {
    cy.api('GET', '/api/v1/agents').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((a) => a.email === email), 'created agent in list').to.be.true
    })
  })

  it('rejects a duplicate email', () => {
    cy.api('POST', '/api/v1/agents', {
      first_name: 'Dupe', email, roles: ['Agent'], send_welcome_email: false
    }, { failOnStatusCode: false }).its('status').should('be.gte', 400)
  })

  it('updates the agent', () => {
    cy.api('PUT', `/api/v1/agents/${agentId}`, {
      first_name: 'Renamed',
      last_name: 'Agent',
      email,
      roles: ['Agent'],
      enabled: true
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/agents/${agentId}`)
      .its('body.data.first_name')
      .should('eq', 'Renamed')
  })

  it('404s on an agent that does not exist', () => {
    cy.api('GET', '/api/v1/agents/99999999', null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)
  })

  it('deletes the agent', () => {
    cy.api('DELETE', `/api/v1/agents/${agentId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/agents/${agentId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)
  })
})
