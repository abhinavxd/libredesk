// Agent lookup remains part of mailbox collaboration; tags, teams and Views are retired.
const stamp = Date.now()
const rowsOf = (body) => body.data.results || body.data

describe('API: retained agent lookup and retired directories', () => {
  const agentEmail = `zz-lookup-agent-${stamp}@example.com`
  const agentFirst = `Zzlookup${stamp}`
  const created = { agents: [] }
  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/agents', {
      first_name: agentFirst, last_name: 'Searchable', email: agentEmail,
      roles: ['Agent'], send_welcome_email: false
    }).then(({ body }) => created.agents.push(body.data.id))
  })
  beforeEach(() => cy.login())

  for (const path of ['/api/v1/tags', '/api/v1/teams', '/api/v1/teams/compact', '/api/v1/views']) {
    it(`keeps ${path} unavailable`, () => {
      cy.api('GET', path, null, { failOnStatusCode: false }).its('status').should('eq', 404)
      cy.api('POST', path, { name: 'Retired' }, { failOnStatusCode: false }).its('status').should('eq', 404)
    })
  }

  describe('agents', () => {
    it('returns every agent when no page params are sent', () => {
      cy.api('GET', '/api/v1/agents/compact').then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(rowsOf(body).find((a) => a.id === created.agents[0]), 'created agent').to.exist
      })
    })

    it('finds an agent by first name, by last name and by email', () => {
      for (const q of [agentFirst, 'Searchable', agentEmail]) {
        cy.api('GET', `/api/v1/agents/compact?q=${encodeURIComponent(q)}`).then(({ body }) => {
          expect(
            rowsOf(body).find((a) => a.id === created.agents[0]),
            `agent matched by ${q}`
          ).to.exist
        })
      }
    })

    it('finds an agent by a full name spanning first and last', () => {
      cy.api('GET', `/api/v1/agents/compact?q=${encodeURIComponent(`${agentFirst} Search`)}`).then(
        ({ body }) => {
          expect(rowsOf(body).find((a) => a.id === created.agents[0])).to.exist
        }
      )
    })

    it('returns nothing for a q that matches nothing', () => {
      cy.api('GET', '/api/v1/agents/compact?q=definitely-no-such-agent-anywhere').then(({ body }) => {
        expect(rowsOf(body)).to.have.length(0)
      })
    })

    it('caps page_size at the server maximum', () => {
      cy.api('GET', '/api/v1/agents/compact?page=1&page_size=100000').then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(rowsOf(body).length).to.be.at.most(500)
      })
    })

    it('resolves the requested id', () => {
      cy.api('GET', `/api/v1/agents/compact?ids=${created.agents[0]}`).then(({ body }) => {
        const rows = rowsOf(body)
        expect(rows).to.have.length(1)
        expect(rows[0].id).to.eq(created.agents[0])
      })
    })

  })

  after(() => {
    cy.login()
    for (const id of created.agents) cy.api('DELETE', `/api/v1/agents/${id}`)
  })
})
