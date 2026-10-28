<template>
  <div
    class="admin-split grid w-full flex-1 min-h-0 items-start gap-6"
    :class="{ 'admin-split-preview': fixedHelp }"
  >
    <div class="relative min-w-0">
      <slot name="content" />
    </div>
    <aside
      v-if="$slots.help"
      class="min-w-0 space-y-3 text-sm leading-relaxed text-muted-foreground [overflow-wrap:anywhere]"
      :class="fixedHelp ? 'admin-preview' : 'admin-help rounded-lg bg-muted p-4 dark:bg-muted/40'"
    >
      <slot name="help" />
    </aside>
  </div>
</template>

<script setup>
defineProps({
  fixedHelp: { type: Boolean, default: false }
})
</script>

<style scoped>
.admin-split {
  grid-template-columns: minmax(0, 1fr);
  align-content: start;
}

@container (min-width: 56rem) {
  .admin-split:has(.admin-help) {
    grid-template-columns: minmax(0, 1fr) 16rem;
  }

  .admin-split-preview:has(.admin-preview) {
    grid-template-columns: minmax(0, 1fr) 27rem;
  }

  .admin-help,
  .admin-preview {
    position: sticky;
    top: 0;
  }
}

@container (min-width: 94rem) {
  .admin-split-preview:has(.admin-preview) {
    grid-template-columns: minmax(0, 1fr) 51.25rem;
  }
}
</style>
