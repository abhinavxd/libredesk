// One global record, so before() captures the live values and after() writes them back.

describe('Notification settings form', () => {
  const stamp = Date.now()
  const path = '/admin/notification'
  const prefix = 'notification.email.'
  const maskedPassword = '••••••••••'

  const seedHost = 'smtp.cypress.test'
  const seedUsername = 'cypress-notify'
  const seedEmail = 'notify@cypress.test'
  const changedHost = `smtp-${stamp}.cypress.test`

  const settingKeys = [
    'username',
    'host',
    'port',
    'password',
    'max_conns',
    'idle_timeout',
    'wait_timeout',
    'auth_protocol',
    'email_address',
    'max_msg_retries',
    'tls_type',
    'tls_skip_verify',
    'hello_hostname',
    'enabled'
  ]

  let original

  const put = (values) =>
    cy.api(
      'PUT',
      '/api/v1/settings/notifications/email',
      Object.fromEntries(Object.entries(values).map(([key, value]) => [prefix + key, value]))
    )

  const getSettings = () =>
    cy.api('GET', '/api/v1/settings/notifications/email').then(({ body }) =>
      Object.fromEntries(
        Object.entries(body.data).map(([key, value]) => [key.replace(prefix, ''), value])
      )
    )

  before(() => {
    cy.login()
    getSettings().then((current) => {
      original = Object.fromEntries(settingKeys.map((key) => [key, current[key]]))
      put({ ...original, host: seedHost, username: seedUsername, email_address: seedEmail, password: `seed-${stamp}` })
    })
  })

  after(() => {
    // Without this guard a failed before() would PUT a record built from nothing,
    // wiping the live settings instead of restoring them.
    if (!original) return
    cy.login()
    // An empty password means "keep current", so the seeded password stays.
    put({ ...original, password: '' })
    getSettings().its('host').should('eq', original.host)
  })

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('loads the stored values into the form', () => {
    cy.visit(path)

    cy.get('input[name="host"]').should('have.value', seedHost)
    cy.get('input[name="username"]').should('have.value', seedUsername)
    cy.get('input[name="email_address"]').should('have.value', seedEmail)
    cy.get('input[name="port"]').should('have.value', String(original.port))
  })

  it('masks the stored password instead of showing it', () => {
    cy.visit(path)

    cy.get('input[name="password"]')
      .should('have.attr', 'type', 'password')
      .and('have.value', maskedPassword)

    getSettings().its('password').should('eq', maskedPassword)
  })

  it('saves a changed host and reloads it', () => {
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveNotification')

    cy.visit(path)
    cy.get('input[name="host"]').should('have.value', seedHost).clear().type(changedHost)
    cy.get('button[type="submit"]').click()

    cy.wait('@saveNotification').its('response.statusCode').should('eq', 200)

    cy.visit(path)
    cy.get('input[name="host"]').should('have.value', changedHost)
    getSettings().its('host').should('eq', changedHost)
  })

  it('keeps the stored password when the masked value is submitted back', () => {
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveNotification')

    cy.visit(path)
    cy.get('input[name="password"]').should('have.value', maskedPassword)
    cy.get('button[type="submit"]').click()

    cy.wait('@saveNotification').then(({ request, response }) => {
      expect(response.statusCode).to.eq(200)
      // The form blanks a masked password so the backend keeps the stored one.
      expect(request.body[`${prefix}password`]).to.eq('')
    })
    getSettings().its('password').should('eq', maskedPassword)
  })

  it('rejects an empty host', () => {
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveNotification')

    cy.visit(path)
    cy.get('input[name="host"]').should('not.have.value', '').clear()
    cy.get('button[type="submit"]').click()

    cy.contains('p.text-destructive', /required/i).should('exist')
    cy.get('@saveNotification.all').should('have.length', 0)
  })

  it('rejects an out of range port', () => {
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveNotification')

    cy.visit(path)
    cy.get('input[name="port"]').should('not.have.value', '').clear().type('70000')
    cy.get('button[type="submit"]').click()

    cy.contains('p.text-destructive', '65535').should('exist')
    cy.get('@saveNotification.all').should('have.length', 0)
  })

  it('rejects a malformed idle timeout', () => {
    cy.intercept('PUT', '**/api/v1/settings/notifications/email').as('saveNotification')

    cy.visit(path)
    cy.get('input[name="idle_timeout"]').should('not.have.value', '').clear().type('forever')
    cy.get('button[type="submit"]').click()

    cy.contains('p.text-destructive', 'Invalid duration format').should('exist')
    cy.get('@saveNotification.all').should('have.length', 0)
  })
})
