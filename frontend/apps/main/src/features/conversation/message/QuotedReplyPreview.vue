<template>
  <div
    class="mb-3 flex shrink-0 items-center gap-3 rounded-md border border-border bg-muted/40 px-3 py-2.5 text-sm"
    role="status"
  >
    <div class="min-w-0 flex-1 space-y-1">
      <p class="font-medium text-muted-foreground">{{ $t('globals.messages.replying') }}</p>
      <p class="line-clamp-2 break-words">
        {{ content || $t('globals.terms.attachment') }}
      </p>
    </div>
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="shrink-0 text-muted-foreground hover:text-foreground max-md:size-11"
      :aria-label="$t('globals.messages.cancelQuotedReply')"
      @click="$emit('clear')"
    >
      <X aria-hidden="true" class="size-4" />
    </Button>
  </div>
</template>
<script setup>
import { X } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
defineProps({ content: { type: String, default: '' } })
defineEmits(['clear'])
</script>
