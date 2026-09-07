import { useRouter } from 'vue-router'
import { Contact, MessageSquare } from 'lucide-vue-next'
import { useUserStore } from '@main/stores/user'
import { permissions as perms } from '@main/constants/permissions'
import api from '@main/api'
import { SECTIONS } from '../sections'

export const ENTITY_SEARCH_MIN_LENGTH = 3

const RESULT_LIMIT = 10

const contactLabel = (contact) =>
  [contact.first_name, contact.last_name].filter(Boolean).join(' ') || contact.email

// Root-level searches that turn typed text into matching records, alongside the static commands.
export function useEntitySearch() {
  const router = useRouter()
  const userStore = useUserStore()

  const searchContacts = async (term) => {
    if (!userStore.can(perms.CONTACTS_READ_ALL)) return []
    const response = await api.searchContacts({ query: term, limit: RESULT_LIMIT })
    return (response.data.data || []).map((contact) => ({
      id: `search.contact.${contact.id}`,
      label: contactLabel(contact),
      hint: contact.email,
      section: SECTIONS.CONTACT_RESULTS,
      icon: Contact,
      run: () => router.push({ name: 'contact-detail', params: { id: contact.id } })
    }))
  }

  const searchConversations = async (term) => {
    const response = await api.searchConversations({ query: term, limit: RESULT_LIMIT })
    return (response.data.data || []).map((conversation) => ({
      id: `search.conversation.${conversation.uuid}`,
      label: conversation.subject || `#${conversation.reference_number}`,
      hint: `#${conversation.reference_number} · ${conversation.status}`,
      section: SECTIONS.CONVERSATION_RESULTS,
      icon: MessageSquare,
      run: () =>
        router.push({
          name: 'inbox-conversation',
          params: { type: 'assigned', uuid: conversation.uuid }
        })
    }))
  }

  const search = async (term) => {
    const results = await Promise.allSettled([searchContacts(term), searchConversations(term)])
    return results.flatMap((result) => (result.status === 'fulfilled' ? result.value : []))
  }

  return { search }
}
