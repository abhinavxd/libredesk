import { h } from 'vue'
import OrganizationDataTableDropDown from '@/features/admin/organizations/dataTableDropdown.vue'
import { format } from 'date-fns'

export const createColumns = (t) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-left' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-left font-medium' }, row.getValue('name'))
    }
  },
  {
    accessorKey: 'website',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.website'))
    },
    cell: function ({ row }) {
      const website = row.getValue('website')
      if (!website) {
        return h('div', { class: 'text-center text-gray-400' }, '-')
      }
      return h(
        'a',
        {
          class: 'text-center text-blue-600 hover:text-blue-800 hover:underline',
          href: website,
          target: '_blank',
          rel: 'noopener noreferrer',
          onClick: (e) => e.stopPropagation()
        },
        website
      )
    }
  },
  {
    accessorKey: 'email_domain',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.emailDomain'))
    },
    cell: function ({ row }) {
      const domain = row.getValue('email_domain')
      return h('div', { class: 'text-center' }, domain || '-')
    }
  },
  {
    accessorKey: 'phone',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.phone'))
    },
    cell: function ({ row }) {
      const phone = row.getValue('phone')
      return h('div', { class: 'text-center' }, phone || '-')
    }
  },
  {
    accessorKey: 'contact_count',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.contacts'))
    },
    cell: function ({ row }) {
      const count = row.getValue('contact_count') || 0
      return h('div', { class: 'text-center' }, count.toString())
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const organization = row.original
      return h(
        'div',
        { class: 'relative' },
        h(OrganizationDataTableDropDown, {
          organization
        })
      )
    }
  }
]
