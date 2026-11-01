
import { defineMessage as msg } from '@lingui/core/macro'


import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '@/constants/conversation.js'

const DEFAULT_STATUS_MESSAGES = {
  Open: msg({ message: 'Open' }),
  Snoozed: msg({ message: 'Snoozed' }),
  Resolved: msg({ message: 'Resolved' }),
  Closed: msg({ message: 'Closed' })
}

export const createColumns = (i18n, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Name} other {Names}}'))
    },
    cell: function ({ row }) {
      const name = row.getValue('name')
      const isDefault = CONVERSATION_DEFAULT_STATUSES_LIST.includes(name)
      const label = isDefault ? i18n._(DEFAULT_STATUS_MESSAGES[name]) : name
      return h('div', null,
        onEdit && !isDefault
          ? h('span', {
              class: 'text-foreground font-medium hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, label)
          : label
      )
    }
  },
  {
    accessorKey: 'category',
    meta: { compact: true },
    header: function () {
      return h('div', null, i18n._('Category'))
    },
    cell: function ({ row }) {
      const category = row.getValue('category')
      const labels = {
        open: i18n._('Open'),
        waiting: i18n._('Waiting'),
        resolved: i18n._('Resolved')
      }
      return h('div', null, labels[category] || category)
    }
  },
  {
    accessorKey: 'created_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, i18n._('Created at'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('created_at') })
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const status = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          status
        })
      )
    }
  }
]
