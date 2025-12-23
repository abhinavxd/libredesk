<template>
  <CommandDialog :open="open" @update:open="toggleOpen" class="transform-gpu z-[51] !min-w-[50vw]">
    <CommandInput :placeholder="t('command.typeCmdOrSearch')" @keydown="onInputKeydown" />
    <CommandList
      class="!min-h-[60vh] h-[60vh] !min-w-[50vw]"
      :class="{ 'overflow-hidden': nestedCommand === 'apply-macro' }"
    >
      <CommandEmpty>
        <p class="text-muted-foreground">{{ $t('command.noCommandAvailable') }}</p>
      </CommandEmpty>

      <!-- Snooze Options -->
      <CommandGroup v-if="nestedCommand === 'snooze'" heading="Snooze for">
        <CommandItem value="1 hour" @select="handleSnooze(60)">
          1 {{ $t('globals.terms.hour') }}
        </CommandItem>
        <CommandItem value="3 hours" @select="handleSnooze(180)"
          >3 {{ $t('globals.terms.hour', 2) }}</CommandItem
        >
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
        <CommandItem value="pick date & time" @select="showCustomDialog">
          {{ $t('globals.messages.pickDateAndTime') }}
        </CommandItem>
      </CommandGroup>

      <!-- Macros -->
      <div
        v-if="
          nestedCommand === 'apply-macro-to-existing-conversation' ||
          nestedCommand === 'apply-macro-to-new-conversation'
        "
      >
        <CommandGroup heading="Apply macro">
          <div class="min-h-[400px]">
            <div class="h-[60vh] grid grid-cols-12">
              <!-- Left Column: Macro List (30%) -->
              <div class="col-span-4 pr-4 border-r overflow-y-auto h-full">
                <CommandItem
                  v-for="(macro, index) in macroStore.macroOptions"
                  :key="macro.value"
                  :value="macro.label + '|' + index"
                  :data-index="index"
                  @select="handleApplyMacro(macro)"
                  class="px-3 py-2 rounded cursor-pointer transition-all duration-200 hover:bg-primary/10 hover:"
                >
                  <div class="flex items-center gap-2">
                    <Zap size="14" class="shrink-0" />
                    <span class="text-sm truncate w-full break-words whitespace-normal">{{
                      macro.label
                    }}</span>
                  </div>
                </CommandItem>
              </div>

              <!-- Right Column: Macro Details (70%) -->
              <div class="col-span-8 px-4 overflow-y-auto h-full pb-12">
                <div class="space-y-3 text-xs">
                  <!-- Reply Preview -->
                  <div v-if="replyContent" class="space-y-2">
                    <p class="text-xs font-semibold text-foreground">
                      {{ $t('command.replyPreview') }}
                    </p>
                    <Letter
                      :html="replyContent"
                      :allowedSchemas="['cid', 'https', 'http', 'mailto']"
                      class="w-full min-h-200 p-2 bg-muted/50 rounded overflow-auto shadow native-html"
                    />
                  </div>

                  <!-- Actions -->
                  <div v-if="otherActions.length > 0" class="space-y-2">
                    <p class="text-xs font-semibold">
                      {{ $t('globals.terms.action', 2) }}
                    </p>
                    <div class="space-y-1.5 max-w-sm">
                      <div
                        v-for="action in otherActions"
                        :key="action.type"
                        class="flex items-center gap-2 px-2 py-1.5 rounded text-xs bg-muted/50 hover:bg-accent hover:text-accent-foreground transition duration-200 group"
                      >
                        <div
                          class="p-1 rounded-full group-hover:bg-muted/20 transition duration-200"
                        >
                          <User v-if="action.type === 'assign_user'" :size="10" class="shrink-0" />
                          <Users
                            v-else-if="action.type === 'assign_team'"
                            :size="10"
                            class="shrink-0"
                          />
                          <Pin
                            v-else-if="action.type === 'set_status'"
                            :size="10"
                            class="shrink-0"
                          />
                          <Rocket
                            v-else-if="action.type === 'set_priority'"
                            :size="10"
                            class="shrink-0"
                          />
                          <Tags
                            v-else-if="action.type === 'add_tags'"
                            :size="10"
                            class="shrink-0"
                          />
                          <Tags
                            v-else-if="action.type === 'set_tags'"
                            :size="10"
                            class="shrink-0"
                          />
                          <Tags
                            v-else-if="action.type === 'remove_tags'"
                            :size="10"
                            class="shrink-0"
                          />
                        </div>
                        <span class="truncate">{{ getActionLabel(action) }}</span>
                      </div>
                    </div>
                  </div>

                  <!-- Empty State -->
                  <div
                    v-if="!replyContent && otherActions.length === 0"
                    class="flex items-center justify-center h-20"
                  >
                    <p class="text-xs text-muted-foreground italic">
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
        v-else-if="conversationStore.hasConversationOpen && !nestedCommand"
      >
        <CommandItem
          value="apply-macro"
          @select="setNestedCommand('apply-macro-to-existing-conversation')"
        >
          {{ $t('globals.messages.apply', { name: t('globals.terms.macro').toLowerCase() }) }}
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
    <div class="mt-2 px-4 py-2 text-xs text-gray-500 flex space-x-4">
      <span> {{ $t('command.enterToSelect') }}</span>
      <span> {{ $t('command.navigateWithArrows') }}</span>
      <span> {{ $t('command.closeWithEsc') }}</span>
      <span> {{ $t('command.backspaceToParent') }}</span>
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
        <Popover>
          <PopoverTrigger as-child>
            <Button variant="outline" class="w-full justify-start text-left font-normal">
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ selectedDate ? selectedDate : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar mode="single" v-model="selectedDate" />
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
import { useConversationStore } from '@/stores/conversation'
import { useMacroStore } from '@/stores/macro'
import { CONVERSATION_DEFAULT_STATUSES, MACRO_CONTEXT } from '@/constants/conversation'
import { Users, User, Pin, Rocket, Tags, Zap } from 'lucide-vue-next'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem
} from '@/components/ui/command'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useI18n } from 'vue-i18n'
import { Letter } from 'vue-letter'

