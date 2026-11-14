<script setup>
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  class: { type: null, required: false }
})
</script>

<template>
  <h3 :class="cn('ui-heading', props.class)">
    <slot />
  </h3>
</template>
