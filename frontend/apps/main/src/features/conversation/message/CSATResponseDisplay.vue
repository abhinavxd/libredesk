<template>
  <div v-if="isCsatMessage" class="mt-3 pt-3 border-t border-border">
    <!-- Submitted CSAT -->
    <div v-if="isSubmitted && hasResponse">
      <div class="text-xs text-muted-foreground mb-2">
        {{ t('globals.terms.feedback', 1) }}
      </div>

      <div v-if="csatResponse.rating" class="flex items-center gap-2 mb-2">
        <span class="text-lg">{{ getRatingEmoji(csatResponse.rating) }}</span>
        <span class="text-sm font-medium">{{ getRatingText(csatResponse.rating) }}</span>
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
      {{ t('globals.terms.awaitingResponse') }}
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

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

const getRatingEmoji = (rating) => {
  const emojis = { 1: '😢', 2: '😕', 3: '😊', 4: '😃', 5: '🤩' }
  return emojis[rating] || ''
}

const getRatingText = (rating) => {
  const keys = {
    1: 'globals.terms.poor',
    2: 'globals.terms.fair',
    3: 'globals.terms.good',
    4: 'globals.terms.great',
    5: 'globals.terms.excellent'
  }
  return keys[rating] ? t(keys[rating]) : ''
}
</script>
