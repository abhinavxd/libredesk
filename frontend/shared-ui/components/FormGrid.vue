<script setup>
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  compact: { type: Boolean, default: false },
  class: { type: null, required: false }
})
</script>

<template>
  <div
    :class="cn('form-grid grid min-w-0 gap-5', { 'form-grid-compact': props.compact }, props.class)"
  >
    <slot />
  </div>
</template>

<style scoped>
.form-grid {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 15rem), 1fr));
}

.form-grid-compact {
  grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
}

.form-grid > :deep(*) {
  min-width: 0;
}
</style>
