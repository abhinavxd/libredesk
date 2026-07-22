<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="mb-6">
          <DotLoader v-if="isLoadingLearning" />
          <SwitchField
            v-else
            :title="t('admin.ai.faqLearning.title')"
            :description="t('admin.ai.faqLearning.description')"
            :checked="learningEnabled"
            @update:checked="toggleLearning"
          />
        </div>

        <Dialog v-model:open="reviewOpen">
          <DialogContent class="sm:max-w-[560px]">
            <DialogHeader>
              <DialogTitle>{{ t('admin.ai.suggestion.review') }}</DialogTitle>
            </DialogHeader>
            <div class="space-y-4">
              <div class="space-y-2">
                <Label>{{ t('globals.terms.question') }}</Label>
                <Input v-model="reviewQuestion" />
              </div>
              <div class="space-y-2">
                <Label>{{ t('globals.terms.answer') }}</Label>
                <Textarea v-model="reviewAnswer" :rows="6" />
              </div>
              <p v-if="reviewUUID" class="text-sm">
                <a
                  :href="`/inboxes/all/conversation/${reviewUUID}`"
                  target="_blank"
                  class="text-primary hover:underline"
                >
                  {{ t('admin.ai.suggestion.viewSource') }}
                </a>
              </p>
            </div>
            <DialogFooter class="gap-2">
              <Button variant="outline" :disabled="submitting" @click="reject">
                {{ t('globals.messages.reject') }}
              </Button>
              <Button :disabled="submitting" @click="approve">
                {{ t('globals.messages.approve') }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <DataTable
          :columns="createSuggestionColumns(t, { onReview: openReview })"
          :data="suggestions"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.ai.faqLearning.help') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ t('globals.terms.learnMore') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { Input } from '@shared-ui/components/ui/input/index.js'
import { Textarea } from '@shared-ui/components/ui/textarea/index.js'
import { Label } from '@shared-ui/components/ui/label/index.js'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle
} from '@shared-ui/components/ui/dialog/index.js'
import SwitchField from '@shared-ui/components/SwitchField.vue'
import { createSuggestionColumns } from '@/features/admin/ai/suggestionColumns.js'
import { DotLoader } from '@shared-ui/components/ui/loader'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const submitting = ref(false)

const suggestions = ref([])
const learningEnabled = ref(false)

const reviewOpen = ref(false)
const reviewId = ref(null)
const reviewQuestion = ref('')
const reviewAnswer = ref('')
const reviewUUID = ref('')
const isLoadingLearning = ref(false)

onMounted(() => {
  getSuggestions()
  getLearning()
})

const getSuggestions = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAIFaqSuggestions('pending')
    suggestions.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const getLearning = async () => {
  try {
    isLoadingLearning.value = true
    const resp = await api.getAIFaqLearning()
    learningEnabled.value = !!resp.data.data?.enabled
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoadingLearning.value = false
  }
}

const toggleLearning = async (val) => {
  try {
    await api.updateAIFaqLearning({ enabled: val })
    learningEnabled.value = val
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const openReview = (item) => {
  reviewId.value = item.id
  reviewQuestion.value = item.question
  reviewAnswer.value = item.answer
  reviewUUID.value = item.conversation_uuid || ''
  reviewOpen.value = true
}

const approve = async () => {
  if (submitting.value) return
  try {
    submitting.value = true
    await api.approveAIFaqSuggestion(reviewId.value, {
      question: reviewQuestion.value,
      answer: reviewAnswer.value
    })
    reviewOpen.value = false
    getSuggestions()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('admin.ai.suggestion.approved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    submitting.value = false
  }
}

const reject = async () => {
  if (submitting.value) return
  try {
    submitting.value = true
    await api.rejectAIFaqSuggestion(reviewId.value)
    reviewOpen.value = false
    getSuggestions()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('admin.ai.suggestion.rejected')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    submitting.value = false
  }
}
</script>
