describe('General settings form', () => {
  const stamp = Date.now()
  const siteName = `Cypress Desk ${stamp}`
  const path = '/admin/general'
  const onePixelPNG =
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII='

  // The PUT replaces the whole record, so the restore has to send every key.
  const generalKeys = [
    'app.site_name',
    'app.lang',
    'app.max_file_upload_size',
    'app.logo_url',
    'app.root_url',
    'app.allowed_file_upload_extensions',
    'app.timezone',
    'app.show_conversation_subject'
  ]

  let original

  const pickOption = (field, optionText) => {
    cy.get(`select[name="${field}"]`).siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains(optionText).click()
  }

  before(() => {
    cy.login()
    cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
      original = Object.fromEntries(generalKeys.map((key) => [key, body.data[key]]))
      expect(original['app.site_name'], 'captured site name').to.be.a('string')
    })
  })

  after(() => {
    cy.login()
    cy.api('PUT', '/api/v1/settings/general', original)
    cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
      expect(body.data['app.site_name']).to.eq(original['app.site_name'])
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('loads the stored values into the form', () => {
    cy.visit(path)

    cy.get('input[name="site_name"]').should('have.value', original['app.site_name'])
    cy.get('input[name="root_url"]').should('have.value', original['app.root_url'])
    cy.get('select[name="timezone"]').should('have.value', original['app.timezone'])
  })

  it('saves a changed site name and reloads it', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[name="site_name"]')
      .should('have.value', original['app.site_name'])
      .clear()
    cy.get('input[name="site_name"]').type(siteName)
    cy.get('input[name="site_name"]').closest('form').find('button[type="submit"]').click()

    cy.wait('@saveGeneral').its('response.statusCode').should('eq', 200)
    cy.contains('Changes saved').should('exist')

    cy.visit(path)
    cy.get('input[name="site_name"]').should('have.value', siteName)
    cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
      expect(body.data['app.site_name']).to.eq(siteName)
    })
  })

  it('saves a changed timezone and reloads it', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('select[name="timezone"]').should('have.value', original['app.timezone'])
    pickOption('timezone', 'UTC (UTC+00:00)')
    cy.get('input[name="site_name"]').closest('form').find('button[type="submit"]').click()

    cy.wait('@saveGeneral').its('response.statusCode').should('eq', 200)

    cy.visit(path)
    cy.get('select[name="timezone"]').should('have.value', 'UTC')
    cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
      expect(body.data['app.timezone']).to.eq('UTC')
    })
  })

  it('rejects a malformed root URL', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[name="root_url"]')
      .should('not.have.value', '')
      .clear()
    cy.get('input[name="root_url"]').type('definitely not a url')
    cy.get('input[name="site_name"]').closest('form').find('button[type="submit"]').click()

    cy.contains('Root URL should be a valid URL').should('exist')
    cy.get('@saveGeneral.all').should('have.length', 0)
  })

  it('saves a blank site name as the Fernmail default', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[name="site_name"]').clear()
    cy.get('input[name="site_name"]').should('have.attr', 'placeholder', 'Fernmail')
    cy.get('input[name="site_name"]').closest('form').find('button[type="submit"]').click()

    cy.wait('@saveGeneral').its('response.statusCode').should('eq', 200)
    cy.visit(path)
    cy.title().should('match', / - Fernmail$/)
  })

  it('uploads a site logo that becomes the favicon', () => {
    // The upload is deliberately not intercepted: Cypress's proxy re-encodes an
    // intercepted multipart body as UTF-8, which corrupts the PNG's bytes.
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[type="file"][accept^="image/png"]').selectFile(
      { contents: Cypress.Buffer.from(onePixelPNG, 'base64'), fileName: 'logo.png', mimeType: 'image/png' },
      { force: true }
    )
    cy.get('input[name="site_name"]')
      .closest('form')
      .find('img[src^="/uploads/"]')
      .should('have.attr', 'src')
      .and('match', /^\/uploads\/[0-9a-f-]{36}$/)
    cy.get('input[name="site_name"]').closest('form').find('button[type="submit"]').click()
    cy.wait('@saveGeneral').its('response.statusCode').should('eq', 200)

    cy.api('GET', '/api/v1/settings/general').then(({ body }) => {
      expect(body.data['app.logo_url']).to.match(/^\/uploads\/[0-9a-f-]{36}$/)
      cy.get('link#app-favicon').should('have.attr', 'href', body.data['app.logo_url'])
    })
  })

  it('rejects an SVG site logo', () => {
    cy.visit(path)
    cy.get('input[type="file"][accept^="image/png"]').selectFile(
      { contents: Cypress.Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>'), fileName: 'logo.svg' },
      { force: true }
    )
    cy.contains('The logo must be a PNG, JPG, WebP or ICO image').should('exist')
  })
})
