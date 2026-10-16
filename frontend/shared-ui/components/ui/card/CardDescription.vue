<script setup>
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  class: { type: null, required: false }
})
</script>

<template>
  <p :class="cn('ui-description', props.class)">
    <slot />
  </p>
</template>
