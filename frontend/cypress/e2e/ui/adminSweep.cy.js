// The href list is read from the navigation source at runtime, so a new nav entry extends this spec.

const NAV_SOURCE = 'apps/main/src/constants/navigation.js'

const normalise = (p) => (p.length > 1 ? p.replace(/\/+$/, '') : p)

describe('Every navigable page loads', () => {
  let hrefs = []
  const failures = []
  let current = null

  // Record the page that threw and keep going, so one broken page does not hide the rest.
  Cypress.on('uncaught:exception', (err) => {
    failures.push(`${current}: ${err.message.split('\n')[0]}`)
    return false
  })

  before(() => {
    cy.readFile(NAV_SOURCE).then((src) => {
      hrefs = [...src.matchAll(/href:\s*'([^']+)'/g)].map((m) => m[1])
      expect(hrefs.length, 'nav hrefs discovered').to.be.greaterThan(10)
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('opens each one without a client-side error', () => {
    cy.wrap(hrefs).each((href) => {
      current = href
      cy.visit(href)
      // Landing anywhere else means the route died or the session broke.
      cy.location('pathname', { timeout: 15000 })
        .then((p) => normalise(p))
        .should('eq', normalise(href))
      cy.get('body').should('be.visible')
      cy.contains(/something went wrong|unexpected error/i).should('not.exist')
    })

    cy.then(() => {
      expect(failures, `pages that threw:\n${failures.join('\n')}`).to.be.empty
    })
  })
})

describe('Admin toolbar', () => {
  const attributes = {
    contact: [{ id: 101, name: 'Customer field', key: 'customer_field', applies_to: 'contact' }],
    conversation: [
      { id: 102, name: 'Conversation field', key: 'conversation_field', applies_to: 'conversation' }
    ]
  }

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
    cy.intercept('GET', '**/api/v1/custom-attributes*', (req) => {
      const rows = attributes[req.query.applies_to] || []
      req.reply({
        data: rows.map((row) => ({
          ...row,
          data_type: 'text',
          created_at: '2026-01-01T00:00:00Z',
          updated_at: '2026-01-01T00:00:00Z'
        }))
      })
    }).as('attributes')
    cy.visit('/admin/custom-attributes')
    cy.contains('[role="tab"]', /^Contact$/).click()
    cy.contains('tr', 'Customer field').should('be.visible')
  })

  it('keeps tabs left and search beside the action on the right', () => {
    cy.get('[data-admin-toolbar]').within(() => {
      cy.get('[role="tablist"]').then(($tabs) => {
        cy.get('input[aria-label="Search"]').then(($search) => {
          expect($tabs[0].getBoundingClientRect().right).to.be.at.most(
            $search[0].getBoundingClientRect().left
          )
        })
      })
      cy.contains('button', 'New custom attribute').should('be.visible')
    })
    cy.get('input[aria-label="Search"]').should('have.length', 1)
  })

  it('expands search on click and keeps a query expanded after blur', () => {
    cy.get('.admin-toolbar-search')
      .invoke('width')
      .then((compactWidth) => {
        cy.get('input[aria-label="Search"]').click()
        cy.get('.admin-toolbar-search').invoke('width').should('be.greaterThan', compactWidth)
        cy.get('input[aria-label="Search"]').type('missing')
        cy.contains('No results found').should('be.visible')
        cy.get('input[aria-label="Search"]').blur()
        cy.get('.admin-toolbar-search').invoke('width').should('be.greaterThan', compactWidth)
        cy.get('button[aria-label="Clear search"]').click()
        cy.contains('tr', 'Customer field').should('be.visible')
      })
  })

  it('clears the previous query when switching attribute tabs', () => {
    cy.get('input[aria-label="Search"]').type('Customer')
    cy.contains('[role="tab"]', /^Conversation$/).click()
    cy.get('input[aria-label="Search"]').should('have.value', '')
    cy.contains('tr', 'Conversation field').should('be.visible')
  })

  it('keeps every control reachable on a narrow screen', () => {
    cy.viewport(390, 844)
    cy.get('input[aria-label="Search"]').click()
    cy.get('input[aria-label="Search"]').type('Customer')
    cy.contains('button', 'New custom attribute').should('be.visible')
    cy.contains('[role="tab"]', /^Conversation$/).click()
    cy.contains('tr', 'Conversation field').should('be.visible')
    cy.document().then((doc) => {
      expect(doc.documentElement.scrollWidth).to.be.at.most(doc.documentElement.clientWidth)
    })
  })
})
