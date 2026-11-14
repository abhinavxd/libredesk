import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import TableDate from '@main/components/table/TableDate.vue'

export const createOutgoingEmailTableColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', null,
        h(RouterLink,
          {
            to: { name: 'edit-template', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },
  {
    accessorKey: 'is_default',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, t('globals.terms.default')),
    cell: ({ row }) => {
      const isDefault = row.getValue('is_default')

      return h('div', null, [
        h('input', {
          type: 'checkbox',
          checked: isDefault,
          disabled: true
        })
      ])
    }
  },
  {
    accessorKey: 'created_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, t('globals.terms.createdAt'))
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
      const template = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          template
        })
      )
    }
  }
]


export const createEmailNotificationTableColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', null,
        h(RouterLink,
          {
            to: { name: 'edit-template', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      )
    }
  },

  {
    accessorKey: 'created_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: function () {
      return h('div', null, t('globals.terms.createdAt'))
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
      const template = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          template
        })
      )
    }
  }
]
