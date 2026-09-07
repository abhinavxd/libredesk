// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const siteName = `Cypress HC ${stamp}`
const siteSlug = `cypress-hc-${stamp}`
const pageTitle = `Cypress help center ${stamp}`
const editedPageTitle = `Cypress help center ${stamp} edited`
const collectionName = `Cypress Collection ${stamp}`
const collectionDescription = `Collection from the help center form spec ${stamp}`
const renamedCollection = `Cypress Collection ${stamp} edited`
const articleTitle = `Cypress Article ${stamp}`
const articleBody = `Article body from the help center form spec ${stamp}`
const articleExcerpt = `Excerpt ${stamp}`
const listPath = '/admin/help-center'

// Hover cannot be faked, but the row wrapper also reveals actions on focus-within.
const openRowMenu = (label) =>
  cy.contains('button.tree-node-title', label).closest('.tree-node').find('.hover-actions button').focus().click()

const sheet = () => cy.get('[role="dialog"]')

describe('Help center forms', () => {
  let helpCenterId
  let collectionId
  let articleId

  beforeEach(() => {
    cy.viewport(1400, 900)
    cy.login()
  })

  it('creates a help center site', () => {
    cy.intercept('POST', '**/api/v1/help-centers').as('createHelpCenter')

    cy.visit(listPath)
    cy.contains('button', 'New').click()

    sheet().find('input[name="name"]').type(siteName)
    sheet().find('input[name="slug"]').clear().type(siteSlug)
    sheet().find('input[name="page_title"]').type(pageTitle)

    sheet().find('button[type="submit"]').click()
    cy.wait('@createHelpCenter').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      helpCenterId = response.body.data.id
      cy.location('pathname').should('eq', `${listPath}/${helpCenterId}/customize`)
    })

    cy.visit(listPath)
    cy.contains(siteName).should('exist')
  })

  it('loads the saved site values back into the customize form', () => {
    expect(helpCenterId, 'help center from the create step').to.be.a('number')

    cy.visit(`${listPath}/${helpCenterId}/customize`)

    cy.get('input[name="name"]').should('have.value', siteName)
    cy.get('input[name="slug"]').should('have.value', siteSlug)
    cy.get('input[name="page_title"]').should('have.value', pageTitle)
    cy.get('select[name="template"]').should('have.value', 'classic')
  })

  it('persists a changed page title', () => {
    cy.intercept('PUT', `**/api/v1/help-centers/${helpCenterId}`).as('updateHelpCenter')

    cy.visit(`${listPath}/${helpCenterId}/customize`)
    cy.get('input[name="page_title"]').should('have.value', pageTitle).clear().type(editedPageTitle)
    cy.get('button[type="submit"]').click()
    cy.wait('@updateHelpCenter').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${helpCenterId}/customize`)
    cy.get('input[name="page_title"]').should('have.value', editedPageTitle)
  })

  it('rejects a site submit with no name', () => {
    cy.intercept('POST', '**/api/v1/help-centers').as('createHelpCenter')

    cy.visit(listPath)
    cy.contains('button', 'New').click()
    sheet().find('button[type="submit"]').click()

    cy.contains(/required/i).should('exist')
    sheet().find('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createHelpCenter.all').should('have.length', 0)
  })

  it('creates a collection', () => {
    cy.intercept('POST', `**/api/v1/help-centers/${helpCenterId}/collections`).as('createCollection')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button', 'New collection').click()

    sheet().find('input[name="name"]').type(collectionName)
    sheet().find('textarea[name="description"]').type(collectionDescription)
    sheet().contains('button', 'Create').click()

    cy.wait('@createCollection').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      collectionId = response.body.data.id
    })

    cy.contains(collectionName).should('exist')
  })

  it('loads the saved collection back into its edit sheet', () => {
    expect(collectionId, 'collection from the create step').to.be.a('number')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', collectionName).click()

    sheet().contains('Edit collection').should('be.visible')
    sheet().find('input[name="name"]').should('have.value', collectionName)
    sheet().find('textarea[name="description"]').should('have.value', collectionDescription)
    sheet().find('select[name="locale"]').should('have.value', 'en')
    sheet().find('button[role="switch"]').should('have.attr', 'data-state', 'checked')
  })

  it('persists a changed collection name', () => {
    cy.intercept('PUT', `**/api/v1/help-centers/${helpCenterId}/collections/${collectionId}`).as(
      'updateCollection'
    )

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', collectionName).click()
    sheet().find('input[name="name"]').clear().type(renamedCollection)
    sheet().contains('button', 'Update').click()
    cy.wait('@updateCollection').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', renamedCollection).click()
    sheet().find('input[name="name"]').should('have.value', renamedCollection)
  })

  it('rejects a collection submit with no name', () => {
    cy.intercept('POST', `**/api/v1/help-centers/${helpCenterId}/collections`).as('createCollection')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button', 'New collection').click()
    sheet().find('textarea[name="description"]').type('No name on this one')
    sheet().contains('button', 'Create').click()

    cy.contains(/required/i).should('exist')
    sheet().find('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createCollection.all').should('have.length', 0)
  })

  it('creates an article', () => {
    cy.intercept('POST', '**/api/v1/collections/*/articles').as('createArticle')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    openRowMenu(renamedCollection)
    cy.get('[role="menuitem"]').contains('New article').click()

    sheet().find('input[name="title"]').type(articleTitle)
    cy.get('.tiptap.ProseMirror').click().type(articleBody)
    sheet().find('textarea[name="excerpt"]').type(articleExcerpt)
    sheet().find('select[name="status"]').siblings('button[role="combobox"]').click()
    cy.contains('[role="option"]', 'Published').click()

    sheet().contains('button', 'Create').click()
    cy.wait('@createArticle').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      articleId = response.body.data.id
    })

    cy.contains(articleTitle).should('exist')
  })

  it('loads the saved article back into its edit sheet', () => {
    expect(articleId, 'article from the create step').to.be.a('number')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', articleTitle).click()

    sheet().contains('Edit article').should('be.visible')
    sheet().find('input[name="title"]').should('have.value', articleTitle)
    cy.get('.tiptap.ProseMirror').should('contain.text', articleBody)
    sheet().find('textarea[name="excerpt"]').should('have.value', articleExcerpt)
    sheet().find('select[name="status"]').should('have.value', 'published')
    sheet().find('select[name="collection_id"]').should('have.value', String(collectionId))
    sheet().find('select[name="locale"]').should('have.value', 'en')
  })

  it('persists a changed article title', () => {
    cy.intercept('PUT', `**/api/v1/articles/${articleId}`).as('updateArticle')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', articleTitle).click()
    sheet().find('input[name="title"]').clear().type(`${articleTitle} edited`)
    sheet().contains('button', 'Update').click()
    cy.wait('@updateArticle').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    cy.contains('button.tree-node-title', `${articleTitle} edited`).click()
    sheet().find('input[name="title"]').should('have.value', `${articleTitle} edited`)
    cy.get('.tiptap.ProseMirror').should('contain.text', articleBody)
  })

  it('rejects an article submit with no title', () => {
    cy.intercept('POST', '**/api/v1/collections/*/articles').as('createArticle')

    cy.visit(`${listPath}/${helpCenterId}/tree`)
    openRowMenu(renamedCollection)
    cy.get('[role="menuitem"]').contains('New article').click()

    cy.get('.tiptap.ProseMirror').click().type(articleBody)
    sheet().contains('button', 'Create').click()

    cy.contains(/required/i).should('exist')
    sheet().find('input[name="title"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createArticle.all').should('have.length', 0)
  })

  it('deletes the article, the collection and the site', () => {
    cy.intercept('DELETE', '**/api/v1/collections/*/articles/*').as('deleteArticle')
    cy.intercept(
      'DELETE',
      `**/api/v1/help-centers/${helpCenterId}/collections/${collectionId}`
    ).as('deleteCollection')
    cy.intercept('DELETE', `**/api/v1/help-centers/${helpCenterId}`).as('deleteHelpCenter')

    cy.visit(`${listPath}/${helpCenterId}/tree`)

    openRowMenu(`${articleTitle} edited`)
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()
    cy.wait('@deleteArticle').its('response.statusCode').should('eq', 200)
    cy.contains(`${articleTitle} edited`).should('not.exist')

    openRowMenu(renamedCollection)
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()
    cy.wait('@deleteCollection').its('response.statusCode').should('eq', 200)
    cy.contains(renamedCollection).should('not.exist')

    cy.contains('button', 'Visit site').parent().find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()
    cy.wait('@deleteHelpCenter').its('response.statusCode').should('eq', 200)

    cy.location('pathname').should('eq', listPath)
    cy.contains(siteName).should('not.exist')
  })
})
