<template>
  <div class="flex flex-wrap">
    <TransitionGroup name="attachment-list" tag="div" class="flex flex-wrap gap-2">
      <div
        v-for="attachment in allAttachments"
        :key="attachment.uuid || attachment.tempId"
        class="flex items-center bg-background border border-gray-200 rounded shadow-sm transition-all duration-300 ease-in-out hover:shadow-md group px-2 gap-2"
      >
        <div class="flex items-center space-x-1 py-1">
          <DotLoader v-if="attachment.loading"/>
          <PaperclipIcon v-else size="16" />

          <Tooltip>
            <TooltipTrigger as-child>
              <div
                class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-primary group-hover:text-gray-900 dark:group-hover:text-foreground"
              >
                {{ getAttachmentName(attachment.filename) }}
                <span class="text-xs text-gray-500 ml-1">
                  {{ formatBytes(attachment.size) }}
                </span>
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p class="text-sm">{{ attachment.filename }}</p>
            </TooltipContent>
          </Tooltip>
        </div>

        <button
          v-if="!attachment.loading"
          @click.prevent="onDelete(attachment.uuid)"
          class="text-gray-400 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 rounded transition-colors duration-300 ease-in-out"
          title="Remove attachment"
        >
          <X size="14" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatBytes } from '@shared-ui/utils/file'
import { X, Paperclip as PaperclipIcon } from 'lucide-vue-next'
import { DotLoader } from '@shared-ui/components/ui/loader'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'

const props = defineProps({
  attachments: {
    type: Array,
    required: true
  },
  uploadingFiles: {
    type: Array,
    default: () => []
  },
  onDelete: {
    type: Function,
    required: true
  }
})

const allAttachments = computed(() => [
  ...props.uploadingFiles.map((file, i) => ({
    tempId: `${file.name}-${i}`,
    filename: file.name,
    size: file.size,
    loading: true
  })),
  ...props.attachments
])

const getAttachmentName = (name) => {
  return name.length > 20 ? name.substring(0, 17) + '...' : name
}
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
