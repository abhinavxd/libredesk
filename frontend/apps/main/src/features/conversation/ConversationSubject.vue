<template>
  <div v-if="canEdit || subject" class="hidden md:flex items-center min-w-0 gap-1">
    <span class="text-muted-foreground select-none" aria-hidden="true">·</span>
    <Input
      v-if="isEditing"
      ref="inputRef"
      v-model="draft"
      :maxlength="SUBJECT_MAX_LENGTH"
      :aria-label="t('conversation.editSubject')"
      class="h-7 py-0 text-sm min-w-0 w-56"
      @keydown.enter.prevent="commit"
      @keydown.esc.prevent="cancel"
      @blur="commit"
    />
    <button
      v-else-if="canEdit"
      type="button"
      class="truncate text-sm text-muted-foreground rounded-md px-1 py-0.5 -mx-1 hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      :title="subject || t('conversation.addSubject')"
      @click="startEditing"
    >
      {{ subject || t('conversation.addSubject') }}
    </button>
    <span v-else class="truncate text-sm text-muted-foreground" :title="subject">{{ subject }}</span>
  </div>
</template>

<script setup>
import { computed, nextTick, ref } from 'vue'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { Input } from '@shared-ui/components/ui/input'
import { SUBJECT_MAX_LENGTH } from '@main/constants/conversation'
import { permissions as perms } from '@main/constants/permissions.js'
import { useI18n } from 'vue-i18n'

const conversationStore = useConversationStore()
const userStore = useUserStore()
const { t } = useI18n()

const isEditing = ref(false)
const draft = ref('')
const inputRef = ref(null)

const subject = computed(() => conversationStore.current?.subject || '')
const canEdit = computed(() => userStore.can(perms.CONVERSATIONS_UPDATE_SUBJECT))

const startEditing = async () => {
  draft.value = subject.value
  isEditing.value = true
  await nextTick()
  inputRef.value?.$el?.focus()
  inputRef.value?.$el?.select()
}

// Escape closes the editor before the blur handler runs, so commit() has nothing left to save.
const cancel = () => {
  isEditing.value = false
  draft.value = ''
}

const commit = () => {
  if (!isEditing.value) return
  const next = draft.value.trim().replace(/\s+/g, ' ')
  isEditing.value = false
  draft.value = ''
  if (next === subject.value) return
  conversationStore.updateSubject(next).catch(() => {})
}
</script>
