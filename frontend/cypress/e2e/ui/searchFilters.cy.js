// Every filter change is written to the URL, so the query string shows the filters the search used.
const term = 'zzsearchfilters'
const searchPath = `/inboxes/search?q=${term}`
const appliedRange = 'created=2026-01-01,2026-01-31'

const filterTrigger = (label) => cy.contains('main button[role="combobox"]', label)
const pickOption = (label) => cy.get('[role="option"]').contains(label).click()
const dayCell = (date) => cy.get(`[data-reka-calendar-cell-trigger][data-value="${date}"]`).first()

describe('Search filters', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('clears a filter when its current value is picked again', () => {
    cy.visit(searchPath)
    filterTrigger('Status').click()
    pickOption('Open')
    cy.location('search').should('match', /status=\d+/)
    cy.contains('main button', 'Clear filters').should('be.visible')

    filterTrigger('Open').click()
    pickOption('Open')
    cy.location('search').should('not.include', 'status=')
    filterTrigger('Status').should('be.visible')
    cy.contains('main button', 'Clear filters').should('not.exist')
  })

  it('clears every filter at once', () => {
    cy.visit(`${searchPath}&assignee=none&${appliedRange}`)
    cy.contains('main button', 'Clear filters').click()
    cy.location('search').should('eq', `?q=${term}`)
  })

  it('keeps the applied date range until the new range is complete', () => {
    cy.visit(`${searchPath}&${appliedRange}`)
    cy.contains('main button', 'Jan 01, 2026 - Jan 31, 2026').click()

    dayCell('2026-01-10').click()
    cy.location('search').should('include', appliedRange)

    dayCell('2026-01-20').click()
    cy.location('search').should('include', 'created=2026-01-10,2026-01-20')
    cy.contains('main button', 'Jan 10, 2026 - Jan 20, 2026').should('be.visible')
  })

  it('clears only the date range', () => {
    cy.visit(`${searchPath}&assignee=none&${appliedRange}`)
    cy.contains('main button', 'Jan 01, 2026 - Jan 31, 2026').click()
    cy.get('[role="dialog"]').contains('button', 'Clear').click()
    cy.location('search').should('not.include', 'created=')
    cy.location('search').should('include', 'assignee=none')
  })
})
