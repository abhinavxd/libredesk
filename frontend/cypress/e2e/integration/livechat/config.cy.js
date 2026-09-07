const startText = 'Cypress start chat'

const withStart = (overrides = {}) => ({
  visitors: { allow_start_conversation: true, start_conversation_button_text: startText },
  users: { allow_start_conversation: true, start_conversation_button_text: startText },
  ...overrides
})

describe('Live chat widget config applied end to end', () => {
  it('shows brand name, logo, greeting and introduction on the home screen', () => {
    cy.createLivechatInbox(
      withStart({
        brand_name: 'Cypress Brand',
        logo_url: `${Cypress.config('baseUrl')}/static/public/launcher-logo.png`,
        greeting_message: 'Hello from Cypress',
        introduction_message: 'We reply fast'
      })
    ).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetLauncher().click()
      cy.widgetBody().contains('Hello from Cypress').should('be.visible')
      cy.widgetBody().contains('We reply fast').should('be.visible')
      cy.widgetBody().find('img[src*="launcher-logo.png"]').should('exist')
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().contains('Cypress Brand').should('be.visible')
    })
  })

  it('applies the launcher color from config', () => {
    cy.createLivechatInbox(
      withStart({ colors: { primary: '#ff0000' }, launcher: { color: '#00ff00', position: 'right', spacing: { side: 20, bottom: 20 } } })
    ).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid)
      cy.widgetLauncher().then((el) => {
        const style = el[0].ownerDocument.defaultView.getComputedStyle(el[0])
        expect(style.backgroundColor, 'launcher color not applied').to.eq('rgb(0, 255, 0)')
      })
    })
  })

  it('shows the notice banner only when enabled', () => {
    cy.createLivechatInbox(
      withStart({ notice_banner: { enabled: true, text: 'Cypress notice text' } })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().contains('Cypress notice text').should('be.visible')
    })

    cy.createLivechatInbox(
      withStart({ notice_banner: { enabled: false, text: 'Hidden notice' } })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().find('textarea').should('be.visible')
      cy.widgetBody().contains('Hidden notice').should('not.exist')
    })
  })

  it('hides the powered-by link when show_powered_by is false', () => {
    cy.createLivechatInbox(withStart({ show_powered_by: true })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().find('a[href="https://libredesk.io"]').should('exist')
    })

    cy.createLivechatInbox(withStart({ show_powered_by: false })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().find('textarea').should('be.visible')
      cy.widgetBody().find('a[href="https://libredesk.io"]').should('not.exist')
    })
  })

  it('toggles the emoji and file upload actions from features', () => {
    cy.createLivechatInbox(withStart({ features: { emoji: true, file_upload: true } })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().find('button[aria-label="Add emoji"]').should('exist')
      // The attach button only renders once a conversation exists to upload against.
      cy.widgetSend(`Message for upload ${Date.now()}`)
      cy.widgetBody().find('button[aria-label="Attach file"]').should('exist')
    })

    cy.createLivechatInbox(withStart({ features: { emoji: false, file_upload: false } })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetBody().find('textarea').should('be.visible')
      cy.widgetBody().find('button[aria-label="Add emoji"]').should('not.exist')
      cy.widgetBody().find('button[aria-label="Attach file"]').should('not.exist')
    })
  })

  it('renders configured home apps', () => {
    cy.createLivechatInbox(
      withStart({
        home_apps: [
          { type: 'announcement', title: 'Cypress announcement', description: 'Read this', url: '' },
          { type: 'external_link', text: 'Cypress docs link', url: 'https://example.test/docs' }
        ]
      })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains('Cypress announcement').should('be.visible')
      cy.widgetBody().contains('Cypress docs link').should('be.visible')
    })
  })

  it('opens straight into the chat when direct_to_conversation is set', () => {
    cy.createLivechatInbox(withStart({ direct_to_conversation: true })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().find('textarea', { timeout: 20000 }).should('be.visible')
    })
  })

  it('hides the start button when visitors may not start conversations', () => {
    let inbox
    cy.createLivechatInbox(
      withStart({ visitors: { allow_start_conversation: false, start_conversation_button_text: startText } })
    ).then((created) => {
      inbox = created
    })
    cy.then(() => cy.openWidget(inbox))
    cy.widgetBody().contains('Home').should('be.visible')
    cy.widgetBody().should('not.contain', startText)
    cy.then(() =>
      cy.conversationForInbox(inbox).then((conversation) => {
        expect(conversation, 'a conversation was created anyway').to.be.null
      })
    )
  })

  it('replaces the start button with the running conversation when multiples are prevented', () => {
    cy.createLivechatInbox(
      withStart({
        visitors: {
          allow_start_conversation: true,
          prevent_multiple_conversations: true,
          start_conversation_button_text: startText
        }
      })
    ).then((inbox) => {
      const firstMessage = `First conversation ${Date.now()}`
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText).click()
      cy.widgetSend(firstMessage)
      cy.widgetBody().find('button[aria-label="Go back"]').click()
      cy.widgetBody().contains('Home').click()
      cy.widgetBody().contains(firstMessage, { timeout: 20000 }).should('be.visible')
      cy.widgetBody().should('not.contain', startText)
    })
  })

  it('blocks replies to a closed conversation when configured', () => {
    let inbox
    cy.createLivechatInbox(
      withStart({
        visitors: {
          allow_start_conversation: true,
          prevent_reply_to_closed_conversation: true,
          start_conversation_button_text: startText
        }
      })
    ).then((created) => {
      inbox = created
    })
    cy.then(() => cy.openWidget(inbox))
    cy.widgetBody().contains(startText).click()
    cy.widgetSend(`Message before close ${Date.now()}`)
    cy.then(() => cy.closeConversation(inbox))
    cy.widgetBody().find('textarea', { timeout: 20000 }).should('not.exist')
  })

  it('applies dark mode', () => {
    cy.createLivechatInbox(withStart({ dark_mode: true })).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().find('.dark').should('exist')
    })
  })

  it('paints the home screen background from config', () => {
    cy.createLivechatInbox(
      withStart({
        home_screen: {
          header_text_color: 'light',
          background: { type: 'gradient', gradient_start: '#112233', gradient_end: '#445566' },
          fade_background: true
        }
      })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains(startText)
      cy.widgetBody().then((body) => {
        const hasGradient = Array.from(body[0].querySelectorAll('div')).some((el) =>
          (el.getAttribute('style') || '').includes('gradient')
        )
        expect(hasGradient, 'gradient background not applied').to.be.true
      })
    })
  })

  it('shows the reply expectation message once business hours are configured', () => {
    let inbox
    cy.login()
    cy.api('POST', '/api/v1/business-hours', {
      name: `Cypress always open ${Date.now()}`,
      is_always_open: true,
      hours: {},
      holidays: []
    })
      .its('body.data.id')
      .then((id) => cy.setDefaultBusinessHours(String(id)))

    cy.createLivechatInbox(
      withStart({
        chat_reply_expectation_message: 'We usually reply in 5 minutes',
        show_office_hours_in_chat: true
      })
    ).then((created) => {
      inbox = created
    })
    cy.then(() => cy.openWidget(inbox))
    cy.widgetBody().contains(startText).click()
    // The message only renders against a live conversation.
    cy.then(() => cy.widgetSend(`Expectation probe ${Date.now()}`))
    cy.widgetBody().contains('We usually reply in 5 minutes', { timeout: 20000 }).should('be.visible')
    cy.then(() => cy.setDefaultBusinessHours(''))
  })

  it('renders the widget in the configured language', () => {
    cy.createLivechatInbox({
      language: 'de-DE',
      fallback_language: 'en-US',
      visitors: { allow_start_conversation: true },
      users: { allow_start_conversation: true }
    }).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains('Senden Sie uns eine Nachricht', { timeout: 20000 }).should('be.visible')
    })
  })

  it('collects pre-chat form fields and attaches them to the contact', () => {
    const stamp = Date.now()
    const email = `prechat.${stamp}@cypress.test`
    let inbox

    cy.createLivechatInbox(
      withStart({
        prechat_form: {
          enabled: true,
          title: 'Tell us about you',
          fields: [
            { key: 'name', type: 'text', label: 'Name', required: true, enabled: true, order: 1, is_default: true },
            { key: 'email', type: 'email', label: 'Email', required: true, enabled: true, order: 2, is_default: true }
          ]
        }
      })
    ).then((created) => {
      inbox = created
    })

    cy.then(() => cy.openWidget(inbox))
    cy.widgetBody().contains(startText).click()
    cy.widgetBody().contains('Tell us about you').should('be.visible')

    cy.widgetBody().contains('button', 'Start chat').should('be.disabled')
    cy.widgetBody().find('input').eq(0).type('Cypress Prechat')
    cy.widgetBody().find('input').eq(1).type(email)
    cy.widgetBody().find('textarea').should('be.visible').type(`Prechat message ${stamp}`)
    cy.widgetBody().contains('button', 'Start chat').click()
    cy.widgetBody().contains(`Prechat message ${stamp}`, { timeout: 20000 }).should('be.visible')

    cy.then(() =>
      cy.latestConversation(inbox).then((conversation) => {
        expect(conversation.contact.email, 'pre-chat email not saved on the contact').to.eq(email)
        expect(JSON.stringify(conversation.contact)).to.include('Cypress Prechat')
      })
    )
  })

  it('keeps the messages-tab button off when visitors may not start one but multiples are prevented', () => {
    cy.createLivechatInbox(
      withStart({
        visitors: {
          allow_start_conversation: true,
          prevent_multiple_conversations: true,
          start_conversation_button_text: startText
        }
      })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains('Messages').click()
      cy.widgetBody().contains(startText).should('be.visible')
    })

    cy.createLivechatInbox(
      withStart({
        visitors: {
          allow_start_conversation: false,
          prevent_multiple_conversations: true,
          start_conversation_button_text: startText
        }
      })
    ).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains('Messages').click()
      cy.widgetBody().contains('Home').should('be.visible')
      cy.widgetBody().should('not.contain', startText)
    })
  })

  it('uses the visitor label on the messages tab, not the agent-side one', () => {
    cy.createLivechatInbox({
      visitors: { allow_start_conversation: true, start_conversation_button_text: 'Visitor label' },
      users: { allow_start_conversation: true, start_conversation_button_text: 'User label' }
    }).then((inbox) => {
      cy.openWidget(inbox)
      cy.widgetBody().contains('Messages').click()
      cy.widgetBody().contains('Visitor label').should('be.visible')
      cy.widgetBody().should('not.contain', 'User label')
    })
  })
})
