// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const hoursName = `Cypress Business Hours ${stamp}`
const renamedHours = `Cypress Business Hours ${stamp} edited`
const holidayName = `Cypress Holiday ${stamp}`
const newPath = '/admin/business-hours/new'
const listPath = '/admin/business-hours'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// A weekday row holds its checkbox label plus the open and close time inputs.
const dayRow = (day) => cy.get(`label[for="${day}"]`).parent().parent()

const setDay = (day, open, close) => {
  cy.get(`button[role="checkbox"]#${day}`).click()
  dayRow(day).find('input[type="time"]').eq(0).clear().type(open)
  dayRow(day).find('input[type="time"]').eq(1).clear().type(close)
}

describe('Business hours form', () => {
  let hoursId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates business hours with custom weekday hours and a holiday', () => {
    cy.intercept('POST', '**/api/v1/business-hours').as('createHours')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(hoursName)
    cy.get('input[name="description"]').type('Created by the business hours form spec')

    cy.contains('label', 'Custom business hours').click()
    setDay('Monday', '08:30', '16:45')
    setDay('Wednesday', '10:00', '18:00')

    cy.contains('button', 'New holiday').click()
    cy.get('#holiday_name').type(holidayName)
    cy.contains('button', 'Pick a date').click()
    cy.get('[data-radix-vue-calendar-cell-trigger]:not([data-outside-view])')
      .contains('15')
      .click()
    cy.get('[role="dialog"]').contains('button', 'Add').click()
    cy.contains('td', holidayName).should('exist')

    cy.get('button[type="submit"]').click()
    cy.wait('@createHours').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      expect(request.body.hours).to.deep.eq({
        Monday: { open: '08:30', close: '16:45' },
        Wednesday: { open: '10:00', close: '18:00' }
      })
      expect(request.body.holidays).to.have.length(1)
      expect(request.body.holidays[0].name).to.eq(holidayName)
      hoursId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(hoursName)
    cy.contains(hoursName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(hoursId, 'business hours from the create step').to.be.a('number')

    cy.visit(`${listPath}/${hoursId}/edit`)

    cy.get('input[name="name"]').should('have.value', hoursName)
    cy.get('input[name="description"]').should(
      'have.value',
      'Created by the business hours form spec'
    )

    cy.get('button[role="checkbox"]#Monday').should('have.attr', 'data-state', 'checked')
    cy.get('button[role="checkbox"]#Wednesday').should('have.attr', 'data-state', 'checked')
    cy.get('button[role="checkbox"]#Tuesday').should('have.attr', 'data-state', 'unchecked')

    dayRow('Monday').find('input[type="time"]').eq(0).should('have.value', '08:30')
    dayRow('Monday').find('input[type="time"]').eq(1).should('have.value', '16:45')
    dayRow('Wednesday').find('input[type="time"]').eq(0).should('have.value', '10:00')
    dayRow('Wednesday').find('input[type="time"]').eq(1).should('have.value', '18:00')

    cy.contains('td', holidayName).should('exist')
  })

  it('persists a changed name', () => {
    cy.intercept('PUT', `**/api/v1/business-hours/${hoursId}`).as('updateHours')

    cy.visit(`${listPath}/${hoursId}/edit`)
    cy.get('input[name="name"]').should('have.value', hoursName).clear().type(renamedHours)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateHours').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${hoursId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedHours)
  })

  it('persists a changed closing time', () => {
    cy.intercept('PUT', `**/api/v1/business-hours/${hoursId}`).as('updateHours')

    cy.visit(`${listPath}/${hoursId}/edit`)
    dayRow('Monday').find('input[type="time"]').eq(1).should('have.value', '16:45')
    dayRow('Monday').find('input[type="time"]').eq(1).clear().type('20:15')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateHours').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${hoursId}/edit`)
    dayRow('Monday').find('input[type="time"]').eq(1).should('have.value', '20:15')
  })

  it('keeps an edited name when a weekday time is changed too', () => {
    cy.visit(`${listPath}/${hoursId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedHours).clear().type(`${renamedHours} v2`)
    dayRow('Wednesday').find('input[type="time"]').eq(1).clear().type('19:30')

    cy.get('input[name="name"]').should('have.value', `${renamedHours} v2`)
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/business-hours').as('createHours')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createHours.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects custom business hours with no weekday selected', () => {
    cy.intercept('POST', '**/api/v1/business-hours').as('createHours')

    cy.visit(newPath)
    cy.get('input[name="name"]').type(`${hoursName} no days`)
    cy.contains('label', 'Custom business hours').click()
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createHours.all').should('have.length', 0)
  })

  it('deletes the business hours', () => {
    cy.intercept('DELETE', `**/api/v1/business-hours/${hoursId}`).as('deleteHours')

    cy.visit(listPath)
    filterList(renamedHours)
    cy.contains('tr', renamedHours).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteHours').its('response.statusCode').should('eq', 200)
    cy.contains(renamedHours).should('not.exist')
  })
})
