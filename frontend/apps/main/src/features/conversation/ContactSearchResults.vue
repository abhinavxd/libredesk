<template>
  <div
    v-if="results.length"
    class="absolute w-full z-50 mt-1 rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
  >
    <ul class="max-h-60 overflow-y-auto" role="listbox">
      <li
        v-for="(contact, index) in results"
        :key="contact.id"
        @click="emit('select', contact)"
        role="option"
        :aria-selected="index === highlightedIndex"
        class="relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors duration-200"
        :class="
          index === highlightedIndex
            ? 'bg-accent text-accent-foreground'
            : 'hover:bg-accent hover:text-accent-foreground'
        "
      >
        <slot :contact="contact" />
      </li>
    </ul>
  </div>
</template>

<script setup>
defineProps({
  results: { type: Array, required: true },
  highlightedIndex: { type: Number, required: true }
})
const emit = defineEmits(['select'])
</script>
