<template>
  <div
    class="relative flex h-14 justify-between"
    :class="{ 'items-end': isFullscreen, 'items-center': !isFullscreen }"
  >
    <EmojiPicker
      ref="emojiPickerRef"
      :native="true"
      @select="onSelectEmoji"
      class="absolute bottom-14 left-0 md:left-14 z-20"
      v-if="isEmojiPickerVisible"
    />
    <div class="flex items-center gap-1">
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
      <Tooltip>
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :aria-label="$t('globals.messages.attachFile')"
            @click="triggerFileUpload"
            :pressed="false"
          >
            <Paperclip class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.messages.attachFile') }}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :aria-label="$t('globals.messages.addEmoji')"
            @click="toggleEmojiPicker"
            :pressed="isEmojiPickerVisible"
          >
            <Smile class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('globals.messages.addEmoji') }}</TooltipContent>
      </Tooltip>
      <Tooltip v-if="showGenerateReply">
        <TooltipTrigger as-child>
          <Toggle
            :class="ICON_BUTTON_CLASS"
            variant="outline"
            :pressed="false"
            :disabled="isGenerating"
            :aria-label="$t('replyBox.generateReply')"
            @click="emit('generateReply')"
          >
            <Loader2 v-if="isGenerating" class="h-4 w-4 animate-spin" />
            <Sparkles v-else class="h-4 w-4" />
          </Toggle>
        </TooltipTrigger>
        <TooltipContent>{{ $t('replyBox.generateReply') }}</TooltipContent>
      </Tooltip>
    </div>
    <div class="flex items-center rounded-md shadow-sm">
      <Button
        class="h-8 max-md:h-11 px-4 rounded-r-none"
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
            class="h-8 max-md:h-11 px-2 rounded-l-none border-l border-primary-foreground/30 [&[data-state=open]>svg]:rotate-180"
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
const ICON_BUTTON_CLASS =
  'border-border/70 bg-background px-2 py-2 text-muted-foreground shadow-none max-md:min-h-11 max-md:min-w-11'

import { ref, defineAsyncComponent } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@shared-ui/components/ui/button'
import { Toggle } from '@shared-ui/components/ui/toggle'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { Paperclip, Smile, ChevronDownIcon, Sparkles, Loader2 } from 'lucide-vue-next'
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
  const [mod] = await Promise.all([import('vue3-emoji-picker'), import('vue3-emoji-picker/css')])
  return mod.default
})

const attachmentInput = ref(null)
// const inlineImageInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['emojiSelect', 'generateReply'])

// Using defineProps for props that don't need two-way binding
defineProps({
  isFullscreen: Boolean,
  isSending: Boolean,
  isGenerating: Boolean,
  enableSend: Boolean,
  handleSend: Function,
  handleSendAndSetStatus: Function,
  showSendButton: {
    type: Boolean,
    default: true
  },
  showGenerateReply: {
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
