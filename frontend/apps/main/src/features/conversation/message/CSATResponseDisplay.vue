<template>
  <div v-if="csatResponse" class="mt-3 p-3 bg-muted/50 rounded-md border">
    <div class="text-sm font-medium text-muted-foreground mb-2">
      {{ $t('globals.terms.feedback') }}:
    </div>

    <!-- Rating -->
    <div v-if="csatResponse.rating" class="flex items-center gap-2 mb-2">
      <span class="text-lg">{{ getRatingEmoji(csatResponse.rating) }}</span>
      <span class="text-sm font-medium">{{ getRatingText(csatResponse.rating) }}</span>
      <span class="text-xs text-muted-foreground">({{ csatResponse.rating }}/5)</span>
    </div>

    <!-- Feedback -->
    <div v-if="csatResponse.feedback" class="text-sm italic text-foreground">
      "{{ csatResponse.feedback }}"
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  message: {
    type: Object,
    required: true
  }
})

const csatResponse = computed(() => {
  const meta = props.message.meta
  if (!meta?.is_csat || !meta?.csat_submitted) {
    return null
  }

  return {
    rating: meta.submitted_rating || null,
    feedback: meta.submitted_feedback || null
  }
})

const getRatingEmoji = (rating) => {
  const ratings = {
    1: '😢',
    2: '😕',
    3: '😊',
    4: '😃',
    5: '🤩'
  }
  return ratings[rating] || ''
}

const getRatingText = (rating) => {
  const ratings = {
    1: 'Poor',
    2: 'Fair',
    3: 'Good',
    4: 'Great',
    5: 'Excellent'
  }
  return ratings[rating] || ''
}
</script>
