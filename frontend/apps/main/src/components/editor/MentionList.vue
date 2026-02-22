<template>
  <div
    v-if="items.length > 0"
    class="mention-list bg-background border rounded-lg shadow-lg overflow-hidden max-h-60 overflow-y-auto"
  >
    <button
      v-for="(item, index) in items"
      :key="item.id"
      class="mention-item w-full text-left px-3 py-2 flex items-center gap-2 hover:bg-muted"
      :class="{ 'bg-muted': index === selectedIndex }"
      @click="selectItem(index)"
    >
      <span v-if="item.type === 'team'" class="text-lg">{{ item.emoji || '👥' }}</span>
      <Avatar v-else class="w-6 h-6">
        <AvatarImage :src="item.avatar_url" :alt="item.label" />
        <AvatarFallback class="text-xs">{{ getInitials(item.label) }}</AvatarFallback>
      </Avatar>
      <span class="flex-1 truncate">{{ item.label }}</span>
      <span class="text-xs text-muted-foreground">{{ getTypeLabel(item.type) }}</span>
    </button>
  </div>
  <div v-else-if="query" class="mention-list bg-background border rounded-lg shadow-lg p-3">
    <span class="text-sm text-muted-foreground">{{ $t('globals.messages.noResults') }}</span>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'

const { t } = useI18n()

const props = defineProps({
  items: {
    type: Array,
    default: () => []
  },
  command: {
    type: Function,
    required: true
  },
  query: {
    type: String,
    default: ''
  }
})

const selectedIndex = ref(0)

const getInitials = (name) => {
  if (!name) return '?'
  const parts = name.split(' ')
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

const getTypeLabel = (type) => {
  if (type === 'agent') return t('globals.terms.agent')
  if (type === 'team') return t('globals.terms.team')
  return type
}

const selectItem = (index) => {
  const item = props.items[index]
  if (item) {
    props.command({ id: item.id, type: item.type, label: item.label })
  }
}

const upHandler = () => {
  selectedIndex.value = (selectedIndex.value + props.items.length - 1) % props.items.length
}

const downHandler = () => {
  selectedIndex.value = (selectedIndex.value + 1) % props.items.length
}

const enterHandler = () => {
  selectItem(selectedIndex.value)
}

watch(
  () => props.items,
  () => {
    selectedIndex.value = 0
  }
)

// Scroll selected item into view on keyboard navigation
watch(selectedIndex, () => {
  nextTick(() => {
    document.querySelector('.mention-item.bg-muted')?.scrollIntoView({ block: 'nearest' })
  })
})

defineExpose({
  onKeyDown: ({ event }) => {
    if (event.key === 'ArrowUp') {
      upHandler()
      return true
    }
    if (event.key === 'ArrowDown') {
      downHandler()
      return true
    }
    if (event.key === 'Enter') {
      enterHandler()
      return true
    }
    return false
  }
})
</script>

<style scoped>
.mention-list {
  min-width: 200px;
}
</style>
