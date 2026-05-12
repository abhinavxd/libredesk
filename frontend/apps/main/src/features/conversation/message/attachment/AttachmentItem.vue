<template>
  <div class="flex items-center group text-left">
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
          class="absolute inset-0 p-1 pr-12 text-gray-50 opacity-0 group-hover:opacity-100 overlay text-wrap"
        >
          <p class="font-bold text-xs">{{ shortName(attachment.name) }}</p>
          <p class="text-xs">{{ formatBytes(attachment.size) }}</p>
        </div>
      </template>

      <template v-else>
        <div class="flex-1 flex items-center justify-center">
          <component :is="fileIcon" class="w-10 h-10" :class="iconColor" />
        </div>
        <div class="w-full text-center">
          <p
            class="text-xs font-medium text-foreground truncate"
            :title="attachment.name"
          >
            {{ shortName(attachment.name) }}
          </p>
          <p class="text-xs text-muted-foreground">{{ formatBytes(attachment.size) }}</p>
        </div>
      </template>

      <a
        :href="attachment.url"
        target="_blank"
        rel="noopener noreferrer"
        class="absolute top-1.5 right-1.5 p-0.5 rounded opacity-0 group-hover:opacity-100 transition-opacity"
        :class="isImage ? 'hover:text-white/80' : 'hover:bg-background'"
        :title="t('globals.terms.download')"
        :aria-label="t('globals.terms.download')"
        @click.stop
      >
        <Download
          class="w-4 h-4"
          :class="isImage ? '' : 'text-muted-foreground'"
        />
      </a>
    </div>

    <Teleport to="body">
      <div
        v-if="showPdfPreview"
        class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80"
        @click.self="showPdfPreview = false"
      >
        <button
          class="absolute top-4 right-4 text-white hover:text-gray-300 z-10"
          :aria-label="t('globals.messages.close')"
          @click="showPdfPreview = false"
        >
          <X :size="28" />
        </button>
        <a
          :href="attachment.url"
          download
          class="absolute top-4 right-14 text-white hover:text-gray-300 z-10"
          :title="t('globals.terms.download')"
          :aria-label="t('globals.terms.download')"
        >
          <Download :size="24" />
        </a>
        <iframe
          :src="attachment.url"
          :title="attachment.name"
          class="w-[90vw] h-[90vh] rounded shadow-2xl bg-white"
        />
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatBytes, getThumbFilepath } from '@shared-ui/utils/file'
import {
  Download,
  X,
  FileText,
  FileSpreadsheet,
  File,
  FileImage,
  FileArchive,
  FileCode
} from 'lucide-vue-next'

const props = defineProps({
  attachment: { type: Object, required: true }
})
const emit = defineEmits(['preview'])

const { t } = useI18n()

const showPdfPreview = ref(false)

const shortName = (name) => (name || '').substring(0, 40)

const isImage = computed(() =>
  (props.attachment.content_type || '').startsWith('image/')
)

const ext = computed(() => {
  const parts = (props.attachment.name || '').split('.')
  return parts.length > 1 ? parts.pop().toLowerCase() : ''
})

const canPreviewPdf = computed(() => !isImage.value && ext.value === 'pdf')

const fileIcon = computed(() => {
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
  } else if (canPreviewPdf.value) {
    showPdfPreview.value = true
  } else {
    window.open(props.attachment.url, '_blank')
  }
}
</script>
