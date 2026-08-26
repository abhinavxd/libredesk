import { h } from 'vue'
import dataTableDropdown from '@main/features/admin/portal/forms/dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center' },
        onEdit
          ? h(
              'span',
              {
                class: 'text-foreground font-medium hover:underline cursor-pointer',
                onClick: () => onEdit(row.original)
              },
              row.getValue('name')
            )
          : row.getValue('name')
      )
    }
  },
  {
    id: 'fields',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('admin.portalForm.fields'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, String((row.original.fields || []).length))
    }
  },
  {
    accessorKey: 'updated_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => h('div', { class: 'relative' }, h(dataTableDropdown, { form: row.original }))
  }
]
