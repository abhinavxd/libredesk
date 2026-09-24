const panel = () => cy.get('section[role="dialog"]')
const emailDraft = {
  values: {
    inbox_id: '901', team_id: null, agent_id: 'none', subject: 'Draft subject',
    content: '<p>Draft message</p>', contact_email: 'user@example.com',
    cc: '', bcc: '', first_name: 'Customer A', last_name: ''
  }, contact: null
}
const template = {
  id: 71, name: 'draft_template', language: 'en', status: 'APPROVED',
  component_types: ['HEADER', 'BODY', 'BUTTONS'], header_type: 'TEXT',
  header_content: 'Order {{1}}', body_content: 'Hello {{1}}',
  buttons: [{ type: 'URL', text: 'Track', url: 'https://example.com/{{1}}' }]
}
const whatsappDraft = {
  inboxId: '902', teamId: null, agentId: 'none', phoneCountryCode: 'IN',
  phoneNumber: '9000000000', firstName: 'Customer A', lastName: '', contact: null,
  templateId: 71, templateParams: { 'body:1': 'Customer A', 'header:1': '123', button_url_0: 'order-123' }
}
const expand = () => panel().find('header').contains('button', 'New conversation').click()
const visitDraft = (channel = 'email', draft = emailDraft, path = '/inboxes/assigned') => {
  cy.visit(path, { onBeforeLoad(win) {
    win.localStorage.removeItem('newConversationDraftEmail')
    win.localStorage.removeItem('newConversationDraftWhatsApp')
    win.localStorage.setItem(channel === 'email' ? 'newConversationDraftEmail' : 'newConversationDraftWhatsApp', JSON.stringify(draft))
  } })
  expand()
}
const reloadDraft = () => {
  cy.reload()
  expand()
}
const submit = () => panel().find('form:visible').find('button[type="submit"]').scrollIntoView().click()
const checkSaved = (channel, assertion) => cy.window().should((win) => {
  assertion(JSON.parse(win.localStorage.getItem(`newConversationDraft${channel}`)))
})

