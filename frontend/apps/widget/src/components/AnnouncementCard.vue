<template>
  <a :href="announcement.url" target="_blank" rel="noopener noreferrer" class="block no-underline">
    <Card class="overflow-hidden rounded-xl shadow-none hover:bg-accent transition-colors cursor-pointer">
      <img
        :src="announcement.image_url"
        :alt="announcement.title"
        class="w-full h-auto"
      />
      <CardContent class="p-3 text-sm">
        <div class="font-semibold">{{ announcement.title }}</div>
        <div v-if="announcement.description" class="text-muted-foreground mt-1 line-height-10">
          {{ announcement.description }}
        </div>
      </CardContent>
    </Card>
  </a>
</template>

<script setup>
import { Card, CardContent } from '@shared-ui/components/ui/card'

defineProps({
  announcement: {
    type: Object,
    required: true
  }
})
</script>
