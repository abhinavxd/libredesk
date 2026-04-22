<script setup>
import { useForwardPropsEmits } from 'radix-vue'
import Command from './Command.vue'
import { Dialog, DialogContent } from '../dialog'

const props = defineProps({
  open: { type: Boolean, required: false },
  defaultOpen: { type: Boolean, required: false },
  modal: { type: Boolean, required: false },
  searchTerm: { type: String, required: false },
  filterFunction: { type: Function, required: false },
  class: { type: String, required: false }
})
const emits = defineEmits(['update:open', 'update:searchTerm'])

const forwarded = useForwardPropsEmits(props, emits)
</script>

<template>
  <Dialog v-bind="forwarded">
    <DialogContent :class="['overflow-hidden p-0 shadow-lg', props.class]">
      <Command
        :search-term="props.searchTerm"
        :filter-function="props.filterFunction"
        @update:search-term="$emit('update:searchTerm', $event)"
        class="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground [&_[cmdk-group]:not([hidden])_~[cmdk-group]]:pt-0 [&_[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-5 [&_[cmdk-input-wrapper]_svg]:w-5 [&_[cmdk-input]]:h-12 [&_[cmdk-item]]:px-2 [&_[cmdk-item]]:py-2 [&_[cmdk-item]_svg]:h-5 [&_[cmdk-item]_svg]:w-5"
      >
        <slot />
      </Command>
    </DialogContent>
  </Dialog>
</template>
