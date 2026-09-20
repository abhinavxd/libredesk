<template>
  <div class="flex flex-wrap gap-2" :class="align === 'end' ? 'justify-end' : 'justify-start'">
    <Button
      v-for="(reply, index) in replies"
      :key="`${index}-${reply}`"
      type="button"
      variant="outline"
      size="sm"
      class="h-auto max-w-full rounded-full py-1.5 text-left whitespace-normal shadow-md"
      :disabled="disabled"
      @click="emit('select', reply)"
    >
      {{ reply }}
    </Button>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'

defineProps({
  replies: { type: Array, required: true },
  disabled: { type: Boolean, default: false },
  align: { type: String, default: 'start' }
})

const emit = defineEmits(['select'])
</script>
