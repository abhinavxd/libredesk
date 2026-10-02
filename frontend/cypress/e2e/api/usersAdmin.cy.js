describe('API: users settings screen', () => {
  const stamp = Date.now()
  const address = `users.${stamp}@example.test`
  let inboxID
  let addressID
  let userID

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/inboxes', {
      name: 'Users mailbox', channel: 'email', from: address, enabled: false,
      config: { auth_type: 'password', imap: [], smtp: [] }
    }).then(({ body }) => { inboxID = body.data.id })
    cy.api('GET', '/api/v1/admin/addresses').then(({ body }) => {
      addressID = body.data.find((item) => item.address === address).id
    })
  })

  after(() => {
    if (userID) cy.api('DELETE', `/api/v1/admin/users/${userID}`, null, { failOnStatusCode: false })
    if (inboxID) cy.api('DELETE', `/api/v1/inboxes/${inboxID}`)
  })

  beforeEach(() => cy.login())

  it('lists people with the built-in roles and every address, but never the System user', () => {
    cy.api('GET', '/api/v1/admin/users').then(({ body }) => {
      expect(body.data.roles.slice(0, 3)).to.deep.eq(['Admin', 'Agent', 'Contributor'])
      expect(body.data.addresses.map((item) => item.id)).to.include(addressID)
      expect(body.data.users.map((user) => user.email)).not.to.include('System')
    })
  })

  it('adds a Contributor with an address, then changes their access', () => {
    cy.api('POST', '/api/v1/admin/users', {
      first_name: 'Casey', last_name: 'Writer', email: `casey.${stamp}@example.test`,
      role: 'Contributor', address_ids: [addressID], send_welcome_email: false
    }).then(({ body }) => {
      userID = body.data.id
      expect(body.data.roles).to.deep.eq(['Contributor'])
      expect(body.data.address_ids).to.deep.eq([addressID])
      cy.api('PUT', `/api/v1/admin/users/${userID}/access`, { role: 'Agent', address_ids: [], enabled: false })
    }).then(({ body }) => {
      expect(body.data.roles).to.deep.eq(['Agent'])
      expect(body.data.address_ids).to.deep.eq([])
      expect(body.data.enabled).to.eq(false)
    })
  })

  it('rejects unknown roles and addresses', () => {
    cy.api('PUT', `/api/v1/admin/users/${userID}/access`, { role: 'Superuser', address_ids: [], enabled: true }, {
      failOnStatusCode: false
    }).its('status').should('eq', 400)
    cy.api('PUT', `/api/v1/admin/users/${userID}/access`, { role: 'Agent', address_ids: [999999999], enabled: true }, {
      failOnStatusCode: false
    }).its('status').should('eq', 400)
  })

  it('refuses to edit the built-in System user', () => {
    cy.api('GET', '/api/v1/agents/me').then(({ body }) => {
      cy.api('PUT', `/api/v1/admin/users/${body.data.id}/access`, { role: 'Agent', address_ids: [], enabled: true }, {
        failOnStatusCode: false
      }).its('status').should('eq', 400)
    })
  })

  it('deletes a user', () => {
    cy.api('DELETE', `/api/v1/admin/users/${userID}`).its('status').should('eq', 200)
    cy.api('GET', '/api/v1/admin/users').then(({ body }) => {
      expect(body.data.users.map((user) => user.id)).not.to.include(userID)
      userID = null
    })
  })

  it('keeps address saves from changing per-person access', () => {
    cy.api('POST', '/api/v1/admin/users', {
      first_name: 'Robin', email: `robin.${stamp}@example.test`, role: 'Agent', address_ids: [addressID], send_welcome_email: false
    }).then(({ body }) => {
      userID = body.data.id
      cy.api('GET', `/api/v1/admin/addresses/${addressID}`)
    }).then(({ body }) => {
      const { id, inbox_id, address: email, display_name, kind, enabled, team_ids } = body.data
      cy.api('PUT', `/api/v1/admin/addresses/${id}`, { inbox_id, address: email, display_name, kind, enabled, team_ids: team_ids || [] })
    }).then(() => cy.api('GET', `/api/v1/admin/addresses/${addressID}/access`))
      .then(({ body }) => {
        expect(body.data.user_ids).to.include(userID)
      })
  })
})
