<template>
  <div class="border-b">
    <button
      type="button"
      class="flex w-full items-center justify-between py-4 text-sm font-semibold"
      :aria-expanded="open"
      @click="emit('toggle')"
    >
      {{ title }}
      <ChevronDown
        class="size-4 shrink-0 text-muted-foreground transition-transform duration-200"
        :class="{ 'rotate-180': open }"
      />
    </button>
    <!-- v-show, not v-if: unmounting a section would drop its field values from the form. -->
    <div v-show="open" class="pb-6 space-y-4">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { ChevronDown } from 'lucide-vue-next'

defineProps({
  title: {
    type: String,
    required: true
  },
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['toggle'])
</script>
