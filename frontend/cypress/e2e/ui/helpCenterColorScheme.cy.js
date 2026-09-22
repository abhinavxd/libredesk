const stamp = Date.now()
const listPath = '/admin/help-center'
const LIGHT_COLOR = '#26583f'
const DARK_COLOR = '#46ce8e'
const LIGHT_LOGO = '/uploads/cypress-light.png'
const DARK_LOGO = '/uploads/cypress-dark.png'

const createHelpCenter = (key, theme, template = 'classic') =>
  cy
    .api('POST', '/api/v1/help-centers', {
      name: `Cypress scheme ${key} ${stamp}`,
      slug: `cypress-scheme-${key}-${stamp}`,
      page_title: 'Help',
      default_locale: 'en',
      allowed_locales: ['en'],
      template,
      ...(theme ? { theme } : {})
    })
    .its('body.data')

// The head script reads the device setting while the page parses, so the stub must be in place before load.
const visitAs = (hc, deviceDark) =>
  cy.visit(`/hc/${hc.slug}/en`, {
    onBeforeLoad(win) {
      cy.stub(win, 'matchMedia').callsFake((query) => ({
        matches: query.includes('dark') ? deviceDark : false,
        media: query,
        onchange: null,
        addListener() {},
        removeListener() {},
        addEventListener() {},
        removeEventListener() {},
        dispatchEvent: () => false
      }))
    }
  })

const html = () => cy.get('html')
const accent = () =>
  cy.document().then((doc) =>
    doc.defaultView.getComputedStyle(doc.documentElement).getPropertyValue('--hc-accent').trim()
  )

