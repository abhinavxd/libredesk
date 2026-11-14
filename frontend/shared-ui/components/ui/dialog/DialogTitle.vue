<script setup>
import { computed } from 'vue'
import { DialogTitle, useForwardProps } from 'radix-vue'
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false }
})

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})

const forwardedProps = useForwardProps(delegatedProps)
</script>

<template>
  <DialogTitle
    v-bind="forwardedProps"
    :class="cn('ui-dialog-heading', props.class)"
  >
    <slot />
  </DialogTitle>
</template>
