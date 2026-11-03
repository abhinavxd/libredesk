const home = { en: 'Home', de: 'Startseite', zh: '主页' }
const browserLanguages = navigator.languages.join(',')
const setBrowserLanguages = (languages) => cy.then(() =>
  Cypress.automation('remote:debugger:protocol', {
    command: 'Network.setUserAgentOverride',
    params: { userAgent: navigator.userAgent, acceptLanguage: languages }
  })
)

afterEach(() => setBrowserLanguages(browserLanguages))

describe('Live chat widget language chosen by the host page', () => {
  for (const [languages, expected] of [
    ['de-AT,de-DE', home.en],
    ['ko-KR,de-DE', home.en],
    ['de-DE,en-US', home.de]
  ]) {
    it(`keeps exact browser matching without a website language for ${languages}`, () => {
      cy.createLivechatInbox({ language: 'auto', fallback_language: 'en-US' }).then((inbox) => {
        setBrowserLanguages(languages)
        cy.visitWidgetHost(inbox.uuid)
        cy.widgetBody().then((body) => {
          expect(body[0].ownerDocument.defaultView.navigator.language).to.eq(languages.split(',')[0])
        })
        cy.widgetLauncher().click()
        cy.widgetBody().contains(expected).should('be.visible')
        cy.window().then((win) => win.Libredesk.setLanguage('de-AT'))
        cy.widgetBody().contains(home.de).should('be.visible')
        cy.window().then((win) => win.Libredesk.setLanguage(null))
        cy.widgetBody().contains(expected).should('be.visible')
      })
    })
  }

  it('loads in the language the host page passes', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de-DE' })
      cy.get('iframe[src*="/widget?inbox_id="]')
        .should('have.attr', 'src')
        .and('contain', 'lang=de-DE')
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('matches a bare language tag to a shipped translation', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('switches language when the host page calls setLanguage', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-US' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.window().then((win) => win.Libredesk.setLanguage('de-DE'))
      cy.widgetBody().contains(home.de).should('be.visible')
    })
  })

  it('keeps the latest language when switches overlap', () => {
    cy.createLivechatInbox({ language: 'auto' }).then((inbox) => {
      cy.intercept('GET', '**/api/v1/lang/de-DE', (req) => {
        req.on('response', (res) => res.setDelay(1000))
      })
      cy.visitWidgetHost(inbox.uuid, { language: 'en-US' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.window().then((win) => {
        win.Libredesk.setLanguage('de-DE')
        win.Libredesk.setLanguage('zh-CN')
      })
      // eslint-disable-next-line cypress/no-unnecessary-waiting -- the delayed German response must land and be ignored
      cy.wait(1500)
      cy.widgetBody().contains(home.zh).should('be.visible')
      cy.widgetBody().contains(home.de).should('not.exist')
    })
  })

  it('keeps a language the admin fixed for the inbox', () => {
    cy.createLivechatInbox({ language: 'en-US' }).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'de-DE' })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
    })
  })
})

