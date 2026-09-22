<script setup>
import { useI18n } from 'vue-i18n'
import { ArrowDown, ArrowUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'

defineProps({
  index: { type: Number, required: true },
  length: { type: Number, required: true }
})

const emit = defineEmits(['move'])
const { t } = useI18n()
</script>

<template>
  <div class="flex shrink-0">
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="size-7"
      :disabled="index === 0"
      :aria-label="t('globals.messages.moveUp')"
      @click="emit('move', -1)"
    >
      <ArrowUp aria-hidden="true" />
    </Button>
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="size-7"
      :disabled="index === length - 1"
      :aria-label="t('globals.messages.moveDown')"
      @click="emit('move', 1)"
    >
      <ArrowDown aria-hidden="true" />
    </Button>
  </div>
</template>
