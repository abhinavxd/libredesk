<template>
  <Badge
    variant="outline"
    class="rounded-full uppercase tracking-[0.07em] font-medium text-[9px] leading-none px-[5.5px] py-[3px] bg-warning/10 text-warning-600 border-warning/50 shrink-0"
  >
    <slot />
  </Badge>
</template>

<script setup>
import { Badge } from '@shared-ui/components/ui/badge'
</script>