describe('Composer drafts', () => {
  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
    cy.intercept('GET', '**/api/v1/inboxes', { body: { data: [
      { id: 901, name: 'Draft email inbox', channel: 'email' },
      { id: 902, name: 'Draft WhatsApp inbox', channel: 'whatsapp' },
      { id: 903, name: 'Other WhatsApp inbox', channel: 'whatsapp' }
    ] } })
    cy.intercept('GET', '**/api/v1/contacts/search*', { body: { data: [] } })
    cy.intercept('GET', '**/api/v1/whatsapp/templates*', { delay: 200, body: { data: [template] } }).as('templates')
    cy.intercept('POST', '**/api/v1/conversations', { body: { data: { uuid: '' } } }).as('send')
  })

  it('sends after repeatedly opening, filling and closing CC and BCC', () => {
    visitDraft()
    for (const field of ['cc', 'bcc', 'cc', 'bcc']) {
      panel().contains('button', new RegExp(`^${field.toUpperCase()}$`)).click()
      panel().find(`input[name="${field}"]`).type('copy@example.com')
      panel().find(`button[aria-label="Remove ${field.toUpperCase()}"]`).click()
      panel().find(`input[name="${field}"]`).should('not.exist')
    }
    submit()
    cy.wait('@send').then(({ request }) => {
      expect(request.body.cc).to.deep.eq([])
      expect(request.body.bcc).to.deep.eq([])
    })
    panel().should('not.exist')
    checkSaved('Email', (draft) => expect(draft).to.eq(null))
  })

  it('validates visible recipients on submit and permits sending after removing an invalid row', () => {
    visitDraft()
    panel().contains('button', /^CC$/).click()
    panel().find('input[name="cc"]').type('unfinished')
    panel().contains('Invalid email address').should('not.exist')
    submit()
    panel().contains('Invalid email address').should('be.visible')
    cy.get('@send.all').should('have.length', 0)
    panel().find('button[aria-label="Remove CC"]').click()
    panel().contains('button', /^BCC$/).click()
    panel().find('input[name="bcc"]').type('copy@example.com, second@example.com')
    submit()
    cy.wait('@send').its('request.body.bcc').should('deep.eq', ['copy@example.com', 'second@example.com'])
  })

  it('keeps Send reachable on a short viewport with recipient errors', () => {
    cy.viewport(1000, 400)
    visitDraft()
    for (const field of ['cc', 'bcc']) {
      panel().contains('button', new RegExp(`^${field.toUpperCase()}$`)).click()
      panel().find(`input[name="${field}"]`).type('unfinished')
    }
    submit()
    panel().find('form:visible').then(($form) => {
      expect($form[0].scrollHeight).to.be.greaterThan($form[0].clientHeight)
      expect(getComputedStyle($form[0]).overflowY).to.eq('auto')
    })
    panel().find('button[type="submit"]:visible').scrollIntoView().should('be.visible').then(($button) => {
      const rect = $button[0].getBoundingClientRect()
      expect(rect.bottom).to.be.at.most(400)
      expect(rect.top).to.be.at.least(0)
    })
  })

  it('restores uploaded attachments and sends their IDs after reload', () => {
    const media = { id: 801, uuid: 'draft-attachment', filename: 'note.txt', size: 16, content_type: 'text/plain' }
    cy.intercept('POST', '**/api/v1/media', { body: { data: media } }).as('upload')
    visitDraft()
    panel().find('input[type="file"]:first').selectFile({
      contents: Cypress.Buffer.from('draft attachment'), fileName: 'note.txt', mimeType: 'text/plain'
    }, { force: true })
    cy.wait('@upload')
    checkSaved('Email', (draft) => expect(draft.attachments).to.deep.eq([media]))
    reloadDraft()
    panel().contains('note.txt').should('be.visible')
    panel().find('.tiptap:visible').should('contain.text', 'Draft message')
    submit()
    cy.wait('@send').its('request.body.attachments').should('deep.eq', [801])
    checkSaved('Email', (draft) => expect(draft).to.eq(null))
  })

  it('persists an attachment-only draft and its removal', () => {
    const media = { id: 801, uuid: 'draft-attachment', filename: 'note.txt', size: 16 }
    visitDraft('email', { values: { ...emailDraft.values, contact_email: '', subject: '', content: '', first_name: '' }, attachments: [media] })
    reloadDraft()
    panel().contains('note.txt').should('be.visible')
    panel().find('button[title="Remove attachment"]').click()
    checkSaved('Email', (draft) => expect(draft).to.eq(null))
  })

  it('keeps the draft and attachments when Send fails', () => {
    const media = { id: 801, uuid: 'draft-attachment', filename: 'note.txt', size: 16 }
    cy.intercept('POST', '**/api/v1/conversations', {
      statusCode: 500, body: { error: { type: 'GeneralException', message: 'Test send failure' } }
    }).as('failedSend')
    visitDraft('email', { ...emailDraft, attachments: [media] })
    submit()
    cy.wait('@failedSend')
    reloadDraft()
    panel().contains('note.txt').should('be.visible')
    panel().find('.tiptap:visible').should('contain.text', 'Draft message')
    cy.intercept('POST', '**/api/v1/conversations', { body: { data: { uuid: '' } } }).as('retrySend')
    submit()
    cy.wait('@retrySend').its('request.body.attachments').should('deep.eq', [801])
    checkSaved('Email', (draft) => expect(draft).to.eq(null))
  })

  it('sends on Enter without toggling recipient fields', () => {
    visitDraft()
    panel().contains('button', /^CC$/).click()
    panel().find('input[name="cc"]').type('copy@example.com')
    panel().find('input[name="subject"]').type('{enter}')
    cy.wait('@send').its('request.body.cc').should('deep.eq', ['copy@example.com'])
  })

  it('persists a template selection before a recipient is entered', () => {
    visitDraft('whatsapp', { inboxId: '902', templateId: null, templateParams: {} })
    cy.wait('@templates')
    panel().contains('button', 'draft_template').click()
    panel().find('input.col-span-2').eq(0).type('Customer A')
    checkSaved('WhatsApp', (draft) => {
      expect(draft.templateId).to.eq(71)
      expect(draft.templateParams['body:1']).to.eq('Customer A')
    })
    reloadDraft()
    cy.wait('@templates')
    panel().contains('Hello Customer A').should('be.visible')
    panel().find('button[type="submit"]:visible').should('be.disabled')
    panel().contains('button', /^Back$/).click()
    checkSaved('WhatsApp', (draft) => expect(draft).to.eq(null))
  })

  it('restores the WhatsApp template and all parameter types across reloads', () => {
    visitDraft('whatsapp', whatsappDraft)
    cy.wait('@templates')
    panel().contains('Hello Customer A').should('be.visible')
    panel().find('input.col-span-2').eq(0).clear().type('Customer B')
    checkSaved('WhatsApp', (draft) => expect(draft.templateParams['body:1']).to.eq('Customer B'))
    reloadDraft()
    cy.wait('@templates')
    panel().contains('Hello Customer B').should('be.visible')
    panel().find('input.col-span-2').then(($inputs) => {
      expect([...$inputs].map((input) => input.value)).to.deep.eq(['Customer B', '123', 'order-123'])
    })
    submit()
    cy.wait('@send').then(({ request }) => {
      expect(request.body.whatsapp_template_id).to.eq(71)
      expect(request.body.whatsapp_template_params).to.deep.eq({ ...whatsappDraft.templateParams, 'body:1': 'Customer B' })
    })
    checkSaved('WhatsApp', (draft) => expect(draft).to.eq(null))
  })

  it('preserves partial WhatsApp parameters without enabling Send', () => {
    visitDraft('whatsapp', { ...whatsappDraft, templateParams: { 'body:1': 'Customer A' } })
    cy.wait('@templates')
    reloadDraft()
    cy.wait('@templates')
    panel().find('input.col-span-2').eq(0).should('have.value', 'Customer A')
    panel().find('input.col-span-2').eq(1).should('have.value', '')
    panel().find('button[type="submit"]:visible').should('be.disabled')
  })

  it('does not restore a deleted template or carry parameters to a different inbox', () => {
    visitDraft('whatsapp', whatsappDraft)
    cy.wait('@templates')
    panel().contains('button[role="combobox"]', 'Draft WhatsApp inbox').click()
    cy.contains('[role="option"]', 'Other WhatsApp inbox').click()
    cy.wait('@templates')
    panel().find('input.col-span-2').should('not.exist')
    checkSaved('WhatsApp', (draft) => expect(draft.templateId).to.eq(null))
    cy.intercept('GET', '**/api/v1/whatsapp/templates*', { body: { data: [] } }).as('emptyTemplates')
    visitDraft('whatsapp', whatsappDraft)
    cy.wait('@emptyTemplates')
    panel().find('input.col-span-2').should('not.exist')
    panel().find('button[type="submit"]:visible').should('be.disabled')
  })

  it('targets the visible composer after minimizing, switching channels and restoring', () => {
    const uuid = '00000000-0000-4000-8000-000000000901'
    cy.intercept('GET', `**/api/v1/conversations/${uuid}`, { body: { data: {
      id: 901, uuid, reference_number: 901, inbox_id: 901, inbox: { id: 901, channel: 'email', name: 'Draft email inbox' },
      channel: 'email', status: 'Open', subject: 'Existing conversation', tags: [],
      contact: { id: 901, first_name: 'Customer A', last_name: '', email: 'user@example.com' },
      created_at: new Date().toISOString(), updated_at: new Date().toISOString()
    } } })
    cy.intercept('GET', `**/api/v1/conversations/${uuid}/messages*`, { body: { data: { results: [], total: 0, page: 1, total_pages: 0 } } })
    cy.intercept('GET', '**/api/v1/macros/search*', { body: { data: [{ id: 501, name: 'Draft test macro', actions: [], has_message_content: true }] } }).as('macros')
    cy.intercept('GET', '**/api/v1/macros/501', { body: { data: { id: 501, message_content: '<p>Macro target text</p>' } } })
    visitDraft('email', emailDraft, `/inboxes/all/conversation/${uuid}`)
    const applyMacro = (view) => {
      cy.get('body').trigger('keydown', { key: 'm', code: 'KeyM', ctrlKey: true })
      cy.wait('@macros').its('request.url').should('include', `view=${view}`)
      cy.contains('[role="option"]', 'Draft test macro').click()
    }
    expand()
    applyMacro('replying')
    cy.get('.tiptap:visible').should('contain.text', 'Macro target text')
    checkSaved('Email', (draft) => expect(draft.values.content).to.eq('<p>Draft message</p>'))
    expand()
    applyMacro('starting_conversation')
    panel().find('.tiptap:visible').should('contain.text', 'Macro target text')
    panel().contains('[role="tab"]', 'WhatsApp').click()
    applyMacro('replying')
    panel().find('header button[aria-label="Close"]').click()
    applyMacro('replying')
  })
})
