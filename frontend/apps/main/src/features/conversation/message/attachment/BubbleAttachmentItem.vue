<template>
  <div class="flex items-center group text-left">
    <Popover :open="showAudio" @update:open="showAudio = $event">
      <PopoverTrigger as-child>
        <div
          class="relative w-36 h-28 rounded border overflow-hidden cursor-pointer transition-colors"
          :class="
            isImage
              ? ''
              : 'flex flex-col items-center justify-between bg-muted/40 hover:bg-muted p-3'
          "
          @click="onClick"
        >
          <template v-if="isImage">
            <img
              :src="getThumbFilepath(attachment.url)"
              :alt="attachment.name"
              class="w-full h-full object-cover"
            />
            <div
              class="absolute inset-x-0 top-0 flex items-start justify-between gap-2 px-2 pt-1.5 pb-5 bg-gradient-to-b from-black/75 via-black/40 to-transparent opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"
            >
              <div class="min-w-0 flex-1 text-white image-meta">
                <p class="font-medium text-xs truncate">{{ shortName(attachment.name) }}</p>
                <p class="text-[10px] opacity-90">{{ formatBytes(attachment.size) }}</p>
              </div>
              <DownloadLink
                :url="attachment.url"
                class="text-white hover:text-white hover:bg-white/15 shrink-0 pointer-events-auto -mr-0.5"
              />
            </div>
          </template>

          <template v-else>
            <div class="flex-1 flex items-center justify-center">
              <component :is="fileIcon" class="w-10 h-10" :class="iconColor" />
            </div>
            <div class="w-full text-center">
              <p class="text-xs font-medium text-foreground truncate" :title="attachment.name">
                {{ shortName(attachment.name) }}
              </p>
              <p class="text-xs text-muted-foreground">{{ formatBytes(attachment.size) }}</p>
            </div>
          </template>

          <DownloadLink
            v-if="!isImage"
            :url="attachment.url"
            class="absolute top-1.5 right-1.5 opacity-0 group-hover:opacity-100 transition-opacity"
          />
        </div>
      </PopoverTrigger>
      <PopoverContent v-if="isAudio" class="w-80 p-3" @click.stop>
        <p class="text-xs font-medium truncate mb-2" :title="attachment.name">
          {{ attachment.name }}
        </p>
        <audio :src="attachment.url" controls autoplay preload="auto" class="w-full h-8" />
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { formatBytes, getThumbFilepath } from '@shared-ui/utils/file'
import DownloadLink from '@/components/DownloadLink.vue'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import {
  FileText,
  FileSpreadsheet,
  File,
  FileImage,
  FileArchive,
  FileCode,
  FileAudio
} from 'lucide-vue-next'

const props = defineProps({
  attachment: { type: Object, required: true }
})
const emit = defineEmits(['preview'])

const showAudio = ref(false)

const shortName = (name) => (name || '').substring(0, 40)

const isImage = computed(() => (props.attachment.content_type || '').startsWith('image/'))

const isAudio = computed(() => (props.attachment.content_type || '').startsWith('audio/'))

const ext = computed(() => {
  const parts = (props.attachment.name || '').split('.')
  return parts.length > 1 ? parts.pop().toLowerCase() : ''
})

const fileIcon = computed(() => {
  if (isAudio.value) return FileAudio
  const e = ext.value
  if (e === 'pdf') return FileText
  if (['xls', 'xlsx', 'csv'].includes(e)) return FileSpreadsheet
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(e)) return FileImage
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(e)) return FileArchive
  if (['html', 'xml', 'json', 'js', 'css'].includes(e)) return FileCode
  if (['doc', 'docx', 'txt', 'rtf'].includes(e)) return FileText
  return File
})

const iconColor = computed(() => {
  if (isAudio.value) return 'text-purple-500'
  const e = ext.value
  if (e === 'pdf') return 'text-red-500'
  if (['xls', 'xlsx', 'csv'].includes(e)) return 'text-green-600'
  if (['doc', 'docx', 'txt', 'rtf'].includes(e)) return 'text-blue-500'
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(e)) return 'text-amber-600'
  return 'text-muted-foreground'
})

const onClick = () => {
  if (isImage.value) {
    emit('preview', props.attachment)
  } else if (isAudio.value) {
    showAudio.value = true
  } else {
    window.open(props.attachment.url, '_blank', 'noopener,noreferrer')
  }
}
</script>

<style scoped>
.image-meta {
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
}
</style>
