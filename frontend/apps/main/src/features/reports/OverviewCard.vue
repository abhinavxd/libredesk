<template>
  <div class="flex flex-1 flex-col gap-5 box p-5">
    <div class="flex items-center">
      <p class="text-2xl font-medium">{{ title }}</p>
    </div>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-6">
      <div
        v-for="(item, key) in filteredCounts"
        :key="key"
        class="flex flex-col items-center gap-2 text-center"
      >
        <span class="text-sm text-muted-foreground">{{ labels[key] }}</span>
        <span class="text-2xl font-semibold">{{ item }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  counts: { type: Object, required: true },
  labels: { type: Object, required: true },
  title: { type: String, required: true }
})

// Filter out counts that don't have a label
const filteredCounts = computed(() => {
  return Object.fromEntries(Object.entries(props.counts).filter(([key]) => props.labels[key]))
})
</script>
