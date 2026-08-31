describe('API: help center', () => {
  const stamp = Date.now()
  const hcSlug = `api-hc-${stamp}`
  const hcName = `API Help Center ${stamp}`
  const collectionName = `Getting started ${stamp}`
  const articleTitle = `How to reset a password ${stamp}`
  let helpCenterId
  let collectionId
  let articleId

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a help center with no name', () => {
    cy.api('POST', '/api/v1/help-centers', { slug: hcSlug, page_title: 'Help' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a help center with no slug', () => {
    cy.api('POST', '/api/v1/help-centers', { name: hcName, page_title: 'Help' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/slug/i)
    })
  })

  it('rejects a help center with no page title', () => {
    cy.api('POST', '/api/v1/help-centers', { name: hcName, slug: hcSlug }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/page_title/i)
    })
  })

  it('rejects a malformed slug', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName, slug: 'Not A Slug!', page_title: 'Help'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a slug reserved by a public route', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName, slug: 'api', page_title: 'Help'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an unsupported default locale', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName, slug: `${hcSlug}-locale`, page_title: 'Help', default_locale: 'xx'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a malformed custom domain', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName, slug: `${hcSlug}-domain`, page_title: 'Help', custom_domain: 'not a url'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a help center and persists every field', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName,
      slug: hcSlug,
      page_title: `Help ${stamp}`,
      meta_description: 'Answers to common questions',
      default_locale: 'en',
      allowed_locales: ['en'],
      template: 'docs'
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      helpCenterId = body.data.id
      expect(helpCenterId).to.be.a('number')
      expect(body.data.name).to.eq(hcName)
      expect(body.data.slug).to.eq(hcSlug)
      expect(body.data.page_title).to.eq(`Help ${stamp}`)
      expect(body.data.meta_description).to.eq('Answers to common questions')
      expect(body.data.default_locale).to.eq('en')
      expect(body.data.allowed_locales).to.deep.eq(['en'])
      expect(body.data.template).to.eq('docs')
      expect(body.data.is_active).to.eq(true)
    })
  })

  it('reads the help center back by id', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.slug).to.eq(hcSlug)
      expect(body.data.name).to.eq(hcName)
    })
  })

  it('lists the help center', () => {
    cy.api('GET', '/api/v1/help-centers').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((hc) => hc.slug === hcSlug), 'created help center in list').to.be.true
    })
  })

  it('rejects a duplicate slug', () => {
    cy.api('POST', '/api/v1/help-centers', {
      name: `${hcName} clone`, slug: hcSlug, page_title: 'Help'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(409)
      expect(body.error_type).to.eq('ConflictException')
    })
  })

  it('updates the help center', () => {
    cy.api('PUT', `/api/v1/help-centers/${helpCenterId}`, {
      name: `${hcName} renamed`,
      slug: hcSlug,
      page_title: 'Renamed title',
      meta_description: 'Renamed meta',
      default_locale: 'en',
      allowed_locales: ['en'],
      template: 'classic'
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/help-centers/${helpCenterId}`).then(({ body }) => {
      expect(body.data.name).to.eq(`${hcName} renamed`)
      expect(body.data.page_title).to.eq('Renamed title')
      expect(body.data.meta_description).to.eq('Renamed meta')
      expect(body.data.template).to.eq('classic')
    })
  })

  it('toggles the help center active flag', () => {
    cy.api('PUT', `/api/v1/help-centers/${helpCenterId}/toggle`)
      .its('body.data.is_active')
      .should('eq', false)
    cy.api('PUT', `/api/v1/help-centers/${helpCenterId}/toggle`)
      .its('body.data.is_active')
      .should('eq', true)
  })

  it('404s on a help center that does not exist', () => {
    cy.api('GET', '/api/v1/help-centers/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a collection with no name', () => {
    cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, { locale: 'en' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a collection in a locale the help center does not serve', () => {
    cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, {
      name: collectionName, locale: 'fr'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('404s creating a collection under a help center that does not exist', () => {
    cy.api('POST', '/api/v1/help-centers/99999999/collections', {
      name: collectionName, locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('creates a collection in the help center', () => {
    cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, {
      name: collectionName,
      description: 'Start here',
      icon: 'book',
      locale: 'en',
      sort_order: 1,
      is_published: true
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      collectionId = body.data.id
      expect(collectionId).to.be.a('number')
      expect(body.data.help_center_id).to.eq(helpCenterId)
      expect(body.data.name).to.eq(collectionName)
      expect(body.data.description).to.eq('Start here')
      expect(body.data.icon).to.eq('book')
      expect(body.data.locale).to.eq('en')
      expect(body.data.sort_order).to.eq(1)
      expect(body.data.is_published).to.eq(true)
      expect(body.data.slug).to.be.a('string').and.not.be.empty
    })
  })

  it('lists the collection under its help center', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/collections`).then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((c) => c.id === collectionId), 'created collection in list').to.be.true
    })
  })

  it('updates the collection', () => {
    cy.api('PUT', `/api/v1/help-centers/${helpCenterId}/collections/${collectionId}`, {
      name: `${collectionName} renamed`,
      description: 'Renamed description',
      icon: 'star',
      locale: 'en',
      sort_order: 5,
      is_published: false
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/collections`).then(({ body }) => {
      const rows = body.data.results || body.data
      const updated = rows.find((c) => c.id === collectionId)
      expect(updated.name).to.eq(`${collectionName} renamed`)
      expect(updated.description).to.eq('Renamed description')
      expect(updated.icon).to.eq('star')
      expect(updated.sort_order).to.eq(5)
      expect(updated.is_published).to.eq(false)
    })
  })

  it('toggles the collection published flag', () => {
    cy.api('PUT', `/api/v1/collections/${collectionId}/toggle`)
      .its('body.data.is_published')
      .should('eq', true)
  })

  it('404s updating a collection that does not exist', () => {
    cy.api('PUT', `/api/v1/help-centers/${helpCenterId}/collections/99999999`, {
      name: 'ghost', locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('rejects an article with no title', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      content: '<p>Body</p>', locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/title/i)
    })
  })

  it('rejects an article with no content', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      title: articleTitle, locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/content/i)
    })
  })

  it('rejects an article whose content is only whitespace', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      title: articleTitle, content: '   \n\t  ', locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/content/i)
    })
  })

  it('rejects an unknown article status', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      title: articleTitle, content: '<p>Body</p>', locale: 'en', status: 'bogus'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an article whose locale differs from its collection', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      title: articleTitle, content: '<p>Body</p>', locale: 'de'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('404s creating an article in a collection that does not exist', () => {
    cy.api('POST', '/api/v1/collections/99999999/articles', {
      title: articleTitle, content: '<p>Body</p>', locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('creates an article in the collection', () => {
    cy.api('POST', `/api/v1/collections/${collectionId}/articles`, {
      title: articleTitle,
      content: '<p>Open settings and click reset.</p>',
      excerpt: 'Reset guide',
      meta_title: 'Reset your password',
      meta_description: 'Steps to reset a password',
      locale: 'en',
      status: 'published',
      sort_order: 2,
      ai_enabled: true
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      articleId = body.data.id
      expect(articleId).to.be.a('number')
      expect(body.data.collection_id).to.eq(collectionId)
      expect(body.data.title).to.eq(articleTitle)
      expect(body.data.content).to.eq('<p>Open settings and click reset.</p>')
      expect(body.data.excerpt).to.eq('Reset guide')
      expect(body.data.meta_title).to.eq('Reset your password')
      expect(body.data.meta_description).to.eq('Steps to reset a password')
      expect(body.data.locale).to.eq('en')
      expect(body.data.status).to.eq('published')
      expect(body.data.sort_order).to.eq(2)
      expect(body.data.ai_enabled).to.eq(true)
      expect(body.data.slug).to.be.a('string').and.not.be.empty
    })
  })

  it('reads the article back by id', () => {
    cy.api('GET', `/api/v1/collections/${collectionId}/articles/${articleId}`)
      .then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(body.data.title).to.eq(articleTitle)
        expect(body.data.content).to.eq('<p>Open settings and click reset.</p>')
      })
  })

  it('appears in the help center tree', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/tree?locale=en`).then(({ status, body }) => {
      expect(status).to.eq(200)
      const collection = body.data.tree.find((c) => c.id === collectionId)
      expect(collection, 'created collection in tree').to.exist
      expect(collection.articles.some((a) => a.id === articleId), 'created article in tree').to.be.true
    })
  })

  it('updates the article', () => {
    cy.api('PUT', `/api/v1/articles/${articleId}`, {
      title: `${articleTitle} renamed`,
      content: '<p>Updated body.</p>',
      excerpt: 'Renamed excerpt',
      locale: 'en',
      status: 'draft',
      ai_enabled: false
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/collections/${collectionId}/articles/${articleId}`).then(({ body }) => {
      expect(body.data.title).to.eq(`${articleTitle} renamed`)
      expect(body.data.content).to.eq('<p>Updated body.</p>')
      expect(body.data.excerpt).to.eq('Renamed excerpt')
      expect(body.data.status).to.eq('draft')
      expect(body.data.ai_enabled).to.eq(false)
    })
  })

  it('rejects an update whose content is only whitespace', () => {
    cy.api('PUT', `/api/v1/articles/${articleId}`, {
      title: articleTitle, content: '   ', locale: 'en'
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/content/i)
    })
  })

  it('publishes the article through the status endpoint', () => {
    cy.api('PUT', `/api/v1/articles/${articleId}/status`, { status: 'published' })
      .its('status')
      .should('eq', 200)
    cy.api('GET', `/api/v1/collections/${collectionId}/articles/${articleId}`)
      .its('body.data.status')
      .should('eq', 'published')
  })

  it('rejects an empty status', () => {
    cy.api('PUT', `/api/v1/articles/${articleId}/status`, { status: '' }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('404s on an article that does not exist', () => {
    cy.api('GET', `/api/v1/collections/${collectionId}/articles/99999999`, null, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('deletes the article', () => {
    cy.api('DELETE', `/api/v1/collections/${collectionId}/articles/${articleId}`)
      .its('status')
      .should('eq', 200)
    cy.api('GET', `/api/v1/collections/${collectionId}/articles/${articleId}`, null, {
      failOnStatusCode: false
    }).its('status').should('eq', 404)
  })

  it('deletes the collection', () => {
    cy.api('DELETE', `/api/v1/help-centers/${helpCenterId}/collections/${collectionId}`)
      .its('status')
      .should('eq', 200)
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/collections`).then(({ body }) => {
      const rows = body.data.results || body.data
      expect(rows.some((c) => c.id === collectionId), 'deleted collection gone').to.be.false
    })
  })

  it('deletes the help center', () => {
    cy.api('DELETE', `/api/v1/help-centers/${helpCenterId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('eq', 404)
  })
})
