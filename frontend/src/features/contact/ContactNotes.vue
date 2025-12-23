<template>
  <div class="w-full space-y-6 pb-8 relative">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4">
      <span class="text-xl font-semibold text-gray-900 dark:text-foreground">
        {{ $t('globals.terms.note', 2) }}
      </span>
      <Button
        variant="outline"
        size="sm"
        @click="startAddingNote"
        v-if="!isAddingNote && !isLoading && notes.length !== 0"
        class="transition-all hover:bg-primary/10 hover:border-primary/30"
      >
        <PlusIcon class="mr-2" size="18" />
        {{ $t('globals.messages.new', { name: $t('globals.terms.note') }) }}
      </Button>
    </div>

    <div class="h-52" v-if="isLoading">
      <Spinner />
    </div>

    <!-- Note input -->
    <div v-if="isAddingNote">
      <form @submit.prevent="addOrUpdateNote" @keydown.ctrl.enter="addOrUpdateNote">
        <div class="space-y-4">
          <div class="box p-2 h-52 min-h-52">
            <Editor
              v-model:htmlContent="newNote"
              @update:htmlContent="(value) => (newNote = value)"
              :placeholder="t('editor.newLine') + t('editor.send')"
            />
          </div>
          <div class="flex justify-end space-x-3 pt-2">
            <Button variant="outline" @click="cancelAddNote"> Cancel </Button>
            <Button type="submit" :disabled="!newNote.trim()">
              {{ $t('globals.messages.save') + ' ' + $t('globals.terms.note').toLowerCase() }}
            </Button>
          </div>
        </div>
      </form>
    </div>

    <!-- Notes card list -->
    <div class="space-y-4">
      <Card
        v-for="note in notes"
        :key="note.id"
        class="overflow-hidden border-gray-2 dark:hover:border-gray-700 hover:border-gray-300 transition-all duration-200 box hover:shadow"
      >
        <!-- Header -->
        <CardHeader class="bg-background border-b p-2">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-3">
              <Avatar class="border shadow-sm">
                <AvatarImage :src="note.avatar_url" />
                <AvatarFallback>
                  {{ getInitials(note.first_name, note.last_name) }}
                </AvatarFallback>
              </Avatar>
              <div>
                <p class="text-sm font-medium text-gray-900 dark:text-foreground">
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
                <Button variant="ghost" size="icon" class="h-8 w-8 rounded-full">
                  <MoreVerticalIcon class="h-4 w-4" />
                  <span class="sr-only">Open menu</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-[180px]">
                <DropdownMenuItem
                  @click="deleteNote(note.id)"
                  class="text-destructive cursor-pointer"
                >
                  <TrashIcon class="mr-2" size="15" />
                  {{
                    $t('globals.messages.delete', { name: $t('globals.terms.note').toLowerCase() })
                  }}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </CardHeader>

        <!-- Note content -->
        <CardContent class="pt-4 pb-5">
          <Letter
            :html="note.note"
            :allowedSchemas="['cid', 'https', 'http', 'mailto']"
            class="whitespace-pre-wrap text-sm native-html"
          />
        </CardContent>
      </Card>
    </div>

    <!-- No notes message -->
    <div
      v-if="notes.length === 0 && !isAddingNote && !isLoading"
      class="box border-dashed p-10 text-center bg-gray-50/50 mt-6 dark:bg-background"
    >
      <div class="flex flex-col items-center">
        <div class="rounded-full bg-gray-100 dark:bg-foreground p-4 mb-2">
          <MessageSquareIcon class="text-gray-400 dark:text-background" size="25" />
        </div>
        <h3 class="mt-2 text-base font-medium text-gray-900 dark:text-foreground">
          {{ $t('contact.notes.empty') }}
        </h3>
        <p class="mt-1 text-sm text-muted-foreground max-w-sm mx-auto">
          {{ $t('contact.notes.help') }}
        </p>
        <Button variant="outline" class="mt-3 border-gray-300" @click="startAddingNote">
          <PlusIcon class="mr-2" size="15" />
          {{ $t('globals.messages.add', { name: $t('globals.terms.note').toLowerCase() }) }}
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { format } from 'date-fns'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardContent } from '@/components/ui/card'
import { Avatar, AvatarImage, AvatarFallback } from '@/components/ui/avatar'
import { Spinner } from '@/components/ui/spinner'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem
} from '@/components/ui/dropdown-menu'
import {
  PlusIcon,
  MoreVerticalIcon,
  TrashIcon,
  ClockIcon,
  MessageSquareIcon
} from 'lucide-vue-next'
import Editor from '@/components/editor/TextEditor.vue'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { getInitials } from '@/utils/strings'
import { useUserStore } from '@/stores/user'
import { Letter } from 'vue-letter'
import api from '@/api'

const props = defineProps({ contactId: Number })
const { t } = useI18n()
const emitter = useEmitter()
const userStore = useUserStore()

const notes = ref([])
const isAddingNote = ref(false)
const newNote = ref('')
const isLoading = ref(false)

const fetchNotes = async () => {
  try {
    isLoading.value = true
    const { data } = await api.getContactNotes(props.contactId)
    notes.value = data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(fetchNotes)

const formatDate = (date) => format(new Date(date), 'PPP p')

const startAddingNote = () => {
  isAddingNote.value = true
}

const cancelAddNote = () => {
  isAddingNote.value = false
  newNote.value = ''
}

const addOrUpdateNote = async () => {
  try {
    await api.createContactNote(props.contactId, { note: newNote.value })
    await fetchNotes()
    cancelAddNote()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const deleteNote = async (noteId) => {
  try {
    await api.deleteContactNote(props.contactId, noteId)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    await fetchNotes()
  }
}
</script>
