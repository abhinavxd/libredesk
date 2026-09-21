// The steps run in order and share the inbox created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Widget Tabs ${stamp}`
const brandName = `Cypress Tabs Brand ${stamp}`
const campaignName = `Cypress Campaign ${stamp}`
const campaignMessage = `Need a hand with pricing? ${stamp}`
const secondCampaignName = `Cypress Second ${stamp}`
const hcName = `Cypress Widget HC ${stamp}`
const hcSlug = `cypress-widget-hc-${stamp}`
const collectionName = `Cypress Collection ${stamp}`
const articleTitle = `Cypress Article ${stamp}`
const listPath = '/admin/inboxes'

const selectOption = (trigger, option) => {
  cy.get(trigger).click()
  cy.get('[role="option"]').contains(option).click()
  // The listbox keeps a focus guard over the form until it has finished closing.
  cy.get('[role="option"]').should('not.exist')
}

const editInbox = (inboxId) => cy.visit(`${listPath}/${inboxId}/edit`)

const openCampaign = (name) => cy.contains('button', name).click()

describe('Live chat inbox form: proactive messages and help center tabs', () => {
  let inboxId
  let helpCenterId
  let articleId

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/help-centers', {
      name: hcName,
      slug: hcSlug,
      page_title: `Help ${stamp}`,
      default_locale: 'en',
      allowed_locales: ['en'],
      template: 'docs'
    })
      .then(({ body }) => {
        helpCenterId = body.data.id
        return cy.api('POST', `/api/v1/help-centers/${helpCenterId}/collections`, {
          name: collectionName,
          locale: 'en',
          sort_order: 1,
          is_published: true
        })
      })
      .then(({ body }) =>
        cy.api('POST', `/api/v1/collections/${body.data.id}/articles`, {
          title: articleTitle,
          content: '<p>Cypress body.</p>',
          locale: 'en',
          status: 'published'
        })
      )
      .then(({ body }) => {
        articleId = body.data.id
        expect(articleId).to.be.a('number')
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
    if (helpCenterId) {
      cy.api('DELETE', `/api/v1/help-centers/${helpCenterId}`, null, { failOnStatusCode: false })
    }
  })

  it('creates the inbox the other steps edit', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    cy.visit('/admin/inboxes/new')
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

  it('saves an inbox created outside the form, which carries no prechat config', () => {
    const apiInboxName = `${inboxName} api`
    cy.api('POST', '/api/v1/inboxes', {
      name: apiInboxName,
      channel: 'livechat',
      enabled: true,
      config: {
        brand_name: brandName,
        colors: { primary: '#112233' },
        launcher: { position: 'right' }
      }
    }).then(({ body }) => {
      const apiInboxId = body.data.id
      cy.intercept('PUT', `**/api/v1/inboxes/${apiInboxId}`).as('updateApiInbox')

      editInbox(apiInboxId)
      cy.openInboxSection('General')
      cy.get('input[name="config.website_url"]').type('https://cypress.test/api-inbox')
      cy.get('button[type="submit"]').click()
      cy.wait('@updateApiInbox').its('response.statusCode').should('eq', 200)

      cy.api('DELETE', `/api/v1/inboxes/${apiInboxId}`, null, { failOnStatusCode: false })
    })
  })

  it('saves a proactive message and loads every field back', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', 'Add proactive message').click()

    cy.get('#campaign-name').clear().type(campaignName)
    cy.get('#campaign-message').clear().type(campaignMessage)
    cy.get('#campaign-include_urls').clear().type('/pricing\nhttps://example.com/docs/*')
    cy.get('#campaign-exclude_urls').clear().type('/checkout/*')
    cy.get('#campaign-delay_seconds').type('{selectall}25')
    cy.get('#campaign-event').clear().type('pricing_viewed')
    selectOption('#campaign-repeat', 'After an interval')
    cy.get('#campaign-repeat_hours').type('{selectall}12')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    openCampaign(campaignName)

    cy.get('#campaign-name').should('have.value', campaignName)
    cy.get('#campaign-message').should('have.value', campaignMessage)
    cy.get('#campaign-include_urls').should(
      'have.value',
      '/pricing\nhttps://example.com/docs/*'
    )
    cy.get('#campaign-exclude_urls').should('have.value', '/checkout/*')
    cy.get('#campaign-delay_seconds').should('have.value', '25')
    cy.get('#campaign-event').should('have.value', 'pricing_viewed')
    cy.get('#campaign-repeat_hours').should('have.value', '12')
  })

  it('rejects a proactive message with an empty message', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    openCampaign(campaignName)
    cy.get('#campaign-message').clear()

    cy.get('button[type="submit"]').click()

    cy.contains('Message cannot be empty').should('be.visible')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('rejects a repeat interval outside the allowed range', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    openCampaign(campaignName)
    cy.get('#campaign-repeat_hours').type('{selectall}9000')

    cy.get('button[type="submit"]').click()

    cy.get('#campaign-repeat_hours').should('have.attr', 'aria-invalid', 'true')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('rejects a proactive message with both devices turned off', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    openCampaign(campaignName)
    cy.get('[data-campaign-devices] button[role="switch"]').each(($switch) => {
      cy.wrap($switch).click()
    })

    cy.get('button[type="submit"]').click()

    cy.contains('Select at least one device').should('be.visible')
    cy.get('@updateInbox.all').should('have.length', 0)
  })

  it('persists the enabled toggle and the saved order of two proactive messages', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains('button', 'Add proactive message').click()
    cy.get('#campaign-name').clear().type(secondCampaignName)
    cy.get('#campaign-message').clear().type(`Second message ${stamp}`)
    cy.contains('button', 'Back to proactive messages').click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    // A new proactive message starts paused, so one click enables it.
    cy.contains('button', campaignName)
      .parents('.border')
      .first()
      .find('button[role="switch"][aria-label="Enabled"]')
      .should('have.attr', 'data-state', 'unchecked')
      .click()
    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.api('GET', `/api/v1/inboxes/${inboxId}`).then(({ body }) => {
      const campaigns = body.data.config.campaigns
      expect(campaigns.map((item) => item.name)).to.deep.eq([campaignName, secondCampaignName])
      expect(campaigns.find((item) => item.name === campaignName).enabled).to.eq(true)
    })
  })

  it('deletes a proactive message', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    openCampaign(secondCampaignName)
    cy.get('[data-campaign-editor]').contains('button', 'Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()
    cy.get('[data-campaign-editor]').should('not.exist')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    editInbox(inboxId)
    cy.openInboxSection('Proactive messages')
    cy.contains(secondCampaignName).should('not.exist')
    cy.contains(campaignName).should('exist')
  })

  it('links a help center, shows the help tab and features an article', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Help center')
    selectOption('#widget-help-center', hcName)

    cy.get('[data-help-audience="visitors"]').contains('Show Help tab').should('be.visible')
    cy.contains('Featured articles').should('be.visible')
    cy.get('[data-featured-articles] button[role="combobox"]').should('not.be.disabled').click()
    cy.get('[role="option"]').contains(articleTitle).click()

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    editInbox(inboxId)
    cy.openInboxSection('Help center')
    cy.get('#widget-help-center').should('contain', hcName)
    cy.contains(articleTitle).scrollIntoView().should('be.visible')

    cy.api('GET', `/api/v1/inboxes/${inboxId}`).then(({ body }) => {
      const help = body.data.config.help
      expect(help.help_center_id).to.eq(helpCenterId)
      expect(help.featured_ids).to.deep.eq([articleId])
    })
  })

  it('removes a featured article', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Help center')
    cy.get('[data-featured-articles] button[aria-label="Remove"]').click()
    cy.get('[data-featured-articles] button[aria-label="Remove"]').should('not.exist')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.api('GET', `/api/v1/inboxes/${inboxId}`).then(({ body }) => {
      expect(body.data.config.help.featured_ids).to.deep.eq([])
    })
  })

  it('unlinks the help center and hides its settings', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    editInbox(inboxId)
    cy.openInboxSection('Help center')
    selectOption('#widget-help-center', 'None')

    cy.contains('Featured articles').should('not.exist')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.api('GET', `/api/v1/inboxes/${inboxId}`).then(({ body }) => {
      expect(body.data.config.help.help_center_id).to.eq(0)
    })
  })
})
