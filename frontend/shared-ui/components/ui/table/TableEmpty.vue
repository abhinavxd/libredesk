<script setup>
import { computed } from 'vue'
import TableRow from './TableRow.vue'
import TableCell from './TableCell.vue'
import { cn } from '../../../lib/utils'

const props = defineProps({
  class: { type: null, required: false },
  colspan: { type: Number, required: false, default: 1 }
})

const delegatedProps = computed(() => {
  const { class: _, ...delegated } = props

  return delegated
})
</script>

<template>
  <TableRow>
    <TableCell
      :class="cn('p-4 whitespace-nowrap align-middle text-sm text-foreground', props.class)"
      v-bind="delegatedProps"
    >
      <div class="flex items-center justify-center py-10">
        <slot />
      </div>
    </TableCell>
  </TableRow>
</template>
