import { h } from 'vue'
import SnippetDataTableDropDown from '@/features/admin/snippets/dataTableDropdown.vue'
import { format } from 'date-fns'
import { getTextFromHTML } from '@/utils/strings.js'

export const createColumns = (t) => [
  {
    accessorKey: 'content',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.content'))
    },
    cell: function ({ row }) {
      const content = getTextFromHTML(row.getValue('content'))
      const truncated = content.length > 30 ? content.substring(0, 30) + '...' : content
      return h('div', {
        class: 'font-medium text-center',
        title: content
      }, truncated)
    }
  },
  {
    accessorKey: 'enabled',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.enabled'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('enabled') ? t('globals.messages.yes') : t('globals.messages.no'))
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center font-medium' },
        format(row.getValue('created_at'), 'PPpp')
      )
    }
  },
  {
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.updatedAt'))
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center font-medium' },
        format(row.getValue('updated_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const snippet = row.original
      return h(
        'div',
        { class: 'relative' },
        h(SnippetDataTableDropDown, {
          snippet
        })
      )
    }
  }
]