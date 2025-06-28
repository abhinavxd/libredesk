<script setup>
import { computed } from 'vue'
import { PopoverContent, PopoverPortal, useForwardPropsEmits } from 'radix-vue'
import { cn } from '../../../lib/utils'

defineOptions({
  inheritAttrs: false
})

const props = defineProps({
  forceMount: { type: Boolean, required: false },
  trapFocus: { type: Boolean, required: false },
  side: { type: null, required: false },
  sideOffset: { type: Number, required: false, default: 4 },
  align: { type: null, required: false, default: 'center' },
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
  'interactOutside',
  'openAutoFocus',
  'closeAutoFocus'
])

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <PopoverPortal>
    <PopoverContent
      v-bind="{ ...forwarded, ...$attrs }"
      :class="
        cn(
          'z-50 w-72 rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2',
          props.class
        )
      "
    >
      <slot />
    </PopoverContent>
  </PopoverPortal>
</template>
