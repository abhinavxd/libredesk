<template>
  <div class="flex items-center gap-5 p-6 border-b min-w-[200px]">
    <Skeleton class="h-12 w-12 rounded-full aspect-square" />
    <div class="space-y-2 flex-grow">
      <Skeleton class="h-4 w-full max-w-[250px]" />
      <Skeleton class="h-4 w-full max-w-[200px]" />
    </div>
  </div>
</template>

<script setup>
import { Skeleton } from '@shared-ui/components/ui/skeleton'
</script>
