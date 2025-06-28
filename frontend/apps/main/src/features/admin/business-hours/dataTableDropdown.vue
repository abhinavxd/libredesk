<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only"></span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="edit(props.role.id)">
        {{ t('globals.messages.edit') }}
      </DropdownMenuItem>
      <DropdownMenuItem @click="() => (alertOpen = true)">
        {{ t('globals.messages.delete') }}
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>
          {{ t('globals.messages.areYouAbsolutelySure') }}
        </AlertDialogTitle>
        <AlertDialogDescription>
          {{
            t('globals.messages.deletionConfirmation', {
              name: t('globals.terms.businessHour').toLowerCase()
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>
          {{ t('globals.messages.cancel') }}
        </AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">
          {{ t('globals.messages.delete') }}
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
import api from '../../../api'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'

const { t } = useI18n()
const router = useRouter()
const emit = useEmitter()
const alertOpen = ref(false)

const props = defineProps({
  role: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function edit(id) {
  router.push({ name: 'edit-business-hours', params: { id } })
}

async function handleDelete() {
  await api.deleteBusinessHours(props.role.id)
  alertOpen.value = false
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'business_hours'
  })
}
</script>
