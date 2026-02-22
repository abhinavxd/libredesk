<template>
  <div class="text-center">
    <div class="text-muted-foreground text-sm">
      {{ message.content }}
      <Tooltip>
        <TooltipTrigger>
          <span class="text-xs ml-1">{{ format(message.updated_at, 'h:mm a') }}</span>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {{ format(message.updated_at, "MMMM dd, yyyy 'at' HH:mm") }}
          </p>
        </TooltipContent>
      </Tooltip>
    </div>
  </div>
</template>

<script setup>
import { format } from 'date-fns'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'

defineProps({
  message: Object
})
</script>
