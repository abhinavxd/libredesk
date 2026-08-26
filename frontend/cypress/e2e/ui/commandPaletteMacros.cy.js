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
      visible_when: ['replying'],
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

  it('serves the paginated compact list without message content', () => {
    cy.api('GET', `/api/v1/macros/compact?page=1&page_size=5&q=${encodeURIComponent(contentMacroName)}`).then(
      ({ status, body }) => {
        expect(status).to.eq(200)
        const { results, total, page } = body.data
        expect(page).to.eq(1)
        expect(total).to.eq(1)
        expect(results[0].id).to.eq(contentMacroId)
        expect(results[0]).to.not.have.property('message_content')
        expect(results[0].has_message_content).to.eq(true)
      }
    )
  })

  it('searches macros by name and view without message content', () => {
    cy.api('GET', `/api/v1/macros/search?q=${encodeURIComponent(actionMacroName)}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.map((m) => m.id)).to.deep.eq([actionMacroId])
      expect(body.data[0]).to.not.have.property('message_content')
      expect(body.data[0].has_message_content).to.eq(false)
    })
    cy.api('GET', `/api/v1/macros/search?q=${encodeURIComponent(actionMacroName)}&view=adding_private_note`)
      .its('body.data')
      .should('have.length', 0)
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

  it('searches macros live in the palette and fetches the preview on demand', () => {
    cy.intercept('GET', '**/api/v1/macros/search*').as('macroSearch')
    cy.intercept('GET', /\/api\/v1\/macros\/\d+$/).as('macroContent')

    cy.visit('/inboxes/assigned')
    cy.contains('My inbox').should('be.visible')

    // useMagicKeys only sees Ctrl_M when the Control and m keydowns arrive in separate tasks.
    cy.window().then(async (win) => {
      win.dispatchEvent(new win.KeyboardEvent('keydown', { key: 'Control', code: 'ControlLeft', bubbles: true }))
      await new Promise((resolve) => setTimeout(resolve, 50))
      win.dispatchEvent(new win.KeyboardEvent('keydown', { key: 'm', code: 'KeyM', ctrlKey: true, bubbles: true }))
    })
    cy.wait('@macroSearch')

    cy.get('[cmdk-input-wrapper] input').type(contentMacroName)
    cy.wait('@macroSearch')

    cy.contains('[role="option"]', contentMacroName)
      .should('exist')
      .trigger('pointerenter')

    cy.wait('@macroContent').its('response.statusCode').should('eq', 200)
    cy.contains(messageBody).should('be.visible')

    // Auto-highlight and pointerenter both hit this macro, the cache must collapse them to one request.
    cy.get('@macroContent.all').then((calls) => {
      const ours = calls.filter((c) => c.request.url.endsWith(`/${contentMacroId}`))
      expect(ours).to.have.length(1)
    })
  })
})
