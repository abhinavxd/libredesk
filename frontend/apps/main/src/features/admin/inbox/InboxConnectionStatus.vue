<template>
  <div v-if="isDisconnected" class="flex justify-center">
    <Tooltip>
      <TooltipTrigger as-child>
        <span
          class="inline-flex items-center gap-1.5 text-destructive text-sm font-medium cursor-default"
        >
          <TriangleAlert class="size-4" />
          {{ $t('admin.inbox.disconnected') }}
        </span>
      </TooltipTrigger>
      <TooltipContent>
        <p class="text-sm max-w-xs">{{ $t('admin.inbox.disconnected.description') }}</p>
      </TooltipContent>
    </Tooltip>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { TriangleAlert } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'

const props = defineProps({
  inbox: {
    type: Object,
    required: true
  }
})

const isDisconnected = computed(
  () => props.inbox.channel === 'email' && Boolean(props.inbox.disconnected_at)
)
</script>
