<script setup>
import { Toaster as Sonner } from 'vue-sonner';
</script>

<template>
  <Sonner
    v-bind="$attrs"
    :toast-options="{
      classes: {
        toast: 'flex items-start gap-3 rounded-md border shadow-lg bg-popover text-popover-foreground border-border',
        description: 'text-muted-foreground',
        icon: 'shrink-0',
      },
    }"
  />
</template>

<style>
[data-sonner-toast][data-type='success'] [data-icon] {
  color: var(--success-text);
}
[data-sonner-toast][data-type='error'] [data-icon] {
  color: hsl(var(--destructive));
}
[data-sonner-toast][data-type='warning'] [data-icon] {
  color: var(--warning-text);
}
[data-sonner-toast][data-type='info'] [data-icon] {
  color: var(--info-text);
}
</style>
