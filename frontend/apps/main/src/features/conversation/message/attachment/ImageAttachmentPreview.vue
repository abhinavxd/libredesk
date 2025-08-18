<template>
  <div class="flex flex-wrap items-center group text-left">
    <div class="relative">
      <img :src="getThumbFilepath(attachment.url)" class="w-36 h-28 flex items-center object-cover" />
      <div class="p-1 absolute inset-0 text-gray-50 opacity-0 group-hover:opacity-100 overlay text-wrap">
        <div class="flex flex-col justify-between h-full">
          <div>
            <p class="font-bold text-xs">{{ trimAttachmentName(attachment.name) }}</p>
            <p class="text-xs">{{ formatBytes(attachment.size) }}</p>
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
import { formatBytes, getThumbFilepath } from '@shared-ui/utils/file'
import { Download } from 'lucide-vue-next';

const props = defineProps({
  attachment: {
    type: Object,
    required: true
  }
})

const trimAttachmentName = (name) => {
  return name.substring(0, 40)
}

const downloadAttachment = () => {
  window.open(props.attachment.url, '_blank');
}
</script>
