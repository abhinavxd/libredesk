describe('API: help center color scheme', () => {
  const stamp = Date.now()
  const created = {}

  const createHelpCenter = (key, theme) =>
    cy
      .api('POST', '/api/v1/help-centers', {
        name: `Color scheme ${key} ${stamp}`,
        slug: `api-hc-scheme-${key}-${stamp}`,
        page_title: 'Help',
        default_locale: 'en',
        allowed_locales: ['en'],
        ...(theme ? { theme } : {})
      })
      .then(({ body }) => {
        created[key] = body.data
      })

  const page = (key, query = '') => cy.request(`/hc/${created[key].slug}/en${query}`).its('body')

  before(() => {
    cy.login()
    createHelpCenter('none')
    createHelpCenter('light', { color_scheme: 'light', color: '#26583f' })
    createHelpCenter('dark', {
      color_scheme: 'dark',
      color: '#26583f',
      color_dark: '#46ce8e',
      logo_url: '/uploads/light.png',
      logo_url_dark: '/uploads/dark.png'
    })
    createHelpCenter('system', { color_scheme: 'system', color: '#26583f' })
    createHelpCenter('footer', {
      color_scheme: 'system',
      color: '#26583f',
      footer: {
        background_color: '#e3e3e3',
        text_color: '#000000',
        background_color_dark: '#101010',
        text_color_dark: '#e5e5e5'
      }
    })
    createHelpCenter('light-footer', {
      color_scheme: 'system',
      color: '#26583f',
      footer: { background_color: '#e3e3e3', text_color: '#000000' }
    })
  })

  beforeEach(() => {
    cy.login()
  })

  it('fills the scheme defaults for a help center saved without them', () => {
    cy.api('GET', `/api/v1/help-centers/${created.none.id}`).then(({ body }) => {
      expect(body.data.theme.color_scheme).to.eq('light')
      expect(body.data.theme.color_dark).to.eq(body.data.theme.color)
    })
    cy.api('GET', `/api/v1/help-centers/${created.light.id}`)
      .its('body.data.theme.color_dark')
      .should('eq', '#26583f')
  })

  it('saves an unknown scheme as light', () => {
    const { name, slug, page_title, default_locale, allowed_locales } = created.none
    cy.api('PUT', `/api/v1/help-centers/${created.none.id}`, {
      name,
      slug,
      page_title,
      default_locale,
      allowed_locales,
      theme: { color_scheme: 'purple' }
    })
      .its('body.data.theme.color_scheme')
      .should('eq', 'light')
  })

  it('renders a light site without dark mode or a toggle', () => {
    page('light').then((html) => {
      expect(html).not.to.match(/<html[^>]*class="hc-dark"/)
      expect(html).not.to.contain('data-hc-theme')
      expect(html).not.to.contain('prefers-color-scheme')
    })
  })

  it('renders a dark site dark, with the dark color and logo', () => {
    page('dark').then((html) => {
      expect(html).to.match(/<html[^>]*class="hc-dark"/)
      expect(html).to.contain(':root.hc-dark { --hc-accent: #46ce8e; }')
      expect(html).to.contain('/uploads/dark.png')
      expect(html).not.to.contain('data-hc-theme')
    })
  })

  it('renders a system site with the toggle and the device check', () => {
    page('system').then((html) => {
      expect(html).not.to.match(/<html[^>]*class="hc-dark"/)
      expect(html).to.contain('data-hc-theme')
      expect(html).to.contain('prefers-color-scheme')
    })
  })

  it('uses the dark footer colors in dark mode', () => {
    page('footer').then((html) => {
      expect(html).to.contain('--hc-footer-bg:#e3e3e3;--hc-footer-text:#000000;')
      expect(html).to.contain(
        ':root.hc-dark { --hc-accent: #26583f; --hc-footer-bg:#101010;--hc-footer-text:#e5e5e5; }'
      )
    })
  })

  it('falls back to the default dark footer when no dark footer color is set', () => {
    page('light-footer').should(
      'contain',
      ':root.hc-dark { --hc-accent: #26583f; --hc-footer-bg:initial;--hc-footer-text:initial; }'
    )
  })

  it('lets an embedded article follow the widget instead of the site', () => {
    page('dark', '?embed=1').should('not.match', /<html[^>]*class="hc-dark"/)
    page('light', '?embed=1&theme=dark').should('match', /<html[^>]*class="hc-dark"/)
    page('system', '?embed=1').then((html) => {
      expect(html).not.to.contain('data-hc-theme')
      expect(html).not.to.contain('prefers-color-scheme')
    })
  })

  after(() => {
    cy.login()
    Object.values(created).forEach((hc) => {
      cy.api('DELETE', `/api/v1/help-centers/${hc.id}`, null, { failOnStatusCode: false })
    })
  })
})
