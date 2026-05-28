<template>
  <div class="w-full space-y-4 relative">
    <!-- Header -->
    <div class="flex items-center mb-4" :class="compact ? 'justify-end' : 'justify-between'">
      <span v-if="!compact" class="text-xl font-semibold text-foreground">
        {{ $t('globals.terms.note', 2) }}
      </span>
      <Button
        variant="outline"
        size="sm"
        @click="startAddingNote"
        v-if="!isAddingNote && !isLoading && (notes.length !== 0 || compact) && userStore.can('contact_notes:write')"
        class="transition-all hover:bg-primary/10 hover:border-primary/30"
      >
        <PlusIcon size="18" />
        {{ $t('contact.newNote') }}
      </Button>
    </div>

    <div
      v-if="isLoading"
      class="flex items-center justify-center"
      :class="compact ? 'py-4' : 'h-52'"
    >
      <Spinner :absolute="false" />
    </div>

    <!-- Note input -->
    <div v-if="isAddingNote && userStore.can('contact_notes:write')">
      <form @submit.prevent="addOrUpdateNote" @keydown.ctrl.enter="addOrUpdateNote">
        <div class="space-y-4">
          <div class="box p-2 h-52 min-h-52">
            <Editor
              v-model:htmlContent="newNote"
              @update:htmlContent="(value) => (newNote = value)"
              :placeholder="t('editor.hint.newLineSend')"
            />
          </div>
          <div class="flex justify-end space-x-3 pt-2">
            <Button variant="outline" @click="cancelAddNote"> {{ $t('globals.messages.cancel') }} </Button>
            <Button type="submit" :disabled="!newNote.trim()">
              {{ $t('contact.saveNote') }}
            </Button>
          </div>
        </div>
      </form>
    </div>

    <!-- Notes card list -->
    <div class="space-y-4">
      <Card
        v-for="note in visibleNotes"
        :key="note.id"
        class="overflow-hidden hover:border-border transition-all duration-200 box hover:shadow"
      >
        <!-- Header -->
        <CardHeader :class="compact ? 'p-3 pb-2' : 'bg-background border-b p-2'">
          <div class="flex items-center justify-between">
            <div class="flex items-center" :class="compact ? 'space-x-2 min-w-0' : 'space-x-3'">
              <Avatar :class="compact ? 'h-5 w-5' : 'border shadow-sm'">
                <AvatarImage :src="note.avatar_url" />
                <AvatarFallback :class="{ 'text-[10px]': compact }">
                  {{ getInitials(note.first_name, note.last_name) }}
                </AvatarFallback>
              </Avatar>
              <div v-if="compact" class="flex items-center gap-1.5 min-w-0 text-xs">
                <span class="font-medium text-foreground truncate">
                  {{ note.first_name }} {{ note.last_name }}
                </span>
                <span class="text-muted-foreground">·</span>
                <span class="text-muted-foreground truncate" :title="formatDate(note.created_at)">
                  {{ relativeDate(note.created_at) }}
                </span>
              </div>
              <div v-else>
                <p class="text-sm font-medium text-foreground">
                  {{ note.first_name }} {{ note.last_name }}
                </p>
                <p class="text-xs text-muted-foreground flex items-center">
                  <ClockIcon class="h-3 w-3 mr-1 inline-block opacity-70" />
                  {{ formatDate(note.created_at) }}
                </p>
              </div>
            </div>
            <!-- Allow owner and `Admin` to delete any note -->
            <DropdownMenu
              v-if="
                (userStore.can('contact_notes:delete') && note.user_id === userStore.userID) ||
                userStore.hasAdminRole
              "
            >
              <DropdownMenuTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  class="rounded-full"
                  :class="compact ? 'h-6 w-6 -mr-1' : 'h-8 w-8'"
                >
                  <MoreVerticalIcon :class="compact ? 'h-3.5 w-3.5' : 'h-4 w-4'" />
                  <span class="sr-only">{{ $t('globals.terms.openMenu') }}</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-[180px]">
                <DropdownMenuItem
                  @click="deleteNote(note.id)"
                  class="text-destructive cursor-pointer"
                >
                  <TrashIcon class="mr-2" size="15" />
                  {{
                    $t('contact.deleteNote')
                  }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardHeader>

        <!-- Note content -->
        <CardContent :class="compact ? 'px-3 pb-3 pt-0' : 'pt-4 pb-5'">
          <Letter
            :html="note.note"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="whitespace-pre-wrap text-sm native-html"
          />
        </CardContent>
      </Card>
      <!-- Load more notes -->
      <div v-if="compact && notes.length > NOTES_LIMIT && !showAll" class="flex justify-center pt-2">
       <Button variant="ghost" size="sm" @click="showAll = true">
         {{ $t('globals.terms.loadMore') }} ({{ notes.length - NOTES_LIMIT }})
       </Button>
      </div>
    </div>

    <!-- No notes message -->
    <template v-if="showEmpty">
      <div v-if="compact" class="text-center text-sm text-muted-foreground py-4">
        {{ $t('contact.notes.empty') }}
      </div>
      <div v-else class="box border-dashed p-10 text-center bg-muted/50 mt-6">
        <div class="flex flex-col items-center">
          <div class="rounded-full bg-muted p-4 mb-2">
            <MessageSquareIcon class="text-muted-foreground" size="25" />
          </div>
          <h3 class="mt-2 text-base font-medium text-foreground">
            {{ $t('contact.notes.empty') }}
          </h3>
          <p class="mt-1 text-sm text-muted-foreground max-w-sm mx-auto">
            {{ $t('contact.notes.help') }}
          </p>
          <Button
            v-if="userStore.can('contact_notes:write')"
            variant="outline"
            class="mt-3"
            @click="startAddingNote"
          >
            <PlusIcon size="15" />
            {{ $t('contact.addNote') }}
          </Button>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
const notesCache = new Map()
</script>

<script setup>
import { ref, watch, computed } from 'vue'
import { format, formatDistanceToNow } from 'date-fns'
import { Button } from '@shared-ui/components/ui/button'
import { Card, CardHeader, CardContent } from '@shared-ui/components/ui/card'
import { Avatar, AvatarImage, AvatarFallback } from '@shared-ui/components/ui/avatar'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'
import {
  PlusIcon,
  MoreVerticalIcon,
  TrashIcon,
  ClockIcon,
  MessageSquareIcon
} from 'lucide-vue-next'
import Editor from '@main/components/editor/TextEditor.vue'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { getInitials } from '@shared-ui/utils/string'
import { useUserStore } from '@main/stores/user'
import { Letter } from 'vue-letter'
import api from '@main/api'

const props = defineProps({ contactId: Number, compact: { type: Boolean, default: false } })

const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()

const notes = ref([])
const isAddingNote = ref(false)
const newNote = ref('')
const isLoading = ref(false)
const NOTES_LIMIT = 10
const showAll = ref(false)
const latestFetchId = ref(0)

const fetchNotes = async (contactId = props.contactId, { useCache = true } = {}) => {
  if (!contactId) return
  const fetchId = ++latestFetchId.value

  if (useCache && notesCache.has(contactId)) {
    notes.value = notesCache.get(contactId)
    isLoading.value = false
    return
  }

  try {
    isLoading.value = true
    const { data } = await api.getContactNotes(contactId)
    if (fetchId !== latestFetchId.value) return
    notes.value = data.data
    notesCache.set(contactId, data.data)
  } catch (error) {
    if (fetchId !== latestFetchId.value) return
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    if (fetchId === latestFetchId.value) {
      isLoading.value = false
    }
  }
}

const formatDate = (date) => format(new Date(date), 'PPP p')
const relativeDate = (date) => formatDistanceToNow(new Date(date), { addSuffix: true })

const startAddingNote = () => {
  isAddingNote.value = true
}

const cancelAddNote = () => {
  isAddingNote.value = false
  newNote.value = ''
}

const addOrUpdateNote = async () => {
  const targetId = props.contactId
  if (!targetId) return
  try {
    await api.createContactNote(targetId, { note: newNote.value })
    notesCache.delete(targetId)
    await fetchNotes(props.contactId, { useCache: false })
    cancelAddNote()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const deleteNote = async (noteId) => {
  const targetId = props.contactId
  if (!targetId) return
  try {
    await api.deleteContactNote(targetId, noteId)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    notesCache.delete(targetId)
    await fetchNotes(props.contactId, { useCache: false })
  }
}

const visibleNotes = computed(() => {
  if (!props.compact || showAll.value) return notes.value
  return notes.value.slice(0, NOTES_LIMIT)
})

const showEmpty = computed(() => notes.value.length === 0 && !isAddingNote.value && !isLoading.value)

watch(() => props.contactId, (newId) => {
  latestFetchId.value++
  showAll.value = false
  cancelAddNote()
  if (!newId) {
    notes.value = []
    isLoading.value = false
    return
  }
  if (!notesCache.has(newId)) notes.value = []
  fetchNotes(newId)
}, { immediate: true })
</script>
