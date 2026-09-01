<template>
  <div v-if="canMerge">
    <Button
      v-if="!global"
      variant="outline"
      size="sm"
      :class="compact ? 'h-7 text-xs shrink-0 px-2' : 'w-full justify-start'"
      :disabled="busy"
      @click="open = true"
    >
      <GitMerge class="size-3.5 mr-1.5" />
      {{ t('conversation.merge.action') }}
    </Button>

    <Dialog :open="open" @update:open="onOpen">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ t('conversation.merge.title') }}</DialogTitle>
          <DialogDescription>{{ t('conversation.merge.confirm') }}</DialogDescription>
        </DialogHeader>

        <Input
          v-model="query"
          type="search"
          :placeholder="t('conversation.merge.searchPlaceholder')"
          @input="onSearch"
        />

        <div class="max-h-64 overflow-y-auto rounded-md border divide-y">
          <button
            v-for="item in candidates"
            :key="item.uuid"
            type="button"
            class="w-full text-left px-3 py-2 text-sm hover:bg-muted"
            :class="{ 'bg-muted': selected === item.uuid }"
            @click="selected = item.uuid"
          >
            <span class="font-medium tabular-nums">#{{ item.reference_number }}</span>
            <span class="ml-2 text-muted-foreground">{{ item.subject || item.last_message || '' }}</span>
          </button>
          <p v-if="!candidates.length" class="px-3 py-6 text-sm text-muted-foreground">
            {{ t('conversation.merge.noCandidates') }}
          </p>
        </div>

        <label v-if="canMergeContacts" class="flex items-start gap-2 text-sm cursor-pointer">
          <Checkbox :checked="mergeContacts" @update:checked="mergeContacts = $event" class="mt-0.5" />
          <span>
            {{ t('conversation.merge.mergeContacts') }}
            <span class="block text-xs text-muted-foreground">{{ t('conversation.merge.mergeContactsHint') }}</span>
          </span>
        </label>

        <DialogFooter>
          <Button variant="outline" @click="open = false">{{ t('globals.messages.cancel') }}</Button>
          <Button :disabled="!selected || busy" @click="runMerge">
            {{ t('conversation.merge.action') }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { GitMerge } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { useConversationStore } from '@main/stores/conversation'
import { useUserStore } from '@main/stores/user'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'
import { conversationRouteForContext } from '@main/composables/useZendeskTabs'

const props = defineProps({
  compact: { type: Boolean, default: false },
  global: { type: Boolean, default: false }
})

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const userStore = useUserStore()

const open = ref(false)
const busy = ref(false)
const query = ref('')
const selected = ref('')
const extras = ref([])
const mergeContacts = ref(false)
const sourceUUID = ref('')

const canMerge = computed(() => userStore.can('conversations:write'))
const canMergeContacts = computed(() => userStore.can('contacts:merge'))
const currentUUID = computed(() => sourceUUID.value || conversationStore.current?.uuid)

const contactTickets = computed(() => {
  const list = conversationStore.current?.previous_conversations || []
  return list.filter((c) => c.uuid && c.uuid !== currentUUID.value)
})

const candidates = computed(() => {
  const seen = new Set()
  const out = []
  for (const item of [...extras.value, ...contactTickets.value]) {
    const uuid = item.uuid
    if (!uuid || uuid === currentUUID.value || seen.has(uuid)) continue
    seen.add(uuid)
    out.push(item)
  }
  return out
})

const onOpen = (value) => {
  open.value = value
  if (!value) {
    query.value = ''
    selected.value = ''
    extras.value = []
    mergeContacts.value = false
    sourceUUID.value = ''
  }
}

const onOpenMerge = (payload) => {
  sourceUUID.value = payload?.uuid || conversationStore.current?.uuid || ''
  open.value = true
}

let searchTimer
const onSearch = () => {
  clearTimeout(searchTimer)
  const q = query.value.trim().replace(/^#/, '')
  if (q.length < 3) {
    extras.value = []
    return
  }
  searchTimer = setTimeout(async () => {
    try {
      const { data } = await api.searchConversations({ query: q })
      extras.value = data.data || []
    } catch {
      extras.value = []
    }
  }, 250)
}

const runMerge = async () => {
  if (!currentUUID.value || !selected.value) return
  busy.value = true
  try {
    await api.mergeConversation(currentUUID.value, {
      target_uuid: selected.value,
      merge_contacts: canMergeContacts.value && mergeContacts.value
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('conversation.merge.success') })
    open.value = false
    await router.push(conversationRouteForContext(route, selected.value))
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    busy.value = false
  }
}

onMounted(() => {
  if (props.global) emitter.on(EMITTER_EVENTS.OPEN_MERGE_DIALOG, onOpenMerge)
})

onUnmounted(() => {
  if (props.global) emitter.off(EMITTER_EVENTS.OPEN_MERGE_DIALOG, onOpenMerge)
})
</script>
