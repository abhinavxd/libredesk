<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ t('navigation.keyboardShortcuts') }}</DialogTitle>
      </DialogHeader>
      <div class="mt-2 space-y-6">
        <div v-for="group in groups" :key="group.title" class="space-y-3">
          <h4 class="text-xs font-medium uppercase tracking-wide text-muted-foreground">
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
import { useI18n } from 'vue-i18n'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'

defineProps({
  open: {
    type: Boolean,
    default: false
  }
})
defineEmits(['update:open'])

const { t } = useI18n()

const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const mod = isMac ? '⌘' : 'Ctrl'

const groups = computed(() => [
  {
    title: t('globals.terms.general'),
    items: [{ label: t('shortcuts.openCommandBar'), keys: [mod, 'K'] }]
  },
  {
    title: t('shortcuts.replyEditor'),
    items: [
      { label: t('actions.openMacros'), keys: [mod, 'M'] },
      { label: t('actions.sendReply'), keys: ['Ctrl', 'Enter'] },
      { label: t('globals.terms.bold'), keys: [mod, 'B'] },
      { label: t('globals.terms.italic'), keys: [mod, 'I'] }
    ]
  }
])
</script>
