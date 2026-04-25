<template>
  <div class="flex flex-wrap items-center group text-left">
    <div class="relative cursor-pointer" @click="$emit('preview', attachment)">
      <img
        :src="getThumbFilepath(attachment.url)"
        :alt="attachment.name"
        class="w-36 h-28 flex items-center object-cover"
      />
      <div class="p-1 absolute inset-0 text-gray-50 opacity-0 group-hover:opacity-100 overlay text-wrap">
        <div class="flex flex-col justify-between h-full">
          <div>
            <p class="font-bold text-xs">{{ trimAttachmentName(attachment.name) }}</p>
            <p class="text-xs">{{ formatBytes(attachment.size) }}</p>
          </div>
          <div class="flex items-center gap-2">
            <Eye :size="20" />
            <a
              :href="attachment.url"
              download
              class="hover:text-white/80"
              :aria-label="t('imageLightbox.download')"
              @click.stop
            >
              <Download :size="20" />
            </a>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'
import { formatBytes, getThumbFilepath } from '@shared-ui/utils/file'
import { Download, Eye } from 'lucide-vue-next'

defineProps({
  attachment: { type: Object, required: true }
})
defineEmits(['preview'])

const { t } = useI18n()

const trimAttachmentName = (name) => (name || '').substring(0, 40)
</script>
