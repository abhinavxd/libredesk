<template>
  <Tooltip v-if="priorityName">
    <TooltipTrigger asChild>
      <PriorityIcon
        :level="marker.level"
        :size="13"
        class="flex-shrink-0"
        :class="marker.class"
        role="img"
        :aria-label="priorityLabel"
      />
    </TooltipTrigger>
    <TooltipContent>{{ priorityLabel }}</TooltipContent>
  </Tooltip>
</template>

<script setup>
import { computed } from 'vue'
import PriorityIcon from '@shared-ui/components/icons/PriorityIcon.vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { useI18n } from 'vue-i18n'

const PRIORITY_MARKERS = {
  low: { level: 1, class: 'text-muted-foreground' },
  medium: { level: 2, class: 'text-warning-600' },
  high: { level: 3, class: 'text-destructive' }
}
const UNKNOWN_MARKER = { level: 0, class: 'text-muted-foreground' }

const props = defineProps({
  priority: {
    type: String,
    default: ''
  }
})

const { t } = useI18n()

const priorityName = computed(() => (props.priority || '').trim())

const marker = computed(() => PRIORITY_MARKERS[priorityName.value.toLowerCase()] || UNKNOWN_MARKER)

const priorityLabel = computed(() => `${t('globals.terms.priority', 1)}: ${priorityName.value}`)
</script>
