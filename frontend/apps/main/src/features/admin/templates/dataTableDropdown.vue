<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editTemplate(props.template.id)">{{
        $t('globals.messages.edit')
      }}</DropdownMenuItem>
      <DropdownMenuItem
        @click="() => (alertOpen = true)"
        v-if="props.template.type !== 'email_notification'"
      >
        {{ $t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('globals.messages.deletionConfirmation', {
              name: $t('globals.terms.template').toLowerCase()
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>
          {{ $t('globals.messages.cancel') }}
        </AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">
          {{ $t('globals.messages.delete') }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
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
import { Button } from '@shared-ui/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../../../api'

const router = useRouter()
const emitter = useEmitter()
const alertOpen = ref(false)

const props = defineProps({
  template: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

const editTemplate = (id) => {
  router.push({ path: `/admin/templates/${id}/edit` })
}

const handleDelete = async () => {
  try {
    await api.deleteTemplate(props.template.id)
    alertOpen.value = false
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'templates'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
