import { h } from 'vue'
import { format } from 'date-fns'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import HelpCenterDropdown from './HelpCenterDropdown.vue'

export const createHelpCenterColumns = (t, { onOpen, onEdit, onDelete, onToggle } = {}) => [
  {
    accessorKey: 'name',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.name')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        h(
          'button',
          {
            type: 'button',
            class: 'text-foreground font-medium hover:underline cursor-pointer',
            onClick: () => onOpen?.(row.original)
          },
          row.getValue('name')
        )
      )
  },
  {
    accessorKey: 'slug',
    header: () => h('div', { class: 'text-center' }, t('globals.terms.slug')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        h(Badge, { variant: 'secondary', class: 'font-normal' }, () => `/${row.getValue('slug')}`)
      )
  },
  {
    accessorKey: 'is_active',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.status')),
    cell: ({ row }) =>
      h(
        'div',
        { class: 'text-center' },
        h(Badge, { variant: row.getValue('is_active') ? 'success' : 'secondary' }, () =>
          row.getValue('is_active') ? t('globals.terms.active') : t('globals.terms.paused')
        )
      )
  },
  {
    accessorKey: 'updated_at',
    enableGlobalFilter: false,
    header: () => h('div', { class: 'text-center' }, t('globals.terms.updatedAt')),
    cell: ({ row }) =>
      h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) =>
      h(
        'div',
        { class: 'relative' },
        h(HelpCenterDropdown, {
          helpCenter: row.original,
          onOpen: (hc) => onOpen?.(hc),
          onEdit: (hc) => onEdit?.(hc),
          onDelete: (hc) => onDelete?.(hc),
          onToggle: (hc) => onToggle?.(hc)
        })
      )
  }
]
