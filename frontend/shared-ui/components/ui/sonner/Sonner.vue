<script setup>
import { Toaster as Sonner } from 'vue-sonner'
</script>

<template>
  <Sonner
    v-bind="$attrs"
    :toast-options="{
      classes: {
        toast:
          'flex items-start gap-3 rounded-md border shadow-lg bg-popover text-popover-foreground border-border',
        description: 'text-muted-foreground',
        closeButton: 'focus-visible:ring-2 focus-visible:ring-ring',
        icon: 'shrink-0'
      }
    }"
  />
</template>

<style>
[data-sonner-toast] [data-close-button] {
  --toast-close-button-start: auto;
  --toast-close-button-end: 0;
  --toast-close-button-transform: translate(35%, -35%);
  --gray1: hsl(var(--popover));
  --gray2: hsl(var(--accent));
  --gray4: hsl(var(--border));
  --gray5: hsl(var(--border));
  --gray12: hsl(var(--popover-foreground));
}

[data-sonner-toast][data-type='success'] [data-icon] {
  color: hsl(var(--success));
}
[data-sonner-toast][data-type='error'] [data-icon] {
  color: hsl(var(--destructive));
}
[data-sonner-toast][data-type='warning'] [data-icon] {
  color: hsl(var(--warning));
}
[data-sonner-toast][data-type='info'] [data-icon] {
  color: hsl(var(--link));
}
</style>
