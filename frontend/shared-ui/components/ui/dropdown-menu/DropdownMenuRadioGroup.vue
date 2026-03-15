<script setup>
import { DropdownMenuRadioGroup, useForwardPropsEmits } from 'radix-vue'

const props = defineProps({
  modelValue: { type: String, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false }
})
const emits = defineEmits(['update:modelValue'])

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <DropdownMenuRadioGroup v-bind="forwarded">
    <slot />
  </DropdownMenuRadioGroup>
</template>
