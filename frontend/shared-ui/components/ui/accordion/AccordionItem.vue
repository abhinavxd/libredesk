<script setup>
import { computed } from 'vue'
import { AccordionItem, useForwardProps } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  disabled: { type: Boolean, required: false },
  value: { type: String, required: true },
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
  <AccordionItem v-bind="forwardedProps" :class="cn('border-b', props.class)">
    <slot />
  </AccordionItem>
</template>
