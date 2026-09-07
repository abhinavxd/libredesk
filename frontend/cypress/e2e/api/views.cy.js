describe('API: views', () => {
  const stamp = Date.now()
  // Filters are a {logic, rules} group or legacy flat array, leaves use conversation-list fields.
  const openFilter = {
    logic: 'AND',
    rules: [{ model: 'conversations', field: 'status_id', operator: 'equals', value: '1' }]
  }
  const highPriorityFilter = {
    logic: 'OR',
    rules: [{ model: 'conversations', field: 'priority_id', operator: 'equals', value: '3' }]
  }
  const legacyFilter = [{ model: 'conversations', field: 'status_id', operator: 'equals', value: '1' }]
  let teamId
  let viewId
  let sharedViewId
  let teamViewId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/teams', {
      name: `Api View Team ${stamp}`,
      conversation_assignment_type: 'Round robin',
      timezone: 'UTC',
      max_auto_assigned_conversations: 0
    }).its('body.data.id').then((id) => {
      teamId = id
    })
  })

  beforeEach(() => cy.login())

  it('rejects a personal view with no name', () => {
    cy.api('POST', '/api/v1/views/me', { filters: openFilter }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a personal view with no filters', () => {
    cy.api('POST', '/api/v1/views/me', { name: `No filters ${stamp}` }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/filters/i)
    })
  })

  it('rejects filters sent as a JSON string', () => {
    cy.api('POST', '/api/v1/views/me', {
      name: `String filters ${stamp}`, filters: JSON.stringify(legacyFilter)
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects filters on a field that is not allowed', () => {
    cy.api('POST', '/api/v1/views/me', {
      name: `Bad field ${stamp}`,
      filters: { logic: 'AND', rules: [{ model: 'conversations', field: 'nope', operator: 'equals', value: '1' }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects filters on a model that is not allowed', () => {
    cy.api('POST', '/api/v1/views/me', {
      name: `Bad model ${stamp}`,
      filters: { logic: 'AND', rules: [{ model: 'secrets', field: 'id', operator: 'equals', value: '1' }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a personal view and persists every field', () => {
    cy.api('POST', '/api/v1/views/me', {
      name: `Api personal view ${stamp}`, filters: openFilter
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      viewId = body.data.id
      expect(viewId).to.be.a('number')
      expect(body.data.name).to.eq(`Api personal view ${stamp}`)
      expect(body.data.visibility).to.eq('user')
      expect(body.data.user_id).to.be.a('number')
      expect(body.data.filters.logic).to.eq('AND')
      expect(body.data.filters.rules).to.have.length(1)
      expect(body.data.filters.rules[0]).to.include({
        model: 'conversations', field: 'status_id', operator: 'equals', value: '1'
      })
    })
  })

  it('accepts the legacy flat array filter form', () => {
    cy.api('POST', '/api/v1/views/me', {
      name: `Api legacy view ${stamp}`, filters: legacyFilter
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.filters).to.have.length(1)
      cy.api('DELETE', `/api/v1/views/me/${body.data.id}`).its('status').should('eq', 200)
    })
  })

  it('lists the personal view', () => {
    cy.api('GET', '/api/v1/views/me').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((v) => v.id === viewId), 'created view in list').to.be.true
    })
  })

  it('lists conversations for the view', () => {
    cy.api('GET', `/api/v1/views/${viewId}/conversations?page=1&page_size=5`)
      .then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(body.data.results).to.be.an('array')
        expect(body.data.page).to.eq(1)
      })
  })

  it('updates the personal view', () => {
    cy.api('PUT', `/api/v1/views/me/${viewId}`, {
      name: `Api personal view renamed ${stamp}`, filters: highPriorityFilter
    }).its('status').should('eq', 200)

    cy.api('GET', '/api/v1/views/me').then(({ body }) => {
      const rows = body.data.results || body.data
      const view = rows.find((v) => v.id === viewId)
      expect(view.name).to.eq(`Api personal view renamed ${stamp}`)
      expect(view.filters.logic).to.eq('OR')
      expect(view.filters.rules[0].field).to.eq('priority_id')
    })
  })

  it('404s on updating a personal view that does not exist', () => {
    cy.api('PUT', '/api/v1/views/me/99999999', { name: 'Ghost', filters: openFilter }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('404s on deleting a personal view that does not exist', () => {
    cy.api('DELETE', '/api/v1/views/me/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a shared view with no name', () => {
    cy.api('POST', '/api/v1/shared-views', { visibility: 'all', filters: openFilter }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a shared view with no filters', () => {
    cy.api('POST', '/api/v1/shared-views', { name: `No filters ${stamp}`, visibility: 'all' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/filters/i)
    })
  })

  it('rejects a shared view with user visibility', () => {
    cy.api('POST', '/api/v1/shared-views', {
      name: `Personal disguised ${stamp}`, visibility: 'user', filters: openFilter
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a team shared view with no team id', () => {
    cy.api('POST', '/api/v1/shared-views', {
      name: `No team ${stamp}`, visibility: 'team', filters: openFilter
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/team_id/i)
    })
  })

  it('creates a shared view visible to everyone', () => {
    cy.api('POST', '/api/v1/shared-views', {
      name: `Api shared view ${stamp}`, visibility: 'all', filters: openFilter
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      sharedViewId = body.data.id
      expect(sharedViewId).to.be.a('number')
      expect(body.data.name).to.eq(`Api shared view ${stamp}`)
      expect(body.data.visibility).to.eq('all')
      expect(body.data.user_id).to.be.undefined
      expect(body.data.filters.rules[0].field).to.eq('status_id')
    })
  })

  it('creates a shared view scoped to a team', () => {
    cy.api('POST', '/api/v1/shared-views', {
      name: `Api team view ${stamp}`, visibility: 'team', team_id: teamId, filters: openFilter
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      teamViewId = body.data.id
      expect(body.data.visibility).to.eq('team')
      expect(body.data.team_id).to.eq(teamId)
    })
  })

  it('reads the shared view back by id', () => {
    cy.api('GET', `/api/v1/shared-views/${sharedViewId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.id).to.eq(sharedViewId)
      expect(body.data.name).to.eq(`Api shared view ${stamp}`)
      expect(body.data.visibility).to.eq('all')
    })
  })

  it('lists both shared views', () => {
    cy.api('GET', '/api/v1/shared-views').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((v) => v.id === sharedViewId), 'all-visibility view in list').to.be.true
      expect(rows.some((v) => v.id === teamViewId), 'team view in list').to.be.true
    })
  })

  it('shows the all-visibility view on the current user shared list', () => {
    cy.api('GET', '/api/v1/views/shared').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((v) => v.id === sharedViewId), 'all-visibility view visible to user').to.be.true
      expect(rows.some((v) => v.id === teamViewId), 'team view hidden from non member').to.be.false
    })
  })

  it('updates the shared view', () => {
    cy.api('PUT', `/api/v1/shared-views/${sharedViewId}`, {
      name: `Api shared view renamed ${stamp}`, visibility: 'all', filters: highPriorityFilter
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/shared-views/${sharedViewId}`).then(({ body }) => {
      expect(body.data.name).to.eq(`Api shared view renamed ${stamp}`)
      expect(body.data.filters.rules[0].field).to.eq('priority_id')
    })
  })

  it('404s on a shared view that does not exist', () => {
    cy.api('GET', '/api/v1/shared-views/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a shared view id of zero', () => {
    cy.api('GET', '/api/v1/shared-views/0', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
  })

  it('does not let the personal view routes touch a shared view', () => {
    cy.api('PUT', `/api/v1/views/me/${sharedViewId}`, { name: 'Hijack', filters: openFilter }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(403)
      expect(body.error_type).to.eq('PermissionException')
    })

    cy.api('DELETE', `/api/v1/views/me/${sharedViewId}`, null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(403)
        expect(body.error_type).to.eq('PermissionException')
      })
  })

  it('does not let the shared view routes touch a personal view', () => {
    cy.api('GET', `/api/v1/shared-views/${viewId}`, null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })

    cy.api('PUT', `/api/v1/shared-views/${viewId}`, {
      name: 'Hijack', visibility: 'all', filters: openFilter
    }, { failOnStatusCode: false }).its('status').should('eq', 404)

    cy.api('DELETE', `/api/v1/shared-views/${viewId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })

  it('deletes the shared views', () => {
    cy.api('DELETE', `/api/v1/shared-views/${sharedViewId}`).its('status').should('eq', 200)
    cy.api('DELETE', `/api/v1/shared-views/${teamViewId}`).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/shared-views/${sharedViewId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })

  it('deletes the personal view', () => {
    cy.api('DELETE', `/api/v1/views/me/${viewId}`).its('status').should('eq', 200)

    cy.api('GET', '/api/v1/views/me').then(({ body }) => {
      const rows = body.data.results || body.data
      expect(rows.some((v) => v.id === viewId), 'deleted view gone from list').to.be.false
    })
  })
})