const conversationStore = useConversationStore()
const macroStore = useMacroStore()
const { t } = useI18n()
const open = ref(false)
const emitter = useEmitter()
const nestedCommand = ref(null)
const showDatePicker = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')

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
    assign_user: t('globals.messages.assign', { name: t('globals.terms.agent').toLowerCase() }),
    assign_team: t('globals.messages.assign', { name: t('globals.terms.team').toLowerCase() }),
    set_status: t('globals.messages.set', { name: t('globals.terms.status').toLowerCase() }),
    set_priority: t('globals.messages.set', { name: t('globals.terms.priority').toLowerCase() }),
    add_tags: t('globals.messages.add', { name: t('globals.terms.tag', 2).toLowerCase() }),
    set_tags: t('globals.messages.set', { name: t('globals.terms.tag', 2).toLowerCase() }),
    remove_tags: t('globals.messages.remove', { name: t('globals.terms.tag', 2).toLowerCase() })
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

onMounted(() => {
  emitter.on(EMITTER_EVENTS.SET_NESTED_COMMAND, (data) => {
    setNestedCommand(data.command)
    open.value = data.open
  })
  watchHighlightedMacro()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SET_NESTED_COMMAND)
})

const watchHighlightedMacro = () => {
  const observer = new MutationObserver(() => {
    const highlightedEl = document.querySelector('[data-highlighted]')?.getAttribute('data-index')
    highlightedMacro.value = highlightedEl ? macroStore.macroOptions[highlightedEl] : null
  })

  observer.observe(document.body, {
    attributes: true,
    subtree: true
  })
}
</script>
