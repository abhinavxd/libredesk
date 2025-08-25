<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        variant="outline"
        role="combobox"
        :aria-expanded="open"
        :class="['w-full justify-between', buttonClass]"
      >
        <slot name="selected" :selected="selectedItem">{{ selectedLabel }}</slot>
        <CaretSortIcon class="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </Button>
    </PopoverTrigger>
    <PopoverContent class="p-0">
      <Command>
        <CommandInput class="h-9" :placeholder="placeholder" />
        <CommandEmpty>Not found.</CommandEmpty>
        <CommandList>
          <CommandGroup>
            <CommandItem
              v-for="item in props.items"
              :key="item.value"
              :value="JSON.stringify({ label: item.label, value: item.value })"
              @select="handleSelect"
            >
              <slot name="item" :item="item">{{ item.label }}</slot>
              <CheckIcon
                :class="
                  cn('ml-auto h-4 w-4', String(value) === item.value ? 'opacity-100' : 'opacity-0')
                "
              />
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { ref, computed } from 'vue'
import { CaretSortIcon, CheckIcon } from '@radix-icons/vue'
import { cn } from '../../../lib/utils'
import { Button } from '../button'
import { Popover, PopoverContent, PopoverTrigger } from '../popover'
import {
  CommandEmpty,
  CommandGroup,
  CommandInput,
  Command,
  CommandItem,
  CommandList
} from '../command'

const props = defineProps({
  items: {
    type: Array,
    required: true
  },
  placeholder: String,
  defaultLabel: String,
  buttonClass: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['select'])
const value = defineModel()
const open = ref(false)

const selectedItem = computed(() => props.items.find((i) => i.value === value.value))
const selectedLabel = computed(() => selectedItem.value?.label || props.defaultLabel)

const handleSelect = (ev) => {
  if (typeof ev.detail.value === 'string') {
    try {
      const selected = JSON.parse(ev.detail.value)
      value.value = selected.value
      open.value = false
      emit('select', selected)
    } catch (e) {
      console.error('Invalid selection value')
    }
  }
}
</script>
