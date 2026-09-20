<template>
  <div class="flex justify-between flex-1 min-h-0">
    <div class="relative pr-6" :class="fixedHelp ? 'flex-1 min-w-0' : 'w-full xl:w-8/12'">
      <slot name="content" />
    </div>
    <div
      class="rounded-md p-2 space-y-2 self-stretch text-sm text-foreground/70"
      :class="fixedHelp ? 'hidden xl:block w-[432px] shrink-0' : 'hidden lg:block w-3/12'"
    >
      <slot name="help" />
    </div>
  </div>
</template>

<script setup>
defineProps({
  fixedHelp: { type: Boolean, default: false }
})
</script>