describe('Help center color scheme', () => {
  const created = {}

  before(() => {
    cy.login()
    createHelpCenter('form').then((hc) => (created.form = hc))
    createHelpCenter('light', { color_scheme: 'light', color: LIGHT_COLOR }).then(
      (hc) => (created.light = hc)
    )
    createHelpCenter('dark', {
      color_scheme: 'dark',
      color: LIGHT_COLOR,
      color_dark: DARK_COLOR,
      logo_url: LIGHT_LOGO,
      logo_url_dark: DARK_LOGO
    }).then((hc) => (created.dark = hc))
    createHelpCenter('system', {
      color_scheme: 'system',
      color: LIGHT_COLOR,
      color_dark: DARK_COLOR
    }).then((hc) => (created.system = hc))
    createHelpCenter(
      'docs',
      { color_scheme: 'system', color: LIGHT_COLOR, color_dark: DARK_COLOR },
      'docs'
    ).then((hc) => (created.docs = hc))
  })

  beforeEach(() => {
    cy.viewport(1400, 900)
    cy.login()
  })

  describe('admin form', () => {
    const openBrand = () => {
      cy.visit(`${listPath}/${created.form.id}/customize`)
      cy.contains('[role="tab"]', 'Appearance').click()
      cy.contains('button[aria-expanded]', 'Brand').should('have.attr', 'aria-expanded', 'true')
    }
    const schemeTab = (label) => cy.contains('[role="tab"]', label)
    const setColor = (name, value) =>
      cy.get(`input[name="${name}"]`).invoke('val', value).trigger('input')
    const preview = () => cy.get('iframe[title="Help center"]')

    it('starts a help center without a saved scheme on Light', () => {
      openBrand()
      cy.get('#hc-scheme-light').should('have.attr', 'data-state', 'checked')
      schemeTab('Light').should('have.attr', 'data-state', 'active')
      cy.get('input[name="theme.logo_url"]').should('be.visible')
    })

    it('edits light and dark branding on their own tabs and saves both', () => {
      cy.intercept('PUT', `**/api/v1/help-centers/${created.form.id}`).as('update')
      openBrand()

      cy.get('input[name="theme.logo_url"]').clear().type(LIGHT_LOGO)
      setColor('theme.color', LIGHT_COLOR)

      schemeTab('Dark').click()
      schemeTab('Dark').should('have.attr', 'data-state', 'active')
      cy.get('input[name="theme.logo_url"]').should('not.exist')
      cy.get('input[name="theme.favicon"]').should('be.visible')
      cy.get('input[name="theme.logo_url_dark"]').should('have.value', '').type(DARK_LOGO)
      setColor('theme.color_dark', DARK_COLOR)
      preview().should('have.attr', 'srcdoc').and('match', /<html[^>]*class="hc-dark"/)

      schemeTab('Light').click()
      cy.get('input[name="theme.logo_url"]').should('have.value', LIGHT_LOGO)
      cy.get('input[name="theme.color"]').should('have.value', LIGHT_COLOR)
      preview().should('have.attr', 'srcdoc').and('not.match', /<html[^>]*class="hc-dark"/)

      cy.contains('label', 'Match system').click()
      cy.get('button[type="submit"]').click()
      cy.wait('@update').its('response.statusCode').should('eq', 200)

      cy.api('GET', `/api/v1/help-centers/${created.form.id}`)
        .its('body.data.theme')
        .should('include', {
          color_scheme: 'system',
          color: LIGHT_COLOR,
          color_dark: DARK_COLOR,
          logo_url: LIGHT_LOGO,
          logo_url_dark: DARK_LOGO
        })
    })

    it('loads saved dark branding back into the Dark tab', () => {
      openBrand()
      cy.get('#hc-scheme-system').should('have.attr', 'data-state', 'checked')
      schemeTab('Dark').click()
      cy.get('input[name="theme.logo_url_dark"]').should('have.value', DARK_LOGO)
      cy.get('input[name="theme.color_dark"]').should('have.value', DARK_COLOR)
    })

    it('switches to the Dark tab and a dark preview when Dark is picked', () => {
      openBrand()
      cy.contains('label', /^Dark$/).click()
      schemeTab('Dark').should('have.attr', 'data-state', 'active')
      preview().should('have.attr', 'srcdoc').and('match', /<html[^>]*class="hc-dark"/)

      cy.contains('label', /^Light$/).click()
      schemeTab('Light').should('have.attr', 'data-state', 'active')
      preview().should('have.attr', 'srcdoc').and('not.match', /<html[^>]*class="hc-dark"/)
    })
  })

  describe('public page', () => {
    it('keeps a light site light on a dark device, with no toggle', () => {
      visitAs(created.light, true)
      html().should('not.have.class', 'hc-dark')
      cy.get('[data-hc-theme]').should('not.exist')
      accent().should('eq', LIGHT_COLOR)
    })

    it('shows a dark site dark on a light device, with the dark color and logo', () => {
      visitAs(created.dark, false)
      html().should('have.class', 'hc-dark')
      cy.get('[data-hc-theme]').should('not.exist')
      accent().should('eq', DARK_COLOR)
      cy.get(`img.hc-logo-dark[src="${DARK_LOGO}"]`).should('be.visible')
      cy.get(`img.hc-logo-light[src="${LIGHT_LOGO}"]`).should('not.be.visible')
    })

    it('follows the device on a system site', () => {
      visitAs(created.system, true)
      html().should('have.class', 'hc-dark')
      accent().should('eq', DARK_COLOR)

      visitAs(created.system, false)
      html().should('not.have.class', 'hc-dark')
      accent().should('eq', LIGHT_COLOR)
    })

    it('lets the visitor override the device and remembers it across reloads', () => {
      visitAs(created.system, false)
      html().should('not.have.class', 'hc-dark')

      cy.get('[data-hc-theme]').filter(':visible').first().click()
      html().should('have.class', 'hc-dark')
      cy.window().its('localStorage').invoke('getItem', 'libredesk-hc-theme').should('eq', 'dark')

      visitAs(created.system, false)
      html().should('have.class', 'hc-dark')

      cy.get('[data-hc-theme]').filter(':visible').first().click()
      html().should('not.have.class', 'hc-dark')

      visitAs(created.system, true)
      html().should('not.have.class', 'hc-dark')
    })

    it('applies the same rules on the docs template', () => {
      visitAs(created.docs, true)
      html().should('have.class', 'hc-dark')
      accent().should('eq', DARK_COLOR)
      cy.get('[data-hc-theme]').filter(':visible').first().click()
      html().should('not.have.class', 'hc-dark')
      visitAs(created.docs, true)
      html().should('not.have.class', 'hc-dark')
    })
  })

  after(() => {
    cy.login()
    Object.values(created).forEach((hc) => {
      cy.api('DELETE', `/api/v1/help-centers/${hc.id}`, null, { failOnStatusCode: false })
    })
  })
})
