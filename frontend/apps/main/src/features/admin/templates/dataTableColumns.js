import { h } from 'vue'
import { RouterLink } from 'vue-router'
import dropdown from './dataTableDropdown.vue'
import { formatDateTime } from '@shared-ui/utils/datetime.js'

export const createOutgoingEmailTableColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' },
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
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.default')),
    cell: ({ row }) => {
      const isDefault = row.getValue('is_default')

      return h('div', { class: 'text-center' }, [
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
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, formatDateTime(row.getValue('created_at')))
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
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' },
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
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, formatDateTime(row.getValue('created_at')))
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
