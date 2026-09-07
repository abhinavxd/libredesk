// The first page is stubbed empty, so these pass or fail on the ids call alone whatever the DB holds.

const stamp = Date.now()
const agentFirst = `Zzlate${stamp}`
const agentName = `${agentFirst} Agent`
const teamName = `zzz-late-team-${stamp}`
const viewName = `ZZ late view ${stamp}`
const sharedViewName = `ZZ late shared view ${stamp}`
const ruleName = `ZZ late rule ${stamp}`
const macroName = `ZZ late macro ${stamp}`
const tagName = `zz-late-tag-${stamp}`
const inboxName = `ZZ late inbox ${stamp}`
const contactLastName = `LateCustomer${stamp}`

const leafFilters = (agentId) => ({
  logic: 'AND',
  rules: [
    {
      logic: 'AND',
      rules: [
        {
          model: 'conversations',
          field: 'assigned_user_id',
          operator: 'equals',
          value: String(agentId)
        }
      ]
    }
  ]
})

const blankFirstPage = (resource) => {
  cy.intercept({ method: 'GET', url: `**/api/v1/${resource}*`, query: { page: '1' } }, { body: { data: [] } })
}

const valueCombobox = () => cy.get('[data-cy="filter-value"] button[role="combobox"]')

const typeInPicker = (text) =>
  cy
    .get('[data-radix-popper-content-wrapper]:visible input')
    .last()
    .invoke('val', text)
    .trigger('input')

