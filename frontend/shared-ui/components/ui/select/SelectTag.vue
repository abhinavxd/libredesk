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
        <div class="flex w-full items-center">
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
          <Spinner
            v-if="searching"
            size="xs"
            variant="muted"
            :absolute="false"
            class="mr-3 h-4 w-4 shrink-0"
          />
        </div>
      </ComboboxAnchor>
      <ComboboxPortal>
        <ComboboxContent>
          <CommandList
            position="popper"
            class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
          >
            <CommandEmpty v-if="!searching">{{ $t('globals.messages.noResultsFound') }}</CommandEmpty>
            <CommandGroup>
              <CommandItem
                v-for="item in visibleOptions"
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
import { CommandEmpty, CommandGroup, CommandItem, CommandList } from '@shared-ui/components/ui/command'
import {
  TagsInput,
  TagsInputInput,
  TagsInputItem,
  TagsInputItemDelete,
  TagsInputItemText
} from '@shared-ui/components/ui/tags-input'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxInput,
  ComboboxPortal,
  ComboboxRoot
} from 'radix-vue'
import { computed, ref, watch, onUnmounted } from 'vue'
import { useField } from 'vee-validate'
import Spinner from '@shared-ui/components/ui/spinner/Spinner.vue'
import { useRemoteSearch } from '@shared-ui/composables/useRemoteSearch'

const RENDER_CAP = 200
const SEARCH_DEBOUNCE_MS = 250

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
  },
  // When set, typing queries the server instead of filtering `items` locally.
  search: {
    type: Function,
    default: null
  }
})

const { handleBlur } = useField(() => props.name, undefined, {
  initialValue: tags.value
})

const open = ref(false)
const searchTerm = ref('')

const {
  results: remoteItems,
  searching,
  update: updateSearch,
  dispose: disposeSearch
} = useRemoteSearch((term) => props.search(term), SEARCH_DEBOUNCE_MS)

watch(searchTerm, (term) => {
  if (props.search) updateSearch(term)
})

watch(open, (isOpen) => {
  if (!isOpen && searchTerm.value) searchTerm.value = ''
})

onUnmounted(() => {
  disposeSearch()
})

const displayedItems = computed(() =>
  props.search && remoteItems.value !== null ? remoteItems.value : props.items
)

const filteredOptions = computed(() => {
  const available = displayedItems.value.filter((item) => !tags.value.includes(item.value))

  if (props.search || !searchTerm.value) return available

  return available.filter((item) =>
    item.label.toLowerCase().includes(searchTerm.value.toLowerCase())
  )
})

const visibleOptions = computed(() => filteredOptions.value.slice(0, RENDER_CAP))

// Reloading the list can drop a chosen row from `items`, which would leave its chip showing a raw id.
const seenLabels = new Map()

watch(
  displayedItems,
  (items) => {
    for (const item of items) seenLabels.set(item.value, item.label)
  },
  { immediate: true, deep: true }
)

const getLabel = (value) => {
  const item = displayedItems.value.find((item) => item.value === value)
  return item?.label || seenLabels.get(value) || value
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

const filterFunc = (remainingItemValues, term) => {
  const remainingItems = displayedItems.value.filter((item) => remainingItemValues.includes(item.value))
  if (props.search) return remainingItems.map((item) => item.value)
  return remainingItems
    .filter((item) => item.label.toLowerCase().includes(term.toLowerCase()))
    .map((item) => item.value)
}
</script>
