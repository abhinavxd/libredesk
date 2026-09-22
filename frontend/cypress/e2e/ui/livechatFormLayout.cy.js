// The steps run in order and share the inbox created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Layout ${stamp}`
const brandName = `Cypress Layout Brand ${stamp}`
const listPath = '/admin/inboxes'
const newPath = '/admin/inboxes/new'

const TABS = ['General', 'Appearance', 'Content', 'Conversations', 'Setup']

const SECTIONS = {
  General: ['General', 'Conversation continuity email inbox'],
  Appearance: ['Theme', 'Branding', 'Launcher position'],
  Content: ['Messages', 'Notice banner', 'Home screen apps', 'Help center', 'Proactive messages'],
  Conversations: ['Features', 'Office hours', 'Pre-chat form', 'Users'],
  Setup: ['Installation', 'Identity verification', 'JavaScript API', 'Security']
}

const openTab = (label) => cy.get('form').contains('[role="tab"]', new RegExp(`^\\s*${label}\\s*$`)).click()

const sectionTrigger = (label) =>
  cy.get('form').contains('button[aria-expanded]', new RegExp(`^\\s*${label}\\s*$`))

const openNewForm = () => {
  cy.visit(newPath)
  cy.contains('Create a live chat inbox').click()
}

describe('Live chat inbox form layout', () => {
  let inboxId

  beforeEach(() => {
    Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))
    cy.viewport(1280, 900)
    cy.login()
  })

  it('creates an inbox with fields spread across every tab', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.openInboxSection('General')
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.brand_name"]').type(brandName)

    cy.openInboxSection('Notice banner')
    cy.inboxSection('Notice banner').find('button[role="switch"]').click()
    cy.get('textarea[name="config.notice_banner.text"]').clear().type('Cypress layout notice')

    cy.openInboxSection('Users')
    cy.get('textarea[name="config.visitors.quick_replies"]').clear().type('One\nTwo\nThree')

    cy.openInboxSection('Security')
    cy.get('input[name="config.session_duration"]').clear().type('6h')

    cy.get('button[type="submit"]').click()
    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
      const config = response.body.data.config
      expect(config.notice_banner.text, 'notice text').to.eq('Cypress layout notice')
      expect(config.visitors.quick_replies, 'quick replies').to.deep.eq(['One', 'Two', 'Three'])
      expect(config.session_duration, 'session duration').to.eq('6h')
    })
  })

  it('shows every tab and its sections, all collapsed on load', () => {
    cy.visit(`${listPath}/${inboxId}/edit`)

    TABS.forEach((tab) => {
      openTab(tab)
      SECTIONS[tab].forEach((section) => {
        sectionTrigger(section).should('be.visible').and('have.attr', 'aria-expanded', 'false')
      })
    })
  })

  it('opens one section at a time', () => {
    cy.visit(`${listPath}/${inboxId}/edit`)

    cy.openInboxSection('General')
    sectionTrigger('General').should('have.attr', 'aria-expanded', 'true')

    cy.openInboxSection('Conversation continuity email inbox')
    sectionTrigger('General').should('have.attr', 'aria-expanded', 'false')

    sectionTrigger('Conversation continuity email inbox').click()
    sectionTrigger('Conversation continuity email inbox').should(
      'have.attr',
      'aria-expanded',
      'false'
    )
  })

  it('keeps values from collapsed sections on submit', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Office hours')
    cy.switchField('Show office hours in chat').click()
    cy.get('input[name="config.chat_reply_expectation_message"]')
      .clear()
      .type('Cypress replies fast')

    // Everything typed above is now inside a collapsed section.
    cy.openInboxSection('Security')
    cy.get('button[type="submit"]').click()

    cy.wait('@updateInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.config.chat_reply_expectation_message).to.eq('Cypress replies fast')
    })
  })

  it('opens the section holding the first error on a failed submit', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Security')
    cy.get('input[name="config.session_duration"]').clear().type('not-a-duration')

    cy.openInboxSection('Messages')
    cy.get('button[type="submit"]').click()

    cy.get('form')
      .contains('[role="tab"]', /^\s*Setup\s*$/)
      .should('have.attr', 'data-state', 'active')
    sectionTrigger('Security').should('have.attr', 'aria-expanded', 'true')
    cy.get('input[name="config.session_duration"]').should('be.visible')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('lands on the right tab for a link that uses an old tab name', () => {
    cy.visit(`${listPath}/${inboxId}/edit?tab=appearance`)
    cy.get('form')
      .contains('[role="tab"]', /^\s*Appearance\s*$/)
      .should('have.attr', 'data-state', 'active')
    sectionTrigger('Branding').should('have.attr', 'aria-expanded', 'true')

    cy.visit(`${listPath}/${inboxId}/edit?tab=installation`)
    cy.get('form')
      .contains('[role="tab"]', /^\s*Setup\s*$/)
      .should('have.attr', 'data-state', 'active')
    sectionTrigger('Installation').should('have.attr', 'aria-expanded', 'true')
  })

  it('keeps both branding themes when the light and dark tabs are switched', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Theme')
    cy.get('#theme-system').click()

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('#bg-gradient').click()
    cy.get('#text-black').click()

    cy.editBrandingTheme('Dark')
    cy.get('#text-white').click()

    // Light's background type and text color must survive the round trip.
    cy.editBrandingTheme('Light')
    cy.get('#bg-gradient').should('have.attr', 'data-state', 'checked')
    cy.get('#text-black').should('have.attr', 'data-state', 'checked')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      const branding = response.body.data.config.branding
      expect(branding.light.home_screen.background.type, 'light background').to.eq('gradient')
      expect(branding.light.home_screen.header_text_color, 'light text').to.eq('black')
      expect(branding.dark.home_screen.header_text_color, 'dark text').to.eq('white')
    })
  })

  it('keeps the preview and the branding tabs on the same theme', () => {
    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.get('[data-preview-theme="light"]').should('be.visible')

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Dark')
    cy.get('[data-preview-theme="dark"]').should('have.attr', 'data-state', 'active')

    cy.get('[data-preview-theme="light"]').click()
    cy.inboxSection('Branding')
      .find('[role="tab"][data-state="active"]')
      .should('contain.text', 'Light')
  })

  it('rejects more than six quick replies', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Users')
    cy.get('textarea[name="config.visitors.quick_replies"]')
      .clear()
      .type('a\nb\nc\nd\ne\nf\ng')

    cy.get('button[type="submit"]').click()
    cy.inboxSection('Users').contains('Add up to 6 quick replies.').should('be.visible')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('shows the inbox uuid in the installation and identity snippets', () => {
    cy.api('GET', `/api/v1/inboxes/${inboxId}`).then(({ body }) => {
      const uuid = body.data.uuid
      cy.visit(`${listPath}/${inboxId}/edit`)

      cy.openInboxSection('Installation')
      cy.inboxSection('Installation').should('contain', uuid)

      cy.openInboxSection('Identity verification')
      cy.inboxSection('Identity verification').should('contain', 'external_user_id')

      cy.openInboxSection('JavaScript API')
      cy.inboxSection('JavaScript API').should('contain', 'Libredesk')
    })
  })

  after(() => {
    if (inboxId) {
      cy.login()
      cy.api('DELETE', `/api/v1/inboxes/${inboxId}`, null, { failOnStatusCode: false })
    }
  })
})
