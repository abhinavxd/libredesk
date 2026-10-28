<script setup>
import {
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxRoot,
  ComboboxViewport,
  ComboboxVirtualizer
} from 'reka-ui'

defineProps({
  items: { type: Array, required: true },
  modelValue: Object,
  placeholder: String
})
const emit = defineEmits(['select'])
const searchTerm = defineModel('searchTerm', { type: String, default: '' })
</script>

<template>
  <ComboboxRoot
    :model-value="modelValue"
    :open="true"
    ignore-filter
    @update:model-value="emit('select', $event)"
  >
    <ComboboxInput
      v-model="searchTerm"
      :display-value="() => searchTerm"
      :placeholder="placeholder"
      :aria-label="placeholder"
      class="flex h-9 w-full border-b bg-transparent px-3 text-sm outline-none placeholder:text-muted-foreground"
    />
    <div v-if="!items.length" class="py-6 text-center text-sm">
      {{ $t('globals.messages.notFound') }}
    </div>
    <ComboboxContent>
      <ComboboxViewport class="h-[300px] overflow-y-auto overflow-x-hidden p-1">
        <ComboboxVirtualizer
          v-slot="{ option }"
          :options="items"
          :estimate-size="32"
          :text-content="(item) => item.label"
        >
          <ComboboxItem
            :key="option.value"
            :value="option"
            class="flex h-8 w-full cursor-default select-none items-center rounded-sm px-2 text-sm outline-none data-[highlighted]:bg-accent"
          >
            <slot :item="option">{{ option.label }}</slot>
          </ComboboxItem>
        </ComboboxVirtualizer>
      </ComboboxViewport>
    </ComboboxContent>
  </ComboboxRoot>
</template>
