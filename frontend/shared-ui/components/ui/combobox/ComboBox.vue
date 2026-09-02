<template>
  <Popover v-model:open="open">
    <PopoverTrigger as-child>
      <slot name="trigger" :selected="selectedItem" :open="open">
        <Button
          variant="outline"
          role="combobox"
          :aria-expanded="open"
          :class="['w-full justify-between', buttonClass]"
        >
          <span class="flex min-w-0 flex-1 items-center text-left">
            <slot name="selected" :selected="selectedItem">
              <span class="min-w-0 truncate">{{ selectedLabel }}</span>
            </slot>
          </span>
          <CaretSortIcon class="h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </slot>
    </PopoverTrigger>
    <PopoverContent class="p-0" :align="align">
      <Command v-model:search-term="searchTerm" :filter-function="passThroughFilter">
        <CommandInput class="h-9" :placeholder="placeholder" :loading="searching" />
        <CommandEmpty v-if="!searching">{{ $t('globals.messages.notFound') }}</CommandEmpty>
        <CommandList>
          <CommandGroup>
            <CommandItem
              v-for="item in visibleItems"
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
import { ref, computed, watch, onUnmounted } from 'vue'
import { CaretSortIcon, CheckIcon } from '@radix-icons/vue'
import { cn } from '@shared-ui/lib/utils'
import { useRemoteSearch } from '@shared-ui/composables/useRemoteSearch'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import {
  CommandEmpty,
  CommandGroup,
  CommandInput,
  Command,
  CommandItem,
  CommandList
} from '@shared-ui/components/ui/command'

const RENDER_CAP = 300
const SEARCH_DEBOUNCE_MS = 250

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
  },
  align: {
    type: String,
    default: 'center'
  },
  // When set, typing queries the server instead of filtering `items` locally.
  search: {
    type: Function,
    default: null
  }
})

const emit = defineEmits(['select'])
const value = defineModel()
const open = ref(false)
const searchTerm = ref('')

const passThroughFilter = (items) => items

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

const filteredItems = computed(() => {
  if (props.search) return displayedItems.value
  const term = searchTerm.value?.trim().toLowerCase()
  if (!term) return displayedItems.value
  return displayedItems.value.filter((item) =>
    [item.label, item.calling_code]
      .filter(Boolean)
      .some((field) => String(field).toLowerCase().includes(term))
  )
})

const visibleItems = computed(() => filteredItems.value.slice(0, RENDER_CAP))

// Reloading the list can drop the picked row from `items`, leaving the trigger with no label.
const pickedItem = ref(null)

const selectedItem = computed(
  () =>
    props.items.find((i) => i.value === value.value) ||
    (pickedItem.value?.value === value.value ? pickedItem.value : null)
)
const selectedLabel = computed(() => selectedItem.value?.label || props.defaultLabel)

watch(
  [value, () => props.items],
  ([currentValue, items]) => {
    const selected = items.find((item) => item.value === currentValue)
    if (selected) pickedItem.value = selected
    else if (pickedItem.value?.value !== currentValue) pickedItem.value = null
  },
  { immediate: true, deep: true }
)

const handleSelect = (ev) => {
  if (typeof ev.detail.value === 'string') {
    try {
      const selected = JSON.parse(ev.detail.value)
      pickedItem.value = selected
      value.value = selected.value
      open.value = false
      emit('select', selected)
    } catch (e) {
      console.error('Invalid selection value')
    }
  }
}
</script>
