import { h } from 'vue'
import { RouterLink } from 'vue-router'
import TeamDataTableDropdown from '@/features/admin/teams/TeamDataTableDropdown.vue'
import { format } from 'date-fns'
import { getI18n } from '@/i18n'

const t = () => getI18n().global.t

export const columns = [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t()('globals.terms.name', 1))
    },
    cell: function ({ row }) {
      const emoji = row.original.emoji
      return h('div', { class: 'text-center' },
        h(RouterLink,
          {
            to: { name: 'edit-team', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => [emoji ? `${emoji} ` : '', row.getValue('name')]
        )
      )
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t()('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        format(row.getValue('created_at'), 'PPpp')
      )
    }
  },
  {
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, t()('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        format(row.getValue('updated_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const team = row.original
      return h(
        'div',
        { class: 'relative' },
        h(TeamDataTableDropdown, {
          team
        })
      )
    }
  }
]
