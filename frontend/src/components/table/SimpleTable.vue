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
        <th scope="col" class="relative px-6 py-3"></th>
      </tr>
    </thead>
    <tbody class="bg-background divide-y divide-border">
      <template v-if="data.length === 0">
        <tr>
          <td :colspan="headers.length + 1" class="px-6 py-12 text-center">
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
import { defineProps, defineEmits } from 'vue'
import { Button } from '@/components/ui/button'

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
  }
})

const emit = defineEmits(['deleteItem'])

function deleteItem(item) {
  emit('deleteItem', item)
}
</script>
