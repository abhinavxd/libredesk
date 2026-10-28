const startButtonText = 'Cypress start chat'

const embedConfig = (overrides = {}) => ({
  visitors: { allow_start_conversation: true, start_conversation_button_text: startButtonText },
  users: { allow_start_conversation: true, start_conversation_button_text: startButtonText },
  ...overrides
})

describe('Live chat widget embedded on a host page', () => {
  it('renders the launcher and keeps the panel closed until clicked', () => {
    cy.createLivechatInbox(embedConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetLauncher().should('be.visible')
      cy.get('iframe[src*="/widget?inbox_id="]').should('not.be.visible')

      cy.widgetLauncher().click()
      cy.get('iframe[src*="/widget?inbox_id="]').should('be.visible')
      cy.widgetBody().contains(startButtonText).should('be.visible')
    })
  })

  it('keeps the launcher usable by keyboard under host button styles', () => {
    cy.createLivechatInbox(embedConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.document().then((doc) => {
        const style = doc.createElement('style')
        style.textContent = 'button { margin: 30px; padding: 40px; min-width: 280px; background-image: linear-gradient(black, white); border: 10px solid black; }'
        doc.head.appendChild(style)
      })
      cy.widgetLauncher().should('be.visible').and('have.prop', 'tagName', 'BUTTON')
        .and('have.attr', 'type', 'button')
        .and('have.attr', 'aria-expanded', 'false')
        .and('have.attr', 'aria-label', 'Open chat')
      cy.widgetLauncher().then(($button) => {
        const button = $button[0]
        const style = button.ownerDocument.defaultView.getComputedStyle(button)
        expect(button.getBoundingClientRect().width).to.eq(60)
        expect(button.getBoundingClientRect().height).to.eq(60)
        expect(style.margin).to.eq('0px')
        expect(style.backgroundImage).to.eq('none')
      })
      cy.viewport(390, 844)
      cy.widgetLauncher().focus()
      cy.widgetLauncher().type('{enter}')
      cy.widgetLauncher().should('have.attr', 'aria-expanded', 'true')
        .and('have.attr', 'aria-label', 'Close chat')
      cy.widgetBody().find('button[aria-label="Close chat"]').focus()
      cy.widgetBody().find('button[aria-label="Close chat"]').click()
      cy.widgetLauncher().should('be.focused').and('have.attr', 'aria-expanded', 'false')
      cy.viewport(1000, 660)
      cy.widgetLauncher().type(' ')
      cy.get('iframe[src*="/widget?inbox_id="]').should('be.visible')
      cy.widgetLauncher().focus()
      cy.widgetLauncher().type(' ')
      cy.get('iframe[src*="/widget?inbox_id="]').should('not.be.visible')
    })
  })

  it('keeps the original image visible after a missing thumbnail and reconnect', () => {
    cy.createLivechatInbox(embedConfig({ features: { file_upload: true } })).then((inbox) => {
      cy.intercept('GET', '**/uploads/thumb_*', { statusCode: 404, body: '' }).as('missingThumbnail')
      cy.intercept('GET', '**/api/v1/widget/chat/conversations/*').as('openedConversation')
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetLauncher().click()
      cy.widgetBody().contains(startButtonText).click()
      cy.widgetSend('Image fallback conversation')
      cy.wait('@openedConversation')
      cy.widgetBody().find('input[type="file"]').selectFile('../static/public/launcher-logo.png', { force: true })
      cy.wait('@missingThumbnail')
      const image = () => cy.widgetBody().find('[data-message-uuid] img[alt="launcher-logo.png"]')
      image().should(($image) => {
        expect($image[0].naturalWidth).to.be.greaterThan(0)
        expect($image.attr('src')).not.to.include('/thumb_')
      })
      let originalImage
      image().then(($image) => { originalImage = $image[0] })
      cy.intercept('GET', '**/api/v1/widget/chat/conversations/*', (req) => {
        req.continue((res) => {
          for (const message of res.body.data.messages) {
            for (const attachment of message.attachments || []) {
              if (attachment.name !== 'launcher-logo.png') continue
              const thumbnail = new URL(attachment.thumbnail_url)
              thumbnail.searchParams.set('refresh', '1')
              attachment.thumbnail_url = thumbnail.href
            }
          }
        })
      }).as('recoveredConversation')
      cy.window().then((win) => {
        const frame = win.Libredesk.iframe.contentWindow
        frame.dispatchEvent(new frame.Event('online'))
      })
      cy.wait('@recoveredConversation').its('response.statusCode').should('eq', 200)
      image().should(($image) => {
        expect($image[0]).to.eq(originalImage)
        expect($image[0].naturalWidth).to.be.greaterThan(0)
        expect($image.attr('src')).not.to.include('/thumb_')
      })
    })
  })

  for (const rating of [0, 5]) {
    it(`shows submitted CSAT feedback live in the agent view with rating ${rating}`, () => {
      const feedback = `Live customer feedback ${rating} ${Date.now()}`
      let inbox
      let session
      let originalDocument
      cy.createLivechatInbox({}, { csat_enabled: true }).then((created) => { inbox = created })
      cy.then(() => cy.widgetInit(inbox.uuid)).then((created) => { session = created })
      cy.then(() => cy.visit(`/inboxes/all/conversation/${session.conversationUuid}`))
      cy.document().then((doc) => { originalDocument = doc })
      cy.then(() => cy.api('PUT', `/api/v1/conversations/${session.conversationUuid}/status`, {
        status: 'Resolved'
      })).its('status').should('eq', 200)
      cy.contains('[data-message-uuid]', 'Awaiting Response', { timeout: 20000 }).should('be.visible')
      cy.then(() => cy.widgetApi('GET', `/api/v1/widget/chat/conversations/${session.conversationUuid}`,
        session.sessionToken, inbox.uuid)).then(({ body }) => {
        const survey = body.data.messages.find((message) => message.meta?.is_csat)
        expect(survey?.meta.csat_uuid).to.be.a('string')
        cy.request('POST', `/api/v1/csat/${survey.meta.csat_uuid}/response`, { rating, feedback })
          .its('status').should('eq', 200)
        cy.get(`[data-message-uuid="${survey.uuid}"]`).should('contain.text', feedback)
          .and('not.contain.text', 'Awaiting Response')
        if (rating) cy.get(`[data-message-uuid="${survey.uuid}"]`).should('contain.text', '5/5')
      })
      cy.document().should((doc) => { expect(doc).to.eq(originalDocument) })
    })
  }

  it('moves the launcher when the inbox launcher position changes', () => {
    cy.createLivechatInbox(
      embedConfig({ launcher: { position: 'right', spacing: { side: 20, bottom: 20 } } })
    ).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetWrapperSide().then((side) => {
        expect(side.right, 'launcher not on the right').to.eq('20px')
        expect(side.left).to.eq('auto')
      })

      cy.then(() =>
        cy.saveLivechatInbox(inbox, { launcher: { position: 'left', spacing: { side: 40, bottom: 60 } } })
      )
      cy.then(() => cy.visitWidgetHost(inbox.uuid))
      cy.widgetWrapperSide().then((side) => {
        expect(side.left, 'launcher did not move to the left').to.eq('40px')
        expect(side.right).to.eq('auto')
        expect(side.bottom).to.eq('60px')
      })
    })
  })

  it('starts a conversation as a visitor and shows the agent reply', () => {
    const visitorMessage = `Visitor from the widget ${Date.now()}`
    const agentMessage = `Agent answer ${Date.now()}`
    let inbox

    cy.createLivechatInbox(embedConfig()).then((created) => {
      inbox = created
    })
    cy.then(() => cy.visitWidgetHost(inbox.uuid))
    cy.widgetLauncher().click()

    cy.widgetBody().contains(startButtonText).click()
    cy.widgetBody().find('textarea').should('be.visible').type(visitorMessage)
    cy.widgetBody().find('button[aria-label="Send"]').click()
    cy.widgetBody().contains(visitorMessage, { timeout: 20000 }).should('be.visible')

    cy.then(() => cy.agentReplyToLatestConversation(inbox, agentMessage))
    cy.widgetBody().contains(agentMessage, { timeout: 20000 }).should('be.visible')
  })

  it('keeps delivering agent replies in the widget after the inbox is saved', () => {
    const visitorMessage = `Visitor before save ${Date.now()}`
    const beforeSave = `Agent before save ${Date.now()}`
    const afterSave = `Agent after save ${Date.now()}`
    let inbox

    cy.createLivechatInbox(embedConfig()).then((created) => {
      inbox = created
    })
    cy.then(() => cy.visitWidgetHost(inbox.uuid))
    cy.widgetLauncher().click()
    cy.widgetBody().contains(startButtonText).click()
    cy.widgetBody().find('textarea').should('be.visible').type(visitorMessage)
    cy.widgetBody().find('button[aria-label="Send"]').click()
    cy.widgetBody().contains(visitorMessage, { timeout: 20000 }).should('be.visible')

    cy.then(() => cy.agentReplyToLatestConversation(inbox, beforeSave))
    cy.widgetBody().contains(beforeSave, { timeout: 20000 }).should('be.visible')

    cy.then(() => cy.saveLivechatInbox(inbox, { brand_name: 'Cypress saved' }))

    cy.then(() => cy.agentReplyToLatestConversation(inbox, afterSave))
    cy.widgetBody().contains(afterSave, { timeout: 30000 }).should('be.visible')
  })

  it('identifies the contact from a signed JWT', () => {
    const stamp = Date.now()
    const email = `jwt.visitor.${stamp}@cypress.test`
    const secret = `cypress-secret-${stamp}`
    const visitorMessage = `JWT visitor message ${stamp}`
    let inbox

    cy.createLivechatInbox(embedConfig(), { secret }).then((created) => {
      inbox = created
    })

    cy.then(() => {
      cy.intercept('POST', '/api/v1/widget/chat/auth/exchange').as('exchange')
      return cy.visitWidgetHost(inbox.uuid, {
        secret,
        jwtPayload: {
          external_user_id: `cypress_${stamp}`,
          email,
          first_name: 'Cypress',
          last_name: 'Visitor'
        }
      })
    })
    cy.wait('@exchange').its('response.statusCode').should('eq', 200)

    cy.widgetLauncher().click()
    cy.widgetBody().contains(startButtonText).click()
    cy.widgetBody().find('textarea').should('be.visible').type(visitorMessage)
    cy.widgetBody().find('button[aria-label="Send"]').click()
    cy.widgetBody().contains(visitorMessage, { timeout: 20000 }).should('be.visible')

    cy.then(() =>
      cy.latestConversation(inbox).then((conversation) => {
        expect(JSON.stringify(conversation), 'conversation not attached to the JWT contact').to.include(email)
      })
    )
  })

  it('enforces a required pre-chat field before the chat opens', () => {
    cy.createLivechatInbox(
      embedConfig({
        prechat_form: {
          enabled: true,
          title: 'Before we start',
          fields: [
            {
              key: 'name',
              type: 'text',
              label: 'Name',
              required: true,
              enabled: true,
              order: 1,
              is_default: true
            }
          ]
        }
      })
    ).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetLauncher().click()
      cy.widgetBody().contains(startButtonText).click()
      cy.widgetBody().contains('Before we start').should('be.visible')
      cy.widgetBody().find('input').should('exist')
    })
  })
})
