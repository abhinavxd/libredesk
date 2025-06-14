import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'
import { Badge } from '@/components/ui/badge'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('name'))
    }
  },
  {
    accessorKey: 'url',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.url'))
    },
    cell: function ({ row }) {
      const url = row.getValue('url')
      return h('div', { class: 'text-center font-mono text-sm max-w-sm truncate' }, url)
    }
  },
  {
    accessorKey: 'events',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.event', 2))
    },
    cell: function ({ row }) {
      const events = row.getValue('events')
      return h('div', { class: 'text-center' }, [
        h(
          Badge,
          { variant: 'secondary', class: 'text-xs' },
          () => `${events.length} ${t('globals.terms.event', 2)}`
        )
      ])
    }
  },
  {
    accessorKey: 'is_active',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.status')),
    cell: ({ row }) => {
      const isActive = row.getValue('is_active')
      return h('div', { class: 'text-center' }, [
        h(
          Badge,
          {
            variant: isActive ? 'default' : 'secondary',
            class: 'text-xs'
          },
          () => isActive ? t('globals.terms.active') : t('globals.terms.inactive')
        )
      ])
    }
  },
  {
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center text-sm' }, format(row.getValue('updated_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const webhook = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          webhook
        })
      )
    }
  }
]
