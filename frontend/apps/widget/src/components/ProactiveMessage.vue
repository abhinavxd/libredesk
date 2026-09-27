<template>
  <div class="flex flex-col items-start">
    <div class="max-w-[85%] px-4 py-3 rounded-2xl text-sm leading-5 break-words bg-muted text-foreground whitespace-pre-wrap">{{ pending.snapshot.message }}</div>
    <div class="text-[10px] text-muted-foreground mt-1">{{ pending.snapshot.sender }}</div>
  </div>
</template>

<script setup>
defineProps({
  pending: { type: Object, required: true }
})
</script>
