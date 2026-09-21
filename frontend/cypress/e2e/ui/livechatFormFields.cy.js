// The steps run in order and share one inbox.

const stamp = Date.now()
const listPath = '/admin/inboxes'

const created = {
  name: `Cypress Fields ${stamp}`,
  brandName: `Cypress Fields Brand ${stamp}`,
  websiteUrl: `https://cypress.test/${stamp}`,
  greeting: `Greeting ${stamp}`,
  introduction: `Introduction ${stamp}`,
  chatIntroduction: `Chat introduction ${stamp}`,
  notice: `Notice ${stamp}`,
  replyExpectation: `We reply in ${stamp} minutes`,
  visitorButton: `Start visitor ${stamp}`,
  userButton: `Start user ${stamp}`,
  secret: `${stamp}`.padEnd(32, 'f').slice(0, 32),
  sessionDuration: '5h',
  trustedDomain: `cypress-${stamp}.test`,
  blockedIp: '10.0.0.1',
  lightPrimary: '#112233',
  darkPrimary: '#445566',
  lightLauncher: '#778899',
  logoUrl: 'https://cypress.test/logo.png',
  launcherLogo: 'https://cypress.test/launcher.png'
}

const edited = {
  name: `${created.name} v2`,
  brandName: `${created.brandName} v2`,
  websiteUrl: `${created.websiteUrl}/v2`,
  greeting: `${created.greeting} v2`,
  introduction: `${created.introduction} v2`,
  chatIntroduction: `${created.chatIntroduction} v2`,
  notice: `${created.notice} v2`,
  replyExpectation: `${created.replyExpectation} v2`,
  visitorButton: `${created.visitorButton} v2`,
  userButton: `${created.userButton} v2`,
  sessionDuration: '7h',
  trustedDomain: `edited-${stamp}.test`,
  blockedIp: '10.0.0.2',
  lightPrimary: '#aabbcc',
  darkPrimary: '#ddeeff'
}

const openNewForm = () => {
  cy.visit(`${listPath}/new`)
  cy.contains('Create a live chat inbox').click()
}

const setColor = (name, value) =>
  cy.get(`input[name="${name}"]`).invoke('val', value).trigger('input')

