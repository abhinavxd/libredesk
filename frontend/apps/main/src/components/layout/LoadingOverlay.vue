<template>
  <div class="relative" :class="{ 'min-h-[60vh]': loading && reserveHeight }">
    <Spinner v-if="loading" />
    <div :class="{ 'opacity-50 transition-opacity duration-300': loading }">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { Spinner } from '@shared-ui/components/ui/spinner'

defineProps({
  loading: {
    type: Boolean,
    default: false
  },
  reserveHeight: {
    type: Boolean,
    default: false
  }
})
</script>
