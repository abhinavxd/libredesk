<template>
  <DropdownMenu>
    <DataTableActionTrigger />
    <DropdownMenuContent>
      <DropdownMenuItem :as-child="true">
        <RouterLink :to="{ name: 'edit-webhook', params: { id: props.webhook.id } }">
          <Pencil class="mr-2 h-4 w-4" />
          {{ i18n._('Edit') }}
        </RouterLink>
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleToggle">
        <component :is="props.webhook.is_active ? PowerOff : Power" class="mr-2 h-4 w-4" />
        {{
          props.webhook.is_active ? i18n._('Disable') : i18n._('Enable')
        }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="handleTest">
        <Send class="mr-2 h-4 w-4" />
        {{ i18n._('Send test') }}
      </DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem
        @click="() => (alertOpen = true)"
        class="text-destructive focus:text-destructive"
      >
        <Trash class="mr-2 h-4 w-4" />
        {{ i18n._('Delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ i18n._('Are you absolutely sure?') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ i18n._('This action cannot be undone. This will permanently delete this webhook.') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ i18n._('Cancel') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="handleDelete">
          {{ i18n._('Delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import DataTableActionTrigger from '@main/components/datatable/DataTableActionTrigger.vue'
import { ref } from 'vue'
import { Pencil, Power, PowerOff, Send, Trash } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator
} from '@shared-ui/components/ui/dropdown-menu'
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
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'

const emit = useEmitter()
const i18n = useI18n()
const alertOpen = ref(false)

const props = defineProps({
  webhook: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: '',
      is_active: false
    })
  }
})

async function handleDelete() {
  try {
    await api.deleteWebhook(props.webhook.id)
    alertOpen.value = false
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'webhook'
    })
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Deleted successfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

async function handleToggle() {
  try {
    await api.toggleWebhook(props.webhook.id)
    emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'webhook'
    })
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: i18n._('Changes saved')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

async function handleTest() {
  try {
    await api.testWebhook(props.webhook.id)
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'success',
      description: i18n._('Webhook sent successfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
