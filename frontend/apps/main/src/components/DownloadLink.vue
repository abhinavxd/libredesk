<template>
  <a
    :href="downloadUrl(url)"
    :title="i18n._('Download')"
    :aria-label="i18n._('Download')"
    class="inline-flex items-center justify-center p-1 rounded-md text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
    @click.stop
  >
    <Download :size="size" />
  </a>
</template>

<script setup>
import { Download } from 'lucide-vue-next'
import { useI18n } from '@shared-ui/i18n'
import { downloadUrl } from '@shared-ui/utils/file'

defineProps({
  url: { type: String, required: true },
  size: { type: Number, default: 16 }
})

const i18n = useI18n()
</script>
