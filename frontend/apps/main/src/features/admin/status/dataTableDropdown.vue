<template>
  <Dialog v-model:open="dialogOpen">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button
          variant="ghost"
          class="w-8 h-8 p-0"
          v-if="!CONVERSATION_DEFAULT_STATUSES_LIST.includes(props.status.name)"
        >
          <span class="sr-only"></span>
          <MoreHorizontal class="w-4 h-4" />
        </Button>
        <div v-else class="w-8 h-8 p-0 invisible"></div>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DialogTrigger as-child>
          <DropdownMenuItem> {{ $t('globals.messages.edit') }} </DropdownMenuItem>
        </DialogTrigger>
        <DropdownMenuItem @click="() => (alertOpen = true)">
          {{ $t('globals.messages.delete') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>
          {{
            $t('globals.messages.edit', {
              name: $t('globals.terms.status')
            })
          }}
        </DialogTitle>
        <DialogDescription>
          {{ $t('admin.conversationStatus.name.description') }}
        </DialogDescription>
      </DialogHeader>
      <StatusForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-10">
            <Button type="submit" :isLoading="isLoading" :disabled="isLoading">{{
              $t('globals.messages.save')
            }}</Button>
          </DialogFooter>
        </template>
      </StatusForm>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle> {{ $t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{
            $t('globals.messages.deletionConfirmation', {
              name: $t('globals.terms.status').toLowerCase()
            })
          }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">{{
          $t('globals.messages.delete')
        }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { watch, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@shared-ui/components/ui/dropdown-menu/index.js'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog/index.js'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import StatusForm from './StatusForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '../../../constants/conversation.js'
import { useEmitter } from '../../../composables/useEmitter.js'
import { handleHTTPError } from '../../../utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api/index.js'

const { t } = useI18n()
const isLoading = ref(false)
const dialogOpen = ref(false)
const alertOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  status: {
    type: Object,
    required: true
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await api.updateStatus(props.status.id, values)
    dialogOpen.value = false
    emitRefreshStatusList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const handleDelete = async () => {
  isLoading.value = true
  try {
    await api.deleteStatus(props.status.id)
    alertOpen.value = false
    emitRefreshStatusList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const emitRefreshStatusList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'status'
  })
}

watch(
  () => props.status,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
