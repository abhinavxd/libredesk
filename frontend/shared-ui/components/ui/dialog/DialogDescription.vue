<script setup>
import { computed } from 'vue'
import { DialogDescription, useForwardProps } from 'radix-vue'
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
  <DialogDescription
    v-bind="forwardedProps"
    :class="cn('ui-description', props.class)"
  >
    <slot />
  </DialogDescription>
</template>
