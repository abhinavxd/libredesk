<template>
  <!-- idk why I named this select tag, should be named multi-select -->
  <TagsInput
    ref="rootRef"
    v-model="tags"
    class="min-h-9 gap-1 px-2 py-0.5"
    :displayValue="getLabel"
  >
    <TagsInputItem
      v-for="tagValue in tags"
      :key="tagValue"
      :value="tagValue"
      class="min-w-0"
      :class="compact ? 'max-w-[calc(100%-4rem)]' : ''"
    >
      <TagsInputItemText
        class="min-w-0"
        :class="compact ? 'truncate' : ''"
        :title="compact ? getLabel(tagValue) : undefined"
      />
      <TagsInputItemDelete
        type="button"
        :tabindex="compact ? 0 : -1"
        class="shrink-0 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        :class="compact ? 'size-4 items-center justify-center text-muted-foreground can-hover:hover:text-foreground focus-visible:text-foreground [&_svg]:size-3' : ''"
        @click="focusAfterRemove"
      />
    </TagsInputItem>

    <Popover v-if="compact && tags.length" v-model:open="open">
      <PopoverTrigger as-child>
        <Button
          ref="addButtonRef"
          type="button"
          variant="ghost"
          size="icon"
          :aria-label="placeholder"
          class="h-7 w-7 shrink-0 text-muted-foreground"
          @blur="handleBlur"
        >
          <Plus aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" class="p-0">
        <Command v-model:search-term="searchTerm" :filter-function="filterFunc">
          <CommandInput :placeholder="placeholder" :aria-label="placeholder" :loading="searching" />
          <CommandEmpty v-if="!searching">{{ i18n._('No results found') }}</CommandEmpty>
          <CommandList>
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
        </Command>
      </PopoverContent>
    </Popover>

    <ComboboxRoot
      v-else
      :model-value="tags"
      v-model:open="open"
      v-model:search-term="searchTerm"
      :filterFunction="filterFunc"
      class="min-w-24 flex-1"
    >
      <ComboboxAnchor as-child>
        <div class="flex w-full items-center">
          <ComboboxInput :placeholder="tags.length ? '' : placeholder" :aria-label="placeholder" as-child>
            <TagsInputInput
              class="w-0 min-w-0 flex-1 px-1 placeholder:text-muted-foreground"
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
            class="mr-1 h-4 w-4 shrink-0"
          />
          <div v-if="$slots.action" class="shrink-0">
            <slot name="action" />
          </div>
        </div>
      </ComboboxAnchor>
      <ComboboxPortal>
        <ComboboxContent>
          <CommandList
            position="popper"
            class="w-[--radix-popper-anchor-width] rounded-md mt-2 border bg-popover text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2"
          >
            <CommandEmpty v-if="!searching">{{
              i18n._('No results found')
            }}</CommandEmpty>
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
    <div v-if="compact && tags.length && $slots.action" class="ml-auto shrink-0">
      <slot name="action" />
    </div>
  </TagsInput>
</template>

<script setup>
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList
} from '@shared-ui/components/ui/command'
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
import { computed, nextTick, ref, watch, onUnmounted } from 'vue'
import { useField } from 'vee-validate'
import { Plus } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
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
  },
  keepOpenOnSelect: {
    type: Boolean,
    default: false
  },
  compact: {
    type: Boolean,
    default: false
  }
})

const { handleBlur } = useField(() => props.name, undefined, {
  initialValue: tags.value
})

const open = ref(false)
const searchTerm = ref('')
const rootRef = ref(null)
const addButtonRef = ref(null)

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

const focusAfterRemove = () => {
  if (!props.compact) return
  nextTick(() => {
    const target = rootRef.value?.$el?.querySelector('input') || addButtonRef.value?.$el
    target?.focus()
  })
}

// Closing the list here, not in the combobox, keeps focus in the input for the next pick.
const handleSelect = (event) => {
  event.preventDefault()

  const selectedValue = event.detail.value
  if (selectedValue) {
    tags.value = [...tags.value, selectedValue]
    searchTerm.value = ''
  }

  if (!props.keepOpenOnSelect || filteredOptions.value.length === 0) {
    open.value = false
  }

  nextTick(() => rootRef.value?.$el?.querySelector('input')?.focus())
}

const filterFunc = (remainingItemValues, term) => {
  const remainingItems = displayedItems.value.filter((item) =>
    remainingItemValues.includes(item.value)
  )
  if (props.search) return remainingItems.map((item) => item.value)
  return remainingItems
    .filter((item) => item.label.toLowerCase().includes(term.toLowerCase()))
    .map((item) => item.value)
}
</script>
