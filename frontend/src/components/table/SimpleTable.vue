<template>
  <table class="min-w-full table-fixed divide-y divide-border">
    <thead class="bg-muted">
      <tr>
        <th
          v-for="(header, index) in headers"
          :key="index"
          scope="col"
          class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
        >
          {{ header }}
        </th>
        <th v-if="showDelete" scope="col" class="relative px-6 py-3"></th>
      </tr>
    </thead>
    <tbody class="bg-background divide-y divide-border">
      <!-- Loading State -->
      <template v-if="loading">
        <tr v-for="i in skeletonRows" :key="`skeleton-${i}`" class="hover:bg-accent">
          <td
            v-for="(header, index) in headers"
            :key="`skeleton-cell-${index}`"
            class="px-6 py-3 text-sm font-medium text-foreground whitespace-normal break-words"
          >
            <Skeleton class="h-4 w-[85%]" />
          </td>
          <td v-if="showDelete" class="px-6 py-4 text-sm text-muted-foreground">
            <Skeleton class="h-8 w-8 rounded" />
          </td>
        </tr>
      </template>

      <!-- No Results State -->
      <template v-else-if="data.length === 0">
        <tr>
          <td :colspan="headers.length + (showDelete ? 1 : 0)" class="px-6 py-12 text-center">
            <div class="flex flex-col items-center space-y-4">
              <span class="text-md text-muted-foreground">
                {{
                  $t('globals.messages.noResults', {
                    name: $t('globals.terms.result', 2).toLowerCase()
                  })
                }}
              </span>
            </div>
          </td>
        </tr>
      </template>

      <!-- Data Rows -->
      <template v-else>
        <tr v-for="(item, index) in data" :key="index" class="hover:bg-accent">
          <td
            v-for="key in keys"
            :key="key"
            class="px-6 py-4 text-sm font-medium text-foreground whitespace-normal break-words"
          >
            {{ item[key] }}
          </td>
          <td v-if="showDelete" class="px-6 py-4 text-sm text-muted-foreground">
            <Button size="xs" variant="ghost" @click.prevent="deleteItem(item)">
              <Trash2 class="h-4 w-4" />
            </Button>
          </td>
        </tr>
      </template>
    </tbody>
  </table>
</template>

<script setup>
import { Trash2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

defineProps({
  headers: {
    type: Array,
    required: true,
    default: () => []
  },
  keys: {
    type: Array,
    required: true,
    default: () => []
  },
  data: {
    type: Array,
    required: true,
    default: () => []
  },
  showDelete: {
    type: Boolean,
    default: true
  },
  loading: {
    type: Boolean,
    default: false
  },
  skeletonRows: {
    type: Number,
    default: 5
  }
})

const emit = defineEmits(['deleteItem'])

function deleteItem(item) {
  emit('deleteItem', item)
}
</script>
