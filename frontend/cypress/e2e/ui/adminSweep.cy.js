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
