<template>
  <CommandDialog
    :open="open"
    v-model:search-term="searchTerm"
    :filter-function="isMacroMode ? passThroughFilter : undefined"
    @update:open="toggleOpen"
    class="transform-gpu z-[51] !min-w-[50vw]"
  >
    <CommandInput :placeholder="t('command.typeCmdOrSearch')" @keydown="onInputKeydown" />
    <CommandList
      class="!min-h-[50vh] h-[50vh] !min-w-[50vw]"
      :class="{ 'overflow-hidden': nestedCommand === 'apply-macro' }"
    >
      <CommandEmpty>
        <p class="text-sm text-muted-foreground">{{ $t('command.noCommandAvailable') }}</p>
      </CommandEmpty>

      <!-- Snooze Options -->
      <CommandGroup v-if="nestedCommand === 'snooze'" :heading="t('command.snoozeFor')">
        <CommandItem value="1 hour" @select="handleSnooze(60)">
          1 {{ $t('globals.terms.hour') }}
        </CommandItem>
        <CommandItem value="3 hours" @select="handleSnooze(180)">
          3 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="6 hours" @select="handleSnooze(360)">
          6 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="12 hours" @select="handleSnooze(720)">
          12 {{ $t('globals.terms.hour', 2) }}
        </CommandItem>
        <CommandItem value="1 day" @select="handleSnooze(1440)">
          1 {{ $t('globals.terms.day') }}
        </CommandItem>
        <CommandItem value="2 days" @select="handleSnooze(2880)">
          2 {{ $t('globals.terms.day', 2) }}
        </CommandItem>
        <CommandItem value="3 days" @select="handleSnooze(4320)">
          3 {{ $t('globals.terms.day', 2) }}
        </CommandItem>
        <CommandItem value="1 week" @select="handleSnooze(10080)">
          1 {{ $t('globals.terms.week') }}
        </CommandItem>
        <CommandItem value="pick date & time" @select="showCustomDialog">
          {{ $t('globals.messages.pickDateAndTime') }}
        </CommandItem>
      </CommandGroup>

      <!-- Macros -->
      <div v-if="isMacroMode">
        <CommandGroup :heading="$t('actions.applyMacro')">
          <div class="min-h-[400px]">
            <div class="h-[60vh] grid grid-cols-12">
              <!-- Left Column: Macro List (30%) -->
              <div ref="macroListRef" class="col-span-4 pr-2 border-r overflow-y-auto h-full">
                <CommandItem
                  v-for="(macro, index) in visibleMacros"
                  :key="macro.value"
                  :value="macro.label + '|' + index"
                  :data-index="index"
                  @select="handleApplyMacro(macro)"
                  @pointerenter="highlightedMacro = macro"
                  class="px-2 py-2 rounded cursor-pointer transition-colors duration-150 hover:bg-accent"
                >
                  <div class="flex items-center gap-2">
                    <Zap :size="16" class="shrink-0" />
                    <span class="text-sm w-full break-words whitespace-normal">{{
                      macro.label
                    }}</span>
                  </div>
                </CommandItem>
              </div>

              <!-- Right Column: Macro Details (70%) -->
              <div class="col-span-8 px-4 overflow-y-auto h-full pb-6">
                <div class="space-y-3 text-sm">
                  <!-- Reply Preview -->
                  <div v-if="replyContent" class="space-y-2">
                    <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      {{ $t('command.replyPreview') }}
                    </p>
                    <Letter
                      :key="highlightedMacro?.value"
                      :html="replyContent"
                      :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                      class="w-full min-h-[120px] p-3 bg-muted/30 rounded-md border overflow-auto native-html"
                    />
                  </div>

                  <!-- Actions -->
                  <div v-if="otherActions.length > 0" class="space-y-2">
                    <p class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                      {{ $t('globals.terms.action', 2) }}
                    </p>
                    <div class="space-y-1.5">
                      <div
                        v-for="action in otherActions"
                        :key="action.type"
                        class="flex items-center gap-2 px-2.5 py-2 rounded-md text-sm bg-muted/50 hover:bg-accent hover:text-accent-foreground transition-colors duration-150 group"
                      >
                        <div
                          class="p-1 rounded-md bg-muted/30 group-hover:bg-muted/50 transition-colors duration-150"
                        >
                          <User v-if="action.type === 'assign_user'" :size="14" class="shrink-0" />
                          <Users
                            v-else-if="action.type === 'assign_team'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Pin
                            v-else-if="action.type === 'set_status'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Rocket
                            v-else-if="action.type === 'set_priority'"
                            :size="14"
                            class="shrink-0"
                          />
                          <Tags v-else :size="14" class="shrink-0" />
                        </div>
                        <span class="truncate">{{ getActionLabel(action) }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Empty State -->
                  <div
                    v-if="!replyContent && otherActions.length === 0"
                    class="flex flex-col items-center justify-center h-32 gap-2"
                  >
                    <Zap :size="20" class="text-muted-foreground/50" />
                    <p class="text-sm text-muted-foreground">
                      {{ $t('command.selectAMacro') }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </CommandGroup>
      </div>

      <!-- Commands requiring a conversation to be open -->
      <CommandGroup
        :heading="t('globals.terms.conversation', 2)"
        value="conversations"
        v-else-if="conversationStore.isConversationOpen && !nestedCommand"
      >
        <CommandItem
          value="apply-macro"
          @select="setNestedCommand('apply-macro-to-existing-conversation')"
        >
          {{ $t('actions.applyMacro') }}
        </CommandItem>
        <CommandItem value="conv-snooze" @select="setNestedCommand('snooze')">
          {{ $t('globals.terms.snooze') }}
        </CommandItem>
        <CommandItem value="conv-resolve" @select="resolveConversation">
          {{ $t('globals.terms.resolve') }}
        </CommandItem>
      </CommandGroup>
    </CommandList>

    <!-- Navigation -->
    <div class="flex items-center gap-4 border-t px-3 py-2">
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd class="inline-flex h-5 items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">Enter</kbd>
        {{ $t('globals.terms.select') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd class="inline-flex h-5 items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">&uarr;&darr;</kbd>
        {{ $t('command.navigate') }}
      </span>
      <span class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd class="inline-flex h-5 items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">Esc</kbd>
        {{ $t('globals.messages.close') }}
      </span>
      <span v-if="nestedCommand" class="flex items-center gap-1 text-xs text-muted-foreground">
        <kbd class="inline-flex h-5 items-center rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">Backspace</kbd>
        {{ $t('globals.messages.back') }}
      </span>
    </div>
  </CommandDialog>

  <!-- Date Picker for Custom Snooze -->
  <Dialog :open="showDatePicker" @update:open="closeDatePicker">
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>{{ $t('command.pickSnoozeTime') }}</DialogTitle>
        <DialogDescription />
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <Popover :open="datePickerOpen" @update:open="datePickerOpen = $event">
          <PopoverTrigger as-child>
            <Button variant="outline" class="w-full justify-start text-left font-normal">
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ selectedDate ? selectedDate : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar mode="single" v-model="selectedDate" @update:model-value="datePickerOpen = false" />
          </PopoverContent>
        </Popover>
        <div class="grid gap-2">
          <Label>{{ $t('globals.terms.time') }}</Label>
          <Input type="time" v-model="selectedTime" />
        </div>
      </div>
      <DialogFooter>
        <Button @click="handleCustomSnooze">{{ $t('globals.terms.snooze') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useMagicKeys } from '@vueuse/core'
import { CalendarIcon } from 'lucide-vue-next'
import { useConversationStore } from '@main/stores/conversation'
import { useMacroStore } from '@main/stores/macro'
import { CONVERSATION_DEFAULT_STATUSES, MACRO_CONTEXT } from '@main/constants/conversation'
import { Users, User, Pin, Rocket, Tags, Zap } from 'lucide-vue-next'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem
} from '@shared-ui/components/ui/command'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { Button } from '@shared-ui/components/ui/button'
import { Calendar } from '@shared-ui/components/ui/calendar'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { useI18n } from 'vue-i18n'
import { Letter } from 'vue-letter'

const RENDER_CAP = 200

const conversationStore = useConversationStore()
const macroStore = useMacroStore()
const { t } = useI18n()
const open = ref(false)
const emitter = useEmitter()
const nestedCommand = ref(null)
const showDatePicker = ref(false)
const datePickerOpen = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')
const searchTerm = ref('')
const macroListRef = ref(null)

const passThroughFilter = (items) => items

const isMacroMode = computed(
  () =>
    nestedCommand.value === 'apply-macro-to-existing-conversation' ||
    nestedCommand.value === 'apply-macro-to-new-conversation'
)

const macroSearchIndex = computed(() =>
  macroStore.macroOptions.map((m) => ({ macro: m, labelLower: String(m.label).toLowerCase() }))
)

const visibleMacros = computed(() => {
  const term = searchTerm.value?.trim().toLowerCase()
  const index = macroSearchIndex.value
  if (!term) {
    const all = macroStore.macroOptions
    return all.length > RENDER_CAP ? all.slice(0, RENDER_CAP) : all
  }
  const matched = []
  for (let i = 0; i < index.length && matched.length < RENDER_CAP; i++) {
    if (index[i].labelLower.includes(term)) matched.push(index[i].macro)
  }
  return matched
})

const { Meta_K, Ctrl_K } = useMagicKeys({
  passive: false,
  onEventFired(e) {
    if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
    }
  }
})

watch([Meta_K, Ctrl_K], ([mac, win]) => {
  if (mac || win) toggleOpen()
})

const highlightedMacro = ref(null)

function handleApplyMacro(macro) {
  // Create a deep copy.
  const plainMacro = JSON.parse(JSON.stringify(macro))
  if (nestedCommand.value === 'apply-macro-to-new-conversation') {
    conversationStore.setMacro(plainMacro, MACRO_CONTEXT.NEW_CONVERSATION)
  } else {
    conversationStore.setMacro(plainMacro, MACRO_CONTEXT.REPLY)
  }
  toggleOpen()
}

const getActionLabel = computed(() => (action) => {
  const prefixes = {
    assign_user: t('actions.assignAgent'),
    assign_team: t('actions.assignTeam'),
    set_status: t('actions.setStatus'),
    set_priority: t('actions.setPriority'),
    add_tags: t('actions.addTags'),
    set_tags: t('actions.setTags'),
    remove_tags: t('actions.removeTags')
  }
  return `${prefixes[action.type]}: ${action.display_value.length > 0 ? action.display_value.join(', ') : action.value.join(', ')}`
})

const replyContent = computed(() => highlightedMacro.value?.message_content || '')

const otherActions = computed(
  () =>
    highlightedMacro.value?.actions?.filter(
      (a) => a.type !== 'send_private_note' && a.type !== 'send_reply'
    ) || []
)

function toggleOpen() {
  if (nestedCommand.value != 'apply-macro-to-new-conversation' && !open.value) {
    nestedCommand.value = null
  }
  open.value = !open.value
}

function setNestedCommand(command) {
  nestedCommand.value = command
}

function formatDuration(minutes) {
  return minutes < 60 ? `${minutes}m` : `${Math.floor(minutes / 60)}h`
}

async function handleSnooze(minutes) {
  await conversationStore.snoozeConversation(formatDuration(minutes))
  toggleOpen()
}

async function resolveConversation() {
  await conversationStore.updateStatus(CONVERSATION_DEFAULT_STATUSES.RESOLVED)
  toggleOpen()
}

function showCustomDialog() {
  toggleOpen()
  showDatePicker.value = true
}

function closeDatePicker() {
  showDatePicker.value = false
}

function handleCustomSnooze() {
  const [hours, minutes] = selectedTime.value.split(':')
  const snoozeDate = new Date(selectedDate.value)
  snoozeDate.setHours(parseInt(hours), parseInt(minutes))
  const diffMinutes = Math.floor((snoozeDate - new Date()) / (1000 * 60))

  if (diffMinutes <= 0) {
    alert(t('globals.messages.selectAFutureTime'))
    return
  }
  handleSnooze(diffMinutes)
  closeDatePicker()
  toggleOpen()
}

function onInputKeydown(e) {
  if (e.key === 'Backspace') {
    const inputVal = e.target.value || ''
    if (!inputVal && nestedCommand.value !== null) {
      e.preventDefault()
      nestedCommand.value = null
    }
  }
}

const nestedCommandHandler = (data) => {
  setNestedCommand(data.command)
  open.value = data.open
}

let highlightObserver = null

watch(macroListRef, (el) => {
  highlightObserver?.disconnect()
  highlightObserver = null
  highlightedMacro.value = null
  if (!el) return
  highlightObserver = new MutationObserver(() => {
    const idx = el.querySelector('[data-highlighted]')?.getAttribute('data-index')
    highlightedMacro.value = idx != null ? visibleMacros.value[idx] : null
  })
  highlightObserver.observe(el, { attributes: true, attributeFilter: ['data-highlighted'], subtree: true })
})

onMounted(() => {
  emitter.on(EMITTER_EVENTS.SET_NESTED_COMMAND, nestedCommandHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SET_NESTED_COMMAND, nestedCommandHandler)
  highlightObserver?.disconnect()
})
</script>
