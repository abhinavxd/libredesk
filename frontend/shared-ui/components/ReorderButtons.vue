<script setup>
import { useI18n } from '@shared-ui/i18n'
import { ArrowDown, ArrowUp } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'

defineProps({
  index: { type: Number, required: true },
  length: { type: Number, required: true }
})

const emit = defineEmits(['move'])
const i18n = useI18n()
</script>

<template>
  <div class="flex shrink-0 can-hover:opacity-0 can-hover:group-hover/reorder:opacity-100 can-hover:group-focus-within/reorder:opacity-100">
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="size-7"
      :disabled="index === 0"
      :aria-label="i18n._('Move up')"
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
      :aria-label="i18n._('Move down')"
      @click="emit('move', 1)"
    >
      <ArrowDown aria-hidden="true" />
    </Button>
  </div>
</template>
