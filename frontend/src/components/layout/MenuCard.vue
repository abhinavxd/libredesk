<template>
  <div
    class="flex flex-col p-4 border rounded shadow-sm hover:shadow transition-colors cursor-pointer max-w-xs"
    @click="handleClick">
    <div class="flex items-center mb-2">
      <component :is="icon" size="24" class="mr-2 text-primary" />
      <h3 class="text-lg font-medium">{{ title }}</h3>
    </div>
    <p class="text-sm text-gray-600 dark:text-gray-400">{{ subTitle }}</p>
  </div>
</template>

<script setup>
import { defineEmits } from 'vue'

const props = defineProps({
  title: String,
  subTitle: String,
  icon: Function,
  onClick: Function
})

const emit = defineEmits(['click'])

const handleClick = () => {
  props.onClick?.()
  emit('click')
}
</script>
