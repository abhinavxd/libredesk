// The steps run in order and share the inbox created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Theme ${stamp}`
const brandName = `Cypress Theme Brand ${stamp}`
const listPath = '/admin/inboxes'
const newPath = '/admin/inboxes/new'

const lightPrimary = '#123456'
const darkPrimary = '#abcdef'

const openNewForm = () => {
  cy.visit(newPath)
  cy.contains('Create a live chat inbox').click()
  cy.openInboxSection('General')
}

describe('Live chat theme and branding', () => {
  let inboxId

  beforeEach(() => {
    Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))
    cy.viewport(1280, 900)
    cy.login()
  })

  it('saves a separate primary color for each theme', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.brand_name"]').type(brandName)

    cy.openInboxSection('Theme')
    cy.get('#theme-system').click()

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.colors.primary"]')
      .invoke('val', lightPrimary)
      .trigger('input')

    cy.editBrandingTheme('Dark')
    cy.get('input[name="config.branding.dark.colors.primary"]')
      .invoke('val', darkPrimary)
      .trigger('input')

    cy.get('button[type="submit"]').click()
    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
      const branding = response.body.data.config.branding
      expect(branding.light.colors.primary, 'light primary').to.eq(lightPrimary)
      expect(branding.dark.colors.primary, 'dark primary').to.eq(darkPrimary)
      expect(response.body.data.config.theme, 'theme').to.eq('system')
    })
  })

  it('loads each theme back into its own tab', () => {
    expect(inboxId, 'inbox from the create step').to.be.a('number')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Branding')

    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.colors.primary"]').should('have.value', lightPrimary)

    cy.editBrandingTheme('Dark')
    cy.get('input[name="config.branding.dark.colors.primary"]').should('have.value', darkPrimary)
  })

  it('seeds a dark background when the theme is switched away from light', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createSeeded')

    openNewForm()
    cy.get('input[name="name"]').type(`${inboxName} seeded`)
    cy.get('input[name="config.brand_name"]').type(brandName)

    cy.openInboxSection('Theme')
    cy.get('#theme-dark').click()

    cy.openInboxSection('Branding')
    cy.get('input[name="config.branding.dark.home_screen.background.color"]').should(
      'have.value',
      '#1a1a1e'
    )
    cy.get('#text-white').should('have.attr', 'data-state', 'checked')

    cy.get('button[type="submit"]').click()
    cy.wait('@createSeeded').then(({ response }) => {
      const branding = response.body.data.config.branding
      expect(branding.dark.home_screen.header_text_color, 'dark header text').to.eq('white')
      expect(branding.light.home_screen.header_text_color, 'light header text').to.eq('black')
      cy.api('DELETE', `/api/v1/inboxes/${response.body.data.id}`, null, {
        failOnStatusCode: false
      })
    })
  })

  it('rejects a launcher size and icon scale outside their range', () => {
    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Launcher position')

    cy.get('input[name="config.launcher.size"]').clear().type('120')
    cy.get('button[type="submit"]').click()
    cy.contains('40').should('exist')

    cy.get('input[name="config.launcher.size"]').clear().type('64')
    cy.get('input[name="config.launcher.icon_scale"]').clear().type('10')
    cy.get('button[type="submit"]').click()
    cy.contains('40').should('exist')
  })

  it('persists the launcher size and icon scale', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Launcher position')
    cy.get('input[name="config.launcher.size"]').clear().type('72')
    cy.get('input[name="config.launcher.icon_scale"]').clear().type('60')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.config.launcher.size, 'launcher size').to.eq(72)
      expect(response.body.data.config.launcher.icon_scale, 'icon scale').to.eq(60)
    })
  })

  after(() => {
    if (inboxId) {
      cy.login()
      cy.api('DELETE', `/api/v1/inboxes/${inboxId}`, null, { failOnStatusCode: false })
    }
  })
})
