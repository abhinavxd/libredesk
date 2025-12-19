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
      <!-- Generate Response Button -->
      <Button
        v-if="showGenerateButton"
        variant="outline"
        size="sm"
        class="h-8 px-3 text-xs"
        @click="handleGenerate"
        :disabled="isGenerating"
      >
        <Sparkles class="h-3.5 w-3.5 mr-1.5" :class="{ 'animate-pulse': isGenerating }" />
        {{ isGenerating ? 'Generating...' : 'Generate Response' }}
      </Button>
    </div>
    <Button class="h-8 w-6 px-8" @click="handleSend" :disabled="!enableSend" :isLoading="isSending" v-if="showSendButton">
      {{ $t('globals.messages.send') }}
    </Button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@/components/ui/button'
import { Toggle } from '@/components/ui/toggle'
import { Paperclip, Smile, Sparkles } from 'lucide-vue-next'
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'

const attachmentInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['emojiSelect', 'generateResponse'])

// Using defineProps for props that don't need two-way binding
defineProps({
  isFullscreen: Boolean,
  isSending: Boolean,
  isGenerating: {
    type: Boolean,
    default: false
  },
  enableSend: Boolean,
  handleSend: Function,
  showSendButton: {
    type: Boolean,
    default: true
  },
  showGenerateButton: {
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

function handleGenerate() {
  emit('generateResponse')
}
</script>
