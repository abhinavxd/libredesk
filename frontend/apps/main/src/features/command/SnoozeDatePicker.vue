<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-[425px]" @open-auto-focus="focusDateButton">
      <DialogHeader>
        <DialogTitle>{{ $t('command.pickSnoozeTime') }}</DialogTitle>
        <DialogDescription />
      </DialogHeader>
      <div class="grid gap-4 py-4">
        <Popover :open="datePickerOpen" @update:open="datePickerOpen = $event">
          <PopoverTrigger as-child>
            <Button
              ref="dateButtonRef"
              variant="outline"
              class="w-full justify-start text-left font-normal"
            >
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ selectedDate ? selectedDate : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar
              mode="single"
              v-model="selectedDate"
              @update:model-value="datePickerOpen = false"
            />
          </PopoverContent>
        </Popover>
        <div class="grid gap-2">
          <Label>{{ $t('globals.terms.time') }}</Label>
          <Input type="time" v-model="selectedTime" />
        </div>
      </div>
      <DialogFooter>
        <Button @click="snooze">{{ $t('globals.terms.snooze') }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { CalendarIcon } from 'lucide-vue-next'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { Button } from '@shared-ui/components/ui/button'
import { Calendar } from '@shared-ui/components/ui/calendar'
import { Input } from '@shared-ui/components/ui/input'
import { Label } from '@shared-ui/components/ui/label'
import { useConversationStore } from '@main/stores/conversation'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { formatSnoozeDuration } from './providers/useConversationCommands'

defineProps({
  open: { type: Boolean, default: false }
})
const emit = defineEmits(['update:open'])

const { t } = useI18n()
const emitter = useEmitter()
const conversationStore = useConversationStore()
const datePickerOpen = ref(false)
const selectedDate = ref(null)
const selectedTime = ref('12:00')
const dateButtonRef = ref(null)

const focusDateButton = (event) => {
  event.preventDefault()
  dateButtonRef.value?.$el?.focus()
}

const snooze = () => {
  const [hours, minutes] = selectedTime.value.split(':')
  const snoozeDate = new Date(selectedDate.value)
  snoozeDate.setHours(parseInt(hours), parseInt(minutes))
  const diffMinutes = Math.floor((snoozeDate - new Date()) / (1000 * 60))

  if (diffMinutes <= 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: t('globals.messages.selectAFutureTime')
    })
    return
  }
  conversationStore.snoozeConversation(formatSnoozeDuration(diffMinutes))
  emit('update:open', false)
}
</script>
