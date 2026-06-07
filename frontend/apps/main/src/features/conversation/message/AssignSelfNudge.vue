<template>
  <Transition
    enter-active-class="transition ease-out duration-200"
    enter-from-class="opacity-0 translate-y-2"
    enter-to-class="opacity-100 translate-y-0"
    leave-active-class="transition ease-in duration-150"
    leave-from-class="opacity-100 translate-y-0"
    leave-to-class="opacity-0 translate-y-2"
  >
    <div v-if="show" class="absolute bottom-4 left-1/2 -translate-x-1/2 z-10 w-[calc(100%-2rem)] max-w-md">
      <div class="flex items-center gap-3 rounded-lg border border-border bg-background px-3 py-2 shadow-lg">
        <span class="flex-1 text-sm text-foreground">{{ $t('conversation.assignSelfPrompt') }}</span>
        <Button size="sm" @click="$emit('assign')">{{ $t('conversation.assignSelfAction') }}</Button>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground"
          :aria-label="$t('globals.messages.close')"
          @click="$emit('dismiss')"
        >
          <X class="size-4" />
        </button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { X } from 'lucide-vue-next'

defineProps({
  show: {
    type: Boolean,
    default: false
  }
})

defineEmits(['assign', 'dismiss'])
</script>
