<script setup>
import { computed } from 'vue'
import { ComboboxRoot, useForwardPropsEmits } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  modelValue: { type: null, required: false, default: '' },
  defaultValue: { type: null, required: false },
  open: { type: Boolean, required: false, default: true },
  defaultOpen: { type: Boolean, required: false },
  searchTerm: { type: String, required: false },
  multiple: { type: Boolean, required: false },
  disabled: { type: Boolean, required: false },
  name: { type: String, required: false },
  dir: { type: String, required: false },
  filterFunction: { type: Function, required: false },
  displayValue: { type: Function, required: false },
  resetSearchTermOnBlur: { type: Boolean, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false }
})

const emits = defineEmits(['update:modelValue', 'update:open', 'update:searchTerm'])

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ComboboxRoot
    v-bind="forwarded"
    :class="
      cn(
        'flex h-full w-full flex-col overflow-hidden rounded-md bg-popover text-popover-foreground',
        props.class
      )
    "
  >
    <slot />
  </ComboboxRoot>
</template>