describe('Saved lookup values outside the first page', () => {
  const ids = {}

  // radix-vue focuses a trigger it just unmounted, and Cypress fails the test on uncaught errors.
  Cypress.on('uncaught:exception', (err) => !err.message.includes("reading 'focus'"))

  before(() => {
    cy.login()
    cy.api('POST', '/api/v1/agents', {
      first_name: agentFirst,
      last_name: 'Agent',
      email: `zz-late-agent-${stamp}@example.com`,
      roles: ['Agent'],
      send_welcome_email: false
    }).then(({ body }) => {
      ids.agent = body.data.id

      cy.api('POST', '/api/v1/teams', {
        name: teamName,
        emoji: '🛰️',
        conversation_assignment_type: 'Round robin',
        timezone: 'UTC',
        max_auto_assigned_conversations: 0
      }).then(({ body: teamBody }) => {
        ids.team = teamBody.data.id

        cy.api('POST', '/api/v1/views/me', {
          name: viewName,
          filters: leafFilters(ids.agent)
        }).then(({ body: viewBody }) => (ids.view = viewBody.data.id))

        cy.api('POST', '/api/v1/shared-views', {
          name: sharedViewName,
          visibility: 'all',
          filters: leafFilters(ids.agent)
        }).then(({ body: sharedBody }) => (ids.sharedView = sharedBody.data.id))

        cy.api('POST', '/api/v1/automations/rules', {
          name: ruleName,
          description: ruleName,
          type: 'new_conversation',
          enabled: false,
          events: [],
          rules: [
            {
              group_operator: 'AND',
              groups: [
                {
                  logical_op: 'AND',
                  rules: [
                    {
                      field: 'assigned_user',
                      value: String(ids.agent),
                      operator: 'equals',
                      field_type: 'conversation'
                    },
                    {
                      field: 'assigned_team',
                      value: String(ids.team),
                      operator: 'equals',
                      field_type: 'conversation'
                    }
                  ]
                }
              ],
              actions: [
                { type: 'assign_user', value: [String(ids.agent)] },
                { type: 'assign_team', value: [String(ids.team)] },
                {
                  type: 'notify',
                  value: [],
                  recipients: [`user:${ids.agent}`, `team:${ids.team}`],
                  subject: 'x',
                  message: 'y'
                }
              ]
            }
          ]
        }).then(({ body: ruleBody }) => (ids.rule = ruleBody.data.id))

        cy.api('POST', '/api/v1/macros', {
          name: macroName,
          message_content: 'probe',
          visibility: 'all',
          visible_when: ['replying'],
          actions: [{ type: 'assign_user', value: [String(ids.agent)] }]
        }).then(({ body: macroBody }) => (ids.macro = macroBody.data.id))

        cy.api('POST', '/api/v1/tags', { name: tagName }).then(({ body: tagBody }) => {
          ids.tag = tagBody.data.id
        })

        cy.api('POST', '/api/v1/inboxes', {
          name: inboxName,
          channel: 'email',
          enabled: true,
          from: `Late <late+${stamp}@cypress.test>`,
          config: {
            auth_type: 'password',
            imap: [],
            smtp: [
              {
                host: '127.0.0.1',
                port: 1025,
                auth_protocol: 'none',
                max_conns: 2,
                idle_timeout: '5s',
                pool_wait_timeout: '5s',
                max_msg_retries: 1,
                tls_type: 'none'
              }
            ]
          }
        }).then(({ body: inboxBody }) => {
          ids.inbox = inboxBody.data.id
          cy.api('POST', '/api/v1/conversations', {
            inbox_id: ids.inbox,
            agent_id: ids.agent,
            team_id: ids.team,
            contact_email: `zz-late-customer-${stamp}@example.com`,
            first_name: 'ZZ',
            last_name: contactLastName,
            subject: `ZZ late conversation ${stamp}`,
            content: '<p>Saved lookup values probe.</p>',
            initiator: 'contact'
          }).then(({ body: convBody }) => {
            ids.conversation = convBody.data.uuid
            cy.api('POST', `/api/v1/conversations/${ids.conversation}/tags`, {
              action: 'set_tags',
              tags: [tagName]
            })
          })
        })
      })
    })
  })

  beforeEach(() => {
    cy.viewport(1400, 1000)
    cy.login()
  })

  describe('user view filters', () => {
    const openEditDialog = () => {
      cy.visit('/inboxes/assigned')
      cy.contains('button', viewName).parent().find('button[aria-haspopup="menu"]').click()
      cy.get('[role="menuitem"]').contains('Edit').click()
      cy.get('[data-cy="filter-row"]').should('have.length', 1)
    }

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      openEditDialog()

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      valueCombobox().should('contain.text', agentName)
    })

    // Inside a dialog, radix closes the picker on Cypress's synthetic typing after the first key,
    // so this pins the search call reaching the server. Real typing lists the results.
    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      openEditDialog()
      valueCombobox().should('contain.text', agentName).click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
    })
  })

  describe('shared view filters', () => {
    const editPath = () => `/admin/conversations/shared-views/${ids.sharedView}/edit`

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      cy.visit(editPath())

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      valueCombobox().should('contain.text', agentName)
    })

    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      valueCombobox().click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  describe('automation rule', () => {
    const editPath = () => `/admin/automations/${ids.rule}/edit`

    it('labels the saved agent and team in conditions and actions', () => {
      blankFirstPage('agents/compact')
      blankFirstPage('teams/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')
      cy.intercept('GET', '**/api/v1/teams/compact?*ids=*').as('teamById')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      cy.wait('@teamById').its('response.statusCode').should('eq', 200)

      cy.get('button[role="combobox"]')
        .filter((_, el) => el.innerText.includes(agentName))
        .should('have.length.at.least', 2)
      cy.get('button[role="combobox"]')
        .filter((_, el) => el.innerText.includes(teamName))
        .should('have.length.at.least', 2)
    })

    it('labels notify recipients instead of showing the raw id', () => {
      blankFirstPage('agents/compact')
      blankFirstPage('teams/compact')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)

      cy.contains(`Agent: ${agentName}`).should('exist')
      cy.contains(`Team: ${teamName}`).should('exist')
      cy.contains(`user:${ids.agent}`).should('not.exist')
      cy.contains(`team:${ids.team}`).should('not.exist')
    })

    it('searches agents from a condition value', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', ruleName)
      cy.contains('button[role="combobox"]', agentName).first().click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  describe('macro actions', () => {
    const editPath = () => `/admin/conversations/macros/${ids.macro}/edit`

    it('labels the saved agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', macroName)

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      cy.contains('button[role="combobox"]', agentName).should('exist')
    })

    it('searches agents on the server', () => {
      cy.intercept('GET', '**/api/v1/agents/compact?*q=*').as('agentSearch')

      cy.visit(editPath())
      cy.get('input[name="name"]').should('have.value', macroName)
      cy.contains('button[role="combobox"]', agentName).click()
      typeInPicker(agentFirst)

      cy.wait('@agentSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', agentName).should('exist')
    })
  })

  describe('conversation sidebar', () => {
    const openConversation = () => {
      cy.intercept('GET', '**/messages?page=*').as('loadMessages')
      cy.visit(`/inboxes/all/conversation/${ids.conversation}`)
      cy.wait('@loadMessages')
      // The open set of sidebar sections is remembered per browser, so blind toggling closes it.
      cy.contains('button', 'Actions').then(($trigger) => {
        if ($trigger.attr('aria-expanded') !== 'true') cy.wrap($trigger).click()
      })
    }

    it('labels the assigned team by asking for it by id', () => {
      blankFirstPage('teams/compact')
      cy.intercept('GET', '**/api/v1/teams/compact?*ids=*').as('teamById')

      openConversation()

      cy.wait('@teamById').its('response.statusCode').should('eq', 200)
      cy.contains('button[role="combobox"]', teamName).should('exist')
    })

    it('labels the assigned agent by asking for it by id', () => {
      blankFirstPage('agents/compact')
      cy.intercept('GET', '**/api/v1/agents/compact?*ids=*').as('agentById')

      openConversation()

      cy.wait('@agentById').its('response.statusCode').should('eq', 200)
      cy.contains('button[role="combobox"]', agentName).should('exist')
    })

    it('shows the assigned tag while the tag list page is empty', () => {
      blankFirstPage('tags')

      openConversation()

      cy.contains('[data-radix-vue-collection-item]', tagName).should('exist')
    })

    it('searches teams on the server', () => {
      cy.intercept('GET', '**/api/v1/teams/compact?*q=*').as('teamSearch')

      openConversation()
      cy.contains('button[role="combobox"]', teamName).click()
      typeInPicker(teamName)

      cy.wait('@teamSearch').its('request.url').should('include', 'q=')
      cy.contains('[role="option"]', teamName).should('exist')
    })
  })

  after(() => {
    cy.login()
    if (ids.view) cy.api('DELETE', `/api/v1/views/me/${ids.view}`)
    if (ids.sharedView) cy.api('DELETE', `/api/v1/shared-views/${ids.sharedView}`)
    if (ids.rule) cy.api('DELETE', `/api/v1/automations/rules/${ids.rule}`)
    if (ids.macro) cy.api('DELETE', `/api/v1/macros/${ids.macro}`)
    if (ids.team) cy.api('DELETE', `/api/v1/teams/${ids.team}`)
    if (ids.agent) cy.api('DELETE', `/api/v1/agents/${ids.agent}`)
    if (ids.tag) cy.api('DELETE', `/api/v1/tags/${ids.tag}`)
    if (ids.inbox) cy.api('DELETE', `/api/v1/inboxes/${ids.inbox}`)
  })
})
