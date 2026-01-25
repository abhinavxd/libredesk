<script setup>
import { AccordionRoot, useForwardPropsEmits } from 'radix-vue'

const props = defineProps({
  collapsible: { type: Boolean, required: false },
  disabled: { type: Boolean, required: false },
  dir: { type: String, required: false },
  orientation: { type: String, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  type: { type: null, required: false },
  modelValue: { type: null, required: false },
  defaultValue: { type: null, required: false }
})
const emits = defineEmits(['update:modelValue'])

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <AccordionRoot v-bind="forwarded">
    <slot />
  </AccordionRoot>
</template>
