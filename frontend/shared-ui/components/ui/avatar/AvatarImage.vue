<script setup>
import { AvatarImage } from 'radix-vue'

const props = defineProps({
  src: { type: String, required: false, default: '' },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false }
})
</script>

<template>
  <AvatarImage v-bind="props" class="h-full w-full object-cover" />
</template>
