<script setup>
import { computed } from 'vue'
import { AlertDialogAction } from 'radix-vue'
import { cn } from '../../../lib/utils'
import { buttonVariants } from '../button'

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false },
  variant: { type: String, required: false }
})

const delegatedProps = computed(() => {
  const { class: _, variant: __, ...delegated } = props

  return delegated
})
</script>

<template>
  <AlertDialogAction
    v-bind="delegatedProps"
    :class="cn(buttonVariants({ variant: props.variant }), props.class)"
  >
    <slot />
  </AlertDialogAction>
</template>
