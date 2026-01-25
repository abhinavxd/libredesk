<script setup>
import { DropdownMenuSub, useForwardPropsEmits } from 'radix-vue'

const props = defineProps({
  defaultOpen: { type: Boolean, required: false },
  open: { type: Boolean, required: false }
})
const emits = defineEmits(['update:open'])

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <DropdownMenuSub v-bind="forwarded">
    <slot />
  </DropdownMenuSub>
</template>
