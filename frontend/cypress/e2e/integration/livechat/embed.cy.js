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
