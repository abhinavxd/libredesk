<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="outline"
        role="combobox"
        class="w-full justify-between font-normal"
        :aria-expanded="open"
        :disabled="disabled"
      >
        <span
          class="min-w-0 truncate"
          :class="modelValue.length ? 'text-foreground' : 'text-muted-foreground'"
        >
          {{ modelValue.length ? selectedLabel || placeholder : placeholder }}
        </span>
        <span class="ml-auto flex shrink-0 items-center gap-1.5">
          <span
            v-if="modelValue.length"
            class="rounded bg-secondary px-1.5 py-0.5 text-xs tabular-nums text-secondary-foreground"
          >
            {{ modelValue.length }}
          </span>
          <CaretSortIcon class="h-4 w-4 opacity-50" aria-hidden="true" />
        </span>
      </Button>
    </PopoverTrigger>
    <PopoverContent align="start" class="p-0">
      <Command
        multiple
        :model-value="modelValue"
        v-model:search-term="searchTerm"
        :filter-function="passThroughFilter"
      >
        <CommandInput :placeholder="placeholder" :loading="searching" />
        <CommandEmpty v-if="!searching">{{ t('globals.messages.noResultsFound') }}</CommandEmpty>
        <CommandList>
          <CommandGroup>
            <CommandItem
              v-for="item in visibleItems"
              :key="item.value"
              :value="item.value"
              @select.prevent="toggle(item.value)"
            >
              <slot name="item" :item="item">
                <span class="truncate">{{ item.label }}</span>
              </slot>
              <CheckIcon
                class="ml-auto h-4 w-4 shrink-0"
                :class="isSelected(item.value) ? 'opacity-100' : 'opacity-0'"
                aria-hidden="true"
              />
            </CommandItem>
          </CommandGroup>
        </CommandList>
      </Command>
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { computed, onUnmounted, ref, watch } from 'vue'
import { CaretSortIcon, CheckIcon } from '@radix-icons/vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@shared-ui/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { useRemoteSearch } from '@shared-ui/composables/useRemoteSearch'

const RENDER_CAP = 200
const SEARCH_DEBOUNCE_MS = 250

const props = defineProps({
  items: {
    type: Array,
    required: true,
    validator: (items) => items.every((item) => 'label' in item && 'value' in item)
  },
  placeholder: {
    type: String,
    default: ''
  },
  selectedLabel: {
    type: String,
    default: ''
  },
  search: {
    type: Function,
    default: null
  },
  disabled: {
    type: Boolean,
    default: false
  }
})

const modelValue = defineModel({ type: Array, default: () => [] })

const { t } = useI18n()
const open = ref(false)
const searchTerm = ref('')
const passThroughFilter = (values) => values

const {
  results: searchResults,
  searching,
  update: updateSearch,
  dispose: disposeSearch
} = useRemoteSearch((term) => props.search(term), SEARCH_DEBOUNCE_MS)

const displayedItems = computed(() => searchResults.value ?? props.items)
const filteredItems = computed(() => {
  if (props.search || !searchTerm.value) return displayedItems.value
  const term = searchTerm.value.trim().toLowerCase()
  return displayedItems.value.filter((item) => item.label.toLowerCase().includes(term))
})
const visibleItems = computed(() => filteredItems.value.slice(0, RENDER_CAP))

const isSelected = (value) =>
  modelValue.value.some((selected) => String(selected) === String(value))

const toggle = (value) => {
  modelValue.value = isSelected(value)
    ? modelValue.value.filter((selected) => String(selected) !== String(value))
    : [...modelValue.value, value]
}

watch(searchTerm, (term) => {
  if (props.search) updateSearch(term)
})

watch(open, (isOpen) => {
  if (!isOpen && searchTerm.value) searchTerm.value = ''
})

onUnmounted(disposeSearch)
</script>
