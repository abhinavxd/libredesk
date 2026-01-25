<script setup>
import { computed } from 'vue'
import { TagsInputInput, useForwardProps } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  placeholder: { type: String, required: false },
  autoFocus: { type: Boolean, required: false },
  maxLength: { type: Number, required: false },
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
  <TagsInputInput
    v-bind="forwardedProps"
    :class="cn('text-sm min-h-5 focus:outline-none flex-1 bg-transparent px-1', props.class)"
  />
</template>
