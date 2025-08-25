<script setup>
import { cn } from '../../../lib/utils'

const props = defineProps({
  class: { type: null, required: false }
})
</script>

<template>
  <div :class="cn('rounded-xl border bg-card text-card-foreground shadow', props.class)">
    <slot />
  </div>
</template>
