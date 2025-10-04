<template>
  <div>
    <div
      v-for="index in count"
      :key="index"
      class="flex items-center space-x-4 mb-4"
      :class="index % 2 === 0 ? 'justify-end' : 'justify-start'"
    >
      <template v-if="index % 2 !== 0">
        <Skeleton class="h-12 w-12 rounded-full" />
        <div class="space-y-2">
          <Skeleton class="h-4 w-[250px]" />
          <Skeleton class="h-4 w-[200px]" />
        </div>
      </template>

      <template v-else>
        <div class="space-y-2">
          <Skeleton class="h-4 w-[250px]" />
          <Skeleton class="h-4 w-[200px]" />
        </div>
        <Skeleton class="h-12 w-12 rounded-full" />
      </template>
    </div>
  </div>
</template>

<script setup>
import { Skeleton } from '@shared-ui/components/ui/skeleton'

defineProps({
  count: Number,
})
</script>
