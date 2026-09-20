<template>
  <a :href="link.url" target="_blank" rel="noopener noreferrer" class="block no-underline">
    <Card
      class="rounded-xl border-border/80 shadow-sm transition-[background-color,box-shadow] can-hover:hover:bg-accent can-hover:hover:shadow-md"
    >
      <CardContent class="flex items-center gap-3 p-4">
        <span class="min-w-0 flex-1 text-sm font-medium leading-snug text-foreground">
          {{ link.text }}
        </span>
        <ExternalLink size="15" class="shrink-0 text-muted-foreground" aria-hidden="true" />
      </CardContent>
    </Card>
  </a>
</template>

<script setup>
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { ExternalLink } from 'lucide-vue-next'

defineProps({
  link: {
    type: Object,
    required: true
  }
})
</script>
