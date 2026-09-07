import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowUpDown, CheckSquare, CircleDot, Filter } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useBulkActionPermissions } from '@main/composables/useBulkActionPermissions'
import { SECTIONS } from '../sections'

const SORT_FIELDS = [
  'oldest',
  'newest',
  'started_first',
  'started_last',
  'waiting_longest',
  'next_sla_target',
  'priority_first'
]

export const isConversationListRoute = (route) =>
  route.path.startsWith('/inboxes') && route.name !== 'search'

export function useListCommands() {
  const route = useRoute()
  const { t } = useI18n()
  const conversationStore = useConversationStore()
  const { canBulkAct } = useBulkActionPermissions()

  const section = SECTIONS.LIST

  return computed(() => {
    if (!isConversationListRoute(route)) return []
    const commands = []

    // Views are pre-filtered, the list header hides the status filter there too.
    if (!route.params.viewID) {
      commands.push(
        {
          id: 'list.status',
          label: t('command.filterByStatus'),
          keywords: [t('globals.terms.filter'), t('globals.terms.status')],
          section,
          icon: Filter,
          group: true
        },
        ...conversationStore.statusOptions
          .filter((status) => status.label !== conversationStore.getListStatus)
          .map((status) => ({
            id: `list.status.${status.value}`,
            label: status.label,
            parent: 'list.status',
            icon: CircleDot,
            run: () => conversationStore.setListStatus(status.label)
          }))
      )
    }

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

    if (canBulkAct.value && conversationStore.conversationsList.length) {
      commands.push({
        id: 'list.select-all',
        label: t('conversation.bulkActions.selectAll'),
        keywords: ['bulk'],
        section,
        icon: CheckSquare,
        run: () => conversationStore.selectAll()
      })
    }

    return commands
  })
}
