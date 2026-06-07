<template>
  <div
    class="flex justify-between h-14 relative"
    :class="{ 'items-end': isFullscreen, 'items-center': !isFullscreen }"
  >
    <EmojiPicker
      ref="emojiPickerRef"
      :native="true"
      @select="onSelectEmoji"
      class="absolute bottom-14 left-14"
      v-if="isEmojiPickerVisible"
    />
    <div class="flex justify-items-start gap-2">
      <!-- File inputs -->
      <input type="file" class="hidden" ref="attachmentInput" multiple @change="handleFileUpload" />
      <!-- <input
        type="file"
        class="hidden"
        ref="inlineImageInput"
        accept="image/*"
        @change="handleInlineImageUpload"
      /> -->
      <!-- Editor buttons -->
      <Toggle
        class="px-2 py-2 border-0"
        variant="outline"
        @click="triggerFileUpload"
        :pressed="false"
      >
        <Paperclip class="h-4 w-4" />
      </Toggle>
      <Toggle
        class="px-2 py-2 border-0"
        variant="outline"
        @click="toggleEmojiPicker"
        :pressed="isEmojiPickerVisible"
      >
        <Smile class="h-4 w-4" />
      </Toggle>
    </div>
    <div class="flex items-center">
      <Button
        class="h-8 px-4 rounded-r-none"
        @click="handleSend"
        :disabled="!enableSend"
        :isLoading="isSending"
        v-if="showSendButton"
      >
        {{ $t('globals.messages.send') }}
      </Button>
      <DropdownMenu v-if="showSendButton">
        <DropdownMenuTrigger as-child>
          <Button
            class="h-8 px-2 rounded-l-none border-l border-primary-foreground/30 [&[data-state=open]>svg]:rotate-180"
            :disabled="!enableSend"
          >
            <ChevronDownIcon class="text-primary-foreground transition-transform" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent>
          <DropdownMenuLabel>{{ $t('replyBox.sendAndSetAs') }}</DropdownMenuLabel>
          <DropdownMenuItem
            v-for="status in conversationStore.statusOptionsNoSnooze"
            :key="status.value"
            @click="handleSendAndSetStatus(status.label)"
          >
            {{ status.label }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>
</template>

<script setup>
import { ref, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Toggle } from '@shared-ui/components/ui/toggle'
import { Paperclip, Smile, ChevronDownIcon } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuItem,
  DropdownMenuContent,
  DropdownMenuLabel
} from '@shared-ui/components/ui/dropdown-menu'
import { useConversationStore } from '@main/stores/conversation'
const conversationStore = useConversationStore()

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([
    import('vue3-emoji-picker'),
    import('vue3-emoji-picker/css'),
  ])
  return mod.default
})

const attachmentInput = ref(null)
// const inlineImageInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['emojiSelect'])

// Using defineProps for props that don't need two-way binding
defineProps({
  isFullscreen: Boolean,
  isSending: Boolean,
  enableSend: Boolean,
  handleSend: Function,
  handleSendAndSetStatus: Function,
  showSendButton: {
    type: Boolean,
    default: true
  },
  handleFileUpload: Function,
  handleInlineImageUpload: Function
})

onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  if (attachmentInput.value) {
    // Clear the value to allow the same file to be uploaded again.
    attachmentInput.value.value = ''
    attachmentInput.value.click()
  }
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

function onSelectEmoji(emoji) {
  emit('emojiSelect', emoji.i)
}
</script>
