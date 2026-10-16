<template>
  <div class="relative isolate h-screen overflow-y-auto overscroll-contain bg-background">
    <div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      <div class="auth-grid absolute inset-0" />
      <div class="auth-light absolute inset-0" />
    </div>
    <div class="relative flex min-h-full flex-col">
      <main class="flex flex-1 items-center justify-center px-4 py-10 sm:px-6 sm:py-16">
        <div class="auth-content w-full max-w-sm">
          <slot />
        </div>
      </main>
      <footer class="flex justify-center px-4 pb-6">
        <PoweredBy />
      </footer>
    </div>
  </div>
</template>

<script setup>
import PoweredBy from '@shared-ui/components/PoweredBy.vue'
</script>

<style scoped>
.auth-grid {
  background-image: linear-gradient(hsl(var(--border) / 0.35) 1px, transparent 1px),
    linear-gradient(90deg, hsl(var(--border) / 0.35) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(ellipse at center, transparent 20%, black 85%);
}

.auth-light {
  background: radial-gradient(ellipse at 50% 35%, hsl(var(--primary) / 0.08), transparent 65%);
}

.auth-content > :deep(*) {
  border-radius: calc(var(--radius) + 0.25rem);
  box-shadow:
    0 20px 60px -20px hsl(var(--foreground) / 0.15),
    0 0 0 4px hsl(var(--background) / 0.6);
}
</style>
