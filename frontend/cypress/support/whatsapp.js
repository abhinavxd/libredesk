export const hoursAgo = (hours) => String(Math.floor(Date.now() / 1000) - hours * 3600)

export const inboundPayload = ({ wabaID, phoneNumberID, waID, contactName = 'E2E Contact', message }) => ({
  object: 'whatsapp_business_account',
  entry: [
    {
      id: wabaID,
      changes: [
        {
          field: 'messages',
          value: {
            messaging_product: 'whatsapp',
            metadata: { display_phone_number: '15550001111', phone_number_id: phoneNumberID },
            contacts: [{ profile: { name: contactName }, wa_id: waID }],
            messages: [message]
          }
        }
      ]
    }
  ]
})

export const statusPayload = ({ wabaID, phoneNumberID, waID, id, status, errors }) => ({
  object: 'whatsapp_business_account',
  entry: [
    {
      id: wabaID,
      changes: [
        {
          field: 'messages',
          value: {
            metadata: { phone_number_id: phoneNumberID },
            statuses: [
              { id, status, timestamp: hoursAgo(0), recipient_id: waID, ...(errors ? { errors } : {}) }
            ]
          }
        }
      ]
    }
  ]
})

export const templateStatusPayload = ({ wabaID, name, event, reason = 'NONE' }) => ({
  object: 'whatsapp_business_account',
  entry: [
    {
      id: wabaID,
      changes: [
        {
          field: 'message_template_status_update',
          value: {
            event,
            message_template_name: name,
            message_template_language: 'en_US',
            reason
          }
        }
      ]
    }
  ]
})

// Meta signs every delivery with the app secret, so an unsigned or wrongly signed body is refused.
Cypress.Commands.add('waPostWebhook', (payload, { inboxID, secret, failOnStatusCode = false } = {}) => {
  const body = JSON.stringify(payload)
  return cy.task('metaMock:sign', { body, secret }).then((signature) =>
    cy.request({
      method: 'POST',
      url: `/webhooks/whatsapp/${inboxID}`,
      headers: { 'Content-Type': 'application/json', 'X-Hub-Signature-256': signature },
      body,
      failOnStatusCode
    })
  )
})

// fetch must yield null for "not there yet". On undefined Cypress passes the previous subject through, ending the poll on the wrong value.
Cypress.Commands.add('waPoll', (label, fetch, check, tries = 30) => {
  const attempt = (left) =>
    fetch().then((result) => {
      if (check(result)) return cy.wrap(result, { log: false })
      if (left <= 0) throw new Error(`timed out waiting for ${label}`)
      // eslint-disable-next-line cypress/no-unnecessary-waiting -- the pause between polls is the retry
      return cy.wait(500, { log: false }).then(() => attempt(left - 1))
    })
  return attempt(tries)
})

Cypress.Commands.add('waMetaCalls', (predicate) =>
  cy.task('metaMock:requests').then((requests) => requests.filter(predicate))
)

Cypress.Commands.add('waMessages', (conversationUUID) =>
  cy
    .api('GET', `/api/v1/conversations/${conversationUUID}/messages`)
    .then(({ body }) => body.data.results)
)
