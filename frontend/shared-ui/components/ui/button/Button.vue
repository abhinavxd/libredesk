<script setup>
import { Primitive } from 'reka-ui'
import { cn } from '../../../lib/utils'
import { buttonVariants } from '.'
import { Loader2 } from 'lucide-vue-next'
const props = defineProps({
  variant: { type: null, required: false },
  size: { type: null, required: false },
  class: { type: null, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false, default: 'button' },
  isLoading: { type: Boolean, required: false, default: false },
  disabled: { type: Boolean, required: false, default: false }
})
</script>

<template>
  <Primitive
    :as="as"
    :as-child="asChild"
    :class="
      cn(
        buttonVariants({ variant, size }),
        'relative',
        { 'text-transparent': isLoading },
        props.class
      )
    "
    :disabled="isLoading || disabled"
  >
    <slot />
    <span
      v-if="isLoading"
      class="absolute inset-0 flex items-center justify-center pointer-events-none text-background"
    >
      <Loader2 class="h-5 w-5 animate-spin" />
    </span>
  </Primitive>
</template>
