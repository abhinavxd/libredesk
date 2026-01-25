<template>
  <div v-if="config.external_links?.length" class="space-y-2">
    <a
      v-for="link in config.external_links"
      :key="link.url"
      :href="link.url"
      target="_blank"
      class="block no-underline"
    >
      <Card class="hover:bg-accent transition-colors cursor-pointer rounded-md">
        <CardContent class="p-4">
          <div class="flex justify-between items-center">
            <span class="text-sm text-primary font-medium">
              {{ link.text }}
            </span>
            <ExternalLink size="18" class="text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    </a>
  </div>
</template>

<script setup>
import { Card, CardContent } from '@shared-ui/components/ui/card'
import { ExternalLink } from 'lucide-vue-next'

defineProps({
  config: {
    type: Object,
    required: true
  }
})
</script>
