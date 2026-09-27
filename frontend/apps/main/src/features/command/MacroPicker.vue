<template>
  <!-- Mounted only with results: radix's group counts its items once on mount and stays hidden when they arrive async. -->
  <CommandGroup v-if="macros.length" class="flex h-full min-h-0 flex-col">
    <div class="min-h-0 flex-1">
      <div class="grid h-full min-h-0 grid-cols-12">
        <div class="col-span-4 h-full min-h-0 overflow-y-auto border-r border-border/70 pr-2">
          <CommandItem
            v-for="macro in macros"
            :key="macro.value"
            :value="itemValue(macro)"
            @select="onSelect($event, macro)"
            class="cursor-pointer"
          >
            <div class="flex min-w-0 items-center">
              <span class="text-sm w-full break-words whitespace-normal">{{ macro.label }}</span>
            </div>
          </CommandItem>
        </div>

        <div class="col-span-8 h-full min-h-0 overflow-y-auto px-5 pb-5 pt-2">
          <div class="flex min-h-full flex-col space-y-4 text-sm">
            <div v-if="contentPending" class="flex flex-1 items-center justify-center">
              <Spinner :absolute="false" />
            </div>
            <div v-else-if="replyContent" class="space-y-2">
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ $t('command.replyPreview') }}
              </p>
              <Letter
                :key="highlightedMacro?.value"
                :html="replyContent"
                :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                class="native-html min-h-[120px] w-full overflow-auto rounded-lg border bg-background p-4 shadow-sm"
              />
            </div>

            <div v-if="otherActions.length > 0" class="space-y-2">
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                {{ $t('globals.terms.action', 2) }}
              </p>
              <div class="grid gap-2 sm:grid-cols-2">
                <div
                  v-for="action in otherActions"
                  :key="action.type"
                  class="flex min-w-0 items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-sm"
                >
                  <div
                    class="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-background text-muted-foreground shadow-sm"
                  >
                    <component :is="ACTION_ICONS[action.type] || Tags" :size="14" class="shrink-0" />
                  </div>
                  <span class="truncate">{{ actionLabel(action) }}</span>
                </div>
              </div>
            </div>

            <div
              v-if="!contentPending && !replyContent && otherActions.length === 0"
              class="flex min-h-40 flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed bg-muted/20"
            >
              <p class="text-sm text-muted-foreground">
                {{ $t('command.selectAMacro') }}
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </CommandGroup>
</template>

<script setup>
const MACRO_VALUE_PREFIX = 'macro.'
const CONTENT_PREVIEW_DELAY = 150

import { computed, ref, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { Letter } from 'vue-letter'
import { Users, User, Pin, Rocket, Tags } from 'lucide-vue-next'
import { CommandGroup, CommandItem } from '@shared-ui/components/ui/command'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useConversationStore } from '@main/stores/conversation'
import { useMacroStore } from '@main/stores/macro'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'

const ACTION_ICONS = {
  assign_user: User,
  assign_team: Users,
  set_status: Pin,
  set_priority: Rocket
}

const props = defineProps({
  searchTerm: { type: String, default: '' },
  // Value of the highlighted command item, forwarded from the combobox root.
  highlightedValue: { type: String, default: '' },
  macroContext: { type: String, required: true }
})
const emit = defineEmits(['applied'])

const { t } = useI18n()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const macroStore = useMacroStore()

const macros = computed(() => macroStore.macroOptions)
const itemValue = (macro) => MACRO_VALUE_PREFIX + macro.value

const highlightedMacro = computed(() => {
  const value = props.highlightedValue
  if (!value?.startsWith(MACRO_VALUE_PREFIX)) return null
  const id = value.slice(MACRO_VALUE_PREFIX.length)
  return macros.value.find((macro) => macro.value === id) || null
})

const search = () => macroStore.searchMacros(props.searchTerm.trim())
const searchDebounced = useDebounceFn(search, 300)
watch(() => props.searchTerm, searchDebounced)
search()

const failedContentIDs = ref(new Set())
let contentFetchTimer = null

watch(highlightedMacro, (macro) => {
  clearTimeout(contentFetchTimer)
  if (!macro?.has_message_content || macro.id in macroStore.macroContents) return
  failedContentIDs.value.delete(macro.id)
  contentFetchTimer = setTimeout(() => {
    macroStore.fetchMacroContent(macro.id).catch(() => failedContentIDs.value.add(macro.id))
  }, CONTENT_PREVIEW_DELAY)
})

const replyContent = computed(() => {
  const macro = highlightedMacro.value
  return macro ? macroStore.macroContents[macro.id] || '' : ''
})

const contentPending = computed(() => {
  const macro = highlightedMacro.value
  if (!macro?.has_message_content) return false
  return !(macro.id in macroStore.macroContents) && !failedContentIDs.value.has(macro.id)
})

const otherActions = computed(
  () =>
    highlightedMacro.value?.actions?.filter(
      (a) => a.type !== 'send_private_note' && a.type !== 'send_reply'
    ) || []
)

const ACTION_LABEL_KEYS = {
  assign_user: 'actions.assignAgent',
  assign_team: 'actions.assignTeam',
  set_status: 'actions.setStatus',
  set_priority: 'actions.setPriority',
  add_tags: 'actions.addTags',
  set_tags: 'actions.setTags',
  remove_tags: 'actions.removeTags'
}

const actionLabel = (action) => {
  const values = action.display_value.length > 0 ? action.display_value : action.value
  return `${t(ACTION_LABEL_KEYS[action.type])}: ${values.join(', ')}`
}

const onSelect = async (event, macro) => {
  event.preventDefault()
  let messageContent = ''
  if (macro.has_message_content) {
    try {
      messageContent = await macroStore.fetchMacroContent(macro.id)
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
      return
    }
  }
  const plainMacro = JSON.parse(JSON.stringify(macro))
  plainMacro.message_content = messageContent
  conversationStore.setMacro(plainMacro, props.macroContext)
  emit('applied')
}

onUnmounted(() => clearTimeout(contentFetchTimer))
</script>
