<script setup>
import { cn } from '../../../lib/utils'

const props = defineProps({
  class: { type: null, required: false }
})
</script>

<template>
  <tfoot :class="cn('border-t bg-muted/50 font-medium [&>tr]:last:border-b-0', props.class)">
    <slot />
  </tfoot>
</template>
