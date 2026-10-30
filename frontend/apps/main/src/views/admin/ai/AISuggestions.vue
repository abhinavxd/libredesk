<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="mb-6">
          <DotLoader v-if="isLoadingLearning" />
          <SwitchField
            v-else
            :title="i18n._('Learn from resolved conversations')"
            :description="i18n._('When on, new suggestions appear here for you to review.')"
            :checked="learningEnabled"
            @update:checked="toggleLearning"
          />
        </div>

        <Dialog v-model:open="reviewOpen">
          <DialogContent class="sm:max-w-[560px]">
            <DialogHeader>
              <DialogTitle>{{ i18n._('Review suggestion') }}</DialogTitle>
            </DialogHeader>
            <div class="space-y-4">
              <div class="space-y-2">
                <Label>{{ i18n._('Question') }}</Label>
                <Input v-model="reviewQuestion" />
              </div>
              <div class="space-y-2">
                <Label>{{ i18n._('Answer') }}</Label>
                <Textarea v-model="reviewAnswer" :rows="6" />
              </div>
              <p v-if="reviewUUID" class="text-sm">
                <a
                  :href="`/inboxes/all/conversation/${reviewUUID}`"
                  target="_blank"
                  class="text-foreground font-medium hover:underline"
                >
                  {{ i18n.pgettext('AI suggestion action', 'View conversation') }}
                </a>
              </p>
            </div>
            <DialogFooter class="gap-2">
              <Button variant="outline" :disabled="submitting" @click="reject">
                {{ i18n._('Reject') }}
              </Button>
              <Button :disabled="submitting" @click="approve">
                {{ i18n._('Approve') }}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <AdminToolbar v-model:search="searchTerm" searchable class="mb-5" />
        <DataTable
          v-model:search="searchTerm"
          :searchable="false"
          :columns="createSuggestionColumns(i18n, { onReview: openReview })"
          :data="suggestions"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ i18n._('Suggestions are drafted from resolved conversations where a human agent gave a reusable answer. Review each one; approving it adds a snippet the assistant can use.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#learn-from-resolved-conversations"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ i18n._('Learn more') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

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
import { useI18n } from '@shared-ui/i18n'
import api from '@/api'

const searchTerm = ref('')
const i18n = useI18n()
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
      description: i18n._('Changes saved')
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
      description: i18n._('Suggestion approved and added to snippets')
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
      description: i18n._('Suggestion rejected')
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
