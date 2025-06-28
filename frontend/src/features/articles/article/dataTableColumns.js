import { h } from 'vue'
import ArticleDataTableDropDown from './dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'title',
    header: function () {
      return h('div', { class: 'text-left' }, t('globals.terms.title'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-left font-medium' }, row.getValue('title'))
    }
  },
  {
    accessorKey: 'section_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.section'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, row.getValue('section_name') || '-')
    }
  },
  {
    accessorKey: 'category_name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.category'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, row.getValue('category_name') || '-')
    }
  },
  {
    accessorKey: 'is_published',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.status'))
    },
    cell: function ({ row }) {
      const isPublished = row.getValue('is_published')
      const statusClass = isPublished
        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
        : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
      const statusText = isPublished ? t('globals.terms.published') : t('globals.terms.draft')

      return h(
        'div',
        { class: 'text-center' },
        h(
          'span',
          { class: `px-2 py-1 rounded-full text-xs font-medium ${statusClass}` },
          statusText
        )
      )
    }
  },
  {
    accessorKey: 'published_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.publishedAt'))
    },
    cell: function ({ row }) {
      const publishedAt = row.getValue('published_at')
      return h(
        'div',
        { class: 'text-center' },
        publishedAt ? format(new Date(publishedAt), 'PPpp') : '-'
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const article = row.original
      return h(
        'div',
        { class: 'relative' },
        h(ArticleDataTableDropDown, {
          article
        })
      )
    }
  }
]
