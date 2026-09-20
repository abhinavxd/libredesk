<template>
  <div class="flex flex-wrap gap-2" v-if="attachments && attachments.length > 0">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    > 
      <!-- Image preview -->
      <div v-if="isImage(attachment)" class="relative">
        <img
          :src="getThumbnailUrl(attachment)"
          :alt="attachment.name"
          class="max-w-48 max-h-32 rounded-lg object-cover"
          @error="fallbackToOriginal($event, attachment.url)"
          @click="openImage(attachment.url)"
        />
      </div>

      <!-- File attachment -->
      <div
        v-else
        class="flex items-center gap-2 p-2 bg-muted rounded-lg border border-border hover:bg-muted/80 transition-colors"
        @click="downloadFile(attachment)"
      >
        <div class="flex-shrink-0">
          <File class="text-muted-foreground" size="20"/>
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-foreground">{{ truncateFileName(attachment.name) }}</p>
          <p class="text-xs text-muted-foreground">{{ formatBytes(attachment.size) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { localMediaURL, isPreviewImage } from '@shared-ui/utils/resourceURL';
import { File } from 'lucide-vue-next';
import { formatBytes, getThumbFilepath } from '@shared-ui/utils/file';
defineProps({
  attachments: {
    type: Array,
    required: true
  }
})

const isImage = (attachment) => {
  return isPreviewImage(attachment.content_type) && !!localMediaURL(attachment.url)
}

const getThumbnailUrl = (attachment) => {
  if (!isImage(attachment)) return attachment.url
  return localMediaURL(attachment.thumbnail_url) || getThumbFilepath(localMediaURL(attachment.url))
}

const fallbackToOriginal = (event, originalUrl) => {
  if (event.target.dataset.originalFallback) return
  event.target.dataset.originalFallback = 'true'
  event.target.src = localMediaURL(originalUrl)
}

const openImage = (url) => {
  if (!localMediaURL(url)) return
  window.open(localMediaURL(url), '_blank', 'noopener,noreferrer')
}

const downloadFile = (attachment) => {
  if (!localMediaURL(attachment.url)) return
  window.open(localMediaURL(attachment.url), '_blank', 'noopener,noreferrer')
}

const truncateFileName = (name) => {
  if (name.length > 20) {
    return name.slice(0, 17) + '...'
  }
  return name
}
</script>
