import { createI18n } from 'vue-i18n'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import mitt from 'mitt'
import MessageBubble from '@main/features/conversation/message/MessageBubble.vue'

const content =
  '<div style="color:rgb(0,0,0);background-color:rgb(255,255,255)"><p>This email keeps its original colors.</p></div>'

const mountMessage = ({
  darkMode = true,
  showOriginalEmailColors = false,
  direction = 'incoming',
  contentType = 'html',
  channel = 'email',
  privateNote = false,
  groupWithPrev = false
} = {}) => {
  const pinia = createPinia()
  pinia.use(({ store }) => {
    if (store.$id === 'conversation') {
      store.conversation.data = { uuid: 'conversation-1', inbox_channel: channel }
    }
  })
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', name: 'contact-detail', component: { template: '<div />' } }]
  })
  return cy.mount(MessageBubble, {
    props: {
      message: {
        uuid: 'message-1', sender_id: 1,
        author: { id: 1, first_name: 'Alex', last_name: 'Morgan', avatar_url: '' },
        attachments: [], content: contentType === 'text' ? 'A plain-text message.' : content,
        content_type: contentType, created_at: '2026-10-24T12:00:00Z',
        meta: {}, private: privateNote, status: 'sent', type: direction
      },
      direction, darkMode, showOriginalEmailColors, groupWithPrev
    },
    global: {
      plugins: [pinia, router, createI18n({ legacy: false, locale: 'en-US', messages: {} })],
      config: { globalProperties: { emitter: mitt() } },
      stubs: {
        BubbleAttachmentPreview: true,
        CSATResponseDisplay: true,
        ImageLightbox: true,
        Tooltip: { template: '<div><slot /></div>' },
        TooltipContent: { template: '<div hidden><slot /></div>' },
        TooltipTrigger: { template: '<div><slot /></div>' }
      }
    }
  })
}

describe('MessageBubble email colors', () => {
  beforeEach(() => {
    document.documentElement.classList.add('dark')
  })

  it('normalizes incoming HTML without adding a white canvas or per-message controls', () => {
    mountMessage({ groupWithPrev: true })
    cy.contains('a', 'Alex Morgan').should('not.exist')
    cy.get('[data-cy="email-color-toggle"], [data-cy="email-color-actions"]').should('not.exist')
    cy.get('.email-light-canvas').should('not.exist')
    cy.get('.message-bubble').should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
    cy.get('.native-html [style]').first()
      .should('have.css', 'background-color', 'rgba(0, 0, 0, 0)')
    cy.contains('.native-html p', 'This email keeps its original colors.')
      .should('not.have.css', 'color', 'rgb(0, 0, 0)')
  })

  it('remounts the HTML when the conversation color preference changes', () => {
    mountMessage().then(({ wrapper }) => {
      cy.then(() => wrapper.setProps({ showOriginalEmailColors: true }))
      cy.contains('.native-html p', 'This email keeps its original colors.')
        .should('have.css', 'color', 'rgb(0, 0, 0)')
      cy.get('.message-bubble')
        .should('have.css', 'background-color', 'rgb(255, 255, 255)')
        .and('have.css', 'color', 'rgb(10, 10, 10)')
      cy.then(() => wrapper.setProps({ showOriginalEmailColors: false }))
      cy.contains('.native-html p', 'This email keeps its original colors.')
        .should('not.have.css', 'color', 'rgb(0, 0, 0)')
      cy.get('.message-bubble')
        .should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
    })
  })

  it('leaves incoming HTML untouched in light mode', () => {
    document.documentElement.classList.remove('dark')
    mountMessage({ darkMode: false })
    cy.contains('.native-html p', 'This email keeps its original colors.')
      .should('have.css', 'color', 'rgb(0, 0, 0)')
    cy.get('.native-html [style]').first()
      .should('have.css', 'background-color', 'rgb(255, 255, 255)')
  })

  for (const [name, options] of [
    ['outgoing messages', { direction: 'outgoing' }],
    ['private notes', { direction: 'outgoing', privateNote: true }],
    ['non-email messages', { channel: 'livechat' }]
  ]) {
    it(`does not normalize ${name} in either color mode`, () => {
      mountMessage(options).then(({ wrapper }) => {
        cy.get('.native-html p').should('have.css', 'color', 'rgb(0, 0, 0)')
        cy.then(() => wrapper.setProps({ showOriginalEmailColors: true }))
        cy.get('.native-html p').should('have.css', 'color', 'rgb(0, 0, 0)')
        cy.get('.message-bubble')
          .should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
      })
    })
  }

  it('does not add a white background to plain-text messages', () => {
    mountMessage({ contentType: 'text', showOriginalEmailColors: true })
    cy.contains('.native-html', 'A plain-text message.').should('be.visible')
    cy.get('.message-bubble')
      .should('not.have.css', 'background-color', 'rgb(255, 255, 255)')
  })
})
