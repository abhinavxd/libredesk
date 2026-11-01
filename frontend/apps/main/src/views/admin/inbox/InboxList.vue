<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
      <template #actions>
        <router-link :to="{ name: 'new-inbox' }">
          <Button>
            {{ i18n._('New inbox') }}
          </Button>
        </router-link>
      </template>
    </AdminToolbar>
    <div>
      <DataTable
        v-model:search="searchTerm"
        :searchable="false"
        :columns="columns"
        :data="data"
        :loading="isLoading"
      />
    </div>
  </LoadingOverlay>
</template>

<script setup>
import { defineMessage as msg } from '@lingui/core/macro'

import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { onMounted, ref } from 'vue'
import { h } from 'vue'
import { RouterLink } from 'vue-router'
import InboxDataTableDropDown from '@main/features/admin/inbox/InboxDataTableDropDown.vue'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { Button } from '@shared-ui/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { TriangleAlert } from 'lucide-vue-next'
import { Badge } from '@shared-ui/components/ui/badge/index.js'
import DataTable from '@main/components/datatable/DataTable.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from '@shared-ui/i18n'
import TableDate from '@main/components/table/TableDate.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useInboxStore } from '@main/stores/inbox'
import api from '@main/api'

const searchTerm = ref('')
const i18n = useI18n()
const router = useRouter()
const route = useRoute()
const emitter = useEmitter()
const inboxStore = useInboxStore()
const isLoading = ref(false)
const data = ref([])

onMounted(async () => {
  // Handle OAuth callback messages
  const errorCode = route.query.error
  const successCode = route.query.success

  if (errorCode) {
    let msg
    if (errorCode === 'oauth_denied') {
      msg = i18n._('Authorization denied')
    } else if (errorCode === 'inbox_already_exists') {
      msg = i18n._('An inbox with this email already exists. Use Reconnect to update credentials.')
    } else if (errorCode === 'inbox_not_found') {
      msg = i18n._('No inbox found with this email to reconnect.')
    } else if (errorCode === 'email_mismatch') {
      msg = i18n._('The authorized email doesn\'\'t match this inbox. Please authorize with the correct account.')
    } else {
      msg = i18n._('Error connecting inbox')
    }
    setTimeout(() => {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: msg
      })
    }, 500)
  } else if (successCode) {
    const msg =
      successCode === 'oauth_reconnected' ? i18n._('Inbox reconnected successfully') : i18n._('Inbox connected successfully')
    setTimeout(() => {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: msg })
    }, 500)
  }

  await getInboxes()
})

const getInboxes = async () => {
  try {
    isLoading.value = true
    await inboxStore.fetchInboxes(true)
    data.value = inboxStore.inboxes
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

// Columns for the data table
const CHANNEL_LABEL_KEYS = {
  email: msg({ message: '{count, plural, one {Email} other {Emails}}', context: 'Communication channel' }),
  livechat: msg({ message: 'Live Chat' }),
  whatsapp: msg({ message: 'WhatsApp' })
}

const columns = [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', null, i18n._('{count, plural, one {Name} other {Names}}'))
    },
    cell: function ({ row }) {
      const children = [
        h(
          RouterLink,
          {
            to: { name: 'edit-inbox', params: { id: row.original.id } },
            class: 'text-foreground font-medium hover:underline'
          },
          () => row.getValue('name')
        )
      ]
      if (row.original.token_invalid) {
        children.push(
          h(Tooltip, null, {
            default: () => [
              h(TooltipTrigger, { asChild: true }, () =>
                h(TriangleAlert, { class: 'size-4 text-destructive shrink-0' })
              ),
              h(TooltipContent, null, () => i18n._('Authentication failed. This inbox can\'\'t send or receive messages until you update its credentials.'))
            ]
          })
        )
      }
      return h('div', { class: 'flex items-center gap-1.5' }, children)
    }
  },
  {
    accessorKey: 'channel',
    meta: { compact: true },
    header: function () {
      return h('div', null, i18n._('Channel'))
    },
    cell: function ({ row }) {
      const channel = row.getValue('channel')
      return h(
        'div',
        null,
        CHANNEL_LABEL_KEYS[channel] ? i18n._(CHANNEL_LABEL_KEYS[channel], 1) : channel
      )
    }
  },
  {
    accessorKey: 'enabled',
    meta: { compact: true },
    header: () => h('div', null, i18n._('{count, plural, one {Status} other {Statuses}}')),
    cell: ({ row }) => {
      const enabled = row.getValue('enabled')
      return h(
        'div',
        null,
        h(Badge, { variant: enabled ? 'success' : 'secondary' }, () =>
          enabled ? i18n._('Enabled') : i18n._('Disabled')
        )
      )
    }
  },
  {
    accessorKey: 'created_at',
    meta: { compact: true },
    header: function () {
      return h('div', null, i18n._('Created at'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('created_at') })
    }
  },
  {
    accessorKey: 'updated_at',
    meta: { compact: true },
    header: function () {
      return h('div', null, i18n._('Updated at'))
    },
    cell: function ({ row }) {
      return h(TableDate, { value: row.getValue('updated_at') })
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const inbox = row.original
      return h(
        'div',
        { class: 'relative' },
        h(InboxDataTableDropDown, {
          inbox,
          onEditInbox: (id) => handleEditInbox(id),
          onDeleteInbox: (id) => handleDeleteInbox(id),
          onToggleInbox: (id) => handleToggleInbox(id)
        })
      )
    }
  }
]

const handleEditInbox = (id) => {
  router.push({ path: `/admin/inboxes/${id}/edit` })
}

const handleDeleteInbox = async (id) => {
  await api.deleteInbox(id)
  getInboxes()
}

const handleToggleInbox = async (id) => {
  await api.toggleInbox(id)
  getInboxes()
}
</script>
