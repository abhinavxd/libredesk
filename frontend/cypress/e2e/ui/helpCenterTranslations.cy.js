// The steps run in order and share the help center created by the first one.

const stamp = Date.now()
const siteName = `Cypress Translations ${stamp}`
const siteSlug = `cypress-tr-${stamp}`
const enArticleTitle = `EN article ${stamp}`
const frArticleTitle = `FR article ${stamp}`
const listPath = '/admin/help-center'

const sheet = () => cy.contains('[role="dialog"]', 'Edit article')
// The sheet holds a button with the same text, so scope the dialog by its heading.
const linkDialog = () => cy.contains('h2', 'Link an existing article').closest('[role="dialog"]')

describe('Help center article translations', () => {
  let helpCenterId
  let enCollectionId
  let frCollectionId
  let enArticleId
  let frArticleId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/help-centers', {
      name: siteName,
      slug: siteSlug,
      page_title: `Help ${stamp}`,
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
        return cy.api('POST', `/api/v1/collections/${enCollectionId}/articles`, {
          title: enArticleTitle,
          content: `<p>${enArticleTitle}</p>`,
          locale: 'en',
          status: 'published'
        })
      })
      .then(({ body }) => {
        enArticleId = body.data.id
        return cy.api('POST', `/api/v1/collections/${frCollectionId}/articles`, {
          title: frArticleTitle,
          content: `<p>${frArticleTitle}</p>`,
          locale: 'fr',
          status: 'published'
        })
      })
      .then(({ body }) => {
        frArticleId = body.data.id
      })
  })

  beforeEach(() => {
    cy.viewport(1400, 900)
    cy.login()
  })

  it('links an article written in another language', () => {
    cy.intercept('PUT', '**/api/v1/articles/*/link-translation').as('linkTranslation')

    cy.visit(`${listPath}/${helpCenterId}/tree/en`)
    cy.contains('button.tree-node-title', enArticleTitle).click()

    sheet().contains('button', 'Link an existing article').click()
    linkDialog().find('button[role="combobox"]').click()
    cy.get('[role="option"]').contains(frArticleTitle).click()
    linkDialog().contains('button', 'Link').click()

    cy.wait('@linkTranslation').its('response.statusCode').should('eq', 200)
    sheet().contains('French').should('exist')
    sheet().contains('button', 'Link an existing article').should('not.exist')
  })

  it('shows the linked article on the other side too', () => {
    cy.visit(`${listPath}/${helpCenterId}/tree/fr`)
    cy.contains('button.tree-node-title', frArticleTitle).click()
    sheet().contains('English').should('exist')
    sheet().contains('French').should('exist')
  })

  it('unlinks the article and keeps its own language listed', () => {
    cy.intercept('PUT', '**/api/v1/articles/*/unlink-translation').as('unlinkTranslation')

    cy.visit(`${listPath}/${helpCenterId}/tree/fr`)
    cy.contains('button.tree-node-title', frArticleTitle).click()

    sheet()
      .contains('button', 'French')
      .parent()
      .find('button[aria-label="Unlink translation"]')
      .click()
    cy.get('[role="alertdialog"]').contains('button', 'Unlink').click()
    cy.wait('@unlinkTranslation').its('response.statusCode').should('eq', 200)

    sheet().contains('French').should('exist')
    sheet().contains('button', 'Link an existing article').should('exist')
    sheet().find('button[aria-label="Unlink translation"]').should('not.exist')
    cy.api('GET', `/api/v1/collections/${frCollectionId}/articles/${frArticleId}`).then(
      ({ body }) => {
        expect(body.data.translations.map((t) => t.locale)).to.deep.eq(['fr'])
      }
    )
  })

  it('leaves no linkable article once every language is taken', () => {
    cy.api('PUT', `/api/v1/articles/${frArticleId}/link-translation`, {
      translation_of_id: enArticleId
    })
      .its('status')
      .should('eq', 200)

    cy.visit(`${listPath}/${helpCenterId}/tree/en`)
    cy.contains('button.tree-node-title', enArticleTitle).click()
    sheet().contains('button', 'Link an existing article').should('not.exist')
    sheet().contains('Add translation').should('not.exist')
  })

  it('links to the published article on the public site', () => {
    cy.visit(`${listPath}/${helpCenterId}/tree/en`)
    cy.contains('button.tree-node-title', enArticleTitle).click()
    sheet()
      .contains('a', 'View article')
      .should('have.attr', 'href')
      .and('include', `/hc/${siteSlug}/en/articles/`)
  })

  after(() => {
    cy.login()
    cy.api('DELETE', `/api/v1/help-centers/${helpCenterId}`, null, { failOnStatusCode: false })
  })
})
