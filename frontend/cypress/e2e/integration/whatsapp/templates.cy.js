import '../../../support/whatsapp'

const stamp = Date.now()
const name = `qa_edit_${stamp}`
const field = (label) =>
  cy
    .contains('label', label)
    .invoke('attr', 'for')
    .then((id) => cy.get(`[id="${id}"]`))

describe('WhatsApp template management', () => {
  let inboxID
  let template

  const approve = () => {
    cy.task('metaMock:setTemplateStatus', { id: template.meta_template_id, status: 'APPROVED' })
    cy.api('POST', `/api/v1/whatsapp/templates/sync?inbox_id=${inboxID}`)
  }

  before(() => {
    cy.login()
    cy.task('metaMock:reset')
    cy.api('POST', '/api/v1/inboxes', {
      name: `Template QA ${stamp}`,
      channel: 'whatsapp',
      enabled: true,
      config: {
        phone_number_id: `PN-${stamp}`,
        waba_id: `WABA-${stamp}`,
        access_token: 'test-token',
        app_secret: 'test-secret',
        webhook_verify_token: 'test-verify',
        api_version: 'v26.0'
      }
    }).then(({ body }) => {
      inboxID = body.data.id
    })
  })

  beforeEach(() => {
    cy.viewport(1280, 900)
    cy.login()
  })

  afterEach(() => {
    cy.task('metaMock:failTemplateEdit', false)
    cy.task('metaMock:failTemplateDelete', false)
  })

  it('creates a template through the form without validating during typing', () => {
    cy.visit(`/admin/whatsapp/templates/new?inbox_id=${inboxID}`)
    field('Name').type('INVALID')
    cy.contains('Lowercase letters, numbers and underscores only').should('not.exist')
    field('Name').clear().type(name)
    field('Body text').type('Original QA message')
    cy.intercept('POST', '**/api/v1/whatsapp/templates').as('create')
    cy.get('button[type="submit"]').click()
    cy.wait('@create').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      template = response.body.data
      expect(template.component_types).to.deep.eq(['BODY'])
    })
    cy.contains('td', name).should('be.visible')
  })

  it('blocks editing while review is pending, including direct API requests', () => {
    cy.visit(`/admin/whatsapp/templates/${template.id}/edit`)
    cy.contains('This template cannot be edited here').should('be.visible')
    cy.get('button[type="submit"]').should('be.disabled')
    cy.api(
      'PUT',
      `/api/v1/whatsapp/templates/${template.id}`,
      { ...template, body_content: 'Not allowed' },
      { failOnStatusCode: false }
    )
      .its('status')
      .should('eq', 400)
  })

  it('edits header, body, footer, samples and buttons through the UI', () => {
    approve()
    cy.visit(`/admin/whatsapp/templates?inbox_id=${inboxID}`)
    cy.contains('a', name).click()
    field('Name').should('be.disabled')
    field('Category').should('be.disabled')
    cy.contains('button[role="combobox"]', 'English').should('be.disabled')
    cy.contains('label', 'Header type')
      .parent()
      .find('button[role="combobox"]')
      .focus()
      .type('{downarrow}')
    cy.focused().type('{end}{enter}')
    field('Header text').type('QA header')
    field('Body text')
      .clear()
      .type('Updated QA message {{1}}', { parseSpecialCharSequences: false })
    field('Footer').type('QA footer')
    cy.get('#sample-1').type('Ruchika')
    cy.contains('button', 'Add button').click()
    cy.get('input[aria-label="Title"]').type('Help')
    cy.get('input[aria-label="URL"]').type('https://example.com/help')
    cy.intercept('PUT', `**/api/v1/whatsapp/templates/${template.id}`).as('update')
    cy.get('button[type="submit"]').click()
    cy.wait('@update').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.body_content).to.eq('Updated QA message {{1}}')
      expect(response.body.data.header_content).to.eq('QA header')
      expect(response.body.data.footer_content).to.eq('QA footer')
      expect(response.body.data.buttons[0].text).to.eq('Help')
      expect(response.body.data.sample_values).to.deep.eq({ 1: 'Ruchika' })
      expect(response.body.data.status).to.eq('PENDING')
      template = response.body.data
    })
    cy.location('pathname').should('eq', '/admin/whatsapp/templates')
  })

  it('keeps the saved content unchanged when Meta rejects the edit', () => {
    approve()
    cy.task('metaMock:failTemplateEdit', true)
    cy.visit(`/admin/whatsapp/templates/${template.id}/edit`)
    field('Body text').should('have.value', template.body_content).clear().type('Rejected change')
    cy.intercept('PUT', `**/api/v1/whatsapp/templates/${template.id}`).as('update')
    cy.get('button[type="submit"]').click()
    cy.wait('@update').its('response.statusCode').should('eq', 400)
    cy.contains('Template edit refused').should('be.visible')
    cy.api('GET', `/api/v1/whatsapp/templates/${template.id}`).then(({ body }) => {
      expect(body.data.body_content).to.eq(template.body_content)
      expect(body.data.status).to.eq('APPROVED')
    })
  })

  it('keeps the row visible when Meta refuses deletion', () => {
    cy.task('metaMock:failTemplateDelete', true)
    cy.visit(`/admin/whatsapp/templates?inbox_id=${inboxID}`)
    cy.contains('tr', name).find('button[aria-label="Delete"]').click()
    cy.intercept('DELETE', `**/api/v1/whatsapp/templates/${template.id}`).as('delete')
    cy.get('[role="alertdialog"]')
      .contains('button', /^Delete$/)
      .click()
    cy.wait('@delete').its('response.statusCode').should('eq', 400)
    cy.contains('Template deletion refused').should('be.visible')
    cy.contains('td', name).should('be.visible')
  })

  it('deletes successfully and stays deleted after sync', () => {
    cy.visit(`/admin/whatsapp/templates?inbox_id=${inboxID}`)
    cy.contains('tr', name).find('button[aria-label="Delete"]').click()
    cy.intercept('DELETE', `**/api/v1/whatsapp/templates/${template.id}`).as('delete')
    cy.get('[role="alertdialog"]')
      .contains('button', /^Delete$/)
      .click()
    cy.wait('@delete').its('response.statusCode').should('eq', 200)
    cy.contains('td', name).should('not.exist')
    cy.contains('button', 'Sync from Meta').click()
    cy.contains('td', name).should('not.exist')
  })
})
