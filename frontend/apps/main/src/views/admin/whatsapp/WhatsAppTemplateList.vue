<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>

  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex flex-wrap items-center justify-between gap-3 mb-5">
      <div class="flex items-center gap-3">
        <label class="text-sm text-muted-foreground">{{ $t('globals.terms.inbox') }}</label>
        <Select v-model="selectedInboxID" @update:model-value="onInboxChange">
          <SelectTrigger class="w-72">
            <SelectValue :placeholder="$t('placeholders.selectInbox')" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="inb in whatsappInboxes" :key="inb.id" :value="inb.id">
              {{ inb.name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div class="flex items-center gap-2">
        <Button variant="outline" :disabled="!selectedInboxID || isSyncing" @click="onSync">
          <RefreshCw class="size-4" :class="{ 'animate-spin': isSyncing }" />
          {{ $t('admin.whatsappTemplates.syncFromMeta') }}
        </Button>
        <Button :disabled="!selectedInboxID" @click="onNew">
          {{ $t('admin.whatsappTemplates.newTemplate') }}
        </Button>
      </div>
    </div>

    <div v-if="!whatsappInboxes.length" class="box p-6 text-sm text-muted-foreground">
      {{ $t('admin.whatsappTemplates.noInboxes') }}
    </div>
    <DataTable v-else :columns="columns" :data="templates" :loading="isLoading" />
  </LoadingOverlay>

  <AlertDialog :open="deleteAlertOpen" @update:open="deleteAlertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('admin.whatsappTemplates.confirmDelete') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, h, onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { RefreshCw, Trash2 } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Badge } from '@shared-ui/components/ui/badge'
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
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb/index.js'
import DataTable from '@main/components/datatable/DataTable.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { useInboxStore } from '@main/stores/inbox'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const emitter = useEmitter()
const inboxStore = useInboxStore()

const isLoading = ref(false)
const isSyncing = ref(false)
const templates = ref([])
const selectedInboxID = ref(null)
const deleteAlertOpen = ref(false)
const deleteID = ref(null)

const breadcrumbLinks = [
  { path: 'inbox-list', label: t('globals.terms.inbox', 2) },
  { path: '', label: t('admin.whatsappTemplates.title') }
]

const whatsappInboxes = computed(() =>
  inboxStore.inboxes.filter((inb) => inb.channel === 'whatsapp')
)

const RESERVED_NAME_PREFIX = 'libredesk_csat_'

const STATUS_BADGE_VARIANT = {
  APPROVED: 'default',
  PENDING: 'secondary',
  REJECTED: 'destructive',
  DISABLED: 'secondary',
  PAUSED: 'secondary'
}

const columns = [
  {
    accessorKey: 'name',
    header: () => h('div', t('globals.terms.name')),
    cell: ({ row }) => h('div', { class: 'font-mono text-xs' }, row.getValue('name'))
  },
  {
    accessorKey: 'language',
    header: () => h('div', t('globals.terms.language'))
  },
  {
    accessorKey: 'category',
    header: () => h('div', t('admin.whatsappTemplates.category')),
    cell: ({ row }) => h(Badge, { variant: 'outline' }, () => row.getValue('category'))
  },
  {
    accessorKey: 'status',
    header: () => h('div', t('globals.terms.status')),
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
    header: () => h('div', t('globals.terms.updatedAt')),
    cell: ({ row }) => {
      const value = row.getValue('updated_at')
      const date = value ? new Date(value) : null
      return date && !isNaN(date.getTime()) ? format(date, 'PPpp') : ''
    }
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
          onClick: () => confirmDelete(row.original.id)
        },
        () => h(Trash2, { class: 'size-4' })
      )
    }
  }
]

let fetchToken = 0
const fetchTemplates = async () => {
  if (!selectedInboxID.value) {
    templates.value = []
    return
  }
  const token = ++fetchToken
  try {
    isLoading.value = true
    const resp = await api.getWhatsAppTemplates(selectedInboxID.value)
    if (token !== fetchToken) return
    templates.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (token === fetchToken) isLoading.value = false
  }
}

const onInboxChange = (id) => {
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
      description: t('admin.whatsappTemplates.synced', { count: resp.data.data?.synced || 0 })
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
      description: t('globals.messages.deletedSuccessfully')
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
