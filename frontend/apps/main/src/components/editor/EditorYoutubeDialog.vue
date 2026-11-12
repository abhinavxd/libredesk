<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ i18n._('Add YouTube video URL') }}</DialogTitle>
        <DialogDescription></DialogDescription>
      </DialogHeader>
      <form @submit.stop.prevent="setYoutubeVideo">
        <div class="grid gap-4 py-4">
          <Input
            v-model="youtubeUrl"
            type="text"
            :placeholder="i18n._('Enter URL')"
            :aria-label="i18n._('Enter URL')"
            @keydown.enter.prevent="setYoutubeVideo"
          />
          <p v-if="showError" class="text-sm text-destructive">
            {{ i18n._('Enter a valid YouTube video URL.') }}
          </p>
        </div>
        <DialogFooter>
          <Button type="submit">
            {{ i18n._('Save') }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'

const props = defineProps({
  editor: { type: Object, default: null }
})

const isOpen = ref(false)
const youtubeUrl = ref('')
const showError = ref(false)

const open = () => {
  youtubeUrl.value = ''
  showError.value = false
  isOpen.value = true
}

const setYoutubeVideo = () => {
  if (youtubeUrl.value) {
    const inserted = props.editor?.chain().focus().setYoutubeVideo({ src: youtubeUrl.value }).run()
    if (!inserted) {
      showError.value = true
      return
    }
  }
  isOpen.value = false
}

defineExpose({ open })
</script>
