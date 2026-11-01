<template>
  <div class="flex items-center gap-1" @keydown.capture="emojiPickerPointerOpened = false">
    <input
      type="file"
      class="hidden"
      ref="fileInput"
      @change="handleFileUpload"
      :accept="fileUploadEnabled ? '*/*' : ''"
      :disabled="!fileUploadEnabled"
    />

    <Button
      type="button"
      v-if="fileUploadEnabled && canUploadFiles"
      @click="triggerFileUpload"
      :disabled="uploading || disabled"
      :aria-label="i18n._('Attach file')"
      variant="ghost"
      size="sm"
      class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
    >
      <Paperclip class="h-4 w-4 text-muted-foreground" aria-hidden="true" />
    </Button>

    <div class="relative">
      <Button
        type="button"
        v-if="emojiEnabled"
        ref="emojiTriggerRef"
        @click="toggleEmojiPicker"
        :disabled="disabled"
        :class="{ 'bg-muted': isEmojiPickerVisible }"
        :title="i18n._('Add emoji')"
        :aria-label="i18n._('Add emoji')"
        variant="ghost"
        size="sm"
        class="h-8 w-8 p-0 hover:bg-muted/50 border-0"
      >
        <Smile class="h-4 w-4 text-muted-foreground" aria-hidden="true" />
      </Button>

      <Transition
        :css="emojiPickerPointerOpened"
        enter-active-class="transition-[opacity,transform] [transition-duration:125ms] ease-out motion-reduce:duration-100"
        enter-from-class="opacity-0 scale-[0.98] motion-reduce:scale-100"
        enter-to-class="opacity-100 scale-100"
        leave-active-class="transition-[opacity,transform] duration-100 ease-out"
        leave-from-class="opacity-100 scale-100"
        leave-to-class="opacity-0 scale-[0.98] motion-reduce:scale-100"
      >
        <EmojiPicker
          v-if="isEmojiPickerVisible && emojiEnabled"
          ref="emojiPickerRef"
          :native="true"
          @select="onSelectEmoji"
          class="origin-bottom-left absolute bottom-12 left-0 z-50"
        />
      </Transition>
    </div>
  </div>
</template>

<script setup>
import { ref, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Smile, Paperclip } from 'lucide-vue-next'

const EmojiPicker = defineAsyncComponent(async () => {
  const [mod] = await Promise.all([import('vue3-emoji-picker'), import('vue3-emoji-picker/css')])
  return mod.default
})

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
  },
  canUploadFiles: {
    type: Boolean,
    default: true
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['fileUpload', 'emojiSelect'])

const fileInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerPointerOpened = ref(false)
const emojiPickerRef = ref(null)
const emojiTriggerRef = ref(null)

onClickOutside(
  emojiPickerRef,
  () => {
    isEmojiPickerVisible.value = false
  },
  { ignore: [emojiTriggerRef] }
)

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

const toggleEmojiPicker = (event) => {
  emojiPickerPointerOpened.value = event.detail > 0
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

const onSelectEmoji = (emoji) => {
  emit('emojiSelect', emoji.i)
  isEmojiPickerVisible.value = false
}
</script>
