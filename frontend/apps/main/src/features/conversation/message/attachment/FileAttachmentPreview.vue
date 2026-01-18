<template>
  <div class="flex items-center group text-left">
    <div class="relative w-36 h-28 flex items-center justify-center">
      <div>
        <span class="size-20">📄</span>
      </div>
      <div class="p-1 absolute inset-0 text-gray-50 opacity-10 group-hover:opacity-100 overlay text-wrap">
        <div class="flex flex-col justify-between h-full">
          <div>
            <p class="font-bold text-xs opacity-80 group-hover:opacity-100">
              {{ getAttachmentName(attachment.name) }}
            </p>
            <p class="text-xs opacity-0 group-hover:opacity-100">{{ formatBytes(attachment.size) }}</p>
          </div>
          <div @click="downloadAttachment">
            <Download size=20></Download>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { formatBytes } from '@shared-ui/utils/file'
import { Download } from 'lucide-vue-next';

const props = defineProps({
  attachment: {
    type: Object,
    required: true
  }
})

const getAttachmentName = (name) => {
  return name.substring(0, 50)
}

const downloadAttachment = () => {
  window.open(props.attachment.url, '_blank')
}
</script>
