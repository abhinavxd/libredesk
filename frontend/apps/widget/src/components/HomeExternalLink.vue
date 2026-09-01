<template>
  <a
    :href="link.url"
    target="_blank"
    rel="noopener noreferrer"
    class="widget-starter no-underline"
  >
    <span>{{ link.text }}</span>
    <ExternalLink class="widget-starter__icon" />
  </a>
</template>

<script setup>
import { ExternalLink } from 'lucide-vue-next'

defineProps({
  link: {
    type: Object,
    required: true
  }
})
</script>
