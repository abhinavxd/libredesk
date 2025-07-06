<template>
  <div class="flex items-center gap-1">
    <!-- File Upload Input -->
    <input
      type="file"
      class="hidden"
      ref="fileInput"
      @change="handleFileUpload"
      :accept="fileUploadEnabled ? '*/*' : ''"
      :disabled="!fileUploadEnabled"
    />

    <!-- File Upload Button -->
    <Button
      v-if="fileUploadEnabled"
      @click="triggerFileUpload"
      :disabled="uploading"
      variant="ghost"
      size="sm"
      class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
    >
      <Paperclip class="h-4 w-4 text-muted-foreground" />
    </Button>

    <!-- Emoji Picker -->
    <div class="relative">
      <Button
        v-if="emojiEnabled"
        @click="toggleEmojiPicker"
        :class="{ 'bg-muted': isEmojiPickerVisible }"
        title="Add emoji"
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
      >
        <Smile class="h-4 w-4 text-muted-foreground" />
      </Button>

      <EmojiPicker
        v-if="isEmojiPickerVisible && emojiEnabled"
        ref="emojiPickerRef"
        :native="true"
        @select="onSelectEmoji"
        class="absolute bottom-12 left-0 z-50"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import EmojiPicker from 'vue3-emoji-picker'
import { Button } from '@shared-ui/components/ui/button'
import { Smile, Paperclip } from 'lucide-vue-next'
import 'vue3-emoji-picker/css'

const props = defineProps({
  fileUploadEnabled: {
    type: Boolean,
    default: false
  },
  emojiEnabled: {
    type: Boolean,
    default: false
  },
  uploading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['fileUpload', 'emojiSelect'])

const fileInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)

// Close emoji picker when clicking outside
onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  if (fileInput.value && props.fileUploadEnabled && !props.uploading) {
    fileInput.value.value = ''
    fileInput.value.click()
  }
}

const handleFileUpload = (event) => {
  const files = event.target.files
  if (files && files.length > 0) {
    emit('fileUpload', files)
  }
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

const onSelectEmoji = (emoji) => {
  emit('emojiSelect', emoji.i)
  isEmojiPickerVisible.value = false
}
</script>
