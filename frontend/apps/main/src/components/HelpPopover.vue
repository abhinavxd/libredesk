<template>
  <Popover>
    <PopoverTrigger as-child>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        class="size-7 text-muted-foreground"
        :aria-label="$t('globals.terms.help')"
      >
        <CircleHelp aria-hidden="true" />
      </Button>
    </PopoverTrigger>
    <PopoverContent align="start" class="w-[28rem] max-w-[calc(100vw-2rem)]">
      <p class="text-xs leading-relaxed">{{ text }}</p>
    </PopoverContent>
  </Popover>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { CircleHelp } from 'lucide-vue-next'

defineProps({
  text: {
    type: String,
    required: true
  }
})
</script>
