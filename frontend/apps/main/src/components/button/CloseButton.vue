<template>
  <Button
    variant="ghost"
    @click.stop="onClose"
    size="xs"
    class="text-gray-400 dark:text-gray-500 hover:text-gray-600 hover:bg-gray-100 dark:hover:bg-gray-800 w-6 h-6 p-0"
  >
    <slot>
      <X size="16" />
    </slot>
  </Button>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { X } from 'lucide-vue-next'

defineProps({
  onClose: {
    type: Function,
    required: true
  }
})
</script>
