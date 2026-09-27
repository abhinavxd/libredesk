<template>
  <div
    v-if="items.length > 0"
    class="conversation-reference-list bg-background border rounded-lg shadow-lg overflow-hidden max-h-60 overflow-y-auto"
  >
    <button
      v-for="(item, index) in items"
      :key="item.id"
      class="conversation-reference-item w-full text-left px-3 py-2 hover:bg-muted"
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
  <div v-else class="conversation-reference-list bg-background border rounded-lg shadow-lg p-3">
    <span v-if="query.length < MIN_REFERENCE_QUERY_LENGTH" class="text-sm text-muted-foreground">
      {{
        $t('search.minQueryLength', {
          length: MIN_REFERENCE_QUERY_LENGTH
        })
      }}
    </span>
    <span v-else class="text-sm text-muted-foreground">{{
      $t('globals.messages.noResultsFound')
    }}</span>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { MIN_REFERENCE_QUERY_LENGTH } from './conversationReference'

const props = defineProps({
  items: { type: Array, default: () => [] },
  command: { type: Function, required: true },
  query: { type: String, default: '' }
})
const selectedIndex = ref(0)
const selectItem = (index) => props.items[index] && props.command(props.items[index])
watch(
  () => props.items,
  () => {
    selectedIndex.value = 0
  }
)
watch(selectedIndex, () =>
  nextTick(() =>
    document.querySelector('.conversation-reference-item.bg-muted')?.scrollIntoView({ block: 'nearest' })
  )
)
defineExpose({
  onKeyDown: ({ event }) => {
    if (!props.items.length) return false
    if (event.key === 'ArrowUp')
      selectedIndex.value = (selectedIndex.value + props.items.length - 1) % props.items.length
    else if (event.key === 'ArrowDown')
      selectedIndex.value = (selectedIndex.value + 1) % props.items.length
    else if (event.key === 'Enter') selectItem(selectedIndex.value)
    else return false
    return true
  }
})
</script>

<style scoped>
.conversation-reference-list {
  min-width: 240px;
  max-width: 360px;
}
</style>
