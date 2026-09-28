<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <template v-for="attachment in attachments" :key="attachment.uuid">
      <TelegramSticker v-if="/\.tgs$/i.test(attachment.name || '')" :attachment="attachment" />
      <div
        v-else-if="channel === TELEGRAM_CHANNEL && attachment.content_type?.startsWith('audio/')"
        class="w-64 min-w-0 max-w-full space-y-2"
      >
        <audio
          :src="attachment.url"
          controls
          preload="metadata"
          class="w-full min-w-0"
          :aria-label="attachment.name"
        />
        <div class="flex min-w-0 items-center justify-between gap-2">
          <p class="truncate text-xs text-muted-foreground" :title="attachment.name">
            {{ attachment.name }}
          </p>
          <DownloadLink :url="attachment.url" class="size-8 shrink-0 max-md:size-11" />
        </div>
      </div>
      <video
        v-else-if="attachment.content_type?.startsWith('video/')"
        :src="attachment.url"
        controls
        playsinline
        preload="metadata"
        class="w-64 max-w-full max-h-64 rounded-md"
        :aria-label="attachment.name"
      />
      <BubbleAttachmentItem v-else :attachment="attachment" @preview="openLightbox" />
    </template>
  </div>

  <ImageLightbox v-model="lightboxOpen" :images="imageAttachments" :start-index="lightboxIndex" />
</template>

<script setup>
import { ref, computed } from 'vue'
import BubbleAttachmentItem from '@/features/conversation/message/attachment/BubbleAttachmentItem.vue'
import TelegramSticker from './TelegramSticker.vue'
import { TELEGRAM_CHANNEL } from '@main/features/conversation/telegramReply'
import ImageLightbox from '@/components/ImageLightbox.vue'
import DownloadLink from '@main/components/DownloadLink.vue'

const props = defineProps({
  attachments: { type: Array, required: true },
  channel: { type: String, default: '' }
})

const isImage = (attachment) => (attachment.content_type || '').startsWith('image/')

const imageAttachments = computed(() => (props.attachments || []).filter(isImage))

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const openLightbox = (attachment) => {
  const idx = imageAttachments.value.findIndex((a) => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  lightboxOpen.value = true
}
</script>
