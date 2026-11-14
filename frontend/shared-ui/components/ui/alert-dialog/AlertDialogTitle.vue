<script setup>
import { computed } from 'vue'
import { AlertDialogTitle } from 'radix-vue'
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
</script>

<template>
  <AlertDialogTitle v-bind="delegatedProps" :class="cn('ui-dialog-heading', props.class)">
    <slot />
  </AlertDialogTitle>
</template>