describe('Widget and help center languages', () => {
  let helpCenter

  before(() => {
    const locales = ['de', 'en-US', 'en-GB', 'zh-Hans', 'zh-Hant']
    cy.login()
    cy.api('POST', '/api/v1/help-centers', {
      name: 'Language regression articles',
      page_title: 'Language regression articles',
      slug: `language-regression-${Date.now()}`,
      default_locale: 'de',
      allowed_locales: locales
    }).then(({ body }) => {
      helpCenter = body.data
      cy.wrap(locales).each((locale) =>
        cy.api('POST', `/api/v1/help-centers/${helpCenter.id}/collections`, {
          name: `Articles ${locale}`,
          locale,
          is_published: true
        }).then(({ body: collection }) =>
          cy.api('POST', `/api/v1/collections/${collection.data.id}/articles`, {
            title: `Guide ${locale}`,
            content: `<p>Article content ${locale}</p>`,
            locale,
            status: 'published'
          })
        )
      )
    })
  })

  const inboxConfig = () => ({
    language: 'auto',
    help: { help_center_id: helpCenter.id, visitors: { tab: true }, users: { tab: true } }
  })

  it('keeps Help aligned with the widget fallback without a website language', () => {
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.createLivechatInbox({ ...inboxConfig(), fallback_language: 'en-US' }).then((inbox) => {
      setBrowserLanguages('de-AT,de-DE')
      cy.visitWidgetHost(inbox.uuid)
      cy.wait('@localizedHelp').then(({ request, response }) => {
        const params = new URL(request.url).searchParams
        expect(params.get('locale')).to.eq('en-US')
        expect(params.get('fallback_locale')).to.eq(null)
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetLauncher().click()
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.widgetBody().contains('[role="tab"]', 'Help').click()
      cy.widgetBody().contains('Articles en-US').should('be.visible')

      cy.window().then((win) => win.Libredesk.setLanguage('de-AT'))
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'de')
      cy.widgetBody().contains(home.de).should('be.visible')
      cy.widgetBody().contains('Articles de').should('be.visible')

      cy.window().then((win) => win.Libredesk.setLanguage(''))
      cy.wait('@localizedHelp').then(({ request, response }) => {
        expect(new URL(request.url).searchParams.get('locale')).to.eq('en-US')
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetBody().contains(home.en).should('be.visible')
      cy.widgetBody().contains('Articles en-US').should('be.visible')
    })
  })

  it('keeps Help usable while an article from the previous language is loading', () => {
    let releasePrevious, releaseCurrent
    let previousRequested = false
    let currentRequested = false
    const previous = new Cypress.Promise((resolve) => { releasePrevious = resolve })
    const current = new Cypress.Promise((resolve) => { releaseCurrent = resolve })
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.intercept('GET', '**/api/v1/widget/chat/help/articles/*', (req) => {
      if (new URL(req.url).searchParams.get('locale') === 'en-GB') {
        req.alias = 'previousArticle'
        previousRequested = true
        return previous.then(() => req.continue())
      }
      req.alias = 'currentArticle'
      currentRequested = true
      return current.then(() => req.continue())
    })
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-GB' })
      cy.wait('@localizedHelp')
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()
      cy.widgetBody().contains('Articles en-GB').click()
      cy.widgetBody().contains('Guide en-GB').click()
      cy.wrap(null).should(() => expect(previousRequested).to.eq(true))

      cy.window().then((win) => win.Libredesk.setLanguage('en-US'))
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-US')
      cy.widgetBody().contains('Articles en-US').click()
      cy.widgetBody().find('[role="status"].inset-0').should('not.exist')
      cy.widgetBody().contains('Guide en-US').click()
      cy.wrap(null).should(() => expect(currentRequested).to.eq(true))

      cy.then(() => { releasePrevious() })
      cy.wait('@previousArticle')
      cy.widgetBody().find('iframe').should('not.exist')
      cy.widgetBody().find('[role="status"].inset-0').should('be.visible')
      cy.then(() => { releaseCurrent() })
      cy.wait('@currentArticle')
      cy.widgetBody().find('iframe').its('0.contentDocument.body')
        .should('contain', 'Article content en-US')
      cy.widgetBody().find('[role="status"].inset-0').should('not.exist')
    })
  })

  it('keeps search progress tied to the current language', () => {
    let releasePrevious, releaseCurrent
    let previousRequested = false
    let currentRequested = false
    const previous = new Cypress.Promise((resolve) => { releasePrevious = resolve })
    const current = new Cypress.Promise((resolve) => { releaseCurrent = resolve })
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.intercept('GET', '**/api/v1/widget/chat/help/search?*', (req) => {
      if (new URL(req.url).searchParams.get('locale') === 'en-GB') {
        req.alias = 'previousSearch'
        previousRequested = true
        return previous.then(() => req.continue())
      }
      req.alias = 'currentSearch'
      currentRequested = true
      return current.then(() => req.continue())
    })
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-GB' })
      cy.wait('@localizedHelp')
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()
      cy.widgetBody().find('input').type('Guide')
      cy.wrap(null).should(() => expect(previousRequested).to.eq(true))

      cy.window().then((win) => win.Libredesk.setLanguage('en-US'))
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-US')
      cy.widgetBody().find('input').should('have.value', '')
      cy.widgetBody().find('form [role="status"]').should('not.exist')
      cy.widgetBody().find('input').type('Guide')
      cy.wrap(null).should(() => expect(currentRequested).to.eq(true))

      cy.then(() => { releasePrevious() })
      cy.wait('@previousSearch')
      cy.widgetBody().find('form [role="status"]').should('be.visible')
      cy.widgetBody().contains('Guide en-GB').should('not.exist')
      cy.then(() => { releaseCurrent() })
      cy.wait('@currentSearch')
      cy.widgetBody().contains('Guide en-US').should('be.visible')
      cy.widgetBody().find('form [role="status"]').should('not.exist')
    })
  })

  it('keeps article regions when labels match and closes the previous article', () => {
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'en-GB' })
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-GB')
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()
      cy.widgetBody().contains('Articles en-GB').click()
      cy.widgetBody().contains('Guide en-GB').click()
      cy.widgetBody().find('iframe').its('0.contentDocument.body')
        .should('contain', 'Article content en-GB')
      cy.window().its('Libredesk.isExpanded').should('eq', true)

      cy.window().then((win) => win.Libredesk.setLanguage('en-US'))
      cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', 'en-US')
      cy.widgetBody().contains('Articles en-US').should('be.visible')
      cy.widgetBody().find('iframe').should('not.exist')
      cy.window().its('Libredesk.isExpanded').should('eq', false)

      cy.request(`/api/v1/public/help-centers/${helpCenter.slug}/tree?locale=en-AU`)
        .its('body.data.tree.0.locale').should('eq', 'en-US')
    })
  })

  it('uses available HC fallbacks and keeps Chinese writing systems separate', () => {
    cy.intercept('GET', '**/api/v1/widget/chat/help?*').as('localizedHelp')
    cy.createLivechatInbox(inboxConfig()).then((inbox) => {
      cy.visitWidgetHost(inbox.uuid, { language: 'ko-KR' })
      cy.wait('@localizedHelp').then(({ request, response }) => {
        const params = new URL(request.url).searchParams
        expect(params.get('locale')).to.eq('ko-KR')
        expect(params.get('fallback_locale')).to.eq('en-US')
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetLauncher().click()
      cy.widgetBody().contains('[role="tab"]', 'Help').click()

      for (const [requested, resolved] of [['zh-TW', 'zh-Hant'], ['zh-CN', 'zh-Hans'], ['fr-CA', 'de']]) {
        cy.window().then((win) => win.Libredesk.setLanguage(requested))
        cy.wait('@localizedHelp').its('response.body.data.locale').should('eq', resolved)
        cy.widgetBody().contains(`Articles ${resolved}`).should('be.visible')
      }

      cy.window().then((win) => win.Libredesk.setLanguage(null))
      cy.wait('@localizedHelp').then(({ request, response }) => {
        expect(new URL(request.url).searchParams.get('fallback_locale')).to.eq(null)
        expect(response.body.data.locale).to.eq('en-US')
      })
      cy.widgetBody().contains('Articles en-US').should('be.visible')
    })
  })
})
