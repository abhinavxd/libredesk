<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>
          {{ editor?.isActive('link') ? $t('editor.editLinkUrl') : $t('editor.addLinkUrl') }}
        </DialogTitle>
        <DialogDescription></DialogDescription>
      </DialogHeader>
      <form @submit.stop.prevent="setLink">
        <div class="grid gap-4 py-4">
          <Input
            v-model="linkUrl"
            type="text"
            :placeholder="$t('placeholders.enterUrl')"
            :aria-label="$t('placeholders.enterUrl')"
            @keydown.enter.prevent="setLink"
          />
          <div v-if="allowButton" class="flex items-center gap-2">
            <Checkbox
              id="link-as-button"
              :checked="linkAsButton"
              @update:checked="(v) => (linkAsButton = v)"
            />
            <Label for="link-as-button" class="font-normal cursor-pointer">
              {{ $t('editor.displayAsButton') }}
            </Label>
          </div>
        </div>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            @click="unsetLink"
            v-if="editor?.isActive('link')"
          >
            {{ $t('actions.removeLink') }}
          </Button>
          <Button type="submit">
            {{ $t('globals.messages.save') }}
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
import { Checkbox } from '@shared-ui/components/ui/checkbox/index.js'
import { Label } from '@shared-ui/components/ui/label'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'

const props = defineProps({
  editor: { type: Object, default: null },
  allowButton: { type: Boolean, default: false }
})

const isOpen = ref(false)
const linkUrl = ref('')
const linkAsButton = ref(false)

const open = () => {
  if (props.editor?.isActive('link')) {
    const attrs = props.editor.getAttributes('link')
    linkUrl.value = attrs.href
    linkAsButton.value = attrs.class === 'hc-button'
  } else {
    linkUrl.value = ''
    linkAsButton.value = false
  }
  isOpen.value = true
}

const setLink = () => {
  if (linkUrl.value) {
    // class must be set explicitly: setLink merges attrs, so omitting it would
    // keep a previously applied hc-button class.
    const attrs = {
      href: linkUrl.value,
      class: props.allowButton && linkAsButton.value ? 'hc-button' : null
    }
    props.editor?.chain().focus().extendMarkRange('link').setLink(attrs).run()
  }
  isOpen.value = false
}

const unsetLink = () => {
  props.editor?.chain().focus().unsetLink().run()
  isOpen.value = false
}

defineExpose({ open })
</script>
