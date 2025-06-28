<template>
  <!--- Dropdown menu for tag actions -->
  <Dialog v-model:open="dialogOpen">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" class="w-8 h-8 p-0">
          <span class="sr-only"></span>
          <MoreHorizontal class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DialogTrigger as-child>
          <DropdownMenuItem> {{ t('globals.messages.edit') }} </DropdownMenuItem>
        </DialogTrigger>
        <DropdownMenuItem @click="openAlertDialog">
          {{ t('globals.messages.delete') }}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{
          t('globals.messages.edit', {
            name: t('globals.terms.tag')
          })
        }}</DialogTitle>
        <DialogDescription> {{ t('admin.conversationTags.edit.description') }} </DialogDescription>
      </DialogHeader>
      <TagsForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-10">
            <Button type="submit"> {{ t('globals.messages.save') }} </Button>
          </DialogFooter>
        </template>
      </TagsForm>
    </DialogContent>
  </Dialog>

  <!-- Alert dialog for delete confirmation -->
  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ t('globals.messages.areYouAbsolutelySure') }}</AlertDialogTitle>
        <AlertDialogDescription>
          {{ $t('admin.tags.deleteConfirmation') }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ t('globals.messages.cancel') }}</AlertDialogCancel>
        <AlertDialogAction @click="deleteTag">{{ t('globals.messages.delete') }}</AlertDialogAction>
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
import { Button } from '@shared-ui/components/ui/button/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from './formSchema.js'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
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
import { useEmitter } from '../../../composables/useEmitter.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import TagsForm from './TagsForm.vue'
import { useI18n } from 'vue-i18n'
import api from '../../../api/index.js'

const { t } = useI18n()
const dialogOpen = ref(false)
const alertOpen = ref(false)
const emitter = useEmitter()

const props = defineProps({
  tag: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: ''
    })
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateTag(props.tag.id, values)
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    description: t('globals.messages.updatedSuccessfully', { name: t('globals.terms.tag') })
  })
  dialogOpen.value = false
  emitRefreshTagsList()
})

const openAlertDialog = () => {
  alertOpen.value = true
}

const deleteTag = async () => {
  await api.deleteTag(props.tag.id)
  dialogOpen.value = false
  emitRefreshTagsList()
}

const emitRefreshTagsList = () => {
  emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'tags'
  })
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.tag,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
