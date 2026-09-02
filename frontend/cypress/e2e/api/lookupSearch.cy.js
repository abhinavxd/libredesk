// Pins the q and ids query params on all three lookup endpoints.

const stamp = Date.now()

const rowsOf = (body) => body.data.results || body.data

describe('API: lookup search and resolve by id', () => {
  const tagAlpha = `zz-lookup-alpha-${stamp}`
  const tagBeta = `zz-lookup-beta-${stamp}`
  const teamAlpha = `zz-lookup-team-alpha-${stamp}`
  const teamBeta = `zz-lookup-team-beta-${stamp}`
  const agentEmail = `zz-lookup-agent-${stamp}@example.com`
  const agentFirst = `Zzlookup${stamp}`

  const created = { tags: [], teams: [], agents: [] }

  before(() => {
    cy.login()
    for (const name of [tagAlpha, tagBeta]) {
      cy.api('POST', '/api/v1/tags', { name }).then(({ body }) => created.tags.push(body.data.id))
    }
    for (const name of [teamAlpha, teamBeta]) {
      cy.api('POST', '/api/v1/teams', {
        name,
        emoji: '🔎',
        conversation_assignment_type: 'Round robin',
        timezone: 'UTC',
        max_auto_assigned_conversations: 0
      }).then(({ body }) => created.teams.push(body.data.id))
    }
    cy.api('POST', '/api/v1/agents', {
      first_name: agentFirst,
      last_name: 'Searchable',
      email: agentEmail,
      roles: ['Agent'],
      send_welcome_email: false
    }).then(({ body }) => created.agents.push(body.data.id))
  })

  beforeEach(() => cy.login())

  describe('tags', () => {
    it('returns every tag when no page params are sent', () => {
      cy.api('GET', '/api/v1/tags').then(({ status, body }) => {
        expect(status).to.eq(200)
        const rows = rowsOf(body)
        expect(rows).to.be.an('array')
        for (const id of created.tags) {
          expect(rows.find((t) => t.id === id), `tag ${id}`).to.exist
        }
      })
    })

    it('narrows the list with q', () => {
      cy.api('GET', `/api/v1/tags?q=zz-lookup-alpha-${stamp}`).then(({ status, body }) => {
        expect(status).to.eq(200)
        const rows = rowsOf(body)
        expect(rows).to.have.length(1)
        expect(rows[0].name).to.eq(tagAlpha)
      })
    })

    it('matches q case insensitively and mid-name', () => {
      cy.api('GET', `/api/v1/tags?q=LOOKUP-BETA-${stamp}`).then(({ body }) => {
        const rows = rowsOf(body)
        expect(rows.map((t) => t.name)).to.include(tagBeta)
      })
    })

    it('returns nothing for a q that matches nothing', () => {
      cy.api('GET', '/api/v1/tags?q=definitely-no-such-tag-anywhere').then(({ body }) => {
        expect(rowsOf(body)).to.have.length(0)
      })
    })

    it('honours page and page_size alongside q', () => {
      cy.api('GET', `/api/v1/tags?q=zz-lookup-&page=1&page_size=1`).then(({ body }) => {
        expect(rowsOf(body)).to.have.length(1)
      })
    })

    it('resolves exactly the requested ids', () => {
      cy.api('GET', `/api/v1/tags?ids=${created.tags.join(',')}`).then(({ status, body }) => {
        expect(status).to.eq(200)
        const rows = rowsOf(body)
        expect(rows).to.have.length(created.tags.length)
        expect(rows.map((t) => t.id).sort()).to.deep.eq([...created.tags].sort())
      })
    })

    it('ignores ids that do not exist rather than erroring', () => {
      cy.api('GET', `/api/v1/tags?ids=${created.tags[0]},99999999`).then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(rowsOf(body)).to.have.length(1)
      })
    })

    it('falls back to the list when every id is junk', () => {
      cy.api('GET', '/api/v1/tags?ids=abc,-1,0&page_size=3').then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(rowsOf(body).length).to.be.at.most(3)
        expect(rowsOf(body).length).to.be.at.least(1)
      })
    })

    it('lets ids win over q so a selected value always resolves', () => {
      cy.api('GET', `/api/v1/tags?ids=${created.tags[0]}&q=definitely-no-such-tag-anywhere`).then(
        ({ body }) => {
          expect(rowsOf(body)).to.have.length(1)
          expect(rowsOf(body)[0].id).to.eq(created.tags[0])
        }
      )
    })
  })

  describe('teams', () => {
    it('returns every team when no page params are sent', () => {
      cy.api('GET', '/api/v1/teams/compact').then(({ status, body }) => {
        expect(status).to.eq(200)
        for (const id of created.teams) {
          expect(rowsOf(body).find((t) => t.id === id), `team ${id}`).to.exist
        }
      })
    })

    it('keeps the compact shape of id, name and emoji', () => {
      cy.api('GET', `/api/v1/teams/compact?ids=${created.teams[0]}`).then(({ body }) => {
        const row = rowsOf(body)[0]
        expect(Object.keys(row).sort()).to.deep.eq(['emoji', 'id', 'name'])
      })
    })

    it('narrows the list with q', () => {
      cy.api('GET', `/api/v1/teams/compact?q=zz-lookup-team-alpha-${stamp}`).then(({ body }) => {
        const rows = rowsOf(body)
        expect(rows).to.have.length(1)
        expect(rows[0].name).to.eq(teamAlpha)
      })
    })

    it('returns nothing for a q that matches nothing', () => {
      cy.api('GET', '/api/v1/teams/compact?q=definitely-no-such-team-anywhere').then(({ body }) => {
        expect(rowsOf(body)).to.have.length(0)
      })
    })

    it('resolves exactly the requested ids', () => {
      cy.api('GET', `/api/v1/teams/compact?ids=${created.teams.join(',')}`).then(({ body }) => {
        const rows = rowsOf(body)
        expect(rows.map((t) => t.id).sort()).to.deep.eq([...created.teams].sort())
      })
    })
  })

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
    for (const id of created.tags) {
      cy.api('DELETE', `/api/v1/tags/${id}`, null, { failOnStatusCode: false })
    }
    for (const id of created.teams) {
      cy.api('DELETE', `/api/v1/teams/${id}`, null, { failOnStatusCode: false })
    }
    for (const id of created.agents) {
      cy.api('DELETE', `/api/v1/agents/${id}`, null, { failOnStatusCode: false })
    }
  })
})
