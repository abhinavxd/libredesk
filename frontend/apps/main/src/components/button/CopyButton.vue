<template>
  <Button
    type="button"
    :variant="variant"
    :size="size"
    class="h-7 w-7 text-muted-foreground hover:text-foreground hover:bg-accent"
    :class="{ 'w-auto gap-1 px-2': showText }"
    @click="copy"
  >
    <Check v-if="copied" class="h-4 w-4 text-success" />
    <Copy v-else class="h-4 w-4" />
    <span v-if="showText" class="text-xs">{{ $t('globals.terms.copy') }}</span>
  </Button>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import { Copy, Check } from 'lucide-vue-next'
import { copyToClipboard } from '@shared-ui/utils/clipboard.js'
import { useEmitter } from '@main/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  text: { type: String, required: true },
  variant: { type: String, required: false, default: 'ghost' },
  size: { type: String, required: false, default: 'icon' },
  showText: { type: Boolean, required: false, default: false }
})

const { t } = useI18n()
const emitter = useEmitter()
const copied = ref(false)

const copy = async () => {
  const success = await copyToClipboard(props.text)
  if (!success) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.copyFailed')
    })
    return
  }

  copied.value = true
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
    description: t('globals.messages.copied')
  })
  setTimeout(() => {
    copied.value = false
  }, 1500)
}
</script>
