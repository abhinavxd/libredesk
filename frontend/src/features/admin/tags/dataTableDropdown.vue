<template>
  <Dialog v-model:open="dialogOpen">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" class="w-8 h-8 p-0">
          <span class="sr-only">Open menu</span>
          <MoreHorizontal class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DialogTrigger as-child>
          <DropdownMenuItem> Edit </DropdownMenuItem>
        </DialogTrigger>
        <DropdownMenuItem @click="openAlertDialog"> Delete </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Edit tag</DialogTitle>
        <DialogDescription> Change the tag name. Click save when you're done. </DialogDescription>
      </DialogHeader>
      <TagsForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-10">
            <Button type="submit"> Save changes </Button>
          </DialogFooter>
        </template>
      </TagsForm>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
        <AlertDialogDescription>
          This action cannot be undone. This will permanently delete the tag and
          <strong>remove it from all conversations.</strong>
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction @click="deleteTag">Delete</AlertDialogAction>
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
import { formSchema } from './formSchema.js'
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
import TagsForm from './TagsForm.vue'
import api from '@/api/index.js'

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
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateTag(props.tag.id, values)
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    title: 'Success',
    description: 'Tag updated successfully'
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
