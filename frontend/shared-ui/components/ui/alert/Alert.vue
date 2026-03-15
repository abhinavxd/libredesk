<script setup>
import { alertVariants } from '.'
import { cn } from '../../../lib/utils'

const props = defineProps({
  class: { type: null, required: false },
  variant: { type: null, required: false }
})
</script>

<template>
  <div :class="cn(alertVariants({ variant }), props.class)" role="alert">
    <slot />
  </div>
</template>
