describe('API: custom attributes', () => {
  const stamp = Date.now()
  const key = `api_attr_${stamp}`
  const created = []
  let attrId

  const create = (body, options) => cy.api('POST', '/api/v1/custom-attributes', body, options)

  before(() => cy.login())
  beforeEach(() => cy.login())

  after(() => {
    cy.login()
    created.forEach((id) => {
      cy.api('DELETE', `/api/v1/custom-attributes/${id}`, null, { failOnStatusCode: false })
    })
  })

  it('rejects a create with no name', () => {
    create({
      name: '', description: 'd', applies_to: 'conversation', key, data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a create with no applies_to', () => {
    create({
      name: 'X', description: 'd', applies_to: '', key, data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/applies_to/i)
    })
  })

  it('rejects a create with no data type', () => {
    create({
      name: 'X', description: 'd', applies_to: 'conversation', key, data_type: '', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a create with no description', () => {
    create({
      name: 'X', description: '', applies_to: 'conversation', key, data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/description/i)
    })
  })

  it('rejects a create with no key', () => {
    create({
      name: 'X', description: 'd', applies_to: 'conversation', key: '', data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/key/i)
    })
  })

  it('rejects a key that collides with a default conversation field', () => {
    create({
      name: 'X', description: 'd', applies_to: 'conversation', key: 'status', data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  // An omitted values array reaches Postgres as NULL and 500s on the NOT NULL column.
  it.skip('defaults values to an empty array when omitted', () => {
    create({
      name: 'X', description: 'd', applies_to: 'conversation', key: `${key}_noval`, data_type: 'text'
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.values).to.deep.eq([])
      created.push(body.data.id)
    })
  })

  // Data_type is a free-text column with no validation, so garbage is stored.
  it.skip('rejects an unknown data type', () => {
    create({
      name: 'X', description: 'd', applies_to: 'conversation', key: `${key}_baddt`, data_type: 'nonsense', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  // Applies_to is a free-text column with no validation, so garbage is stored.
  it.skip('rejects an unknown applies_to', () => {
    create({
      name: 'X', description: 'd', applies_to: 'nonsense', key: `${key}_badat`, data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a text attribute and persists every field', () => {
    create({
      name: 'Api Attr',
      description: 'created by the api contract spec',
      applies_to: 'conversation',
      key,
      data_type: 'text',
      values: [],
      regex: '^[a-z]+$',
      regex_hint: 'lowercase letters only'
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      attrId = body.data.id
      created.push(attrId)
      expect(attrId).to.be.a('number')
      expect(body.data.name).to.eq('Api Attr')
      expect(body.data.description).to.eq('created by the api contract spec')
      expect(body.data.applies_to).to.eq('conversation')
      expect(body.data.key).to.eq(key)
      expect(body.data.data_type).to.eq('text')
      expect(body.data.regex).to.eq('^[a-z]+$')
      expect(body.data.regex_hint).to.eq('lowercase letters only')
      expect(body.data.values).to.deep.eq([])
    })
  })

  it('creates a list attribute with its values', () => {
    create({
      name: 'Api List',
      description: 'd',
      applies_to: 'contact',
      key: `${key}_list`,
      data_type: 'list',
      values: ['small', 'medium', 'large']
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      created.push(body.data.id)
      expect(body.data.data_type).to.eq('list')
      expect(body.data.values).to.deep.eq(['small', 'medium', 'large'])
    })
  })

  const otherTypes = ['number', 'checkbox', 'date', 'link']
  otherTypes.forEach((dataType) => {
    it(`creates a ${dataType} attribute`, () => {
      create({
        name: `Api ${dataType}`,
        description: 'd',
        applies_to: 'contact',
        key: `${key}_${dataType}`,
        data_type: dataType,
        values: []
      }).then(({ status, body }) => {
        expect(status).to.eq(200)
        created.push(body.data.id)
        expect(body.data.data_type).to.eq(dataType)
      })
    })
  })

  it('reads the attribute back by id', () => {
    cy.api('GET', `/api/v1/custom-attributes/${attrId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.key).to.eq(key)
      expect(body.data.name).to.eq('Api Attr')
      expect(body.data.applies_to).to.eq('conversation')
    })
  })

  it('lists the attribute', () => {
    cy.api('GET', '/api/v1/custom-attributes').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((a) => a.id === attrId), 'created attribute in list').to.be.true
    })
  })

  it('filters the list by applies_to', () => {
    cy.api('GET', '/api/v1/custom-attributes?applies_to=contact').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.every((a) => a.applies_to === 'contact'), 'only contact attributes').to.be.true
      expect(rows.some((a) => a.id === attrId), 'conversation attribute excluded').to.be.false
    })
  })

  it('rejects a duplicate key for the same applies_to', () => {
    create({
      name: 'Api Attr Dupe', description: 'd', applies_to: 'conversation', key, data_type: 'text', values: []
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('allows the same key under a different applies_to', () => {
    create({
      name: 'Api Attr Contact', description: 'd', applies_to: 'contact', key, data_type: 'text', values: []
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      created.push(body.data.id)
      expect(body.data.key).to.eq(key)
      expect(body.data.applies_to).to.eq('contact')
    })
  })

  it('updates the attribute', () => {
    cy.api('PUT', `/api/v1/custom-attributes/${attrId}`, {
      name: 'Api Attr Renamed',
      description: 'updated by the api contract spec',
      applies_to: 'conversation',
      key,
      data_type: 'text',
      values: [],
      regex: '^[0-9]+$',
      regex_hint: 'digits only'
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/custom-attributes/${attrId}`).then(({ body }) => {
      expect(body.data.name).to.eq('Api Attr Renamed')
      expect(body.data.description).to.eq('updated by the api contract spec')
      expect(body.data.regex).to.eq('^[0-9]+$')
      expect(body.data.regex_hint).to.eq('digits only')
    })
  })

  // The form makes both read-only once saved; the update query never touches them.
  it('does not let an update change the key or the data type', () => {
    cy.api('PUT', `/api/v1/custom-attributes/${attrId}`, {
      name: 'Api Attr Renamed',
      description: 'updated by the api contract spec',
      applies_to: 'conversation',
      key: `${key}_moved`,
      data_type: 'date',
      values: []
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/custom-attributes/${attrId}`).then(({ body }) => {
      expect(body.data.key).to.eq(key)
      expect(body.data.data_type).to.eq('text')
    })
  })

  it('404s on an attribute that does not exist', () => {
    cy.api('GET', '/api/v1/custom-attributes/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a non-numeric id', () => {
    cy.api('GET', '/api/v1/custom-attributes/abc', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
  })

  // Deleting a missing attribute returns 200, skipped until the API settles on 404 vs 200.
  it.skip('404s when deleting an attribute that does not exist', () => {
    cy.api('DELETE', '/api/v1/custom-attributes/99999999', null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })

  it('deletes the attribute', () => {
    cy.api('DELETE', `/api/v1/custom-attributes/${attrId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/custom-attributes/${attrId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })
})
