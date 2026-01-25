<template>
  <div class="p-4 rounded-2xl text-sm bg-background text-foreground border border-border">
    <div v-if="!isSubmitted">
      <p class="mb-3">{{ t('globals.messages.pleaseRateConversation') }}</p>

      <div class="flex gap-3 mb-4">
        <button
          v-for="rating in ratings"
          :key="rating.value"
          @click="selectedRating = rating.value"
          class="flex flex-col items-center p-2 rounded-lg"
          :class="{ 'scale-150': selectedRating === rating.value }"
        >
          <span class="text-xl mb-1">{{ rating.emoji }}</span>
          <span class="text-xs text-muted-foreground">{{ rating.text }}</span>
        </button>
      </div>

      <div class="mb-4">
        <label class="text-xs text-muted-foreground mb-2 block">
          {{ t('globals.messages.additionalFeedbackOptional') }}
        </label>
        <textarea
          v-model="feedback"
          :placeholder="$t('globals.terms.tellUsMore')"
          class="w-full p-2 text-sm border border-border rounded-md bg-background text-foreground placeholder:text-muted-foreground"
          rows="2"
          maxlength="500"
        ></textarea>
        <div class="text-xs text-muted-foreground text-right mt-1">{{ feedback.length }}/500</div>
      </div>

      <button
        @click="submitRating"
        :disabled="(!selectedRating && !feedback.trim()) || isSubmitting"
        class="w-full py-2 bg-primary text-primary-foreground rounded-md text-sm disabled:opacity-50 flex items-center justify-center gap-2"
      >
        <div v-if="isSubmitting" class="w-4 h-4 border border-primary-foreground border-t-transparent rounded-full animate-spin"></div>
        <span v-if="isSubmitting">{{ t('globals.messages.submitting') }}</span>
        <span v-else>{{ t('globals.messages.submitFeedback') }}</span>
      </button>
    </div>

    <div v-else class="text-center py-2">
      <p class="mb-3">{{ t('globals.messages.thankYouFeedback') }}</p>
      
      <!-- Show submitted rating if provided -->
      <div v-if="csatMeta.submitted_rating" class="mb-2">
        <span class="text-lg">{{ getRatingEmoji(csatMeta.submitted_rating) }}</span>
        <span class="text-xs text-muted-foreground ml-2">{{ getRatingText(csatMeta.submitted_rating) }}</span>
      </div>
      
      <!-- Show submitted feedback if provided -->
      <div v-if="csatMeta.submitted_feedback" class="text-xs text-muted-foreground italic">
        "{{ csatMeta.submitted_feedback }}"
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@widget/api/index.js'

const props = defineProps({
  message: { type: Object, required: true }
})

const emit = defineEmits(['submitted'])

const selectedRating = ref(null)
const feedback = ref('')
const isSubmitting = ref(false)

const csatMeta = computed(() => {
  return props.message.meta
})

const isSubmitted = computed(() => csatMeta.value.csat_submitted === true)
const csatUuid = computed(() => csatMeta.value.csat_uuid || '')

const { t } = useI18n()

const ratings = [
  { value: 1, emoji: '😢', text: t('globals.terms.poor') },
  { value: 2, emoji: '😕', text: t('globals.terms.fair') },
  { value: 3, emoji: '😊', text: t('globals.terms.good') },
  { value: 4, emoji: '😃', text: t('globals.terms.great') },
  { value: 5, emoji: '🤩', text: t('globals.terms.excellent') }
]

const submitRating = async () => {
  if ((!selectedRating.value && !feedback.value.trim()) || !csatUuid.value) return
  isSubmitting.value = true
  try {
    await api.submitCSATResponse(csatUuid.value, selectedRating.value || 0, feedback.value)
    emit('submitted', {
      rating: selectedRating.value,
      feedback: feedback.value,
      message_uuid: props.message.uuid
    })
  } finally {
    isSubmitting.value = false
  }
}

const getRatingEmoji = (rating) => {
  const ratingObj = ratings.find(r => r.value === rating)
  return ratingObj ? ratingObj.emoji : ''
}

const getRatingText = (rating) => {
  const ratingObj = ratings.find(r => r.value === rating)
  return ratingObj ? ratingObj.text : ''
}
</script>
