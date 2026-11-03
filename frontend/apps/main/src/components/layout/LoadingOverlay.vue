<template>
  <div class="relative" :class="{ 'min-h-[60vh]': loading && reserveHeight }">
    <Spinner v-if="loading" />
    <div :class="{ 'opacity-50 transition-opacity duration-300': loading }">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { provide, toRef } from 'vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { LOADING_OVERLAY_INJECTION_KEY } from './loadingOverlay'

const props = defineProps({
  loading: {
    type: Boolean,
    default: false
  },
  reserveHeight: {
    type: Boolean,
    default: false
  }
})

provide(LOADING_OVERLAY_INJECTION_KEY, toRef(props, 'loading'))
</script>
