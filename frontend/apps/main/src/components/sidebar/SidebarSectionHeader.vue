<template>
  <SidebarHeader class="h-14 shrink-0 flex-row items-center justify-between gap-2 px-3">
    <div class="flex min-w-0 items-baseline gap-1.5">
      <h2 class="ui-heading truncate">{{ title }}</h2>
      <div v-if="$slots.suffix" class="shrink-0 whitespace-nowrap text-xs text-muted-foreground">
        <slot name="suffix" />
      </div>
    </div>
    <div v-if="$slots.actions" class="flex shrink-0 items-center gap-2">
      <slot name="actions" />
    </div>
  </SidebarHeader>
</template>

<script setup>
import { SidebarHeader } from '@shared-ui/components/ui/sidebar'

defineProps({ title: { type: String, required: true } })
</script>
