import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowUpDown } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { SECTIONS } from '../sections'

const SORT_FIELDS = ['oldest', 'newest', 'started_first', 'started_last', 'waiting_longest']

export const isConversationListRoute = (route) => route.path.startsWith('/addresses')

export function useListCommands() {
  const route = useRoute()
  const { t } = useI18n()
  const conversationStore = useConversationStore()

  const section = SECTIONS.LIST

  return computed(() => {
    if (!isConversationListRoute(route)) return []
    const commands = []

    commands.push(
      {
        id: 'list.sort',
        label: t('globals.messages.sortBy'),
        keywords: ['order'],
        section,
        icon: ArrowUpDown,
        group: true
      },
      ...SORT_FIELDS.map((field) => ({
        id: `list.sort.${field}`,
        label: t(conversationStore.sortFieldI18nKeys[field]),
        parent: 'list.sort',
        icon: ArrowUpDown,
        run: () => conversationStore.setListSortField(field)
      }))
    )

    return commands
  })
}
