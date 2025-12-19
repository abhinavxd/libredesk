<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>Merge Conversations</DialogTitle>
        <DialogDescription>
          Search for conversations to merge into this one. This action cannot be undone.
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <Input
          v-model="searchQuery"
          placeholder="Search by reference #, subject, or contact..."
          @input="debouncedSearch"
        />

        <div v-if="isSearching" class="flex justify-center py-4">
          <Spinner />
        </div>

        <div v-else-if="searchResults.length > 0" class="max-h-64 overflow-y-auto space-y-2">
          <div
            v-for="conv in searchResults"
            :key="conv.uuid"
            class="flex items-center gap-3 p-3 border rounded-lg cursor-pointer hover:bg-muted"
            :class="{ 'border-primary bg-primary/5': selectedUuids.includes(conv.uuid) }"
            @click="toggleSelection(conv.uuid)"
          >
            <Checkbox :checked="selectedUuids.includes(conv.uuid)" />
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2">
                <span class="font-mono text-sm text-muted-foreground">#{{ conv.reference_number }}</span>
                <Badge variant="outline" class="text-xs">{{ conv.status }}</Badge>
              </div>
              <div class="text-sm font-medium truncate">{{ conv.subject || 'No subject' }}</div>
              <div class="text-xs text-muted-foreground">
                {{ conv.contact.first_name }} {{ conv.contact.last_name }}
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="searchQuery && !isSearching" class="text-center py-4 text-muted-foreground">
          No conversations found
        </div>
      </div>

      <DialogFooter class="flex gap-2">
        <Button variant="outline" @click="$emit('update:open', false)">Cancel</Button>
        <Button
          variant="destructive"
          :disabled="selectedUuids.length === 0 || isMerging"
          @click="confirmMerge"
        >
          <Spinner v-if="isMerging" class="mr-2 h-4 w-4" />
          Merge {{ selectedUuids.length }} conversation{{ selectedUuids.length !== 1 ? 's' : '' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="showConfirm" @update:open="showConfirm = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Confirm Merge</AlertDialogTitle>
        <AlertDialogDescription>
          You are about to merge {{ selectedUuids.length }} conversation(s) into this one.
          All messages will be combined and the merged conversations will be closed.
          This action cannot be undone.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction @click="executeMerge" class="bg-destructive text-destructive-foreground hover:bg-destructive/90">
          Merge
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@/utils/http'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'

const props = defineProps({
  open: Boolean,
  conversationUuid: String
})

const emit = defineEmits(['update:open', 'merged'])

const emitter = useEmitter()
const searchQuery = ref('')
const searchResults = ref([])
const selectedUuids = ref([])
const isSearching = ref(false)
const isMerging = ref(false)
const showConfirm = ref(false)

const search = async () => {
  if (!searchQuery.value.trim()) {
    searchResults.value = []
    return
  }

  isSearching.value = true
  try {
    const resp = await api.searchConversationsForMerge(props.conversationUuid, searchQuery.value)
    searchResults.value = resp.data.data || []
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    isSearching.value = false
  }
}

const debouncedSearch = useDebounceFn(search, 300)

const toggleSelection = (uuid) => {
  const idx = selectedUuids.value.indexOf(uuid)
  if (idx === -1) {
    selectedUuids.value.push(uuid)
  } else {
    selectedUuids.value.splice(idx, 1)
  }
}

const confirmMerge = () => {
  showConfirm.value = true
}

const executeMerge = async () => {
  showConfirm.value = false
  isMerging.value = true

  try {
    await api.mergeConversations(props.conversationUuid, selectedUuids.value)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: `Merged ${selectedUuids.value.length} conversation(s) successfully`
    })
    emit('merged')
    emit('update:open', false)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    isMerging.value = false
  }
}

watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    searchQuery.value = ''
    searchResults.value = []
    selectedUuids.value = []
  }
})
</script>
