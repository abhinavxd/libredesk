<template>
  <TransitionGroup name="attachment-list" tag="div" class="flex flex-wrap gap-2 px-2 pt-2">
    <div
      v-for="attachment in allAttachments"
      :key="attachment.uuid || attachment.tempId"
      class="relative"
    >
      <div v-if="isImage(attachment)" class="group relative">
        <img
          :src="attachment.url"
          :alt="attachment.filename"
          class="h-20 w-20 rounded-md border object-cover"
        />
        <button
          type="button"
          class="absolute -right-1 -top-1 rounded-full border bg-background p-0.5 text-muted-foreground shadow-sm transition-colors hover:text-destructive focus:outline-none"
          :aria-label="`${$t('globals.terms.remove')} ${attachment.filename}`"
          @click.prevent="$emit('delete', attachment.uuid)"
        >
          <X :size="14" />
        </button>
      </div>
      <div
        v-else
        class="flex items-center gap-2 rounded-md border bg-background px-2 transition-colors duration-150 hover:bg-accent/50"
      >
        <div class="flex items-center space-x-1 py-1">
          <DotLoader v-if="attachment.loading" />
          <Paperclip v-else :size="16" />
          <div
            class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-foreground"
            :title="attachment.filename"
          >
            {{ getAttachmentName(attachment.filename) }}
            <span class="ml-1 text-xs text-muted-foreground">
              {{ formatBytes(attachment.size) }}
            </span>
          </div>
        </div>
        <button
          v-if="!attachment.loading"
          type="button"
          class="rounded-md text-muted-foreground transition-colors duration-150 hover:text-destructive focus:outline-none"
          :aria-label="`${$t('globals.terms.remove')} ${attachment.filename}`"
          @click.prevent="$emit('delete', attachment.uuid)"
        >
          <X :size="14" />
        </button>
      </div>
    </div>
  </TransitionGroup>
</template>

<script setup>
import { computed } from 'vue'
import { Paperclip, X } from 'lucide-vue-next'
import { DotLoader } from '@shared-ui/components/ui/loader'
import { formatBytes } from '@shared-ui/utils/file'

const props = defineProps({
  attachments: {
    type: Array,
    default: () => []
  },
  uploadingFiles: {
    type: Array,
    default: () => []
  }
})

defineEmits(['delete'])

const allAttachments = computed(() => [
  ...props.uploadingFiles.map((file) => ({
    tempId: file.tempId,
    filename: file.name,
    size: file.size,
    loading: true
  })),
  ...props.attachments
])

const getAttachmentName = (name) => {
  if (!name) return ''
  return name.length > 20 ? `${name.substring(0, 17)}...` : name
}

const isImage = (attachment) =>
  !attachment.loading && attachment.content_type?.startsWith('image/') && !!attachment.url
</script>

<style scoped>
.attachment-list-move,
.attachment-list-enter-active,
.attachment-list-leave-active {
  transition: all 0.3s ease;
}

.attachment-list-enter-from,
.attachment-list-leave-to {
  opacity: 0;
  transform: translateX(10px);
}

.attachment-list-leave-active {
  position: absolute;
}
</style>
