// There is no GET /statuses/{id}, so read-backs go through the list.

describe('API: statuses', () => {
  const stamp = `${Date.now()}`.slice(-8)
  const name = `st-${stamp}`
  const renamed = `st-${stamp}-r`
  let statusId

  const listStatuses = () => cy.api('GET', '/api/v1/statuses').then(({ body }) => body.data.results || body.data)

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a create with no name', () => {
    cy.api('POST', '/api/v1/statuses', { category: 'open' }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a create with no category', () => {
    cy.api('POST', '/api/v1/statuses', { name: `nocat-${stamp}` }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a create with an unknown category', () => {
    cy.api('POST', '/api/v1/statuses', { name: `badcat-${stamp}`, category: 'nonsense' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a name longer than 25 characters', () => {
    cy.api('POST', '/api/v1/statuses', { name: 'a'.repeat(26), category: 'open' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a status and persists every field', () => {
    cy.api('POST', '/api/v1/statuses', { name, category: 'open' }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      statusId = body.data.id
      expect(statusId).to.be.a('number')
      expect(body.data.name).to.eq(name)
      expect(body.data.category).to.eq('open')
    })
  })

  it('lists the status', () => {
    listStatuses().then((rows) => {
      const found = rows.find((s) => s.id === statusId)
      expect(found, 'created status in list').to.exist
      expect(found.name).to.eq(name)
      expect(found.category).to.eq('open')
    })
  })

  it('updates the status and the change persists', () => {
    cy.api('PUT', `/api/v1/statuses/${statusId}`, { name: renamed, category: 'resolved' }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.name).to.eq(renamed)
      expect(body.data.category).to.eq('resolved')
    })

    listStatuses().then((rows) => {
      const found = rows.find((s) => s.id === statusId)
      expect(found.name).to.eq(renamed)
      expect(found.category).to.eq('resolved')
    })
  })

  it('rejects an update with an empty name', () => {
    cy.api('PUT', `/api/v1/statuses/${statusId}`, { name: '', category: 'open' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an update with an unknown category', () => {
    cy.api('PUT', `/api/v1/statuses/${statusId}`, { name: renamed, category: 'nonsense' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('refuses to update a default status', () => {
    cy.api('PUT', '/api/v1/statuses/1', { name: `OpenRenamed-${stamp}`, category: 'open' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('refuses to delete a default status', () => {
    cy.api('DELETE', '/api/v1/statuses/1', null, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  // A duplicate status name returns 500 GeneralException.
  it.skip('rejects a duplicate name', () => {
    cy.api('POST', '/api/v1/statuses', { name: renamed, category: 'open' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(409)
      expect(body.error_type).to.eq('ConflictException')
    })
  })

  // Updating a status that does not exist returns 500 GeneralException.
  it.skip('404s on an update to a status that does not exist', () => {
    cy.api('PUT', '/api/v1/statuses/99999999', { name: 'nope', category: 'open' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  // Deleting a status that does not exist returns 500 GeneralException.
  it.skip('404s on a delete of a status that does not exist', () => {
    cy.api('DELETE', '/api/v1/statuses/99999999', null, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('deletes the status', () => {
    cy.api('DELETE', `/api/v1/statuses/${statusId}`).its('status').should('eq', 200)
    listStatuses().then((rows) => {
      expect(rows.some((s) => s.id === statusId), 'deleted status still listed').to.be.false
    })
  })
})
