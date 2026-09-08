<template>
  <Tooltip v-if="priorityName">
    <TooltipTrigger asChild>
      <span
        class="h-2 w-2 rounded-full flex-shrink-0"
        :class="dotClass"
        role="img"
        :aria-label="priorityLabel"
      />
    </TooltipTrigger>
    <TooltipContent>{{ priorityLabel }}</TooltipContent>
  </Tooltip>
</template>

<script setup>
import { computed } from 'vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useI18n } from 'vue-i18n'

// Priorities are seeded rows (Low, Medium, High) and are not translated, so match on the
// name and keep anything unrecognised neutral rather than inventing a colour for it.
const PRIORITY_DOT_CLASSES = {
  low: 'bg-muted-foreground',
  medium: 'bg-warning-600',
  high: 'bg-destructive'
}

const props = defineProps({
  priority: {
    type: String,
    default: ''
  }
})

const { t } = useI18n()

const priorityName = computed(() => (props.priority || '').trim())

const dotClass = computed(
  () => PRIORITY_DOT_CLASSES[priorityName.value.toLowerCase()] || 'bg-muted-foreground'
)

const priorityLabel = computed(() => `${t('globals.terms.priority', 1)}: ${priorityName.value}`)
</script>
