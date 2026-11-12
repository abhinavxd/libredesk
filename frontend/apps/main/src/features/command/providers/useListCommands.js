

import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
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
  const i18n = useI18n()
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
          label: i18n._('Filter by status'),
          keywords: [i18n._('{count, plural, one {Filter} other {Filters}}'), i18n._('{count, plural, one {Status} other {Statuses}}')],
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
        label: i18n._('Sort by'),
        keywords: ['order'],
        section,
        icon: ArrowUpDown,
        group: true
      },
      ...SORT_FIELDS.map((field) => ({
        id: `list.sort.${field}`,
        label: i18n._(conversationStore.sortFieldMessages[field]),
        parent: 'list.sort',
        icon: ArrowUpDown,
        run: () => conversationStore.setListSortField(field)
      }))
    )

    if (canBulkAct.value && conversationStore.conversationsList.length) {
      commands.push({
        id: 'list.select-all',
        label: i18n._('Select all conversations'),
        keywords: ['bulk'],
        section,
        icon: CheckSquare,
        run: () => conversationStore.selectAll()
      })
    }

    return commands
  })
}
