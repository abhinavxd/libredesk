const stamp = Date.now()
const contentMacroName = `Palette content macro ${stamp}`
const actionMacroName = `Palette action macro ${stamp}`
const messageBody = `Reply body from the palette spec ${stamp}`

describe('Command palette macros', () => {
  let contentMacroId
  let actionMacroId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/macros', {
      name: contentMacroName,
      message_content: `<p>${messageBody}</p>`,
      visibility: 'all',
      visible_when: ['replying', 'starting_conversation', 'adding_private_note'],
      actions: []
    }).then((resp) => {
      contentMacroId = resp.body.data.id
    })
    cy.api('POST', '/api/v1/macros', {
      name: actionMacroName,
      message_content: '',
      visibility: 'all',
      visible_when: ['replying', 'starting_conversation', 'adding_private_note'],
      actions: [{ type: 'add_tags', value: ['palette-spec-tag'], display_value: [] }]
    }).then((resp) => {
      actionMacroId = resp.body.data.id
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  after(() => {
    cy.login()
    if (contentMacroId) cy.api('DELETE', `/api/v1/macros/${contentMacroId}`)
    if (actionMacroId) cy.api('DELETE', `/api/v1/macros/${actionMacroId}`)
  })

  it('serves the compact list without message content', () => {
    cy.api('GET', '/api/v1/macros/compact').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data
      rows.forEach((row) => {
        expect(row).to.not.have.property('message_content')
        expect(row).to.have.property('has_message_content').that.is.a('boolean')
      })
      expect(rows.find((r) => r.id === contentMacroId).has_message_content).to.eq(true)
      expect(rows.find((r) => r.id === actionMacroId).has_message_content).to.eq(false)
    })
  })

  it('keeps the legacy list response unchanged', () => {
    cy.api('GET', '/api/v1/macros').then(({ status, body }) => {
      expect(status).to.eq(200)
      const row = body.data.find((r) => r.id === contentMacroId)
      expect(row.message_content).to.contain(messageBody)
      expect(row).to.not.have.property('has_message_content')
    })
  })

  it('serves a single macro with its content to an authenticated agent', () => {
    cy.api('GET', `/api/v1/macros/${contentMacroId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.message_content).to.contain(messageBody)
    })
  })

  it('searches macros in the palette and fetches the preview on demand', () => {
    cy.intercept('GET', '**/api/v1/macros/compact').as('macrosCompact')
    cy.intercept('GET', /\/api\/v1\/macros\/\d+$/).as('macroContent')

    cy.visit('/inboxes/assigned')
    cy.wait('@macrosCompact')

    // useMagicKeys only sees Ctrl_M when the Control and m keydowns arrive in separate tasks.
    cy.window().then(async (win) => {
      win.dispatchEvent(new win.KeyboardEvent('keydown', { key: 'Control', code: 'ControlLeft', bubbles: true }))
      await new Promise((resolve) => setTimeout(resolve, 50))
      win.dispatchEvent(new win.KeyboardEvent('keydown', { key: 'm', code: 'KeyM', ctrlKey: true, bubbles: true }))
    })
    cy.get('[cmdk-input-wrapper] input').type(contentMacroName)

    cy.contains('[role="option"]', contentMacroName)
      .should('exist')
      .trigger('pointerenter')

    cy.wait('@macroContent').its('response.statusCode').should('eq', 200)
    cy.contains(messageBody).should('be.visible')

    // Auto-highlight and pointerenter both hit the same macro, the cache must collapse them to one request.
    cy.get('@macroContent.all').should('have.length', 1)
  })
})
