describe('API: help center article translations', () => {
  const stamp = Date.now()
  const hcSlug = `api-hc-tr-${stamp}`
  const otherSlug = `api-hc-tr-other-${stamp}`
  let helpCenterId
  let otherHelpCenterId
  let enCollectionId
  let frCollectionId
  let otherEnCollectionId
  let enArticleId
  let frArticleId
  let frSpareArticleId
  let otherArticleId

  const createArticle = (collectionId, title, locale) =>
    cy
      .api('POST', `/api/v1/collections/${collectionId}/articles`, {
        title,
        content: `<p>${title}</p>`,
        locale,
        status: 'published'
      })
      .its('body.data.id')

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/help-centers', {
      name: `API translations ${stamp}`,
      slug: hcSlug,
      page_title: 'Help',
      default_locale: 'en',
      allowed_locales: ['en', 'fr']
    })
      .then(({ body }) => {
        helpCenterId = body.data.id
        return cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, {
          name: `EN ${stamp}`,
          locale: 'en'
        })
      })
      .then(({ body }) => {
        enCollectionId = body.data.id
        return cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, {
          name: `FR ${stamp}`,
          locale: 'fr'
        })
      })
      .then(({ body }) => {
        frCollectionId = body.data.id
        return createArticle(enCollectionId, `EN article ${stamp}`, 'en')
      })
      .then((id) => {
        enArticleId = id
        return createArticle(frCollectionId, `FR article ${stamp}`, 'fr')
      })
      .then((id) => {
        frArticleId = id
        return createArticle(frCollectionId, `FR spare ${stamp}`, 'fr')
      })
      .then((id) => {
        frSpareArticleId = id
        return cy.api('POST', '/api/v1/help-centers', {
          name: `API translations other ${stamp}`,
          slug: otherSlug,
          page_title: 'Help',
          default_locale: 'en',
          allowed_locales: ['en']
        })
      })
      .then(({ body }) => {
        otherHelpCenterId = body.data.id
        return cy.api('POST', `/api/v1/help-centers/${otherHelpCenterId}/collections`, {
          name: `Other EN ${stamp}`,
          locale: 'en'
        })
      })
      .then(({ body }) => {
        otherEnCollectionId = body.data.id
        return createArticle(otherEnCollectionId, `Other EN article ${stamp}`, 'en')
      })
      .then((id) => {
        otherArticleId = id
      })
  })

  beforeEach(() => cy.login())

  it('rejects a linkable list with no locale to exclude', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/linkable-articles`, null, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/exclude_locale/i)
    })
  })

  it('lists only articles in the other locales', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/linkable-articles?exclude_locale=en`).then(
      ({ status, body }) => {
        expect(status).to.eq(200)
        const ids = body.data.map((a) => a.id)
        expect(ids).to.include(frArticleId)
        expect(ids).to.include(frSpareArticleId)
        expect(ids).to.not.include(enArticleId)
        expect(ids, 'articles from another help center').to.not.include(otherArticleId)
        const fr = body.data.find((a) => a.id === frArticleId)
        expect(fr.locale).to.eq('fr')
        expect(fr.collection_name).to.eq(`FR ${stamp}`)
      }
    )
  })

  it('rejects a link with no target article', () => {
    cy.api(
      'PUT',
      `/api/v1/articles/${frArticleId}/link-translation`,
      {},
      {
        failOnStatusCode: false
      }
    ).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/translation_of_id/i)
    })
  })

  it('rejects linking an article to itself', () => {
    cy.api(
      'PUT',
      `/api/v1/articles/${frArticleId}/link-translation`,
      { translation_of_id: frArticleId },
      { failOnStatusCode: false }
    ).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects linking across help centers', () => {
    cy.api(
      'PUT',
      `/api/v1/articles/${otherArticleId}/link-translation`,
      { translation_of_id: frArticleId },
      { failOnStatusCode: false }
    ).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('404s linking an article that does not exist', () => {
    cy.api(
      'PUT',
      '/api/v1/articles/99999999/link-translation',
      { translation_of_id: enArticleId },
      { failOnStatusCode: false }
    ).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  it('links the French article as a translation of the English one', () => {
    cy.api('PUT', `/api/v1/articles/${frArticleId}/link-translation`, {
      translation_of_id: enArticleId
    })
      .its('status')
      .should('eq', 200)

    cy.api('GET', `/api/v1/collections/${enCollectionId}/articles/${enArticleId}`).then(
      ({ body }) => {
        const locales = body.data.translations.map((t) => t.locale)
        expect(locales).to.have.members(['en', 'fr'])
      }
    )
    cy.api('GET', `/api/v1/collections/${frCollectionId}/articles/${frArticleId}`).then(
      ({ body }) => {
        const ids = body.data.translations.map((t) => t.id)
        expect(ids).to.have.members([enArticleId, frArticleId])
      }
    )
  })

  it('drops linked articles from the linkable list', () => {
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/linkable-articles?exclude_locale=en`).then(
      ({ body }) => {
        const ids = body.data.map((a) => a.id)
        expect(ids).to.not.include(frArticleId)
        expect(ids).to.include(frSpareArticleId)
      }
    )
  })

  it('rejects linking an article that already belongs to a translation set', () => {
    cy.api(
      'PUT',
      `/api/v1/articles/${enArticleId}/link-translation`,
      { translation_of_id: frSpareArticleId },
      { failOnStatusCode: false }
    ).then(({ status, body }) => {
      expect(status).to.eq(409)
      expect(body.error_type).to.eq('ConflictException')
    })
  })

  it('rejects a second translation in a locale the set already has', () => {
    cy.api(
      'PUT',
      `/api/v1/articles/${frSpareArticleId}/link-translation`,
      { translation_of_id: enArticleId },
      { failOnStatusCode: false }
    ).then(({ status, body }) => {
      expect(status).to.eq(409)
      expect(body.error_type).to.eq('ConflictException')
    })
  })

  it('unlinks the French article and leaves both standing on their own', () => {
    cy.api('PUT', `/api/v1/articles/${frArticleId}/unlink-translation`)
      .its('status')
      .should('eq', 200)

    cy.api('GET', `/api/v1/collections/${frCollectionId}/articles/${frArticleId}`).then(
      ({ body }) => {
        expect(body.data.translations.map((t) => t.id)).to.deep.eq([frArticleId])
      }
    )
    cy.api('GET', `/api/v1/collections/${enCollectionId}/articles/${enArticleId}`).then(
      ({ body }) => {
        expect(body.data.translations.map((t) => t.id)).to.deep.eq([enArticleId])
      }
    )
    cy.api('GET', `/api/v1/help-centers/${helpCenterId}/linkable-articles?exclude_locale=en`).then(
      ({ body }) => {
        expect(body.data.map((a) => a.id)).to.include(frArticleId)
      }
    )
  })

  it('404s unlinking an article that does not exist', () => {
    cy.api('PUT', '/api/v1/articles/99999999/unlink-translation', null, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(404)
      expect(body.error_type).to.eq('NotFoundException')
    })
  })

  after(() => {
    cy.login()
    cy.api('DELETE', `/api/v1/help-centers/${helpCenterId}`, null, { failOnStatusCode: false })
    cy.api('DELETE', `/api/v1/help-centers/${otherHelpCenterId}`, null, {
      failOnStatusCode: false
    })
  })
})
