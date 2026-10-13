const home = { en: 'Home', de: 'Startseite', zh: '主页' }

describe('Live chat widget language chosen by the host page', () => {
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
