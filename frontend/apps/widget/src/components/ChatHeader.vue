<template>
  <div class="flex items-center p-2 border-b border-border bg-background gap-3 relative">
    <div class="flex items-center gap-2 justify-start">
      <Button
        type="button"
        @click="$emit('goBack')"
        variant="ghost"
        size="sm"
        :aria-label="$t('globals.messages.goBack')"
      >
        <ArrowLeft />
      </Button>
      <ChatTitle />
    </div>
    <div class="flex items-center gap-2 ml-auto" :class="{ 'mr-12': widgetStore.isMobileFullScreen }">
      <DropdownMenu v-if="!widgetStore.isMobileFullScreen || canDownloadTranscript">
        <DropdownMenuTrigger as-child>
          <Button type="button" variant="ghost" size="sm" :aria-label="$t('globals.terms.more')">
            <Ellipsis class="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            v-if="!widgetStore.isMobileFullScreen"
            @select="widgetStore.toggleExpand"
          >
            <Minimize2 v-if="widgetStore.isExpanded" class="mr-2 h-4 w-4" aria-hidden="true" />
            <Maximize2 v-else class="mr-2 h-4 w-4" aria-hidden="true" />
            {{ widgetStore.isExpanded ? $t('globals.terms.collapse') : $t('globals.terms.expand') }}
          </DropdownMenuItem>
          <DropdownMenuItem
            v-if="canDownloadTranscript"
            :disabled="downloading"
            @select="downloadTranscript"
          >
            <Download class="mr-2 h-4 w-4" aria-hidden="true" />
            {{ $t('conversation.downloadTranscript') }}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { ArrowLeft, Ellipsis, Maximize2, Minimize2, Download } from 'lucide-vue-next'
import { computed, ref } from 'vue'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import { useChatStore } from '@widget/store/chat.js'
import api from '@widget/api/index.js'
import { downloadBlobResponse, parseBlobError } from '@shared-ui/utils/file'
import { handleHTTPError } from '@shared-ui/utils/http'
import ChatTitle from './ChatTitle.vue'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()
const chatStore = useChatStore()
const downloading = ref(false)
const canDownloadTranscript = computed(
  () => widgetStore.config.features?.transcript && !!chatStore.currentConversation?.uuid
)

const emit = defineEmits(['goBack', 'error'])
const downloadTranscript = async () => {
  if (downloading.value) return
  downloading.value = true
  try {
    const conversation = chatStore.currentConversation
    downloadBlobResponse(
      await api.downloadTranscript(conversation.uuid),
      `transcript-${conversation.reference_number || conversation.uuid}.txt`
    )
  } catch (error) {
    emit('error', handleHTTPError(await parseBlobError(error)).message)
  } finally {
    downloading.value = false
  }
}
</script>