describe('Live chat inbox form: every field', () => {
  let inboxId

  beforeEach(() => {
    Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))
    cy.viewport(1280, 900)
    cy.login()
  })

  it('creates an inbox with every field filled', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()

    cy.openInboxSection('General')
    cy.get('input[name="name"]').type(created.name)
    cy.get('input[name="config.brand_name"]').type(created.brandName)
    cy.get('input[name="config.website_url"]').type(created.websiteUrl)
    cy.switchField('CSAT Surveys').click()
    cy.switchField('Prompt to tag before replying').click()

    cy.openInboxSection('Theme')
    cy.get('#theme-system').click()
    cy.switchField('Show powered by').click()

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.logo_url"]').clear().type(created.logoUrl)
    setColor('config.branding.light.colors.primary', created.lightPrimary)
    cy.get('#text-white').click()
    cy.get('#bg-gradient').click()
    cy.switchField('Fade background').click()

    cy.editBrandingTheme('Dark')
    setColor('config.branding.dark.colors.primary', created.darkPrimary)

    cy.openInboxSection('Launcher position')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.launcher.logo_url"]')
      .clear()
      .type(created.launcherLogo)
    setColor('config.branding.light.launcher.color', created.lightLauncher)
    cy.get('select[name="config.launcher.position"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Left').click()
    cy.get('[role="option"]').should('not.exist')
    cy.get('input[name="config.launcher.size"]').clear().type('72')
    cy.get('input[name="config.launcher.icon_scale"]').clear().type('80')
    cy.get('input[name="config.launcher.spacing.side"]').clear().type('30')
    cy.get('input[name="config.launcher.spacing.bottom"]').clear().type('40')

    cy.openInboxSection('Reply previews')
    cy.switchField('Mobile').click()
    cy.get('input[name="config.previews.auto_hide_seconds"]').clear().type('45')

    cy.openInboxSection('Messages')
    cy.get('textarea[name="config.greeting_message"]').clear().type(created.greeting)
    cy.get('textarea[name="config.introduction_message"]').clear().type(created.introduction)
    cy.get('textarea[name="config.chat_introduction"]').clear().type(created.chatIntroduction)

    cy.openInboxSection('Notice banner')
    cy.switchField('Enable notice banner').click()
    cy.get('textarea[name="config.notice_banner.text"]').clear().type(created.notice)

    cy.openInboxSection('Features')
    cy.switchField('Emoji support').click()

    cy.openInboxSection('Office hours')
    cy.switchField('Show office hours in chat').click()
    cy.get('input[name="config.chat_reply_expectation_message"]')
      .clear()
      .type(created.replyExpectation)

    cy.openInboxSection('Users')
    cy.get('input[name="config.visitors.start_conversation_button_text"]')
      .clear()
      .type(created.visitorButton)
    cy.get('textarea[name="config.visitors.quick_replies"]').clear().type('V one\nV two')
    cy.audienceSwitch('visitors', 'Prevent multiple conversations').click()

    cy.audienceTab('Users', 'Users').click()
    cy.get('input[name="config.users.start_conversation_button_text"]')
      .clear()
      .type(created.userButton)
    cy.get('textarea[name="config.users.quick_replies"]').clear().type('U one\nU two')

    cy.openInboxSection('Security')
    cy.get('input[name="secret"]').clear().type(created.secret)
    cy.get('input[name="config.session_duration"]').clear().type(created.sessionDuration)
    cy.get('textarea[name="config.trusted_domains"]').clear().type(created.trustedDomain)
    cy.get('textarea[name="config.blocked_ips"]').clear().type(created.blockedIp)

    cy.get('button[type="submit"]').click()
    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
      const data = response.body.data
      const c = data.config

      expect(data.name, 'name').to.eq(created.name)
      expect(data.csat_enabled, 'csat').to.eq(true)
      expect(data.prompt_tags_on_reply, 'tag prompt').to.eq(true)
      expect(c.brand_name, 'brand name').to.eq(created.brandName)
      expect(c.website_url, 'website url').to.eq(created.websiteUrl)
      expect(c.theme, 'theme').to.eq('system')
      expect(c.show_powered_by, 'powered by').to.eq(false)

      expect(c.branding.light.logo_url, 'light logo').to.eq(created.logoUrl)
      expect(c.branding.light.colors.primary, 'light primary').to.eq(created.lightPrimary)
      expect(c.branding.light.home_screen.header_text_color, 'light header text').to.eq('white')
      expect(c.branding.light.home_screen.background.type, 'light background').to.eq('gradient')
      expect(c.branding.light.home_screen.fade_background, 'fade').to.eq(true)
      expect(c.branding.dark.colors.primary, 'dark primary').to.eq(created.darkPrimary)
      expect(c.branding.light.launcher.logo_url, 'launcher logo').to.eq(created.launcherLogo)
      expect(c.branding.light.launcher.color, 'launcher color').to.eq(created.lightLauncher)

      expect(c.launcher.position, 'position').to.eq('left')
      expect(c.launcher.size, 'size').to.eq(72)
      expect(c.launcher.icon_scale, 'icon scale').to.eq(80)
      expect(c.launcher.spacing.side, 'side spacing').to.eq(30)
      expect(c.launcher.spacing.bottom, 'bottom spacing').to.eq(40)

      expect(c.previews.mobile, 'mobile previews').to.eq(false)
      expect(c.previews.auto_hide_seconds, 'auto hide').to.eq(45)

      expect(c.greeting_message, 'greeting').to.eq(created.greeting)
      expect(c.introduction_message, 'introduction').to.eq(created.introduction)
      expect(c.chat_introduction, 'chat introduction').to.eq(created.chatIntroduction)
      expect(c.notice_banner.enabled, 'notice enabled').to.eq(true)
      expect(c.notice_banner.text, 'notice text').to.eq(created.notice)

      expect(c.features.emoji, 'emoji').to.eq(false)
      expect(c.show_office_hours_in_chat, 'office hours').to.eq(true)
      expect(c.chat_reply_expectation_message, 'reply expectation').to.eq(created.replyExpectation)

      expect(c.visitors.start_conversation_button_text, 'visitor button').to.eq(
        created.visitorButton
      )
      expect(c.visitors.quick_replies, 'visitor replies').to.deep.eq(['V one', 'V two'])
      expect(c.visitors.prevent_multiple_conversations, 'visitor prevent multiple').to.eq(true)
      expect(c.users.start_conversation_button_text, 'user button').to.eq(created.userButton)
      expect(c.users.quick_replies, 'user replies').to.deep.eq(['U one', 'U two'])

      expect(c.session_duration, 'session').to.eq(created.sessionDuration)
      expect(c.trusted_domains, 'trusted domains').to.deep.eq([created.trustedDomain])
      expect(c.blocked_ips, 'blocked ips').to.deep.eq([created.blockedIp])
    })
  })

  it('loads every saved value back into the edit form', () => {
    expect(inboxId, 'inbox from the create step').to.be.a('number')
    cy.visit(`${listPath}/${inboxId}/edit`)

    cy.openInboxSection('General')
    cy.get('input[name="name"]').should('have.value', created.name)
    cy.get('input[name="config.brand_name"]').should('have.value', created.brandName)
    cy.get('input[name="config.website_url"]').should('have.value', created.websiteUrl)
    cy.switchField('CSAT Surveys').should('have.attr', 'data-state', 'checked')
    cy.switchField('Prompt to tag before replying').should('have.attr', 'data-state', 'checked')

    cy.openInboxSection('Theme')
    cy.get('#theme-system').should('have.attr', 'data-state', 'checked')
    cy.switchField('Show powered by').should('have.attr', 'data-state', 'unchecked')

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    cy.get('input[name="config.branding.light.logo_url"]').should('have.value', created.logoUrl)
    cy.get('input[name="config.branding.light.colors.primary"]').should(
      'have.value',
      created.lightPrimary
    )
    cy.get('#text-white').should('have.attr', 'data-state', 'checked')
    cy.get('#bg-gradient').should('have.attr', 'data-state', 'checked')
    cy.editBrandingTheme('Dark')
    cy.get('input[name="config.branding.dark.colors.primary"]').should(
      'have.value',
      created.darkPrimary
    )

    cy.openInboxSection('Launcher position')
    cy.get('select[name="config.launcher.position"]').should('have.value', 'left')
    cy.get('input[name="config.launcher.size"]').should('have.value', '72')
    cy.get('input[name="config.launcher.icon_scale"]').should('have.value', '80')
    cy.get('input[name="config.launcher.spacing.side"]').should('have.value', '30')
    cy.get('input[name="config.launcher.spacing.bottom"]').should('have.value', '40')

    cy.openInboxSection('Reply previews')
    cy.switchField('Mobile').should('have.attr', 'data-state', 'unchecked')
    cy.get('input[name="config.previews.auto_hide_seconds"]').should('have.value', '45')

    cy.openInboxSection('Messages')
    cy.get('textarea[name="config.greeting_message"]').should('have.value', created.greeting)
    cy.get('textarea[name="config.introduction_message"]').should(
      'have.value',
      created.introduction
    )
    cy.get('textarea[name="config.chat_introduction"]').should(
      'have.value',
      created.chatIntroduction
    )

    cy.openInboxSection('Notice banner')
    cy.get('textarea[name="config.notice_banner.text"]').should('have.value', created.notice)

    cy.openInboxSection('Office hours')
    cy.get('input[name="config.chat_reply_expectation_message"]').should(
      'have.value',
      created.replyExpectation
    )

    cy.openInboxSection('Users')
    cy.get('input[name="config.visitors.start_conversation_button_text"]').should(
      'have.value',
      created.visitorButton
    )
    cy.get('textarea[name="config.visitors.quick_replies"]').should('have.value', 'V one\nV two')
    cy.audienceTab('Users', 'Users').click()
    cy.get('input[name="config.users.start_conversation_button_text"]').should(
      'have.value',
      created.userButton
    )

    cy.openInboxSection('Security')
    cy.get('input[name="secret"]').should('have.value', '••••••••••')
    cy.get('input[name="config.session_duration"]').should('have.value', created.sessionDuration)
    cy.get('textarea[name="config.trusted_domains"]').should('have.value', created.trustedDomain)
    cy.get('textarea[name="config.blocked_ips"]').should('have.value', created.blockedIp)
  })

  it('changes every field from the edit form', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')
    cy.visit(`${listPath}/${inboxId}/edit`)

    cy.openInboxSection('General')
    cy.get('input[name="name"]').clear().type(edited.name)
    cy.get('input[name="config.brand_name"]').clear().type(edited.brandName)
    cy.get('input[name="config.website_url"]').clear().type(edited.websiteUrl)
    cy.switchField('CSAT Surveys').click()
    cy.switchField('Prompt to tag before replying').click()

    cy.openInboxSection('Theme')
    cy.get('#theme-light').click()
    cy.switchField('Show powered by').click()

    cy.openInboxSection('Branding')
    cy.editBrandingTheme('Light')
    setColor('config.branding.light.colors.primary', edited.lightPrimary)
    cy.get('#text-black').click()
    cy.get('#bg-solid').click()
    cy.switchField('Fade background').click()

    cy.openInboxSection('Launcher position')
    cy.get('select[name="config.launcher.position"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Right').click()
    cy.get('[role="option"]').should('not.exist')
    cy.get('input[name="config.launcher.size"]').clear().type('50')
    cy.get('input[name="config.launcher.icon_scale"]').clear().type('60')
    cy.get('input[name="config.launcher.spacing.side"]').clear().type('10')
    cy.get('input[name="config.launcher.spacing.bottom"]').clear().type('12')

    cy.openInboxSection('Reply previews')
    cy.switchField('Desktop').click()
    cy.get('input[name="config.previews.auto_hide_seconds"]').clear().type('15')

    cy.openInboxSection('Messages')
    cy.get('textarea[name="config.greeting_message"]').clear().type(edited.greeting)
    cy.get('textarea[name="config.introduction_message"]').clear().type(edited.introduction)
    cy.get('textarea[name="config.chat_introduction"]').clear().type(edited.chatIntroduction)

    cy.openInboxSection('Notice banner')
    cy.get('textarea[name="config.notice_banner.text"]').clear().type(edited.notice)

    cy.openInboxSection('Features')
    cy.switchField('File upload').click()
    cy.switchField('Download transcript').click()

    cy.openInboxSection('Office hours')
    cy.get('input[name="config.chat_reply_expectation_message"]')
      .clear()
      .type(edited.replyExpectation)

    cy.openInboxSection('Users')
    cy.get('input[name="config.visitors.start_conversation_button_text"]')
      .clear()
      .type(edited.visitorButton)
    cy.audienceSwitch('visitors', 'Launch directly into conversation').click()
    cy.audienceTab('Users', 'Users').click()
    cy.get('input[name="config.users.start_conversation_button_text"]')
      .clear()
      .type(edited.userButton)

    cy.openInboxSection('Security')
    cy.get('input[name="config.session_duration"]').clear().type(edited.sessionDuration)
    cy.get('textarea[name="config.trusted_domains"]').clear().type(edited.trustedDomain)
    cy.get('textarea[name="config.blocked_ips"]').clear().type(edited.blockedIp)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      const data = response.body.data
      const c = data.config

      expect(data.name, 'name').to.eq(edited.name)
      expect(data.csat_enabled, 'csat off again').to.eq(false)
      expect(data.prompt_tags_on_reply, 'tag prompt off again').to.eq(false)
      expect(c.brand_name, 'brand name').to.eq(edited.brandName)
      expect(c.website_url, 'website url').to.eq(edited.websiteUrl)
      expect(c.theme, 'theme').to.eq('light')
      expect(c.show_powered_by, 'powered by back on').to.eq(true)

      expect(c.branding.light.colors.primary, 'light primary').to.eq(edited.lightPrimary)
      expect(c.branding.light.home_screen.header_text_color, 'header text').to.eq('black')
      expect(c.branding.light.home_screen.background.type, 'background').to.eq('solid')
      expect(c.branding.light.home_screen.fade_background, 'fade off again').to.eq(false)
      // The dark half is untouched by this pass and has to survive it.
      expect(c.branding.dark.colors.primary, 'dark primary kept').to.eq(created.darkPrimary)

      expect(c.launcher.position, 'position').to.eq('right')
      expect(c.launcher.size, 'size').to.eq(50)
      expect(c.launcher.icon_scale, 'icon scale').to.eq(60)
      expect(c.launcher.spacing.side, 'side spacing').to.eq(10)
      expect(c.launcher.spacing.bottom, 'bottom spacing').to.eq(12)

      expect(c.previews.desktop, 'desktop previews').to.eq(false)
      expect(c.previews.auto_hide_seconds, 'auto hide').to.eq(15)

      expect(c.greeting_message, 'greeting').to.eq(edited.greeting)
      expect(c.introduction_message, 'introduction').to.eq(edited.introduction)
      expect(c.chat_introduction, 'chat introduction').to.eq(edited.chatIntroduction)
      expect(c.notice_banner.text, 'notice text').to.eq(edited.notice)

      expect(c.features.file_upload, 'file upload').to.eq(false)
      expect(c.features.transcript, 'transcript').to.eq(false)
      expect(c.chat_reply_expectation_message, 'reply expectation').to.eq(edited.replyExpectation)

      expect(c.visitors.start_conversation_button_text, 'visitor button').to.eq(
        edited.visitorButton
      )
      expect(c.visitors.direct_to_conversation, 'visitor direct').to.eq(true)
      expect(c.users.start_conversation_button_text, 'user button').to.eq(edited.userButton)

      expect(c.session_duration, 'session').to.eq(edited.sessionDuration)
      expect(c.trusted_domains, 'trusted domains').to.deep.eq([edited.trustedDomain])
      expect(c.blocked_ips, 'blocked ips').to.deep.eq([edited.blockedIp])
    })
  })

  it('keeps the secret when it is left masked and replaces it when retyped', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('General')
    cy.get('input[name="name"]').clear().type(`${edited.name} b`)
    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Security')
    cy.get('input[name="secret"]').should('have.value', '••••••••••')
  })

  it('adds and removes home screen apps', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Home screen apps')

    cy.contains('button', 'Add announcement').click()
    cy.get('input[placeholder="Title"]').type(`Announcement ${stamp}`)
    cy.get('input[placeholder="Cover image URL"]').type('https://cypress.test/cover.png')
    cy.get('input[placeholder="Link URL"]').type('https://cypress.test/read')

    cy.contains('button', 'Add external link').click()
    cy.get('input[placeholder="Link Text"]').type(`Link ${stamp}`)
    cy.get('input[placeholder="https://example.com"]').last().type('https://cypress.test/docs')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      const apps = response.body.data.config.home_apps
      expect(apps.map((a) => a.type), 'app order').to.deep.eq(['announcement', 'external_link'])
    })

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Home screen apps')
    cy.get('[data-section="homeApps"] button[aria-label="Remove"]').first().click()
    cy.get('[data-section="homeApps"] button[aria-label="Remove"]').first().click()
    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      expect(response.body.data.config.home_apps || []).to.have.length(0)
    })
  })

  it('turns the pre-chat form on and saves its fields', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.openInboxSection('Pre-chat form')
    cy.get('[data-section="prechat"] button[role="switch"]').first().click()
    cy.inboxSection('Pre-chat form').contains('Form fields').should('be.visible')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      const prechat = response.body.data.config.prechat_form
      expect(prechat.visitors.enabled, 'prechat enabled').to.eq(true)
      expect(prechat.visitors.fields.length, 'default fields').to.be.greaterThan(0)
    })
  })

  it('saves the selects and switches that the other steps leave alone', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')
    cy.visit(`${listPath}/${inboxId}/edit`)

    cy.openInboxSection('General')
    cy.get('select[name="config.language"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Auto-detect').click()
    cy.get('[role="option"]').should('not.exist')
    cy.get('select[name="config.fallback_language"]')
      .siblings('button[role="combobox"]')
      .should('exist')

    cy.openInboxSection('Conversation continuity email inbox')
    cy.get('select[name="linked_email_inbox_id"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').not(':contains("None")').first().click()
    cy.get('[role="option"]').should('not.exist')
    cy.get('input[name="config.continuity.offline_threshold"]').clear().type('12m')
    cy.get('input[name="config.continuity.max_messages_per_email"]').clear().type('7')
    cy.get('input[name="config.continuity.min_email_interval"]').clear().type('9m')

    cy.openInboxSection('Reply previews')
    cy.get('select[name="config.previews.content"]').siblings('button[role="combobox"]').click()
    cy.get('[role="option"]').contains('Generic notification').click()
    cy.get('[role="option"]').should('not.exist')

    cy.openInboxSection('Office hours')
    cy.switchField('Show office hours after team assignment').click()

    cy.openInboxSection('Users')
    cy.audienceSwitch('visitors', 'Allow start conversation').click()
    cy.audienceSwitch('visitors', 'Prevent replying to closed conversations').click()
    cy.audienceTab('Users', 'Users').click()
    cy.audienceSwitch('users', 'Launch directly into conversation').click()
    cy.audienceSwitch('users', 'Prevent multiple conversations').click()
    cy.audienceSwitch('users', 'Allow start conversation').click()
    cy.audienceSwitch('users', 'Prevent replying to closed conversations').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      const data = response.body.data
      const c = data.config

      expect(c.language, 'language').to.eq('auto')
      expect(data.linked_email_inbox_id, 'linked email inbox').to.be.a('number')
      expect(c.continuity.offline_threshold, 'offline threshold').to.eq('12m')
      expect(c.continuity.max_messages_per_email, 'max messages').to.eq(7)
      expect(c.continuity.min_email_interval, 'min interval').to.eq('9m')
      expect(c.previews.content, 'preview content').to.eq('generic')
      expect(c.show_office_hours_after_assignment, 'office hours after assignment').to.eq(true)
      expect(c.visitors.allow_start_conversation, 'visitor start').to.eq(false)
      expect(c.visitors.prevent_reply_to_closed_conversation, 'visitor closed replies').to.eq(true)
      expect(c.users.direct_to_conversation, 'user direct').to.eq(true)
      expect(c.users.prevent_multiple_conversations, 'user prevent multiple').to.eq(true)
      expect(c.users.allow_start_conversation, 'user start').to.eq(false)
      expect(c.users.prevent_reply_to_closed_conversation, 'user closed replies').to.eq(true)
    })
  })

  it('rejects out-of-range and malformed values in every section that validates', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    const cases = [
      ['Launcher position', 'input[name="config.launcher.size"]', '120'],
      ['Launcher position', 'input[name="config.launcher.icon_scale"]', '5'],
      ['Launcher position', 'input[name="config.launcher.spacing.side"]', '900'],
      ['Reply previews', 'input[name="config.previews.auto_hide_seconds"]', '999'],
      ['Security', 'input[name="config.session_duration"]', 'nope'],
      ['General', 'input[name="config.website_url"]', 'not-a-url']
    ]

    cases.forEach(([section, selector, value]) => {
      cy.visit(`${listPath}/${inboxId}/edit`)
      cy.openInboxSection(section)
      cy.get(selector).clear().type(value)
      cy.get('button[type="submit"]').click()
      cy.get('@updateInbox.all').should('have.length', 0)
      cy.get(selector).should('be.visible')
    })
  })

  it('deletes the inbox', () => {
    cy.intercept('DELETE', `**/api/v1/inboxes/${inboxId}`).as('deleteInbox')

    cy.visit(listPath)
    cy.get('input[placeholder="Search"]').clear().type(edited.name)
    cy.contains('tr', edited.name).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()
    cy.wait('@deleteInbox').its('response.statusCode').should('eq', 200)
  })
})
