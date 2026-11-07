<template>
  <div class="mb-1 min-w-0 space-y-1 text-xs [overflow-wrap:anywhere]">
    <p v-if="message.meta?.subject" class="text-foreground">
      {{ message.meta.subject }}
    </p>
    <div class="space-y-0.5">
      <p v-if="message.meta?.from?.length">
        <span class="text-muted-foreground">From:</span> {{ message.meta.from.join(', ') }}
      </p>
      <p v-if="message.meta?.to?.length">
        <span class="text-muted-foreground">To:</span> {{ message.meta.to.join(', ') }}
      </p>
      <p v-if="message.meta?.cc?.length">
        <span class="text-muted-foreground">Cc:</span> {{ message.meta.cc.join(', ') }}
      </p>
      <p v-if="message.meta?.bcc?.length">
        <span class="text-muted-foreground">Bcc:</span> {{ message.meta.bcc.join(', ') }}
      </p>
    </div>
    <Popover v-if="senderHeader && senderAddress">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          class="h-auto min-h-8 max-w-full justify-start gap-2 whitespace-normal bg-warning/10 px-2 py-1.5 text-left font-normal text-warning-600 can-hover:hover:bg-warning/15 can-hover:hover:text-warning-600 [&_svg]:size-3.5"
        >
          <TriangleAlert aria-hidden="true" />
          <span>{{ $t('conversation.originalSenderWarning.title') }}</span>
          <ChevronRight aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        class="w-80 space-y-3 text-xs leading-relaxed"
        :aria-label="$t('conversation.originalSenderWarning.title')"
      >
        <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-2">
          <template v-for="entry in senderDetails" :key="entry.header">
            <dt class="text-muted-foreground">{{ entry.header }}</dt>
            <dd class="min-w-0 select-text [overflow-wrap:anywhere]">{{ entry.address }}</dd>
          </template>
        </dl>
        <p class="border-t pt-3 text-muted-foreground">
          {{ $t('conversation.originalSenderWarning', { header: senderHeader }) }}
        </p>
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronRight, TriangleAlert } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'

const props = defineProps({
  message: {
    type: Object,
    required: true
  }
})
const { t } = useI18n()
const senderAddress = computed(() => props.message.meta?.contact_address)
const senderHeader = computed(() => props.message.meta?.contact_header)
const senderDetails = computed(() => [
  { header: t('globals.terms.from'), address: props.message.meta?.from?.join(', ') },
  { header: senderHeader.value, address: senderAddress.value }
])
</script>
