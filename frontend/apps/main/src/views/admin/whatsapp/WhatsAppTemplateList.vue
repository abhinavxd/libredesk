<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <AdminToolbar
          v-model:search="searchTerm"
          :searchable="whatsappInboxes.length > 0"
          class="mb-5"
        >
          <template #left>
            <div class="flex items-center gap-3">
              <label class="text-sm text-muted-foreground">{{ i18n.ngettext('Inbox', 'Inboxes', 1) }}</label>
              <Select v-model="selectedInboxID" @update:model-value="onInboxChange">
                <SelectTrigger class="w-48 sm:w-60">
                  <SelectValue :placeholder="i18n._('Select inbox')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="inb in whatsappInboxes" :key="inb.id" :value="inb.id">
                    {{ inb.name }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </template>
          <template #actions>
            <Button
              type="button"
              variant="outline"
              :disabled="!selectedInboxID || isSyncing"
              @click="onSync"
            >
              <RefreshCw class="size-4" :class="{ 'animate-spin': isSyncing }" />
              {{ i18n._('Sync from Meta') }}
            </Button>
            <Button type="button" :disabled="!selectedInboxID" @click="onNew">
              {{ i18n._('New template') }}
            </Button>
          </template>
        </AdminToolbar>

        <div v-if="!whatsappInboxes.length" class="box p-6 text-sm text-muted-foreground">
          {{ i18n._('Create a WhatsApp inbox first to author templates.') }}
        </div>
        <DataTable
          v-else
          v-model:search="searchTerm"
          :searchable="false"
          :columns="columns"
          :data="templates"
          :loading="isLoading"
        />
      </LoadingOverlay>

      <AlertDialog :open="deleteAlertOpen" @update:open="deleteAlertOpen = $event">
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{{ i18n._('Are you absolutely sure?') }}</AlertDialogTitle>
            <AlertDialogDescription>
              {{ i18n._('Delete this template on libredesk and Meta?') }}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
            <AlertDialogAction @click="handleDelete">
              {{ i18n._('Delete') }}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </template>

    <template #help>
      <div class="space-y-4">
        <div class="space-y-1">
          <p class="text-sm font-medium text-foreground">
            {{ i18n._('WhatsApp templates') }}
          </p>
          <p class="ui-description">
            {{ i18n._('Message templates are pre-approved formats required to start a WhatsApp conversation or reply outside the 24-hour window. Create one here, or sync templates you already approved in Meta.') }}
          </p>
          <a
            href="https://developers.facebook.com/docs/whatsapp/business-management-api/message-templates"
            target="_blank"
            rel="noopener noreferrer"
            class="link-style text-sm"
          >
            {{ i18n._('Learn more') }}
          </a>
        </div>
      </div>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { computed, h, onMounted, ref } from 'vue'
import { RouterLink, useRouter, useRoute } from 'vue-router'
import TableDate from '@main/components/table/TableDate.vue'
import { useI18n } from '@shared-ui/i18n'
import { Info, RefreshCw, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import DataTable from '@main/components/datatable/DataTable.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { useInboxStore } from '@main/stores/inbox'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { canEditWhatsAppTemplate } from '@/features/conversation/whatsappTemplate.js'

const i18n = useI18n()
const router = useRouter()
const route = useRoute()
const emitter = useEmitter()
const inboxStore = useInboxStore()

const isLoading = ref(false)
const isSyncing = ref(false)
const templates = ref([])
const searchTerm = ref('')
const selectedInboxID = ref(null)
const deleteAlertOpen = ref(false)
const deleteID = ref(null)

const whatsappInboxes = computed(() =>
  inboxStore.inboxes.filter((inb) => inb.channel === 'whatsapp')
)

const RESERVED_NAME_PREFIX = 'libredesk_csat_'

const STATUS_BADGE_VARIANT = {
  APPROVED: 'success',
  PENDING: 'secondary',
  REJECTED: 'destructive',
  DISABLED: 'secondary',
  PAUSED: 'secondary'
}

const columns = [
  {
    accessorKey: 'name',
    header: () => h('div', i18n._('{count, plural, one {Name} other {Names}}')),
    cell: ({ row }) => {
      const name = row.getValue('name')
      const children = [
        canEditWhatsAppTemplate(row.original)
          ? h(
              RouterLink,
              {
                to: { name: 'whatsapp-template-edit', params: { id: row.original.id } },
                class: 'font-mono text-xs hover:underline'
              },
              () => name
            )
          : h(
              'span',
              {
                class: 'font-mono text-xs',
                title: i18n._('This template cannot be edited here in its current state or format. Sync templates from Meta, or edit it in WhatsApp Manager.')
              },
              name
            )
      ]
      if ((row.original.name || '').startsWith(RESERVED_NAME_PREFIX)) {
        children.push(
          h(Tooltip, null, {
            default: () => [
              h(TooltipTrigger, { asChild: true }, () =>
                h(Info, { class: 'size-3.5 text-muted-foreground cursor-help' })
              ),
              h(TooltipContent, { class: 'max-w-xs' }, () =>
                i18n._('This template is reserved for CSAT surveys and can\'\'t be deleted. Edit it in the inbox\'\'s CSAT settings.')
              )
            ]
          })
        )
      }
      return h('div', { class: 'flex items-center gap-1.5' }, children)
    }
  },
  {
    accessorKey: 'language',
    header: () => h('div', i18n._('{count, plural, one {Language} other {Languages}}'))
  },
  {
    accessorKey: 'category',
    header: () => h('div', i18n._('Category')),
    cell: ({ row }) => h(Badge, { variant: 'outline' }, () => row.getValue('category'))
  },
  {
    accessorKey: 'status',
    header: () => h('div', i18n._('{count, plural, one {Status} other {Statuses}}')),
    cell: ({ row }) => {
      const status = row.getValue('status')
      return h(
        Badge,
        {
          variant: STATUS_BADGE_VARIANT[status] || 'secondary',
          title: row.original.rejection_reason || undefined
        },
        () => status
      )
    }
  },
  {
    accessorKey: 'updated_at',
    meta: { compact: true },
    header: () => h('div', i18n._('Updated at')),
    cell: ({ row }) => h(TableDate, { value: row.getValue('updated_at') })
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      if ((row.original.name || '').startsWith(RESERVED_NAME_PREFIX)) return null
      return h(
        Button,
        {
          variant: 'ghost',
          size: 'sm',
          type: 'button',
          'aria-label': i18n._('Delete'),
          onClick: () => confirmDelete(row.original.id)
        },
        () => h(Trash2, { class: 'size-4' })
      )
    }
  }
]

let fetchToken = 0
const fetchTemplates = async () => {
  const token = ++fetchToken
  if (!selectedInboxID.value) {
    templates.value = []
    isLoading.value = false
    return
  }
  try {
    isLoading.value = true
    const resp = await api.getWhatsAppTemplates(selectedInboxID.value)
    if (token !== fetchToken) return
    templates.value = resp.data.data
  } catch (error) {
    if (token !== fetchToken) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (token === fetchToken) isLoading.value = false
  }
}

const onInboxChange = (id) => {
  searchTerm.value = ''
  selectedInboxID.value = id
  router.replace({ query: { ...route.query, inbox_id: id } })
  fetchTemplates()
}

const onNew = () => {
  router.push({ name: 'whatsapp-template-new', query: { inbox_id: selectedInboxID.value } })
}

const onSync = async () => {
  if (!selectedInboxID.value) return
  try {
    isSyncing.value = true
    const resp = await api.syncWhatsAppTemplates(selectedInboxID.value)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Synced {count} template(s) from Meta.', { count: resp.data.data?.synced || 0 })
    })
    await fetchTemplates()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSyncing.value = false
  }
}

const confirmDelete = (id) => {
  deleteID.value = id
  deleteAlertOpen.value = true
}

const handleDelete = async () => {
  const id = deleteID.value
  deleteAlertOpen.value = false
  if (!id) return
  try {
    await api.deleteWhatsAppTemplate(id)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Deleted successfully')
    })
    await fetchTemplates()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(async () => {
  await inboxStore.fetchInboxes()
  const queryInboxID = Number(route.query.inbox_id)
  if (queryInboxID && whatsappInboxes.value.some((i) => i.id === queryInboxID)) {
    selectedInboxID.value = queryInboxID
  } else if (whatsappInboxes.value.length > 0) {
    selectedInboxID.value = whatsappInboxes.value[0].id
  }
  if (selectedInboxID.value) {
    await fetchTemplates()
  }
})
</script>
