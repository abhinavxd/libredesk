// The steps run in order and share the record created by the first one.

const stamp = Date.now()
const inboxName = `Cypress Livechat ${stamp}`
const renamedInbox = `Cypress Livechat ${stamp} edited`
const brandName = `Cypress Brand ${stamp}`
const websiteUrl = `https://cypress.test/${stamp}`
const greeting = `Hello from Cypress ${stamp}`
const chatIntroduction = `Ask Cypress ${stamp} anything.`
const trustedDomain = `cypress-${stamp}.test`
const editedBrandName = `Cypress Brand ${stamp} edited`
const homeAppTitle = `Cypress Announcement ${stamp}`
const editedHomeAppTitle = `Cypress Announcement ${stamp} edited`
const secret = `${stamp}`.padEnd(32, 'c').slice(0, 32)
const maskedSecret = '••••••••••'
const newPath = '/admin/inboxes/new'
const listPath = '/admin/inboxes'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

// Tab panels stay in the DOM and are only hidden, so a field is reachable when its tab is open.
const openTab = (label) => cy.get('[role="tab"]').contains(label).click()

const openNewForm = () => {
  cy.visit(newPath)
  cy.contains('Create a live chat inbox').click()
}

describe('Live chat inbox form', () => {
  let inboxId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a live chat inbox', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()

    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.brand_name"]').type(brandName)
    cy.get('input[name="config.website_url"]').type(websiteUrl)

    openTab('Appearance')
    cy.get('input[name="config.launcher.spacing.side"]').clear().type('35')
    cy.get('input[name="config.launcher.spacing.bottom"]').clear().type('45')
    cy.get('select[name="config.launcher.position"]')
      .siblings('button[role="combobox"]')
      .click()
    cy.get('[role="option"]').contains('Left').click()

    openTab('Messages')
    cy.get('textarea[name="config.greeting_message"]').clear().type(greeting)
    cy.get('textarea[name="config.chat_introduction"]').clear().type(chatIntroduction)

    openTab('Security')
    cy.get('input[name="secret"]').type(secret)
    cy.get('input[name="config.session_duration"]').clear().type('4h')
    cy.get('textarea[name="config.trusted_domains"]').type(trustedDomain)

    cy.get('button[type="submit"]').click()
    cy.wait('@createInbox').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      inboxId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(inboxName)
    cy.contains(inboxName).should('exist')
  })

  it('loads the saved values back into every tab of the edit form', () => {
    expect(inboxId, 'inbox from the create step').to.be.a('number')

    cy.visit(`${listPath}/${inboxId}/edit`)

    cy.get('input[name="name"]').should('have.value', inboxName)
    cy.get('input[name="config.brand_name"]').should('have.value', brandName)
    cy.get('input[name="config.website_url"]').should('have.value', websiteUrl)

    openTab('Appearance')
    cy.get('input[name="config.launcher.spacing.side"]').should('have.value', '35')
    cy.get('input[name="config.launcher.spacing.bottom"]').should('have.value', '45')
    cy.get('select[name="config.launcher.position"]').should('have.value', 'left')

    openTab('Messages')
    cy.get('textarea[name="config.greeting_message"]').should('have.value', greeting)
    cy.get('textarea[name="config.chat_introduction"]').should('have.value', chatIntroduction)

    openTab('Security')
    cy.get('input[name="config.session_duration"]').should('have.value', '4h')
    cy.get('textarea[name="config.trusted_domains"]').should('have.value', trustedDomain)
    // The signing secret comes back masked, never as the real value.
    cy.get('input[name="secret"]').should('have.value', maskedSecret)
  })

  it('persists a changed name and greeting message', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.get('input[name="name"]').should('have.value', inboxName).clear().type(renamedInbox)

    openTab('Messages')
    cy.get('textarea[name="config.greeting_message"]').clear().type(`${greeting} v2`)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedInbox)
    openTab('Messages')
    cy.get('textarea[name="config.greeting_message"]').should('have.value', `${greeting} v2`)
  })

  it('rejects a submit with no name and no brand name', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.get('button[type="submit"]').click()

    cy.contains(/required/i).should('be.visible')
    cy.get('@createInbox.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('rejects a malformed website url and jumps back to its tab', () => {
    cy.intercept('POST', '**/api/v1/inboxes').as('createInbox')

    openNewForm()
    cy.get('input[name="name"]').type(inboxName)
    cy.get('input[name="config.brand_name"]').type(brandName)
    cy.get('input[name="config.website_url"]').type('not-a-url')

    openTab('Security')
    cy.get('button[type="submit"]').click()

    cy.contains('Invalid URL').should('be.visible')
    // The offending field is on another tab, so the form has to switch back to it.
    cy.get('input[name="config.website_url"]').should('be.visible')
    cy.get('[role="tab"][data-state="active"]').should('not.contain', 'Security')
    cy.get('@createInbox.all').should('have.length', 0)
  })

  it('persists a brand name edit made alongside a home screen app edit', () => {
    cy.intercept('PUT', `**/api/v1/inboxes/${inboxId}`).as('updateInbox')

    cy.visit(`${listPath}/${inboxId}/edit`)
    openTab('Appearance')
    cy.contains('button', 'Add announcement').click()
    cy.get('input[placeholder="Title"]').type(homeAppTitle)
    cy.get('input[placeholder="Cover image URL"]').type('https://cypress.test/cover.png')
    cy.get('input[placeholder="Link URL"]').type('https://cypress.test/announcement')
    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.get('input[name="config.brand_name"]').clear().type(editedBrandName)
    openTab('Appearance')
    cy.get('input[placeholder="Title"]').clear().type(editedHomeAppTitle)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateInbox').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${inboxId}/edit`)
    cy.get('input[name="config.brand_name"]').should('have.value', editedBrandName)
    openTab('Appearance')
    cy.get('input[placeholder="Title"]').should('have.value', editedHomeAppTitle)
  })

  it('deletes the inbox', () => {
    cy.intercept('DELETE', `**/api/v1/inboxes/${inboxId}`).as('deleteInbox')

    cy.visit(listPath)
    filterList(renamedInbox)
    cy.contains('tr', renamedInbox).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteInbox').its('response.statusCode').should('eq', 200)
    cy.contains(renamedInbox).should('not.exist')
  })
})
