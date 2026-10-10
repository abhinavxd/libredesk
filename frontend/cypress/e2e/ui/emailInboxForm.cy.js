const inboxId = 10001
const editPath = `/admin/inboxes/${inboxId}/edit`
const newPath = '/admin/inboxes/new'
const maskedSecret = '••••••••••'
const field = (name) => cy.get(`input[name="${name}"]`)
const submit = () => cy.get('form button[type="submit"]').click()

const inboxFixture = () => ({
  id: inboxId,
  name: 'Support',
  channel: 'email',
  from: 'Support <support@example.com>',
  enabled: true,
  csat_enabled: true,
  prompt_tags_on_reply: true,
  from_name_template: '{{ .Agent.FirstName }} at {{ .Inbox.Name }}',
  aliases: [
    { email: 'billing@example.com', verification_status: 'verified' },
    { email: 'sales@example.com', verification_status: 'not_verified' }
  ],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  config: {
    auth_type: 'password',
    reply_to: 'replies@example.com',
    enable_plus_addressing: true,
    imap: [
      {
        host: 'imap.example.com',
        port: 993,
        mailbox: 'INBOX',
        username: 'support@example.com',
        password: maskedSecret,
        tls_type: 'tls',
        read_interval: '5m',
        scan_inbox_since: '48h',
        tls_skip_verify: false
      }
    ],
    smtp: [
      {
        host: 'smtp.example.com',
        port: 587,
        username: 'support@example.com',
        password: maskedSecret,
        max_conns: 10,
        max_msg_retries: 3,
        idle_timeout: '25s',
        pool_wait_timeout: '120s',
        auth_protocol: 'login',
        tls_type: 'starttls',
        hello_hostname: 'mail.example.com',
        tls_skip_verify: false
      }
    ]
  }
})

const openManualForm = () => {
  cy.visit(newPath)
  cy.contains('Create an email inbox').click()
  cy.contains('button', 'Configure IMAP and SMTP manually').click()
}

const fillRequiredFields = () => {
  field('name').type('New support inbox')
  field('from').type('support@example.com')
  field('imap.username').type('support@example.com')
  field('imap.password').type('test-password')
  field('smtp.username').type('support@example.com')
  field('smtp.password').type('test-password')
}

