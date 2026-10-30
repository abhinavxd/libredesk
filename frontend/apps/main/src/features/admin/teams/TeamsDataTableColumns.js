import { h } from 'vue'
import { RouterLink } from 'vue-router'
import TeamDataTableDropdown from '@/features/admin/teams/TeamDataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'
import { getI18n } from '@/i18n'

const t = () => getI18n().global.t

export const columns = [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, t()('globals.terms.name', 1))
    },
    cell: function ({ row }) {
      const emoji = row.original.emoji
      return h('div', null,
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
    meta: { compact: true },
    header: function () {
      return h('div', null, t()('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('created_at') })
    }
  },
  {
    accessorKey: 'updated_at',
    meta: { compact: true },
    header: function () {
      return h('div', null, t()('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('updated_at') })
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
