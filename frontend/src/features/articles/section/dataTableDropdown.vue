<template>
  <!--- Dropdown menu for section actions -->
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
      <SectionForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-10">
            <Button type="submit"> {{ t('globals.messages.save') }} </Button>
          </DialogFooter>
        </template>
      </SectionForm>
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
        <AlertDialogAction @click="deleteSection">{{
          t('globals.messages.delete')
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
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
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
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api/index.js'
import SectionForm from './SectionForm.vue'

const { t } = useI18n()
const dialogOpen = ref(false)
const alertOpen = ref(false)
const emitter = useEmitter()

const props = defineProps({
  section: {
    type: Object,
    required: true,
    default: () => ({
      id: '',
      name: '',
      category_id: '',
      category_name: ''
    })
  }
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateArticleSection(props.section.id, values)
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    description: t('globals.messages.updatedSuccessfully', { name: t('globals.terms.tag') })
  })
  dialogOpen.value = false
  emitRefreshSectionsList()
})

const openAlertDialog = () => {
  alertOpen.value = true
}

const deleteSection = async () => {
  await api.deleteArticleSection(props.section.id)
  dialogOpen.value = false
  emitRefreshSectionsList()
}

const emitRefreshSectionsList = () => {
  emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'article_section'
  })
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.section,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>
