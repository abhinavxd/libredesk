<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ i18n._('Keyboard shortcuts') }}</DialogTitle>
      </DialogHeader>
      <div class="mt-2 space-y-6">
        <div v-for="group in groups" :key="group.title" class="space-y-3">
          <h4 class="sidebar-section-label">
            {{ group.title }}
          </h4>
          <div class="space-y-2">
            <div
              v-for="item in group.items"
              :key="item.label"
              class="flex items-center justify-between text-sm"
            >
              <span>{{ item.label }}</span>
              <span class="flex items-center gap-1">
                <kbd
                  v-for="key in item.keys"
                  :key="key"
                  class="inline-flex h-5 min-w-5 items-center justify-center rounded-md border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground"
                >
                  {{ key }}
                </kbd>
              </span>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'

defineProps({
  open: {
    type: Boolean,
    default: false
  }
})
defineEmits(['update:open'])

const i18n = useI18n()

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const mod = isMac ? '⌘' : 'Ctrl'
const alt = isMac ? '⌥' : 'Alt'

const groups = computed(() => [
  {
    title: i18n._('General'),
    items: [
      { label: i18n._('Open command bar'), keys: [mod, 'K'] },
      { label: i18n._('Keyboard shortcuts'), keys: [mod, '/'] },
      { label: i18n._('New conversation'), keys: [alt, 'C'] }
    ]
  },
  {
    title: i18n._('{count, plural, one {Conversation} other {Conversations}}', 2),
    items: [
      { label: i18n._('Previous conversation'), keys: [alt, 'J'] },
      { label: i18n._('Next conversation'), keys: [alt, 'K'] },
      { label: i18n._('Snooze'), keys: [alt, 'Z'] },
      { label: i18n._('Set priority'), keys: [alt, 'P'] },
      { label: i18n._('Assign agent'), keys: [alt, 'A'] },
      { label: i18n._('Switch to reply'), keys: [alt, 'R'] },
      { label: i18n._('Switch to private note'), keys: [alt, 'N'] },
      { label: i18n._('Select conversation'), keys: [alt, 'X'] },
      { label: i18n._('Resolve'), keys: [alt, 'E'] },
      { label: i18n._('Reopen'), keys: [alt, 'O'] }
    ]
  },
  {
    title: i18n._('Reply editor'),
    items: [
      { label: i18n._('Open macros'), keys: [mod, 'M'] },
      { label: i18n._('Switch between reply and private note'), keys: [mod, 'P'] },
      { label: i18n._('Minimize reply box'), keys: [alt, 'M'] },
      { label: i18n._('Send reply'), keys: ['Ctrl', 'Enter'] },
      { label: i18n._('Bold'), keys: [mod, 'B'] },
      { label: i18n._('Italic'), keys: [mod, 'I'] }
    ]
  }
])
</script>
