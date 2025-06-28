<script setup>
import { computed } from 'vue'
import { TagsInputRoot, useForwardPropsEmits } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  modelValue: { type: Array, required: false },
  defaultValue: { type: Array, required: false },
  addOnPaste: { type: Boolean, required: false },
  addOnTab: { type: Boolean, required: false },
  addOnBlur: { type: Boolean, required: false },
  duplicate: { type: Boolean, required: false },
  disabled: { type: Boolean, required: false },
  delimiter: { type: String, required: false },
  dir: { type: String, required: false },
  max: { type: Number, required: false },
  required: { type: Boolean, required: false },
  name: { type: String, required: false },
  id: { type: String, required: false },
  convertValue: { type: Function, required: false },
  displayValue: { type: Function, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  class: { type: null, required: false }
})
const emits = defineEmits(['update:modelValue', 'invalid'])

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <TagsInputRoot
    v-bind="forwarded"
    :class="
      cn(
        'flex flex-wrap gap-2 items-center rounded-md border border-input bg-background px-3 py-1.5 text-sm',
        props.class
      )
    "
  >
    <slot />
  </TagsInputRoot>
</template>
