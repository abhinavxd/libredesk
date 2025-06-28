<script setup>
import { computed } from 'vue'
import { ComboboxContent, useForwardPropsEmits } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  forceMount: { type: Boolean, required: false },
  position: { type: String, required: false },
  bodyLock: { type: Boolean, required: false },
  dismissable: { type: Boolean, required: false, default: false },
  side: { type: null, required: false },
  sideOffset: { type: Number, required: false },
  align: { type: null, required: false },
  alignOffset: { type: Number, required: false },
  avoidCollisions: { type: Boolean, required: false },
  collisionBoundary: { type: null, required: false },
  collisionPadding: { type: [Number, Object], required: false },
  arrowPadding: { type: Number, required: false },
  sticky: { type: String, required: false },
  hideWhenDetached: { type: Boolean, required: false },
  updatePositionStrategy: { type: String, required: false },
  prioritizePosition: { type: Boolean, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false },
  disableOutsidePointerEvents: { type: Boolean, required: false },
  class: { type: null, required: false }
})
const emits = defineEmits([
  'escapeKeyDown',
  'pointerDownOutside',
  'focusOutside',
  'interactOutside'
])

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <ComboboxContent
    v-bind="forwarded"
    :class="cn('max-h-[300px] overflow-y-auto overflow-x-hidden', props.class)"
  >
    <div role="presentation">
      <slot />
    </div>
  </ComboboxContent>
</template>
