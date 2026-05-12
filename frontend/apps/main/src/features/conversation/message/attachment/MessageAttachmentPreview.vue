<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <div
          v-if="isAudio(attachment)"
          class="flex items-center gap-2 rounded border bg-muted/40 px-3 py-2"
        >
          <audio controls preload="auto" class="h-8 max-w-[260px]">
            <source :src="attachment.url" />
          </audio>
          <a
            :href="attachment.url"
            download
            class="p-1 rounded hover:bg-muted shrink-0"
            :title="t('globals.terms.download')"
            :aria-label="t('globals.terms.download')"
            @click.stop
          >
            <Download class="w-4 h-4 text-muted-foreground" />
          </a>
        </div>
        <AttachmentItem v-else :attachment="attachment" @preview="openLightbox" />
      </div>
    </div>
  </div>

  <ImageLightbox
    v-model="lightboxOpen"
    :images="imageAttachments"
    :start-index="lightboxIndex"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Download } from 'lucide-vue-next'
import AttachmentItem from '@/features/conversation/message/attachment/AttachmentItem.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'

const props = defineProps({
  attachments: { type: Array, required: true }
})

const { t } = useI18n()

const isImage = (attachment) => (attachment.content_type || '').startsWith('image/')
const isAudio = (attachment) => (attachment.content_type || '').startsWith('audio/')

const imageAttachments = computed(() =>
  (props.attachments || []).filter(isImage)
)

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const openLightbox = (attachment) => {
  const idx = imageAttachments.value.findIndex((a) => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  lightboxOpen.value = true
}
</script>
