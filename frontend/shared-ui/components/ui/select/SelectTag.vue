<template>
  <!-- idk why I named this select tag, should be named multi-select -->
  <TagsInput v-model="tags" class="px-0 gap-0" :displayValue="getLabel">
    <!-- Tags visible to the user -->
    <div class="flex gap-2 flex-wrap items-center px-3">
      <TagsInputItem v-for="tagValue in tags" :key="tagValue" :value="tagValue">
        <TagsInputItemText />
        <TagsInputItemDelete />
      </TagsInputItem>
    </div>

    <!-- Combobox for selecting new tags -->
    <ComboboxRoot
      :model-value="tags"
      v-model:open="open"
      v-model:search-term="searchTerm"
      :filterFunction="filterFunc"
      class="w-full"
    >
      <ComboboxAnchor as-child>
        <ComboboxInput :placeholder="placeholder" as-child>
          <TagsInputInput
            class="w-full px-3"
            :class="tags.length > 0 ? 'mt-2' : ''"
            @keydown.enter.prevent
            @blur="handleBlur"
            @click="open = true"
            @input.stop
          />
        </ComboboxInput>
      </ComboboxAnchor>
      <ComboboxPortal>
        <ComboboxContent>
          <CommandList
            position="popper"
            class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
          >
            <CommandEmpty> No results found </CommandEmpty>
            <CommandGroup>
              <CommandItem
                v-for="item in filteredOptions"
                :key="item.value"
                :value="item.value"
                @select="handleSelect"
              >
                {{ item.label }}
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </ComboboxContent>
      </ComboboxPortal>
    </ComboboxRoot>
  </TagsInput>
</template>

<script setup>
import { CommandEmpty, CommandGroup, CommandItem, CommandList } from '../command'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '../tags-input'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxInput,
  ComboboxPortal,
  ComboboxRoot
} from 'radix-vue'
import { computed, ref } from 'vue'
import { useField } from 'vee-validate'

const tags = defineModel({
  required: false,
  default: () => []
})

const props = defineProps({
  name: {
    type: String,
    required: false,
    default: 'tags'
  },
  placeholder: {
    type: String,
    default: 'Select...'
  },
  items: {
    type: Array,
    required: true,
    validator: (value) => value.every((item) => 'label' in item && 'value' in item)
  }
})

const { handleBlur } = useField(() => props.name, undefined, {
  initialValue: tags.value
})

const open = ref(false)
const searchTerm = ref('')

// Get all options that are not already selected and match the search term
// If not search term is provided, return all available options
const filteredOptions = computed(() => {
  const available = props.items.filter((item) => !tags.value.includes(item.value))

  if (!searchTerm.value) return available

  return available.filter((item) =>
    item.label.toLowerCase().includes(searchTerm.value.toLowerCase())
  )
})

const getLabel = (value) => {
  const item = props.items.find((item) => item.value === value)
  return item?.label || value
}

const handleSelect = (event) => {
  const selectedValue = event.detail.value
  if (selectedValue) {
    tags.value = [...tags.value, selectedValue]
    searchTerm.value = ''
  }

  if (filteredOptions.value.length === 0) {
    open.value = false
  }
}

// Custom filter function to filter items based on the search term
const filterFunc = (remainingItemValues, term) => {
  const remainingItems = props.items.filter((item) => remainingItemValues.includes(item.value))
  return remainingItems
    .filter((item) => item.label.toLowerCase().includes(term.toLowerCase()))
    .map((item) => item.value)
}
</script>
