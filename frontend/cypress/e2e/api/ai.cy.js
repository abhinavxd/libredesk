describe('API: ai tools', () => {
  const stamp = Date.now()
  const name = `probe_tool_${stamp}`
  let toolId

  before(() => cy.login())
  beforeEach(() => cy.login())

  const validTool = (overrides = {}) => ({
    name,
    description: 'A probe tool',
    url: 'https://example.com/probe',
    method: 'GET',
    auth: { headers: [] },
    parameters: { type: 'object', properties: {} },
    enabled: true,
    ...overrides
  })

  it('rejects a create with no name', () => {
    cy.api('POST', '/api/v1/ai/tools', validTool({ name: '' }), {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a name with disallowed characters', () => {
    cy.api('POST', '/api/v1/ai/tools', validTool({ name: 'bad name!' }), {
      failOnStatusCode: false
    }).its('status').should('eq', 400)
  })

  it('rejects a name longer than 64 characters', () => {
    cy.api('POST', '/api/v1/ai/tools', validTool({ name: 'a'.repeat(65) }), {
      failOnStatusCode: false
    }).its('status').should('eq', 400)
  })

  it('rejects a url that is not http or https', () => {
    cy.api('POST', '/api/v1/ai/tools', validTool({ url: 'notaurl' }), {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a tool and persists every field', () => {
    cy.api('POST', '/api/v1/ai/tools', validTool()).then(({ status, body }) => {
      expect(status).to.eq(200)
      toolId = body.data.id
      expect(body.data.name).to.eq(name)
      expect(body.data.description).to.eq('A probe tool')
      expect(body.data.url).to.eq('https://example.com/probe')
      expect(body.data.method).to.eq('GET')
      expect(body.data.enabled).to.eq(true)
      expect(body.data.requires_verification).to.eq(false)
    })
  })

  it('reads the tool back by id', () => {
    cy.api('GET', `/api/v1/ai/tools/${toolId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.name).to.eq(name)
    })
  })

  it('lists the tool', () => {
    cy.api('GET', '/api/v1/ai/tools').then(({ body }) => {
      const rows = body.data.results || body.data
      expect(rows.some((t) => t.name === name)).to.be.true
    })
  })

  it('updates the tool', () => {
    cy.api('PUT', `/api/v1/ai/tools/${toolId}`, validTool({
      description: 'Updated description',
      requires_verification: true
    })).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/ai/tools/${toolId}`).then(({ body }) => {
      expect(body.data.description).to.eq('Updated description')
      expect(body.data.requires_verification).to.eq(true)
    })
  })

  it('deletes the tool', () => {
    cy.api('DELETE', `/api/v1/ai/tools/${toolId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/ai/tools/${toolId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)
  })
})

describe('API: ai snippets', () => {
  const stamp = Date.now()
  const title = `Probe snippet ${stamp}`
  let snippetId

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a create with a blank title', () => {
    cy.api('POST', '/api/v1/ai/snippets', { title: '', content: 'x' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a snippet', () => {
    cy.api('POST', '/api/v1/ai/snippets', {
      title,
      content: 'Snippet body text'
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      snippetId = body.data.id
      expect(body.data.title).to.eq(title)
      expect(body.data.content).to.eq('Snippet body text')
      expect(body.data.source).to.eq('manual')
    })
  })

  it('lists the snippet', () => {
    cy.api('GET', '/api/v1/ai/snippets').then(({ body }) => {
      const rows = body.data.results || body.data
      expect(rows.some((s) => s.title === title)).to.be.true
    })
  })

  it('updates the snippet', () => {
    cy.api('PUT', `/api/v1/ai/snippets/${snippetId}`, {
      title,
      content: 'Edited body text',
      enabled: true
    }).its('status').should('eq', 200)

    cy.api('GET', '/api/v1/ai/snippets').then(({ body }) => {
      const rows = body.data.results || body.data
      const found = rows.find((s) => s.id === snippetId)
      expect(found.content).to.eq('Edited body text')
    })
  })

  it('deletes the snippet', () => {
    cy.api('DELETE', `/api/v1/ai/snippets/${snippetId}`).its('status').should('eq', 200)
    cy.api('GET', '/api/v1/ai/snippets').then(({ body }) => {
      const rows = body.data.results || body.data
      expect(rows.some((s) => s.id === snippetId)).to.be.false
    })
  })
})
