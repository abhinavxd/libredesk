<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <ImageAttachmentPreview v-if="isImage(attachment)" :attachment="attachment" />
        <FileAttachmentPreview v-else :attachment="attachment" />
      </div>
    </div>
  </div>
</template>

<script setup>
import ImageAttachmentPreview from '@/features/conversation/message/attachment/ImageAttachmentPreview.vue'
import FileAttachmentPreview from '@/features/conversation/message/attachment/FileAttachmentPreview.vue'

defineProps({
  attachments: {
    type: Array,
    required: true
  }
})

const isImage = (attachment) => {
  return attachment.content_type.includes('image')
}
</script>
