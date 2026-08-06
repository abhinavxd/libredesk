<template>
  <div
    v-if="items.length > 0"
    class="ticket-reference-list bg-background border rounded-lg shadow-lg overflow-hidden max-h-60 overflow-y-auto"
  >
    <button
      v-for="(item, index) in items"
      :key="item.id"
      class="ticket-reference-item w-full text-left px-3 py-2 hover:bg-muted"
      :class="{ 'bg-muted': index === selectedIndex }"
      @click="selectItem(index)"
    >
      <div class="flex items-center gap-2">
        <span class="font-medium">#{{ item.label }}</span>
        <span class="text-xs text-muted-foreground">{{ item.status }}</span>
      </div>
      <div v-if="item.subject" class="text-sm text-muted-foreground truncate">
        {{ item.subject }}
      </div>
    </button>
  </div>
  <div
    v-else
    class="ticket-reference-list bg-background border rounded-lg shadow-lg p-3"
  >
    <span v-if="query.length < MIN_REFERENCE_QUERY_LENGTH" class="text-sm text-muted-foreground">
      Type at least {{ MIN_REFERENCE_QUERY_LENGTH }} digits to search tickets
    </span>
    <span v-else class="text-sm text-muted-foreground">
      {{ $t('globals.messages.noResultsFound') }}
    </span>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { MIN_REFERENCE_QUERY_LENGTH } from './ticketReference'

const props = defineProps({
  items: { type: Array, default: () => [] },
  command: { type: Function, required: true },
  query: { type: String, default: '' }
})

const selectedIndex = ref(0)

const selectItem = (index) => {
  const item = props.items[index]
  if (item) props.command(item)
}

watch(
  () => props.items,
  () => {
    selectedIndex.value = 0
  }
)

watch(selectedIndex, () => {
  nextTick(() => {
    document.querySelector('.ticket-reference-item.bg-muted')?.scrollIntoView({ block: 'nearest' })
  })
})

defineExpose({
  onKeyDown: ({ event }) => {
    if (props.items.length === 0) return false
    if (event.key === 'ArrowUp') {
      selectedIndex.value = (selectedIndex.value + props.items.length - 1) % props.items.length
      return true
    }
    if (event.key === 'ArrowDown') {
      selectedIndex.value = (selectedIndex.value + 1) % props.items.length
      return true
    }
    if (event.key === 'Enter') {
      selectItem(selectedIndex.value)
      return true
    }
    return false
  }
})
</script>

<style scoped>
.ticket-reference-list {
  min-width: 240px;
  max-width: 360px;
}
</style>
