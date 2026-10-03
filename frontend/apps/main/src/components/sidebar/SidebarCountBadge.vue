<script setup>
import { Badge } from '@shared-ui/components/ui/badge'

defineProps({
  count: { type: Number, default: 0 },
  ariaLabel: { type: String, default: '' },
  // success: unread mail. review: emails waiting in the review queue.
  tone: { type: String, default: 'success' },
  // corner pins a smaller badge to the top-right of a round icon button.
  corner: { type: Boolean, default: false }
})
</script>

<template>
  <Badge
    v-if="count > 0"
    class="flex shrink-0 items-center justify-center rounded-full font-semibold leading-none tabular-nums shadow-sm"
    :class="[
      corner
        ? 'pointer-events-none absolute -right-1.5 -top-1.5 min-h-4 min-w-4 px-1 py-0 text-xs ring-2 ring-sidebar'
        : 'ml-auto min-h-6 min-w-6 px-2 py-0.5 text-xs max-md:min-h-7 max-md:min-w-7 max-md:px-2.5 max-md:text-sm',
      tone === 'review' ? 'border-review bg-review text-review-foreground hover:bg-review' : 'border-success bg-success text-success-foreground'
    ]"
    :aria-label="ariaLabel"
  >
    {{ count > 99 ? '99+' : String(count) }}
  </Badge>
</template>
