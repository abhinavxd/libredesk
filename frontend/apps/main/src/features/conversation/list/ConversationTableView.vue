<template>
  <table class="conversation-table w-full text-sm table-fixed border-collapse">
    <colgroup>
      <col :style="{ width: columnWidths.checkbox + 'px' }">
      <col v-for="col in resizableCols" :key="'col-' + col" :style="{ width: columnWidths[col] + 'px' }">
    </colgroup>

    <TableHeader class="sticky top-0 bg-background z-10">
      <TableRow class="hover:bg-transparent">
        <TableHead class="px-2 py-2">
          <Checkbox
            :checked="conversationStore.allSelected"
            @update:checked="toggleSelectAll"
            :aria-label="t('conversation.bulkActions.selectAll')"
          />
        </TableHead>
        <TableHead
          v-for="col in resizableCols"
          :key="'th-' + col"
          class="px-2 py-2 text-xs font-medium relative select-none group/th"
          :class="col === 'updated' ? 'text-right' : ''"
        >
          {{ t('conversation.list.column.' + col) }}
          <div
            class="absolute -right-px top-1 bottom-1 w-1 cursor-col-resize border-r-2 border-transparent hover:border-primary/50 active:border-primary z-20 group-hover/th:border-muted-foreground/25"
            :title="t('conversation.list.column.resetWidth')"
            @pointerdown.prevent="startResize($event, col)"
            @dblclick="resetColumn(col)"
          />
        </TableHead>
      </TableRow>
    </TableHeader>

    <TableBody class="divide-y">
      <TableRow
        v-for="conversation in conversations"
        :key="conversation.uuid"
        class="conversation-table-row cursor-pointer transition-colors hover:bg-accent/20"
        :class="{
          'bg-accent/60': conversation.uuid === conversationStore.current?.uuid,
          'bg-primary/5': conversationStore.isSelected(conversation.uuid) && conversation.uuid !== conversationStore.current?.uuid
        }"
        @click="onRowClick(conversation)"
      >
        <!-- Checkbox -->
        <TableCell
          class="px-2 py-2"
          @click.stop="(e) => conversationStore.toggleSelect(conversation.uuid, e.shiftKey)"
        >
          <Checkbox
            :checked="conversationStore.isSelected(conversation.uuid)"
            tabindex="-1"
            class="pointer-events-none"
            :aria-label="t('conversation.bulkActions.selectConversation')"
          />
        </TableCell>

        <!-- Contact -->
        <TableCell class="px-2 py-2">
          <div class="flex items-center gap-2 min-w-0">
            <Avatar class="w-6 h-6 rounded-full shrink-0">
              <AvatarImage
                v-if="conversation.contact.avatar_url"
                :src="conversation.contact.avatar_url"
              />
              <AvatarFallback class="text-xs">
                {{ initials(conversation.contact) }}
              </AvatarFallback>
            </Avatar>
            <span class="truncate text-xs font-medium">
              {{ conversationStore.getContactFullName(conversation.uuid) }}
            </span>
          </div>
        </TableCell>

        <!-- Subject -->
        <TableCell class="px-2 py-2">
          <div class="flex items-center gap-1.5 min-w-0">
            <span
              v-if="conversation.reference_number"
              class="text-xs text-muted-foreground whitespace-nowrap"
            >
              #{{ conversation.reference_number }}
            </span>
            <span class="truncate font-medium text-xs">
              {{ conversation.subject || t('conversation.list.noSubject') }}
            </span>
            <div
              v-if="conversation.unread_message_count > 0"
              class="shrink-0 w-4 h-4 flex items-center justify-center bg-primary text-primary-foreground text-xs font-medium rounded-full"
            >
              {{ conversation.unread_message_count }}
            </div>
          </div>
        </TableCell>

        <!-- Status -->
        <TableCell class="px-2 py-2">
          <span class="text-xs text-muted-foreground truncate">
            {{ conversation.status }}
          </span>
        </TableCell>

        <!-- Priority -->
        <TableCell class="px-2 py-2">
          <div class="flex items-center gap-1.5">
            <span
              class="w-2 h-2 rounded-full shrink-0"
              :class="priorityDotClass(conversation.priority)"
            ></span>
            <span class="text-xs text-muted-foreground truncate">
              {{ conversation.priority || '—' }}
            </span>
          </div>
        </TableCell>

        <!-- Updated -->
        <TableCell class="px-2 py-2 text-right">
          <span class="text-xs text-muted-foreground whitespace-nowrap">
            {{ getRelativeTime(conversation.last_message_at, now) }}
          </span>
        </TableCell>
      </TableRow>
    </TableBody>
  </table>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import { Avatar, AvatarFallback, AvatarImage } from '@shared-ui/components/ui/avatar'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import { TableBody, TableCell, TableHead, TableHeader, TableRow } from '@shared-ui/components/ui/table'
import { getRelativeTime } from '@shared-ui/utils/datetime.js'
import { useConversationStore } from '@/stores/conversation'
import { useConversationRoute } from '@/composables/useConversationRoute'

const { t } = useI18n()
const conversationStore = useConversationStore()
const router = useRouter()
const { buildConversationRoute } = useConversationRoute()

const now = ref(new Date())
let timer = null

defineProps({
  conversations: { type: Array, required: true }
})

const resizableCols = ['contact', 'subject', 'status', 'priority', 'updated']

// Column widths persist per-browser.
const defaultWidths = {
  checkbox: 36,
  contact: 130,
  subject: 220,
  status: 100,
  priority: 100,
  updated: 90
}
const MIN_WIDTH = 60
const columnWidths = useStorage('conversationTableColumnWidths', { ...defaultWidths }, undefined, {
  mergeDefaults: true
})

// Pointer-driven resize. Pointer events cover mouse, pen, and touch.
let resizingCol = null
let startX = 0
let startWidth = 0

const startResize = (event, col) => {
  resizingCol = col
  startX = event.clientX
  startWidth = columnWidths.value[col]
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}

const onPointerMove = (event) => {
  if (!resizingCol) return
  const delta = event.clientX - startX
  columnWidths.value[resizingCol] = Math.max(MIN_WIDTH, startWidth + delta)
}

const onPointerUp = () => {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  resizingCol = null
}

const resetColumn = (col) => {
  columnWidths.value[col] = defaultWidths[col]
}

onMounted(() => {
  timer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  // Defensive cleanup in case the user unmounts mid-drag.
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
})

const toggleSelectAll = () => {
  if (conversationStore.allSelected) {
    conversationStore.clearSelection()
  } else {
    conversationStore.selectAll()
  }
}

const initials = (contact) => {
  return contact?.first_name?.substring(0, 2)?.toUpperCase() || '?'
}

const priorityDotClass = (priority) => {
  switch ((priority || '').toLowerCase()) {
    case 'urgent': return 'bg-red-500'
    case 'high': return 'bg-orange-500'
    case 'medium': return 'bg-yellow-500'
    case 'low': return 'bg-blue-500'
    default: return 'bg-muted'
  }
}

const onRowClick = (conversation) => {
  router.push(buildConversationRoute(conversation))
}
</script>
