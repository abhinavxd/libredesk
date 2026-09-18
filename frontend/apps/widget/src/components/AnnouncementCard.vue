<template>
  <a :href="announcement.url" target="_blank" rel="noopener noreferrer" class="block no-underline">
    <Card
      class="overflow-hidden rounded-xl border-border/80 transition-[background-color,box-shadow] can-hover:hover:bg-accent can-hover:hover:shadow-md"
    >
      <img :src="announcement.image_url" :alt="announcement.title" class="w-full h-auto" />
      <CardContent class="p-4 text-sm">
        <div class="font-semibold leading-snug">{{ announcement.title }}</div>
        <div v-if="announcement.description" class="mt-1 text-muted-foreground leading-relaxed">
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
