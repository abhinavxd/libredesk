import { h } from 'vue'
import TableDate from '@main/components/table/TableDate.vue'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import HelpCenterDropdown from './HelpCenterDropdown.vue'

export const createHelpCenterColumns = (t, { onOpen, onEdit, onDelete, onToggle } = {}) => [
  {
    accessorKey: 'name',
    header: () => h('div', null, t('globals.terms.name')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        h(
          'button',
          {
            type: 'button',
            class: 'text-left text-foreground font-medium hover:underline cursor-pointer',
            onClick: () => onOpen?.(row.original)
          },
          row.getValue('name')
        )
      )
  },
  {
    accessorKey: 'slug',
    header: () => h('div', null, t('globals.terms.slug')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        h(Badge, { variant: 'secondary', class: 'font-normal' }, () => `/${row.getValue('slug')}`)
      )
  },
  {
    accessorKey: 'is_active',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, t('globals.terms.status')),
    cell: ({ row }) =>
      h(
        'div',
        null,
        h(Badge, { variant: row.getValue('is_active') ? 'success' : 'secondary' }, () =>
          row.getValue('is_active') ? t('globals.terms.active') : t('globals.terms.paused')
        )
      )
  },
  {
    accessorKey: 'updated_at',
    meta: { compact: true },
    enableGlobalFilter: false,
    header: () => h('div', null, t('globals.terms.updatedAt')),
    cell: ({ row }) =>
      h(TableDate, { value: row.getValue('updated_at') })
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
