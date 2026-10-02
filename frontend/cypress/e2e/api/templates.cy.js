describe('API: templates', () => {
  const stamp = Date.now()
  const name = `api-template-${stamp}`
  let templateId

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a list with no type', () => {
    cy.api('GET', '/api/v1/templates', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
        expect(body.message).to.match(/type/i)
      })
  })

  it('rejects a create with no name', () => {
    cy.api('POST', '/api/v1/templates', { type: 'email_outgoing', body: 'b' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a create with no type', () => {
    cy.api('POST', '/api/v1/templates', { name: `${name}-notype`, body: 'b' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/type/i)
    })
  })

  it('rejects a create with an unknown type', () => {
    cy.api('POST', '/api/v1/templates', {
      name: `${name}-badtype`, type: 'not_a_type', body: 'b'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a list with an unknown type', () => {
    cy.api('GET', '/api/v1/templates?type=not_a_type', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
  })

  it('rejects a name over the length limit', () => {
    cy.api('POST', '/api/v1/templates', {
      name: 'x'.repeat(200), type: 'email_outgoing', body: 'b'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a template and persists every field', () => {
    cy.api('POST', '/api/v1/templates', {
      name,
      type: 'email_outgoing',
      subject: 'Api subject',
      body: '<p>api body</p>',
      is_default: false
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      templateId = body.data.id
      expect(templateId).to.be.a('number')
      expect(body.data.name).to.eq(name)
      expect(body.data.type).to.eq('email_outgoing')
      expect(body.data.subject).to.eq('Api subject')
      expect(body.data.body).to.eq('<p>api body</p>')
      expect(body.data.is_default).to.eq(false)
      expect(body.data.is_builtin).to.eq(false)
    })
  })

  it('reads the template back by id', () => {
    cy.api('GET', `/api/v1/templates/${templateId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.name).to.eq(name)
      expect(body.data.subject).to.eq('Api subject')
    })
  })

  it('lists the template under its type', () => {
    cy.api('GET', '/api/v1/templates?type=email_outgoing').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((t) => t.id === templateId), 'created template in list').to.be.true
    })
  })



  it('updates the template', () => {
    cy.api('PUT', `/api/v1/templates/${templateId}`, {
      name: `${name}-renamed`,
      type: 'email_outgoing',
      subject: 'Api subject updated',
      body: '<p>updated</p>',
      is_default: false
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/templates/${templateId}`).then(({ body }) => {
      expect(body.data.name).to.eq(`${name}-renamed`)
      expect(body.data.subject).to.eq('Api subject updated')
      expect(body.data.body).to.eq('<p>updated</p>')
    })
  })

  it('rejects an update with no name', () => {
    cy.api('PUT', `/api/v1/templates/${templateId}`, {
      type: 'email_outgoing', body: 'b'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })



  it('404s on a template that does not exist', () => {
    cy.api('GET', '/api/v1/templates/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a non numeric id', () => {
    cy.api('GET', '/api/v1/templates/not-a-number', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
  })

  it('404s on an update of a template that does not exist', () => {
    cy.api('PUT', '/api/v1/templates/99999999', {
      name: 'ghost', type: 'email_outgoing', body: 'b'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('rejects a second default template without changing the existing default', () => {
    cy.api('GET', '/api/v1/templates?type=email_outgoing').then(({ body }) => {
      const existing = (body.data.results || body.data).find((row) => row.is_default)
      const getDefault = existing
        ? cy.wrap(existing)
        : cy.api('POST', '/api/v1/templates', {
          name: `${name}-default-a`, type: 'email_outgoing', body: 'b', is_default: true
        }).then(({ body: created }) => created.data)
      getDefault.then((first) => {
        cy.api('POST', '/api/v1/templates', {
          name: `${name}-default-b`, type: 'email_outgoing', body: 'b', is_default: true
        }, { failOnStatusCode: false }).then((res) => {
          expect(res.status).to.eq(409)
          expect(res.body.error_type).to.eq('ConflictException')
        })
        cy.api('PUT', `/api/v1/templates/${templateId}`, {
          name: `${name}-renamed`, type: 'email_outgoing', body: 'b', is_default: true
        }, { failOnStatusCode: false }).then((res) => {
          expect(res.status).to.eq(409)
          expect(res.body.error_type).to.eq('ConflictException')
        })
        cy.api('GET', `/api/v1/templates/${first.id}`).its('body.data.is_default').should('eq', true)
        if (!existing) cy.api('DELETE', `/api/v1/templates/${first.id}`)
      })
    })
  })

  it('deletes the template', () => {
    cy.api('DELETE', `/api/v1/templates/${templateId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/templates/${templateId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })
})

it('rejects retired notification template requests', () => {
  cy.login()
  cy.api('GET', '/api/v1/templates?type=email_notification', null, { failOnStatusCode: false }).its('status').should('eq', 400)
})
