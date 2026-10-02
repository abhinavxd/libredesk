describe('Address read action', () => {
  const address = `read.${Date.now()}@example.test`
  let inboxID
  let addressID

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: 'Read action mailbox', channel: 'email', from: address, enabled: false,
      config: { auth_type: 'password', imap: [], smtp: [] }
    }).then(({ body }) => { inboxID = body.data.id })
    cy.api('GET', '/api/v1/admin/addresses').then(({ body }) => {
      addressID = body.data.find((item) => item.address === address).id
    })
  })

  after(() => {
    if (inboxID) cy.api('DELETE', `/api/v1/inboxes/${inboxID}`)
  })

  for (const [name, width, height] of [['desktop', 1280, 900], ['mobile', 390, 844]]) {
    it(`marks the selected address read on ${name}`, () => {
      cy.login()
      cy.viewport(width, height)
      cy.intercept('POST', '**/api/v1/addresses/*/mark-read').as('markRead')
      cy.visit(`/addresses/${addressID}`)
      cy.get('button[aria-label="Address actions"]').should('be.visible').click()
      cy.contains('[role="menuitem"]', 'Mark all as read').should('be.visible')
      cy.screenshot(`address-read-${name}`, { capture: 'viewport' })
      cy.contains('[role="menuitem"]', 'Mark all as read').click()
      cy.wait('@markRead').then(({ request, response }) => {
        expect(request.url).to.contain(`/addresses/${addressID}/mark-read`)
        expect(response.statusCode).to.eq(200)
        expect(response.body.data.address_id).to.eq(addressID)
        expect(response.body.data.marked_at).to.be.a('string')
      })
      cy.contains(`Marked all messages in ${address} as read for you.`).should('be.visible')
      cy.contains('New conversation').should('not.exist')
      cy.get('a[href*="/views"]').should('not.exist')
    })
  }
})
