import english from '../../../i18n/en-US.json'
import german from '../../../i18n/de-DE.json'

const standaloneGerman = {
  'globals.actions.create': 'Erstellen',
  'globals.actions.edit': 'Bearbeiten',
  'globals.actions.delete': 'Löschen',
  'globals.labels.new': 'Neu'
}
const agent = {
  id: 10001,
  first_name: 'Localization',
  last_name: 'Agent',
  email: 'localization@example.com',
  enabled: true,
  roles: ['Agent'],
  teams: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z'
}

describe('Standalone admin localization', () => {
  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
    cy.intercept('**/api/v1/**', { body: { data: [] } })
    cy.intercept('GET', '**/api/v1/agents/me', {
      body: {
        data: {
          ...agent,
          id: 1,
          roles: ['Admin'],
          permissions: ['business_hours:manage', 'sla:manage', 'users:manage', 'inboxes:manage']
        }
      }
    })
    cy.intercept('GET', '**/api/v1/agents', { body: { data: [agent] } })
    cy.intercept('GET', `**/api/v1/agents/${agent.id}`, { body: { data: agent } })
    cy.intercept('GET', '**/api/v1/business-hours/10001', {
      body: { data: { id: 10001, name: 'Office', is_always_open: true, hours: [], holidays: [] } }
    })
    cy.intercept('GET', '**/api/v1/sla/10001', {
      body: { data: { id: 10001, name: 'Support', first_response_time: '1h', notifications: [] } }
    })
  })

  for (const translated of [true, false]) {
    describe(
      translated ? 'German standalone translations' : 'English fallback before Crowdin',
      () => {
        const labels = translated ? standaloneGerman : english

        beforeEach(() => {
          // Mirror the server's English-first merge. Keep the German phrase keys,
          // including their {name} placeholders, to catch accidental reuse.
          const translations = { ...german }
          for (const key of Object.keys(standaloneGerman)) delete translations[key]
          cy.intercept('GET', '**/api/v1/config', {
            body: { data: { 'app.lang': 'de-DE' } }
          })
          cy.intercept('GET', '**/api/v1/lang/de-DE', {
            body: { ...english, ...translations, ...(translated ? standaloneGerman : {}) }
          })
        })

        for (const section of ['business-hours', 'sla']) {
          it(`uses standalone create and breadcrumb labels for ${section}`, () => {
            cy.visit(`/admin/${section}/new`)
            cy.get('form button[type="submit"]')
              .should('be.visible')
              .and('have.text', labels['globals.actions.create'])
            cy.get('nav[aria-label="breadcrumb"] li')
              .last()
              .should('be.visible')
              .and('have.text', labels['globals.labels.new'])

            cy.visit(`/admin/${section}/10001/edit`)
            cy.get('nav[aria-label="breadcrumb"] li')
              .last()
              .should('have.text', labels['globals.actions.edit'])
            cy.get('form button[type="submit"]').should(
              'have.text',
              german['globals.messages.save']
            )
          })
        }

        it('uses standalone agent actions and preserves edit navigation', () => {
          cy.visit('/admin/teams/agents')
          cy.contains('tr', agent.email).find('button[aria-haspopup="menu"]').click()
          cy.contains('[role="menuitem"]', labels['globals.actions.edit']).should('be.visible')
          cy.contains('[role="menuitem"]', labels['globals.actions.delete']).click()
          cy.get('[role="alertdialog"]').within(() => {
            cy.contains('button', labels['globals.actions.delete']).should('be.visible')
            cy.contains('button', german['globals.messages.cancel']).click()
          })
          cy.get('[role="alertdialog"]').should('not.exist')
          cy.contains('tr', agent.email).find('button[aria-haspopup="menu"]').click()
          cy.contains('[role="menuitem"]', labels['globals.actions.edit']).click()
          cy.location('pathname').should('eq', `/admin/teams/agents/${agent.id}/edit`)
        })
      }
    )
  }

  it('pluralizes the English sidebar while retaining the singular channel column', () => {
    cy.intercept('GET', '**/api/v1/config', { body: { data: { 'app.lang': 'en-US' } } })
    cy.intercept('GET', '**/api/v1/lang/en-US', { body: english })
    cy.visit('/admin/inboxes')
    cy.contains('[data-sidebar="menu-button"]', /^Channels$/).should('be.visible')
    cy.contains('th', /^Channel$/).should('be.visible')
  })
})
