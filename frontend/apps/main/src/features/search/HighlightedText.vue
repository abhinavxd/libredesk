<template>
  <span>
    <template v-for="(part, index) in parts" :key="index">
      <mark v-if="part.match" class="bg-warning/30 text-foreground rounded-sm px-0.5">{{ part.text }}</mark>
      <template v-else>{{ part.text }}</template>
    </template>
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { splitHighlight } from './highlight'

const props = defineProps({
  text: { type: String, default: '' },
  term: { type: String, default: '' }
})

const parts = computed(() => splitHighlight(props.text, props.term))
</script>
