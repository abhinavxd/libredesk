// The steps run in order and share the inbox created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Branches ${stamp}`
const brandName = `Cypress Branches Brand ${stamp}`
const emailInboxName = `Cypress Branch Email ${stamp}`
const attributeName = `Cypress Branch Attr ${stamp}`
const attributeKey = `cypress_branch_attr_${stamp}`
const campaignName = `Cypress Branch Campaign ${stamp}`
const listPath = '/admin/inboxes'

const selectOption = (trigger, option) => {
  cy.get(trigger).click()
  cy.get('[role="option"]').contains(option).click()
  // The listbox keeps a focus guard over the form until it has finished closing.
  cy.get('[role="option"]').should('not.exist')
}

const editInbox = (inboxId) => cy.visit(`${listPath}/${inboxId}/edit`)

const pickFromLabelledSelect = (label, optionIndex) => {
  cy.contains('label', label).parent().find('button').click()
  cy.get('[role="option"]').eq(optionIndex).click()
  cy.get('[role="option"]').should('not.exist')
}

const linkEmailInbox = (name) => {
  cy.inboxSection('Conversation continuity email inbox').find('button[role="combobox"]').click()
  cy.get('[role="option"]').contains(name).click()
  cy.get('[role="option"]').should('not.exist')
}

// The accordion's open animation makes ':visible' flaky, so these scope by section instead.
const audienceEnabledSwitch = () =>
  cy
    .inboxSection('Pre-chat form')
    .contains('p', 'Show form to this audience')
    .parent()
    .find('button[role="switch"]')

const enablePreChatForm = () =>
  cy.switchField('Enable pre-chat form').then(($switch) => {
    if ($switch.attr('data-state') !== 'checked') cy.wrap($switch).click()
  })

const savedConfig = (alias) => cy.wait(alias).then(({ response }) => response.body.data.config)

