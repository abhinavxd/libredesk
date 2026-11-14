<script setup>
import { AvatarFallback } from 'radix-vue'

const props = defineProps({
  delayMs: { type: Number, required: false },
  asChild: { type: Boolean, required: false },
  as: { type: null, required: false }
})
</script>

<template>
  <AvatarFallback
    v-bind="props"
    class="avatar-initials flex size-full items-center justify-center font-medium"
  >
    <slot />
  </AvatarFallback>
</template>

<style scoped>
.avatar-initials {
  background-color: color-mix(in srgb, hsl(var(--primary)) 12%, hsl(var(--card)));
  color: hsl(var(--foreground));
  background-image:
    radial-gradient(circle at 27% 17%, hsl(var(--card) / 0.45), transparent 55%),
    radial-gradient(circle at 30% 30%, hsl(var(--primary) / 0.25), transparent 95%),
    radial-gradient(circle at 70% 70%, hsl(var(--success) / 0.2), transparent 90%);
  letter-spacing: 0.02em;
}
</style>