describe('Email inbox form', () => {
  let inbox

  beforeEach(() => {
    inbox = inboxFixture()
    cy.viewport(1280, 900)
    cy.login()
    cy.intercept('**/api/v1/**', { body: { data: [] } })
    cy.readFile('../i18n/en-US.json').then((messages) => {
      cy.intercept('GET', '**/api/v1/lang/en-US', { body: messages })
    })
    cy.intercept('GET', '**/api/v1/config', { body: { data: { 'app.lang': 'en-US' } } })
    cy.intercept('GET', '**/api/v1/settings/general', {
      body: { data: { 'app.root_url': 'http://localhost:8000' } }
    })
    cy.intercept('GET', '**/api/v1/agents/me', {
      body: {
        data: {
          id: 1,
          first_name: 'Test',
          last_name: 'Admin',
          email: 'admin@example.com',
          roles: ['Admin'],
          permissions: ['inboxes:manage'],
          teams: []
        }
      }
    })
    cy.intercept('GET', '**/api/v1/inboxes', (req) => req.reply({ data: [inbox] }))
    cy.intercept('GET', `**/api/v1/inboxes/${inboxId}`, (req) => req.reply({ data: inbox }))
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`, (req) => {
      inbox = { ...inbox, ...req.body }
      req.reply({ data: inbox })
    }).as('updateInbox')
    cy.intercept('POST', '**/api/v1/inboxes', (req) => {
      inbox = { ...inbox, ...req.body }
      req.reply({ data: inbox })
    }).as('createInbox')
  })

  it('shows setup choices before a new inbox and keeps manual connection fields open', () => {
    cy.visit(newPath)
    cy.contains('Create an email inbox').click()
    cy.get('form button[type="submit"]').should('not.be.visible')
    cy.contains('button', 'Configure IMAP and SMTP manually').focus()
    cy.focused().type(' ')
    field('name').should('be.visible')
    field('imap.host').should('have.length', 1).and('be.visible')
    field('smtp.host').scrollIntoView().should('have.length', 1).and('be.visible')
    field('reply_to').should('not.be.visible')
    field('imap.read_interval').should('not.be.visible')
  })

  it('creates an inbox with values from closed sections', () => {
    openManualForm()
    fillRequiredFields()
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    submit()
    cy.wait('@createInbox').then(({ request }) => {
      expect(request.body.name).to.eq('New support inbox')
      expect(request.body.config.imap[0].read_interval).to.eq('5m')
      expect(request.body.config.smtp[0].max_conns).to.eq(10)
      expect(request.body.config.enable_plus_addressing).to.eq(true)
    })
    cy.location('pathname').should('eq', '/admin/inboxes')
    cy.contains('New support inbox').should('be.visible')
  })

  it('loads saved fields once and saves values from closed sections', () => {
    cy.visit(editPath)
    field('name').should('have.value', 'Support')
    field('imap.host')
      .should('have.length', 1)
      .and('have.value', 'imap.example.com')
      .and('not.be.visible')
    field('smtp.host')
      .should('have.length', 1)
      .and('have.value', 'smtp.example.com')
      .and('not.be.visible')
    field('imap.password').should('have.value', maskedSecret)
    field('smtp.password').should('have.value', maskedSecret)
    field('name').clear().type('Renamed support')
    submit()
    cy.wait('@updateInbox').then(({ request }) => {
      expect(request.body.name).to.eq('Renamed support')
      expect(request.body.aliases).to.deep.eq(inboxFixture().aliases)
      expect(request.body.from_name_template).to.eq(inboxFixture().from_name_template)
      expect(request.body.csat_enabled).to.eq(true)
      expect(request.body.prompt_tags_on_reply).to.eq(true)
      expect(request.body.config).to.deep.eq({
        ...inboxFixture().config,
        imap: [{ ...inboxFixture().config.imap[0], password: '' }],
        smtp: [{ ...inboxFixture().config.smtp[0], password: '' }]
      })
    })
    cy.reload()
    field('name').should('have.value', 'Renamed support')
  })

  it('retains edits inside closed sections and advanced settings', () => {
    cy.visit(editPath)
    cy.setEmailInboxSection('options', true)
    field('reply_to').clear().type('new-replies@example.com')
    field('aliases[0].email').clear().type('accounts@example.com')
    cy.setEmailInboxSection('imap', true)
    cy.setEmailInboxSection('smtp', true)
    cy.toggleEmailInboxAdvanced('imap')
    field('imap.read_interval').clear().type('10m')
    field('imap.mailbox').clear().type('Archive')
    cy.toggleEmailInboxAdvanced('imap')
    cy.toggleEmailInboxAdvanced('smtp')
    field('smtp.max_conns').clear().type('5')
    cy.toggleEmailInboxAdvanced('smtp')
    cy.setEmailInboxSection('options', true)
    cy.contains('[data-section="options"] p', /^CSAT surveys$/i)
      .parent()
      .find('[role="switch"]')
      .click()
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    cy.setEmailInboxSection('options', false)
    submit()
    cy.wait('@updateInbox').then(({ request }) => {
      expect(request.body.config.reply_to).to.eq('new-replies@example.com')
      expect(request.body.aliases[0].email).to.eq('accounts@example.com')
      expect(request.body.config.imap[0].read_interval).to.eq('10m')
      expect(request.body.config.imap[0].mailbox).to.eq('Archive')
      expect(request.body.config.smtp[0].max_conns).to.eq(5)
      expect(request.body.csat_enabled).to.eq(false)
    })
  })

  it('reveals every invalid section and focuses the first invalid field', () => {
    cy.visit(editPath)
    cy.setEmailInboxSection('options', true)
    field('aliases[0].email').clear().type('invalid')
    field('reply_to').clear().type('invalid')
    field('from_name_template').clear().type('{{ invalid }}', { parseSpecialCharSequences: false })
    cy.setEmailInboxSection('imap', true)
    cy.setEmailInboxSection('smtp', true)
    cy.toggleEmailInboxAdvanced('imap')
    field('imap.mailbox').clear()
    field('imap.read_interval').clear().type('forever')
    cy.toggleEmailInboxAdvanced('imap')
    cy.toggleEmailInboxAdvanced('smtp')
    field('smtp.idle_timeout').clear().type('forever')
    cy.toggleEmailInboxAdvanced('smtp')
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    cy.setEmailInboxSection('options', false)
    submit()
    field('aliases[0].email').should('be.visible').and('have.focus')
    for (const name of ['reply_to', 'from_name_template']) {
      field(name).scrollIntoView().should('be.visible').and('have.attr', 'aria-invalid', 'true')
    }
    for (const name of ['imap.mailbox', 'imap.read_interval', 'smtp.idle_timeout']) {
      field(name).scrollIntoView().should('be.visible').and('have.attr', 'aria-invalid', 'true')
    }
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('validates on submit and clears errors as fields are corrected', () => {
    cy.visit(editPath)
    cy.setEmailInboxSection('options', true)
    field('reply_to').clear().type('half-filled')
    field('reply_to').should('have.attr', 'aria-invalid', 'false').blur()
    field('reply_to').should('have.attr', 'aria-invalid', 'false')
    submit()
    field('reply_to').should('have.attr', 'aria-invalid', 'true')
    field('reply_to').clear().type('replies@example.com')
    field('reply_to').should('have.attr', 'aria-invalid', 'false')
  })

  it('opens connection settings when required new-inbox fields are missing', () => {
    openManualForm()
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    submit()
    field('name').should('have.focus').and('have.attr', 'aria-invalid', 'true')
    field('imap.username')
      .scrollIntoView()
      .should('be.visible')
      .and('have.attr', 'aria-invalid', 'true')
    field('smtp.password')
      .scrollIntoView()
      .should('be.visible')
      .and('have.attr', 'aria-invalid', 'true')
    cy.get('@createInbox.all').should('have.length', 0)
  })

  it('reveals a blank or duplicate alias on save', () => {
    cy.visit(editPath)
    cy.get('[data-section="aliases"]').contains('button', 'Add').click()
    field('aliases[2].email').should('have.focus').and('have.attr', 'aria-invalid', 'false')
    cy.get('[data-alias-row]')
      .last()
      .contains('[role="status"]', 'Save the inbox first to verify this alias.')
      .should('be.visible')
    cy.get('[data-alias-row]')
      .last()
      .contains('button', /^Verify$/)
      .should('be.disabled')
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    cy.setEmailInboxSection('options', false)
    submit()
    field('aliases[2].email').should('be.visible').and('have.focus')
    field('aliases[2].email').type('billing@example.com')
    submit()
    field('aliases[2].email').should('have.attr', 'aria-invalid', 'true')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('shows verification states and verifies an alias without submitting or losing edits', () => {
    inbox.aliases = ['not_verified', 'pending', 'verified', 'failed'].map((status, index) => ({
      email: `alias${index}@example.com`,
      verification_status: status
    }))
    cy.intercept('POST', `**/api/v1/inboxes/${inboxId}/aliases/verify`, (req) => {
      inbox.aliases = inbox.aliases.map((alias) =>
        alias.email === req.body.email ? { ...alias, verification_status: 'pending' } : alias
      )
      req.reply({ data: true })
    }).as('verifyAlias')
    cy.visit(editPath)
    field('name').clear().type('Unsaved name')
    for (const label of ['Not verified', 'Pending', 'Verified', 'Failed']) {
      cy.get('[data-alias-row]').contains('[role="status"]', label).scrollIntoView()
      cy.get('[data-alias-row]').contains('[role="status"]', label).should('be.visible')
    }
    cy.get('[data-section="aliases"]').scrollIntoView()
    cy.screenshot('email-alias-states', { capture: 'viewport' })
    cy.get('[data-alias-row]')
      .first()
      .contains('button', /^Verify$/)
      .click()
    cy.wait('@verifyAlias').its('request.body.email').should('eq', 'alias0@example.com')
    cy.get('[data-alias-row]').first().contains('Pending').should('be.visible')
    field('name').should('have.value', 'Unsaved name')
    field('aliases[2].email').clear().type('renamed@example.com')
    cy.get('[data-alias-row]')
      .eq(2)
      .contains('[role="status"]', 'Save the inbox first to verify this alias.')
      .should('be.visible')
    cy.get('[data-alias-row]')
      .eq(2)
      .contains('button', /^Verify$/)
      .should('be.disabled')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('keeps alias removal and other controls from submitting the form', () => {
    cy.visit(editPath)
    cy.get('[data-alias-row]').first().find('button[aria-label="Remove"]').click()
    field('aliases[0].email').should('have.value', 'sales@example.com')
    cy.get('form button')
      .not('[type="submit"]')
      .each(($button) => {
        expect($button).to.have.attr('type', 'button')
      })
    cy.get('@updateInbox.all').should('have.length', 0)
    field('name').type('{enter}')
    cy.wait('@updateInbox').its('request.body.aliases').should('have.length', 1)
  })

  it('preserves OAuth connection values in a closed section', () => {
    inbox.config.auth_type = 'oauth2'
    inbox.config.oauth = {
      provider: 'google',
      client_id: 'test-client',
      client_secret: maskedSecret,
      access_token: maskedSecret,
      refresh_token: maskedSecret
    }
    cy.visit(editPath)
    field('name').should('have.value', 'Support')
    submit()
    cy.wait('@updateInbox').then(({ request }) => {
      expect(request.body.config.auth_type).to.eq('oauth2')
      expect(request.body.config.oauth).to.deep.eq({
        provider: 'google',
        client_id: 'test-client',
        client_secret: '',
        access_token: '',
        refresh_token: ''
      })
      expect(request.body.config.imap[0].host).to.eq('imap.example.com')
    })
    cy.setEmailInboxSection('imap', true)
    cy.setEmailInboxSection('smtp', true)
    cy.contains('Connected via OAuth - Google').scrollIntoView().should('be.visible')
    field('imap.username').should('not.be.visible')
    field('smtp.password').should('not.be.visible')
    cy.contains('button', 'Reconnect').click()
    cy.get('[role="dialog"]').should('be.visible')
    cy.get('#oauth-client_id').should('have.value', 'test-client')
    cy.get('@updateInbox.all').should('have.length', 1)
  })

  it('reveals OAuth advanced validation errors', () => {
    inbox.config.auth_type = 'oauth2'
    inbox.config.oauth = { provider: 'microsoft', client_id: 'test-client' }
    inbox.config.smtp[0].idle_timeout = 'invalid'
    cy.visit(editPath)
    submit()
    field('smtp.idle_timeout').should('be.visible').and('have.focus')
    cy.setEmailInboxSection('options', true)
    cy.contains('[data-section="options"] p', 'Enable plus addressing')
      .parent()
      .find('[role="switch"]')
      .should('be.disabled')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('fits a narrow viewport and keeps the save action visible', () => {
    cy.viewport(390, 844)
    cy.visit(editPath)
    field('aliases[0].email').should('be.visible')
    cy.get('[data-section="aliases"]').scrollIntoView()
    cy.get('form').should(($form) => {
      expect($form[0].scrollWidth).to.be.at.most($form[0].clientWidth)
    })
    cy.screenshot('email-inbox-mobile', { capture: 'viewport' })
    cy.setEmailInboxSection('options', true)
    cy.setEmailInboxSection('imap', true)
    cy.setEmailInboxSection('smtp', true)
    cy.get('form').should(($form) => {
      expect($form[0].scrollWidth).to.be.at.most($form[0].clientWidth)
    })
    cy.get('form button[type="submit"]').should('be.visible')
    cy.screenshot('email-inbox-mobile-connection', { capture: 'viewport' })
  })

  it('keeps aliases visible and supports keyboard access to independent connection sections', () => {
    cy.viewport(1600, 1000)
    cy.visit(editPath, {
      onBeforeLoad(win) {
        win.localStorage.setItem('vueuse-color-scheme', 'light')
      }
    })
    field('name').should('be.visible')
    cy.get('form [role="tab"]').should('not.exist')
    field('aliases[0].email').should('be.visible')
    field('imap.host').should('not.be.visible')
    field('reply_to').should('not.be.visible')
    cy.screenshot('email-inbox-desktop', { capture: 'viewport' })
    cy.get('[data-section="imap"] > h3 > button').focus()
    cy.focused().type(' ')
    cy.get('[data-section="imap"] > h3 > button').should('have.attr', 'aria-expanded', 'true')
    field('smtp.host').should('not.be.visible')
    cy.get('[data-connection="imap"]').scrollIntoView()
    field('imap.host').should('be.visible')
    cy.screenshot('email-inbox-connection', { capture: 'viewport' })
    cy.setEmailInboxSection('imap', false)
    cy.setEmailInboxSection('smtp', false)
    cy.setEmailInboxSection('options', true)
    cy.get('[data-section="options"]').scrollIntoView()
    cy.screenshot('email-inbox-options', { capture: 'viewport' })
    cy.get('@updateInbox.all').should('have.length', 0)
  })
})
