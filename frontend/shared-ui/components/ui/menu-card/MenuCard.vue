<template>
  <Card class="menu-card" @click="$emit('click')">
    <CardHeader class="pb-3">
      <div class="menu-card-header">
        <CardTitle class="menu-card-title">
          <slot name="title">
            {{ title }}
          </slot>
        </CardTitle>
      </div>
    </CardHeader>
    <CardContent class="pt-0">
      <slot name="subtitle">
        <p v-if="subtitle" class="menu-card-subtitle">{{ subtitle }}</p>
      </slot>
      <slot />
    </CardContent>
  </Card>
</template>

<script setup>
import { Card, CardContent, CardHeader, CardTitle } from '../card'

defineProps({
  title: {
    type: String,
    default: ''
  },
  subtitle: {
    type: String,
    default: ''
  }
})

defineEmits(['click'])
</script>

<style scoped>
.menu-card {
  @apply border rounded shadow-sm transition-colors cursor-pointer p-2 bg-background;
}

.menu-card:hover {
  @apply border-gray-600;
}

.menu-card-header {
  @apply flex items-center justify-between;
}

.menu-card-title {
  @apply flex items-center text-lg font-medium;
}

.menu-card-subtitle {
  @apply text-sm mb-3;
}
</style>
