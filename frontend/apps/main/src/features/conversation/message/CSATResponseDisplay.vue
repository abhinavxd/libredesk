<template>
  <div v-if="isCsatMessage" class="mt-3 pt-3 border-t border-border">
    <!-- Submitted CSAT -->
    <div v-if="isSubmitted && hasResponse">
      <div class="text-xs text-muted-foreground mb-2">
        {{ i18n.ngettext('Feedback', 'Feedbacks', 1) }}
      </div>

      <div v-if="csatResponse.rating" class="flex items-center gap-2 mb-2">
        <span class="text-lg">{{ csatRatingEmoji(csatResponse.rating) }}</span>
        <span class="text-sm font-medium">{{ i18n._(csatRatingMessage(csatResponse.rating)) }}</span>
        <span class="text-xs text-muted-foreground">{{ csatResponse.rating }}/5</span>
      </div>

      <p
        v-if="csatResponse.feedback"
        class="text-sm text-muted-foreground italic pl-3 border-l-2 border-muted"
      >
        {{ csatResponse.feedback }}
      </p>
    </div>

    <span v-else-if="!isSubmitted" class="text-xs text-muted-foreground italic">
      {{ i18n._('Awaiting Response') }}
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { csatRatingEmoji, csatRatingMessage } from '@shared-ui/utils/csat.js'

const i18n = useI18n()

const props = defineProps({
  message: {
    type: Object,
    required: true
  }
})

const isCsatMessage = computed(() => props.message.meta?.is_csat === true)
const isSubmitted = computed(() => props.message.meta?.csat_submitted === true)

const csatResponse = computed(() => {
  if (!isSubmitted.value) return null
  return {
    rating: props.message.meta.submitted_rating || null,
    feedback: props.message.meta.submitted_feedback || null
  }
})

const hasResponse = computed(() =>
  csatResponse.value && (csatResponse.value.rating || csatResponse.value.feedback)
)
</script>
