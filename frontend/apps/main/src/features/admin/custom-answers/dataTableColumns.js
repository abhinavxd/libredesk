import { h } from 'vue'
import CustomAnswerDataTableDropDown from '@/features/admin/custom-answers/dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'question',
    header: function () {
      return h('div', { class: 'text-left' }, t('globals.terms.question'))
    },
    cell: function ({ row }) {
      const question = row.getValue('question')
      const truncated = question.length > 80 ? question.substring(0, 80) + '...' : question
      return h('div', { 
        class: 'text-left font-medium max-w-xs',
        title: question // Show full text on hover
      }, truncated)
    }
  },
  {
    accessorKey: 'answer',
    header: function () {
      return h('div', { class: 'text-left' }, t('globals.terms.answer'))
    },
    cell: function ({ row }) {
      const answer = row.getValue('answer')
      const truncated = answer.length > 100 ? answer.substring(0, 100) + '...' : answer
      return h('div', { 
        class: 'text-left font-medium max-w-sm',
        title: answer // Show full text on hover
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
      const customAnswer = row.original
      return h(
        'div',
        { class: 'relative' },
        h(CustomAnswerDataTableDropDown, {
          customAnswer
        })
      )
    }
  }
]