import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import WhatsAppConversationForm from '@main/features/conversation/WhatsAppConversationForm.vue'

describe('WhatsAppConversationForm', () => {
  it('starts with the supplied contact selected', () => {
    const pinia = createPinia()
    const i18n = createI18n({ legacy: false, missingWarn: false, fallbackWarn: false })
    const contact = {
      id: 42,
      first_name: 'E2E',
      last_name: 'Contact',
      phone_number: '9912345678',
      phone_number_country_code: 'IN'
    }

    cy.mount(WhatsAppConversationForm, {
      props: { initialContact: contact },
      global: {
        plugins: [pinia, i18n],
        config: { globalProperties: { emitter: { emit: cy.stub() } } },
        stubs: {
          ComboBox: true,
          ContactSearchResults: true,
          Select: true,
          SelectAgentCombobox: true,
          SelectTeamCombobox: true,
          WhatsAppTemplatePicker: true
        }
      }
    })

    cy.get('input[type="tel"]').should('have.value', contact.phone_number)
    cy.get('input[type="text"]').eq(0).should('have.value', contact.first_name).and('be.disabled')
    cy.get('input[type="text"]').eq(1).should('have.value', contact.last_name).and('be.disabled')
  })
})
