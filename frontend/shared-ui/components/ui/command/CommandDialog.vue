<script setup>
import { computed } from 'vue'
import { useForwardPropsEmits } from 'radix-vue'
import Command from './Command.vue'
import { Dialog, DialogContent } from '../dialog'

const props = defineProps({
  open: { type: Boolean, required: false },
  defaultOpen: { type: Boolean, required: false },
  modal: { type: Boolean, required: false },
  searchTerm: { type: String, required: false },
  selectedValue: { type: null, required: false },
  filterFunction: { type: Function, required: false },
  class: { type: String, required: false },
  commandClass: { type: String, required: false }
})
const emits = defineEmits(['update:open', 'update:searchTerm', 'update:selectedValue'])

const delegatedProps = computed(() => {
  const delegated = { ...props }
  delete delegated.class
  delete delegated.commandClass
  delete delegated.selectedValue
  return delegated
})

const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <Dialog v-bind="forwarded">
    <DialogContent :class="['overflow-hidden p-0 shadow-lg', props.class]">
      <Command
        :search-term="props.searchTerm"
        :selected-value="props.selectedValue"
        :filter-function="props.filterFunction"
        @update:search-term="$emit('update:searchTerm', $event)"
        @update:selected-value="$emit('update:selectedValue', $event)"
        :class="[
          '[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-medium [&_[cmdk-group-heading]]:text-muted-foreground [&_[cmdk-group]:not([hidden])_~[cmdk-group]]:pt-0 [&_[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-5 [&_[cmdk-input-wrapper]_svg]:w-5 [&_[cmdk-input]]:h-12 [&_[cmdk-item]]:px-2 [&_[cmdk-item]]:py-2 [&_[cmdk-item]_svg]:h-5 [&_[cmdk-item]_svg]:w-5',
          props.commandClass
        ]"
      >
        <slot />
      </Command>
    </DialogContent>
  </Dialog>
</template>
