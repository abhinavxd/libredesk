<script setup>
import { Badge } from '@shared-ui/components/ui/badge'

defineProps({
  count: { type: Number, default: 0 },
  ariaLabel: { type: String, default: '' },
  // success: unread mail. review: emails waiting in the review queue.
  tone: { type: String, default: 'success' }
})
</script>

<template>
  <Badge
    v-if="count > 0"
    class="ml-auto flex min-h-6 min-w-6 shrink-0 items-center justify-center rounded-full px-2 py-0.5 text-xs font-semibold leading-none tabular-nums shadow-sm max-md:min-h-7 max-md:min-w-7 max-md:px-2.5 max-md:text-sm"
    :class="tone === 'review' ? 'border-review bg-review text-review-foreground hover:bg-review' : 'border-success bg-success text-success-foreground'"
    :aria-label="ariaLabel"
  >
    {{ count > 99 ? '99+' : String(count) }}
  </Badge>
</template>
