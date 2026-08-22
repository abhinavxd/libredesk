describe('Live chat widget settings and init rules', () => {
  it('serves launcher settings without a session', () => {
    cy.createLivechatInbox({ launcher: { position: 'left' } }).then((inbox) => {
      cy.widgetApi('GET', '/api/v1/widget/chat/settings/launcher', null, inbox.uuid).then(
        ({ status, body }) => {
          expect(status).to.eq(200)
          expect(body.data.launcher.position).to.eq('left')
        }
      )
    })
  })

  it('withholds server-side config from the public settings response', () => {
    cy.createLivechatInbox({
      blocked_ips: ['203.0.113.7'],
      trusted_domains: ['example.test'],
      session_duration: '4h'
    }).then((inbox) => {
      cy.widgetApi('GET', '/api/v1/widget/chat/settings', null, inbox.uuid).then(({ status, body }) => {
        expect(status).to.eq(200)
        expect(body.data.brand_name).to.eq('Cypress')
        expect(body.data).to.not.have.property('blocked_ips')
        expect(body.data).to.not.have.property('trusted_domains')
        expect(body.data).to.not.have.property('session_duration')
      })
    })
  })

  it('publishes the enabled pre-chat form fields', () => {
    cy.createLivechatInbox({
      prechat_form: {
        enabled: true,
        title: 'Before we start',
        fields: [
          { key: 'name', type: 'text', label: 'Name', required: true, enabled: true, order: 1, is_default: true },
          { key: 'email', type: 'email', label: 'Email', required: false, enabled: false, order: 2, is_default: true }
        ]
      }
    }).then((inbox) => {
      cy.widgetApi('GET', '/api/v1/widget/chat/settings', null, inbox.uuid).then(({ body }) => {
        const keys = body.data.prechat_form.fields.map((f) => f.key)
        expect(body.data.prechat_form.enabled).to.be.true
        expect(keys).to.include('name')
      })
    })
  })

  it('refuses init when a required pre-chat field is missing', () => {
    cy.createLivechatInbox({
      prechat_form: {
        enabled: true,
        fields: [
          { key: 'name', type: 'text', label: 'Name', required: true, enabled: true, order: 1, is_default: true }
        ]
      }
    }).then((inbox) => {
      cy.widgetInit(inbox.uuid, { form_data: {} }, { failOnStatusCode: false }).then((res) => {
        expect(res.status, 'init accepted a missing required field').to.eq(400)
        expect(res.body.message, 'error did not name the field').to.match(/name/i)
      })
      cy.widgetInit(inbox.uuid, { form_data: { name: 'Cypress Visitor' } }).its('status').should('eq', 200)
    })
  })

  it('refuses init when an email pre-chat field is malformed', () => {
    cy.createLivechatInbox({
      prechat_form: {
        enabled: true,
        fields: [
          { key: 'email', type: 'email', label: 'Email', required: true, enabled: true, order: 1, is_default: true }
        ]
      }
    }).then((inbox) => {
      cy.widgetInit(inbox.uuid, { form_data: { email: 'not-an-email' } }, { failOnStatusCode: false }).then(
        (res) => {
          expect(res.status, 'init accepted a malformed email').to.eq(400)
          expect(res.body.message, 'error did not mention the email').to.match(/email/i)
        }
      )
      cy.widgetInit(inbox.uuid, { form_data: { email: 'visitor@example.test' } })
        .its('status')
        .should('eq', 200)
    })
  })

  it('refuses init when visitors may not start conversations', () => {
    cy.createLivechatInbox({ visitors: { allow_start_conversation: false } }).then((inbox) => {
      cy.widgetInit(inbox.uuid, {}, { failOnStatusCode: false }).its('status').should('eq', 400)
    })
  })

  it('refuses a second conversation when multiples are prevented', () => {
    cy.createLivechatInbox({ visitors: { prevent_multiple_conversations: true } }).then((inbox) => {
      let token
      cy.widgetInit(inbox.uuid).then((res) => {
        token = res.sessionToken
      })
      cy.then(() =>
        cy
          .request({
            method: 'POST',
            url: '/api/v1/widget/chat/conversations/init',
            headers: {
              'X-Libredesk-Inbox-ID': inbox.uuid,
              Authorization: `Bearer ${token}`
            },
            body: { message: 'Second conversation' },
            failOnStatusCode: false
          })
          .its('status')
          .should('eq', 403)
      )
    })
  })

  it('rejects init without a message', () => {
    cy.createLivechatInbox().then((inbox) => {
      cy.request({
        method: 'POST',
        url: '/api/v1/widget/chat/conversations/init',
        headers: { 'X-Libredesk-Inbox-ID': inbox.uuid },
        body: {},
        failOnStatusCode: false
      })
        .its('status')
        .should('eq', 400)
    })
  })

  it('names the field when a pre-chat value is too long', () => {
    cy.createLivechatInbox({
      prechat_form: {
        enabled: true,
        fields: [
          { key: 'name', type: 'text', label: 'Name', required: true, enabled: true, order: 1, is_default: true }
        ]
      }
    }).then((inbox) => {
      cy.widgetInit(inbox.uuid, { form_data: { name: 'x'.repeat(300) } }, { failOnStatusCode: false }).then(
        (res) => {
          expect(res.status).to.eq(400)
          expect(res.body.message, 'error does not name the field').to.match(/name/i)
        }
      )
    })
  })
})
