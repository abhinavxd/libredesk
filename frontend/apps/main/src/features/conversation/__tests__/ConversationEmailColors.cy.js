import { defineComponent, h } from 'vue'
import { useColorMode } from '@vueuse/core'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import mitt from 'mitt'
import { initI18n } from '@main/i18n'
import { useConversationStore } from '@main/stores/conversation'
import Conversation from '@main/features/conversation/Conversation.vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'

const toggle = '[data-cy="conversation-email-color-toggle"]'
const contact = { id: 42, first_name: 'Alex', last_name: 'Morgan', email: 'alex@example.test' }
const html = '<div style="color:#000;background-color:#fff"><p>Email content</p></div>'
const message = (id, overrides = {}) => ({
  id, uuid: `message-${id}`, type: 'incoming', content: html,
  content_type: 'html', author: contact, sender_id: contact.id,
  created_at: `2026-10-24T12:00:0${id}Z`, attachments: [], meta: {},
  private: false, status: 'sent', ...overrides
})

let store
let colorMode
const mountConversation = ({ channel = 'email', dark = true } = {}) => {
  localStorage.setItem('vueuse-color-scheme', dark ? 'dark' : 'light')
  document.documentElement.classList.toggle('dark', dark)
  cy.readFile('../i18n/en-US.json').then((messages) => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', name: 'contact-detail', component: { template: '<div />' } }]
    })
    const Host = defineComponent({
      components: { Conversation, TooltipProvider },
      setup() {
        colorMode = useColorMode()
        store = useConversationStore()
        store.conversation.data = {
          uuid: 'conversation-1', inbox_channel: channel, inbox_mail: 'support@example.test',
          contact, status: 'Open', assigned_user_id: 7
        }
        store.messages.data.addMessages('conversation-1', [
          message(1), message(2),
          message(3, { type: 'outgoing', sender_id: 7 }),
          message(4, { type: 'outgoing', sender_id: 7, private: true }),
          message(5, { content: 'A plain-text message.', content_type: 'text' })
        ], 1, 1)
        store.messages.version++
        store.updateAssigneeLastSeen = () => Promise.resolve()
        return () => h(TooltipProvider, null, {
          default: () => h(Conversation, { style: 'height:650px' })
        })
      },
    })
    cy.mount(Host, {
      global: {
        plugins: [
          createPinia(), router,
          initI18n({ legacy: false, locale: 'en-US', messages: { 'en-US': messages } })
        ],
        config: {
          globalProperties: { emitter: mitt() },
          errorHandler(error) { throw error }
        }
      }
    })
  })
}

const expectIncomingColor = (original) => {
  for (const id of [1, 2]) {
    cy.get(`[data-message-uuid="message-${id}"] .native-html p`)
      .should(original ? 'have.css' : 'not.have.css', 'color', 'rgb(0, 0, 0)')
  }
}

describe('Conversation email color preference', () => {
  it('toggles all incoming emails from one header action, including newly loaded messages', () => {
    cy.viewport(1440, 900)
    mountConversation()
    cy.get(toggle)
      .should('have.length', 1)
      .and('have.attr', 'aria-label', 'Show original email colors')
      .and('have.attr', 'aria-pressed', 'false')
      .and('have.text', '')
      .and('have.css', 'width', '32px')
    cy.get(`${toggle} svg`).should('have.class', 'lucide-sun')
      .and('have.attr', 'aria-hidden', 'true')
    cy.get(toggle).next('button').should('have.attr', 'aria-label', 'Details')
    cy.get('.message-bubble button[data-cy="email-color-toggle"]').should('not.exist')
    cy.get('.email-light-canvas').should('not.exist')
    expectIncomingColor(false)
    cy.get(toggle).focus()
    cy.get('[role="tooltip"]').should('have.text', 'Show original email colors')
    cy.get(toggle).type('{enter}')
    cy.get(toggle).should('have.attr', 'aria-pressed', 'true')
      .and('have.attr', 'aria-label', 'Show adjusted dark-mode colors')
      .and('have.text', '')
    cy.get(`${toggle} svg`).should('have.class', 'lucide-moon')
    cy.get(toggle).trigger('pointerleave', { pointerType: 'mouse' })
    cy.get(toggle).trigger('pointermove', { pointerType: 'mouse' })
    cy.get('[role="tooltip"]').should('have.text', 'Show adjusted dark-mode colors')
    expectIncomingColor(true)
    for (const id of [3, 4]) {
      cy.get(`[data-message-uuid="message-${id}"] .native-html p`)
        .should('have.css', 'color', 'rgb(0, 0, 0)')
      cy.get(`[data-message-uuid="message-${id}"] .message-bubble`)
        .should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
    }
    cy.get('[data-message-uuid="message-5"] .message-bubble')
      .should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
    cy.then(() => {
      store.messages.data.addMessage('conversation-1', message(6))
      store.messages.version++
    })
    cy.get('[data-message-uuid="message-6"] .native-html p')
      .should('have.css', 'color', 'rgb(0, 0, 0)')
    cy.get(toggle).click()
    cy.get(toggle).should('have.attr', 'aria-label', 'Show original email colors')
    cy.get(`${toggle} svg`).should('have.class', 'lucide-sun')
    expectIncomingColor(false)
    cy.get('[data-message-uuid="message-6"] .native-html p')
      .should('not.have.css', 'color', 'rgb(0, 0, 0)')
  })

  it('resets the preference when switching conversations in the same mounted view', () => {
    mountConversation()
    cy.get(toggle).click()
    cy.then(() => {
      store.messages.data.addMessages('conversation-2', [message(1), message(2)], 1, 1)
      store.conversation.data = { ...store.current, uuid: 'conversation-2' }
      store.messages.version++
    })
    cy.get(toggle).should('have.attr', 'aria-pressed', 'false')
    expectIncomingColor(false)
    cy.then(() => { store.conversation.data = { ...store.current, uuid: 'conversation-1' } })
    cy.get(toggle).should('have.attr', 'aria-pressed', 'false')
    expectIncomingColor(false)
  })

  it('hides and resets the override when leaving dark mode', () => {
    mountConversation()
    cy.get(toggle).click()
    cy.then(() => { colorMode.value = 'light' })
    cy.get(toggle).should('not.exist')
    expectIncomingColor(true)
    cy.then(() => { colorMode.value = 'dark' })
    cy.get(toggle).should('have.attr', 'aria-pressed', 'false')
    expectIncomingColor(false)
  })

  it('does not show the action in light mode', () => {
    mountConversation({ dark: false })
    cy.get(toggle).should('not.exist')
    expectIncomingColor(true)
  })

  it('does not show the action for other channels', () => {
    mountConversation({ channel: 'livechat' })
    cy.get(toggle).should('not.exist')
    expectIncomingColor(true)
  })
})
