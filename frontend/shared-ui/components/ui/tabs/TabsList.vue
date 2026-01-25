<script setup>
import { computed } from 'vue'
import { TabsList } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  loop: { type: Boolean, required: false },
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
  <TabsList
    v-bind="delegatedProps"
    :class="
      cn(
        'inline-flex h-9 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground',
        props.class
      )
    "
  >
    <slot />
  </TabsList>
</template>
