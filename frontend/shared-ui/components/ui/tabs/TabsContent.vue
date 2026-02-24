<script setup>
import { computed } from 'vue'
import { TabsContent } from 'radix-vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  value: { type: [String, Number], required: true },
  forceMount: { type: Boolean, required: false },
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
  <TabsContent
    :class="
      cn(
        'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        props.class
      )
    "
    v-bind="delegatedProps"
  >
    <slot />
  </TabsContent>
</template>
