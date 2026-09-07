// Credentials here are dummies: saving a WhatsApp inbox always makes Meta verify them, so a create is expected to be rejected.

describe('API: whatsapp', () => {
  const stamp = Date.now()
  const name = `api.whatsapp.${stamp}`

  const config = {
    phone_number_id: `pn-${stamp}`,
    waba_id: `waba-${stamp}`,
    access_token: 'dummy-access-token',
    app_secret: 'dummy-app-secret',
    webhook_verify_token: `verify-${stamp}`,
    api_version: 'v25.0'
  }

  const createInbox = (overrides, options = {}) =>
    cy.api('POST', '/api/v1/inboxes', {
      name,
      channel: 'whatsapp',
      enabled: true,
      config: { ...config, ...overrides }
    }, { failOnStatusCode: false, ...options })

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects an inbox with no phone number id', () => {
    createInbox({ phone_number_id: '' }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/phone_number_id/i)
    })
  })

  it('rejects an inbox with no waba id', () => {
    createInbox({ waba_id: '' }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/waba_id/i)
    })
  })

  // Without the verify token Meta's webhook handshake can never succeed, so inbound would stay dead.
  it('rejects an inbox with no webhook verify token', () => {
    createInbox({ webhook_verify_token: '' }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/webhook_verify_token/i)
    })
  })

  it('rejects an inbox with no access token', () => {
    createInbox({ access_token: '' }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/access_token/i)
    })
  })

  // The app secret signs inbound webhooks, so without it every delivery is rejected.
  it('rejects an inbox with no app secret', () => {
    createInbox({ app_secret: '' }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/app_secret/i)
    })
  })

  it('rejects an inbox whose credentials Meta does not accept', () => {
    // Deterministic either way: the stand-in Graph API is told to refuse, and real Meta refuses anyway.
    cy.task('metaMock:failValidate', true)
    createInbox({}, { timeout: 90000 }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
    cy.task('metaMock:failValidate', false)
  })

  it('does not create an inbox for any of the rejected attempts', () => {
    cy.api('GET', '/api/v1/inboxes').then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.filter((inbox) => inbox.name === name)).to.have.length(0)
    })
  })

  it('requires an inbox id to list templates', () => {
    cy.api('GET', '/api/v1/whatsapp/templates', null, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/inbox_id/i)
    })
  })

  it('returns not found for an unknown template', () => {
    cy.api('GET', '/api/v1/whatsapp/templates/99999999', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
  })

  it('rejects a template with no inbox, name, language, category or body', () => {
    cy.api('POST', '/api/v1/whatsapp/templates', { name: 'no_inbox' }, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
        expect(body.message).to.match(/inbox_id/i)
      })
  })

  it('requires an inbox id to sync templates', () => {
    cy.api('POST', '/api/v1/whatsapp/templates/sync', null, { failOnStatusCode: false })
      .then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
  })

  describe('webhook endpoint', () => {
    const payload = {
      object: 'whatsapp_business_account',
      entry: [{
        id: 'waba-1',
        changes: [{
          field: 'messages',
          value: {
            metadata: { phone_number_id: 'pn-1' },
            messages: [{ from: '919876543210', id: 'wamid.CYPRESS', timestamp: '1716000000', type: 'text', text: { body: 'hi' } }]
          }
        }]
      }]
    }

    // The webhook is public by design - Meta authenticates with a signature, not a session.
    it('rejects a delivery with no signature instead of asking for a login', () => {
      cy.request({
        method: 'POST',
        url: '/webhooks/whatsapp/99999999',
        body: payload,
        failOnStatusCode: false
      }).then(({ status, body }) => {
        expect(status).to.not.eq(401)
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
    })

    it('rejects a delivery aimed at a non-numeric inbox id', () => {
      cy.request({
        method: 'POST',
        url: '/webhooks/whatsapp/not-an-id',
        body: payload,
        failOnStatusCode: false
      }).then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
    })

    it('rejects a delivery aimed at an inbox that is not WhatsApp', () => {
      cy.api('GET', '/api/v1/inboxes').then(({ body }) => {
        const other = body.data.find((inbox) => inbox.channel !== 'whatsapp')
        if (!other) return
        cy.request({
          method: 'POST',
          url: `/webhooks/whatsapp/${other.id}`,
          body: payload,
          headers: { 'X-Hub-Signature-256': 'sha256=deadbeef' },
          failOnStatusCode: false
        }).then(({ status }) => {
          expect(status).to.eq(404)
        })
      })
    })

    it('rejects a verification handshake with the wrong hub.mode', () => {
      cy.request({
        url: '/webhooks/whatsapp/99999999?hub.mode=unsubscribe&hub.verify_token=x&hub.challenge=y',
        failOnStatusCode: false
      }).then(({ status, body }) => {
        expect(status).to.eq(400)
        expect(body.error_type).to.eq('InputException')
      })
    })

    it('rejects a verification handshake for an unknown inbox', () => {
      cy.request({
        url: '/webhooks/whatsapp/99999999?hub.mode=subscribe&hub.verify_token=x&hub.challenge=y',
        failOnStatusCode: false
      }).then(({ status, body }) => {
        expect(status).to.eq(404)
        expect(body.error_type).to.eq('NotFoundException')
      })
    })
  })
})
