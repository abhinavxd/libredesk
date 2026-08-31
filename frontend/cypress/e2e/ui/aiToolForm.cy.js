// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const toolName = `cypress_tool_${stamp}`
const toolDescription = `Looks up an order for the tool form spec ${stamp}`
const toolUrl = `https://api.example.com/orders/${stamp}`
const editedUrl = `https://api.example.com/orders/${stamp}/v2`
const headerKey = 'X-Api-Key'
const headerValue = `secret-${stamp}`
const parameters = '{"type":"object","properties":{"order_id":{"type":"string"}},"required":["order_id"]}'
const newPath = '/admin/ai/tools/new'
const listPath = '/admin/ai/tools'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const pickOption = (field, optionText) => {
  cy.get(`select[name="${field}"]`).siblings('button[role="combobox"]').click()
  cy.get('[role="option"]').contains(optionText).click()
}

const headerKeys = () => cy.get('input[placeholder="Authorization"]')
const headerValues = () => cy.get('input[placeholder="Bearer xxx"]')

describe('AI tool form', () => {
  let toolId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a tool', () => {
    cy.intercept('POST', '**/api/v1/ai/tools').as('createTool')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(toolName)
    cy.get('textarea[name="description"]').type(toolDescription)
    cy.get('input[name="url"]').type(toolUrl)
    pickOption('method', 'GET')

    cy.contains('button', 'Add header').click()
    headerKeys().type(headerKey)
    headerValues().type(headerValue)

    cy.get('textarea[name="parameters"]').type(parameters, { parseSpecialCharSequences: false })

    cy.get('button[type="submit"]').click()
    cy.wait('@createTool').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      toolId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(toolName)
    cy.contains(toolName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(toolId, 'tool from the create step').to.be.a('number')

    cy.visit(`${listPath}/${toolId}/edit`)

    cy.get('input[name="name"]').should('have.value', toolName)
    cy.get('textarea[name="description"]').should('have.value', toolDescription)
    cy.get('input[name="url"]').should('have.value', toolUrl)
    cy.get('select[name="method"]').should('have.value', 'GET')

    headerKeys().should('have.length', 1).and('have.value', headerKey)
    // The backend masks header values on read, so only the row's presence can be asserted.
    headerValues().should('have.length', 1).and('not.have.value', '')

    cy.get('textarea[name="parameters"]')
      .invoke('val')
      .then((value) => {
        expect(JSON.parse(value)).to.deep.eq(JSON.parse(parameters))
      })
  })

  it('adds and removes a header row without submitting the form', () => {
    cy.intercept('PUT', `**/api/v1/ai/tools/${toolId}`).as('updateTool')

    cy.visit(`${listPath}/${toolId}/edit`)
    headerKeys().should('have.length', 1)

    cy.contains('button', 'Add header').click()
    headerKeys().should('have.length', 2)
    headerKeys().eq(1).type('X-Extra')

    cy.get('button[aria-label="Remove"]').eq(1).click()
    cy.get('@updateTool.all').should('have.length', 0)

    headerKeys().should('have.length', 1).and('have.value', headerKey)
    cy.location('pathname').should('eq', `${listPath}/${toolId}/edit`)
  })

  it('persists a changed url and method', () => {
    cy.intercept('PUT', `**/api/v1/ai/tools/${toolId}`).as('updateTool')

    cy.visit(`${listPath}/${toolId}/edit`)
    cy.get('input[name="url"]').should('have.value', toolUrl).clear().type(editedUrl)
    pickOption('method', 'POST')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateTool').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${toolId}/edit`)
    cy.get('input[name="url"]').should('have.value', editedUrl)
    cy.get('select[name="method"]').should('have.value', 'POST')
  })

  it('rejects a submit with no name', () => {
    cy.intercept('POST', '**/api/v1/ai/tools').as('createTool')

    cy.visit(newPath)
    cy.get('input[name="url"]').type(toolUrl)
    cy.get('button[type="submit"]').click()

    // The scrolling admin pane clips the message, so assert it exists rather than is visible.
    cy.contains(/required/i).should('exist')
    cy.get('input[name="name"]').should('have.attr', 'aria-invalid', 'true')
    cy.get('@createTool.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the tool', () => {
    cy.intercept('DELETE', `**/api/v1/ai/tools/${toolId}`).as('deleteTool')

    cy.visit(listPath)
    filterList(toolName)
    cy.contains('tr', toolName).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteTool').its('response.statusCode').should('eq', 200)
    cy.contains(toolName).should('not.exist')
  })
})
