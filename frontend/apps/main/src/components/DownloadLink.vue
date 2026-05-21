<template>
  <a
    :href="downloadUrl(url)"
    :title="t('globals.terms.download')"
    :aria-label="t('globals.terms.download')"
    class="inline-flex items-center justify-center p-1 rounded text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
    @click.stop
  >
    <Download :size="size" />
  </a>
</template>

<script setup>
import { Download } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { downloadUrl } from '@shared-ui/utils/file'

defineProps({
  url: { type: String, required: true },
  size: { type: Number, default: 16 }
})

const { t } = useI18n()
</script>
