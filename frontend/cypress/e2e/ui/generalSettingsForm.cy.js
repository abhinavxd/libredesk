describe('General settings form', () => {
  const stamp = Date.now()
  const siteName = `Cypress Desk ${stamp}`
  const path = '/admin/general'

  // The PUT replaces the whole record, so the restore has to send every key.
  const generalKeys = [
    'app.site_name',
    'app.lang',
    'app.max_file_upload_size',
    'app.favicon_url',
    'app.logo_url',
    'app.root_url',
    'app.allowed_file_upload_extensions',
    'app.timezone',
    'app.business_hours_id',
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
    cy.get('input[name="favicon_url"]').should('have.value', original['app.favicon_url'])
    cy.get('select[name="timezone"]').should('have.value', original['app.timezone'])
  })

  it('saves a changed site name and reloads it', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[name="site_name"]')
      .should('have.value', original['app.site_name'])
      .clear()
      .type(siteName)
    cy.get('button[type="submit"]').click()

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
    cy.get('button[type="submit"]').click()

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
      .type('definitely not a url')
    cy.get('button[type="submit"]').click()

    cy.contains('Root URL should be a valid URL').should('exist')
    cy.get('@saveGeneral.all').should('have.length', 0)
  })

  it('rejects an empty site name', () => {
    cy.intercept('PUT', '**/api/v1/settings/general').as('saveGeneral')

    cy.visit(path)
    cy.get('input[name="site_name"]').should('not.have.value', '').clear()
    cy.get('button[type="submit"]').click()

    cy.contains('Site name should be at least 1 character').should('exist')
    cy.get('@saveGeneral.all').should('have.length', 0)
  })
})
