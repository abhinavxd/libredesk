const stamp = Date.now()
const roleName = `Cypress Role ${stamp}`
const renamedRole = `Cypress Role ${stamp} edited`
const finalRole = `Cypress Role ${stamp} again`
const roleDescription = 'Role created by the role form spec'
const updatedDescription = 'Role edited by the role form spec'
const newPath = '/admin/teams/roles/new'
const listPath = '/admin/teams/roles'

const filterList = (text) => cy.get('input[placeholder="Search"]').clear().type(text)

const permission = (label) => cy.contains('label', label).siblings('button[role="checkbox"]')

const togglePermission = (label) => permission(label).click()

describe('Role form', () => {
  let roleId

  beforeEach(() => {
    cy.viewport(1280, 800)
    cy.login()
  })

  it('creates a role', () => {
    cy.intercept('POST', '**/api/v1/roles').as('createRole')

    cy.visit(newPath)

    cy.get('input[name="name"]').type(roleName)
    cy.get('input[name="description"]').type(roleDescription)
    togglePermission('View all conversations')
    togglePermission('Manage tags')
    togglePermission('Manage conversation statuses')

    cy.get('button[type="submit"]').click()
    cy.wait('@createRole').then(({ response }) => {
      expect(response.statusCode).to.eq(200)
      expect(response.body.data.permissions).to.include.members([
        'conversations:read_all',
        'tags:manage',
        'status:manage'
      ])
      roleId = response.body.data.id
    })

    cy.location('pathname').should('eq', listPath)
    filterList(roleName)
    cy.contains(roleName).should('exist')
  })

  it('loads the saved values back into the edit form', () => {
    expect(roleId, 'role from the create step').to.be.a('number')

    cy.visit(`${listPath}/${roleId}/edit`)

    cy.get('input[name="name"]').should('have.value', roleName)
    cy.get('input[name="description"]').should('have.value', roleDescription)
    permission('View all conversations').should('have.attr', 'data-state', 'checked')
    permission('Manage tags').should('have.attr', 'data-state', 'checked')
    permission('Manage conversation statuses').should('have.attr', 'data-state', 'checked')
    permission('Manage webhooks').should('have.attr', 'data-state', 'unchecked')
    permission('Delete contacts').should('have.attr', 'data-state', 'unchecked')
  })

  it('persists a changed name and description', () => {
    cy.intercept('PUT', `**/api/v1/roles/${roleId}`).as('updateRole')

    cy.visit(`${listPath}/${roleId}/edit`)
    cy.get('input[name="name"]').should('have.value', roleName).clear().type(renamedRole)
    cy.get('input[name="description"]').clear().type(updatedDescription)

    cy.get('button[type="submit"]').click()
    cy.wait('@updateRole').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${roleId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedRole)
    cy.get('input[name="description"]').should('have.value', updatedDescription)
  })

  it('persists a changed permission set', () => {
    cy.intercept('PUT', `**/api/v1/roles/${roleId}`).as('updateRole')

    cy.visit(`${listPath}/${roleId}/edit`)
    togglePermission('Manage tags')
    togglePermission('Manage webhooks')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateRole').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${roleId}/edit`)
    permission('Manage tags').should('have.attr', 'data-state', 'unchecked')
    permission('Manage webhooks').should('have.attr', 'data-state', 'checked')
    permission('Manage conversation statuses').should('have.attr', 'data-state', 'checked')
  })

  it('persists text edits made alongside a permission toggle', () => {
    cy.intercept('PUT', `**/api/v1/roles/${roleId}`).as('updateRole')

    cy.visit(`${listPath}/${roleId}/edit`)
    cy.get('input[name="name"]').should('have.value', renamedRole).clear().type(finalRole)
    togglePermission('Manage macros')

    cy.get('button[type="submit"]').click()
    cy.wait('@updateRole').its('response.statusCode').should('eq', 200)

    cy.visit(`${listPath}/${roleId}/edit`)
    cy.get('input[name="name"]').should('have.value', finalRole)
    permission('Manage macros').should('have.attr', 'data-state', 'checked')
  })

  it('rejects a submit with no name or description', () => {
    cy.intercept('POST', '**/api/v1/roles').as('createRole')

    cy.visit(newPath)
    cy.get('button[type="submit"]').click()

    // Submitting scrolls to the bottom of the permission matrix, so bring the message back in view.
    cy.contains('label', 'Description')
      .parent()
      .contains(/required/i)
      .scrollIntoView()
      .should('be.visible')
    cy.get('@createRole.all').should('have.length', 0)
    cy.location('pathname').should('eq', newPath)
  })

  it('deletes the role', () => {
    cy.intercept('DELETE', `**/api/v1/roles/${roleId}`).as('deleteRole')

    cy.visit(listPath)
    filterList(finalRole)
    cy.contains('tr', finalRole).find('button[aria-haspopup="menu"]').click()
    cy.get('[role="menuitem"]').contains('Delete').click()
    cy.get('[role="alertdialog"]').contains('button', 'Delete').click()

    cy.wait('@deleteRole').its('response.statusCode').should('eq', 200)
    cy.contains(finalRole).should('not.exist')
  })
})