describe('Live chat inbox form: branches the other specs skip', () => {
  let inboxId
  let emailInboxId
  let attributeId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/custom-attributes', {
      name: attributeName,
      key: attributeKey,
      description: 'Cypress branch coverage',
      applies_to: 'contact',
      data_type: 'text',
      values: [],
      regex: '',
      regex_hint: ''
    }).then(({ body }) => {
      attributeId = body.data.id
    })
    cy.api('POST', '/api/v1/inboxes', {
      name: emailInboxName,
      channel: 'email',
      enabled: true,
      from: `Branch ${stamp} <branch.${stamp}@example.com>`,
      config: {
        auth_type: 'password',
        from: `Branch ${stamp} <branch.${stamp}@example.com>`,
        smtp: [
          {
            host: '127.0.0.1',
            port: Number(Cypress.env('SMTP_PORT') || 1025),
            username: '',
            password: '',
            auth_protocol: 'none',
            tls_type: 'none',
            max_conns: 2,
            max_msg_retries: 1,
            idle_timeout: '5s',
            pool_wait_timeout: '5s'
          }
        ],
        imap: []
      }
    }).then(({ body }) => {
      emailInboxId = body.data.id
    })
  })

  beforeEach(() => {
    // radix-vue's select focuses a node that is already gone in headless runs.
    Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))
    cy.viewport(1280, 900)
    cy.login()
  })

  after(() => {
    if (inboxId) cy.api('DELETE', `/api/v1/inboxes/${inboxId}`, null, { failOnStatusCode: false })
    if (emailInboxId) {
      cy.api('DELETE', `/api/v1/inboxes/${emailInboxId}`, null, { failOnStatusCode: false })
    }
    if (attributeId) {
      cy.api('DELETE', `/api/v1/custom-attributes/${attributeId}`, null, {
        failOnStatusCode: false
      })
    }
  })

  it('creates the inbox the other steps edit', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    cy.visit(`${listPath}/new`)
    cy.contains('Create a live chat inbox').click()
    cy.openInboxSection('General')
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.brand_name"]').type(brandName)
    cy.get('button[type="submit"]').click()

    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
    })
  })

  it('rejects an image background with no url, then saves one', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')
    const imageUrl = `https://cypress.test/${stamp}/background.jpg`

    editInbox(inboxId)
    cy.openInboxSection('Branding')
    cy.get('#bg-image').click()
    cy.get('button[type="submit"]').click()
    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Branding').should('have.attr', 'data-state', 'open')

    cy.get('input[name="config.branding.light.home_screen.background.image_url"]').type(imageUrl)
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.branding.light.home_screen.background.type).to.eq('image')
      expect(config.branding.light.home_screen.background.image_url).to.eq(imageUrl)
    })
  })

  it('rejects an enabled notice banner with no text', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Notice banner')
    cy.switchField('Enable notice banner').click()
    cy.get('textarea[name="config.notice_banner.text"]').clear()
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Notice banner').should('have.attr', 'data-state', 'open')
  })

  it('requires continuity values once an email inbox is linked', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Conversation continuity email inbox')
    linkEmailInbox(emailInboxName)
    cy.get('input[name="config.continuity.offline_threshold"]').clear()
    cy.get('button[type="submit"]').click()
    cy.get('@updateInbox.all').should('have.length', 0)

    cy.get('input[name="config.continuity.offline_threshold"]').type('20m')
    cy.get('input[name="config.continuity.min_email_interval"]').clear()
    cy.get('input[name="config.continuity.min_email_interval"]').type('30m')
    cy.get('input[name="config.continuity.max_messages_per_email"]').clear()
    cy.get('input[name="config.continuity.max_messages_per_email"]').type('5')
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.continuity.offline_threshold).to.eq('20m')
      expect(config.continuity.min_email_interval).to.eq('30m')
      expect(config.continuity.max_messages_per_email).to.eq(5)
    })
  })

  it('rejects a continuity message count outside its range', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Conversation continuity email inbox')
    cy.get('input[name="config.continuity.max_messages_per_email"]').clear()
    cy.get('input[name="config.continuity.max_messages_per_email"]').type('200')
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Conversation continuity email inbox').should(
      'have.attr',
      'data-state',
      'open'
    )
  })

  it('saves the launch-directly toggle separately for each audience', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    enablePreChatForm()

    cy.openInboxSection('Users')
    cy.switchFieldIn('Users', 'Launch directly into conversation').click()

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.prechat_form.enabled, 'pre-chat on').to.eq(true)
      expect(config.visitors.direct_to_conversation, 'visitors launch direct').to.eq(true)
      expect(config.users.direct_to_conversation, 'users launch direct').to.eq(false)
    })
  })

  it('adds a pre-chat custom attribute field and removes it again', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    enablePreChatForm()
    cy.inboxSection('Pre-chat form').contains('div.font-medium', attributeName).click()
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      const keys = config.prechat_form.visitors.fields.map((field) => field.key)
      expect(keys).to.include(attributeKey)
    })

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    cy.inboxSection('Pre-chat form')
      .contains('div.font-medium', attributeName)
      .closest('div.border')
      .find('button[aria-label="Remove"]')
      .click()
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      const keys = config.prechat_form.visitors.fields.map((field) => field.key)
      expect(keys).to.not.include(attributeKey)
    })
  })

  it('rejects a business-hours-limited proactive message with no schedule picked', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', 'Add proactive message').click()
    cy.get('#campaign-name').type(campaignName)
    cy.get('#campaign-message').type('Need a hand?')
    selectOption('#campaign-business_hours', 'During business hours')
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.get('#campaign-hours').should('be.visible')

    selectOption('#campaign-business_hours', 'Any time')
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.campaigns).to.have.length(1)
      expect(config.campaigns[0].business_hours).to.eq('any')
    })
  })

  it('saves a proactive message condition and removes it again', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', campaignName).click()
    cy.contains('button', 'Add condition').click()
    cy.contains('button[role="combobox"]', 'Select field').click()
    cy.get('[role="option"]').contains(attributeName).click()
    cy.contains('button[role="combobox"]', 'Select operator').click()
    cy.get('[role="option"]').contains('equals').click()
    cy.get('input[placeholder="Set value"]').type('gold')
    cy.inboxSection('Proactive messages').find('[role="radio"][value="OR"]').click()
    cy.get('button[type="submit"]').click()

    savedConfig('@updateInbox').then((config) => {
      const conditions = config.campaigns[0].conditions
      expect(conditions.logical_op).to.eq('OR')
      expect(conditions.rules).to.have.length(1)
      expect(conditions.rules[0].field).to.eq(attributeKey)
      expect(conditions.rules[0].value).to.eq('gold')
    })

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', campaignName).click()
    cy.inboxSection('Proactive messages').contains('button[role="combobox"]', attributeName)
    cy.inboxSection('Proactive messages').find('button[aria-label="Close"]').first().click()
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.campaigns[0].conditions.rules).to.have.length(0)
    })
  })

  it('saves a different pre-chat form for visitors and for users', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    enablePreChatForm()

    cy.audienceTab('Pre-chat form', 'Visitors').click()
    audienceEnabledSwitch().should('have.attr', 'data-state', 'checked')
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .clear()
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .type('Visitor questions')

    cy.audienceTab('Pre-chat form', 'Users').click()
    audienceEnabledSwitch().then(($switch) => {
      if ($switch.attr('data-state') !== 'checked') cy.wrap($switch).click()
    })
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .clear()
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .type('Signed-in questions')

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.prechat_form.visitors.enabled, 'visitors on').to.eq(true)
      expect(config.prechat_form.users.enabled, 'users on').to.eq(true)
      expect(config.prechat_form.visitors.title).to.eq('Visitor questions')
      expect(config.prechat_form.users.title).to.eq('Signed-in questions')
    })

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .should('have.value', 'Visitor questions')
    cy.audienceTab('Pre-chat form', 'Users').click()
    cy.inboxSection('Pre-chat form')
      .find('input[placeholder="Tell us about yourself"]')
      .should('have.value', 'Signed-in questions')
  })

  it('turns the pre-chat form off for visitors while it stays on for users', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    cy.audienceTab('Pre-chat form', 'Visitors').click()
    audienceEnabledSwitch().click()
    cy.get('button[type="submit"]').click()

    savedConfig('@updateInbox').then((config) => {
      expect(config.prechat_form.enabled, 'form still on').to.eq(true)
      expect(config.prechat_form.visitors.enabled, 'visitors off').to.eq(false)
      expect(config.prechat_form.users.enabled, 'users on').to.eq(true)
    })

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    cy.audienceTab('Pre-chat form', 'Visitors').click()
    audienceEnabledSwitch().should('have.attr', 'data-state', 'unchecked')
    cy.audienceTab('Pre-chat form', 'Users').click()
    audienceEnabledSwitch().should('have.attr', 'data-state', 'checked')
  })

  it('keeps the pre-chat fields of one audience out of the other', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Pre-chat form')
    cy.audienceTab('Pre-chat form', 'Users').click()
    cy.inboxSection('Pre-chat form').contains('div.font-medium', attributeName).click()
    cy.get('button[type="submit"]').click()

    savedConfig('@updateInbox').then((config) => {
      const userKeys = config.prechat_form.users.fields.map((field) => field.key)
      const visitorKeys = config.prechat_form.visitors.fields.map((field) => field.key)
      expect(userKeys, 'users carry the attribute').to.include(attributeKey)
      expect(visitorKeys, 'visitors do not').to.not.include(attributeKey)
    })
  })

  it('saves branding for each theme and keeps the halves apart', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Theme')
    cy.get('#theme-system').click()

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('#bg-solid').click()
    cy.get('input[name="config.branding.light.colors.primary"]').invoke(
      'val',
      '#0f766e'
    ).trigger('input')
    cy.get('#text-black').click()

    cy.editBrandingTheme('Dark')
    cy.get('#bg-solid').click()
    cy.get('input[name="config.branding.dark.colors.primary"]').invoke(
      'val',
      '#f59e0b'
    ).trigger('input')
    cy.get('#text-white').click()

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.theme).to.eq('system')
      expect(config.branding.light.colors.primary).to.eq('#0f766e')
      expect(config.branding.dark.colors.primary).to.eq('#f59e0b')
      expect(config.branding.light.home_screen.header_text_color).to.eq('black')
      expect(config.branding.dark.home_screen.header_text_color).to.eq('white')
    })

    editInbox(inboxId)
    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.colors.primary"]').should('have.value', '#0f766e')
    cy.editBrandingTheme('Dark')
    cy.get('input[name="config.branding.dark.colors.primary"]').should('have.value', '#f59e0b')
  })

  it('edits the dark half when the theme is forced to dark', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Theme')
    cy.get('#theme-dark').click()

    cy.openInboxSection('Branding')
    cy.inboxSection('Branding')
      .find('[role="tab"][data-state="active"]')
      .should('contain.text', 'Dark')
    cy.get('input[name="config.branding.dark.logo_url"]').clear()
    cy.get('input[name="config.branding.dark.logo_url"]').type('https://cypress.test/dark.png')

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.theme).to.eq('dark')
      expect(config.branding.dark.logo_url).to.eq('https://cypress.test/dark.png')
      expect(config.branding.light.colors.primary, 'light half untouched').to.eq('#0f766e')
    })

    editInbox(inboxId)
    cy.openInboxSection('Theme')
    cy.get('#theme-light').click()
    cy.openInboxSection('Branding')
    cy.inboxSection('Branding')
      .find('[role="tab"][data-state="active"]')
      .should('contain.text', 'Light')
    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.theme).to.eq('light')
      expect(config.branding.dark.logo_url, 'dark half kept').to.eq('https://cypress.test/dark.png')
    })
  })

  it('saves both gradient colours of the home screen background', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('#bg-gradient').click()
    cy.get('input[name="config.branding.light.home_screen.background.gradient_start"]')
      .invoke('val', '#0f766e')
      .trigger('input')
    cy.get('input[name="config.branding.light.home_screen.background.gradient_end"]')
      .invoke('val', '#99f6e4')
      .trigger('input')
    cy.switchFieldIn('Branding', 'Fade background').click()

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      const background = config.branding.light.home_screen.background
      expect(background.type).to.eq('gradient')
      expect(background.gradient_start).to.eq('#0f766e')
      expect(background.gradient_end).to.eq('#99f6e4')
      expect(config.branding.light.home_screen.fade_background).to.eq(true)
    })

    editInbox(inboxId)
    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.home_screen.background.gradient_start"]').should(
      'have.value',
      '#0f766e'
    )
    cy.get('input[name="config.branding.light.home_screen.background.gradient_end"]').should(
      'have.value',
      '#99f6e4'
    )
  })

  it('saves the proactive message sender, team, audience and cooldown', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.get('#campaign-cooldown').clear()
    cy.get('#campaign-cooldown').type('8h')
    cy.contains('button', campaignName).click()

    // SelectAgentCombobox and SelectTeamCombobox drop the id they are given.
    pickFromLabelledSelect('Sender', 1)
    pickFromLabelledSelect('Route replies to team', 1)
    selectOption('#campaign-audience', 'Signed-in users')

    cy.get('button[type="submit"]').click()
    savedConfig('@updateInbox').then((config) => {
      expect(config.campaign_cooldown).to.eq('8h')
      expect(config.campaigns[0].sender_id, 'sender picked').to.be.greaterThan(0)
      expect(config.campaigns[0].team_id, 'team picked').to.be.greaterThan(0)
      expect(config.campaigns[0].audience).to.eq('users')
    })

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.get('#campaign-cooldown').should('have.value', '8h')
    cy.contains('button', campaignName).click()
    cy.get('#campaign-audience').should('contain.text', 'Signed-in users')
  })

  it('rejects a campaign cooldown that is not a duration', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.get('#campaign-cooldown').clear()
    cy.get('#campaign-cooldown').type('soon')
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Proactive messages').should('have.attr', 'data-state', 'open')
  })

  it('duplicates a proactive message', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', campaignName).click()
    cy.contains('button', 'Duplicate').click()
    cy.get('button[type="submit"]').click()

    savedConfig('@updateInbox').then((config) => {
      expect(config.campaigns).to.have.length(2)
      const ids = config.campaigns.map((campaign) => campaign.id)
      expect(new Set(ids).size, 'each copy keeps its own id').to.eq(2)
    })
  })

  it('rejects a home screen app with a malformed link', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Home screen apps')
    cy.contains('button', 'Add external link').click()
    cy.get('[data-section="homeApps"] input[placeholder="Link Text"]').type('Status')
    cy.get('[data-section="homeApps"] input[placeholder="https://example.com"]').type('not-a-url')
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Home screen apps').should('have.attr', 'data-state', 'open')
  })

  it('rejects a quick reply longer than the limit', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Users')
    cy.get('textarea[name="config.visitors.quick_replies"]').clear()
    cy.get('textarea[name="config.visitors.quick_replies"]').type('x'.repeat(121), { delay: 0 })
    cy.get('button[type="submit"]').click()

    cy.get('@updateInbox.all').should('have.length', 0)
    cy.inboxSection('Users').should('have.attr', 'data-state', 'open')
  })

  it('saves the auto language option', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('General')
    cy.inboxSection('General').contains('label', 'Language').parent().find('button').click()
    cy.get('[role="option"]').contains('Auto').click()
    cy.get('[role="option"]').should('not.exist')
    cy.get('button[type="submit"]').click()

    savedConfig('@updateInbox').then((config) => {
      expect(config.language).to.eq('auto')
    })
  })

  it('keeps the help articles card out until a help center is linked', () => {
    editInbox(inboxId)
    cy.openInboxSection('Home screen apps')
    cy.contains('button', 'Add help articles').should('be.disabled')
    cy.contains('Choose a help center under Help center before adding this card.').should(
      'be.visible'
    )
  })
})
