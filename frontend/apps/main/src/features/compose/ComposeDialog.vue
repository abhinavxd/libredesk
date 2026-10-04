<template>
  <Dialog :open="composeStore.isOpen" @update:open="(open) => !open && requestClose()">
    <DialogContent
      class="flex max-h-[100dvh] flex-col gap-0 overflow-hidden p-0 max-md:h-[100dvh] max-md:max-w-none max-md:rounded-none md:h-[min(46rem,90dvh)] md:max-w-3xl"
      @escapeKeyDown.prevent="requestClose"
      @pointerDownOutside.prevent
    >
      <DialogHeader class="border-b px-5 py-4">
        <DialogTitle>{{ isResubmit ? $t('compose.editTitle') : $t('compose.title') }}</DialogTitle>
        <DialogDescription v-if="reviewStore.needsReview" class="flex items-center gap-1.5 text-review">
          <ShieldCheck class="size-4 shrink-0" aria-hidden="true" />{{ $t('compose.reviewNotice') }}
        </DialogDescription>
        <DialogDescription v-else class="sr-only">{{ $t('compose.title') }}</DialogDescription>
      </DialogHeader>

      <div v-if="!sendableAddresses.length" class="flex flex-1 items-center justify-center p-8 text-center text-sm text-muted-foreground">
        {{ $t('compose.noAddresses') }}
      </div>

      <form v-else id="compose-form" class="flex min-h-0 flex-1 flex-col" @submit.prevent="send">
        <div class="space-y-2 border-b px-5 py-3 text-sm">
          <label class="flex items-center gap-3">
            <span class="w-14 shrink-0 text-sm text-muted-foreground">{{ $t('compose.from') }}</span>
            <NativeSelect v-model.number="form.addressID" class="min-w-0 flex-1" :disabled="sending" required>
              <option v-for="address in sendableAddresses" :key="address.id" :value="address.id">
                {{ addressLabel(address) }} &lt;{{ address.address }}&gt;
              </option>
            </NativeSelect>
          </label>
          <label class="flex items-center gap-3">
            <span class="w-14 shrink-0 text-sm text-muted-foreground">{{ $t('compose.to') }}</span>
            <Input v-model="form.to" type="text" inputmode="email" autocomplete="off" class="min-w-0 flex-1" :placeholder="$t('compose.recipientsPlaceholder')" :disabled="sending" />
            <Button v-if="!showCopies" type="button" variant="ghost" size="sm" @click="showCopies = true">{{ $t('compose.addCopies') }}</Button>
          </label>
          <template v-if="showCopies">
            <label class="flex items-center gap-3">
              <span class="w-14 shrink-0 text-sm text-muted-foreground">{{ $t('compose.cc') }}</span>
              <Input v-model="form.cc" type="text" inputmode="email" autocomplete="off" class="min-w-0 flex-1" :disabled="sending" />
            </label>
            <label class="flex items-center gap-3">
              <span class="w-14 shrink-0 text-sm text-muted-foreground">{{ $t('compose.bcc') }}</span>
              <Input v-model="form.bcc" type="text" inputmode="email" autocomplete="off" class="min-w-0 flex-1" :disabled="sending" />
            </label>
          </template>
          <label class="flex items-center gap-3">
            <span class="w-14 shrink-0 text-sm text-muted-foreground">{{ $t('compose.subject') }}</span>
            <Input v-model="form.subject" type="text" maxlength="998" autocomplete="off" class="min-w-0 flex-1" :disabled="sending" />
          </label>
        </div>

        <div class="flex min-h-0 flex-1 flex-col px-5 py-3">
          <Editor
            v-if="editorKey"
            :key="editorKey"
            ref="editorRef"
            v-model:htmlContent="form.html"
            v-model:textContent="form.text"
            :placeholder="$t('globals.terms.typeMessage')"
            :autoFocus="false"
            :disabled="sending"
            :enableInlineImages="true"
            :typingIndicator="false"
            @send="send"
            @filesDropped="uploadFiles"
          />
          <ActivationKeyCard
            v-if="editorKey"
            class="mt-2 shrink-0"
            :html="form.html"
            :editor="editorRef"
            :disabled="sending"
          />
          <ReplyBoxAttachmentPreview
            v-if="mediaFiles.length || uploadingFiles.length"
            :attachments="mediaFiles"
            :uploadingFiles="uploadingFiles"
            :onDelete="handleFileDelete"
            class="mt-2"
          />
        </div>

        <ul v-if="shownProblems.length" class="mx-5 mb-2 space-y-0.5 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive" role="alert">
          <li v-for="problem in shownProblems" :key="problem.key">{{ $t(problem.key, problem.values || {}) }}</li>
        </ul>

        <div class="flex items-center justify-between gap-2 border-t px-5 py-3">
          <div class="flex items-center gap-1">
            <input ref="fileInput" type="file" class="hidden" multiple @change="handleFileUpload" />
            <Button type="button" variant="ghost" size="icon" :aria-label="$t('globals.messages.attachFile')" :disabled="sending" @click="fileInput?.click()">
              <Paperclip class="size-4" />
            </Button>
          </div>
          <div class="flex items-center gap-2">
            <Button type="button" variant="outline" :disabled="sending" @click="requestClose">{{ $t('globals.messages.cancel') }}</Button>
            <Button type="submit" :disabled="sending || uploadingFiles.length > 0" :class="{ 'bg-review text-review-foreground hover:bg-review/90': reviewStore.needsReview }">
              <Send class="size-4" />{{ sending ? $t('compose.sending') : reviewStore.needsReview ? $t('compose.sendForReview') : $t('compose.send') }}
            </Button>
          </div>
        </div>
      </form>
    </DialogContent>
  </Dialog>

  <AlertDialog :open="confirmDiscard" @update:open="confirmDiscard = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>{{ $t('compose.discardTitle') }}</AlertDialogTitle>
        <AlertDialogDescription>{{ isResubmit ? $t('compose.discardReturnedDescription') : $t('compose.discardDescription') }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>{{ $t('compose.keepEditing') }}</AlertDialogCancel>
        <AlertDialogAction variant="destructive" @click="close">{{ $t('compose.discard') }}</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Paperclip, Send, ShieldCheck } from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { NativeSelect } from '@shared-ui/components/ui/native-select'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@shared-ui/components/ui/alert-dialog'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import Editor from '@main/components/editor/ConversationEditor.vue'
import ReplyBoxAttachmentPreview from '@/features/conversation/message/attachment/ReplyBoxAttachmentPreview.vue'
import ActivationKeyCard from '@/features/conversation/ActivationKeyCard.vue'
import { hasInlineImage, hasPendingInlineUpload } from '@main/composables/useInlineImageUpload'
import { useFileUpload } from '@main/composables/useFileUpload'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useAddressStore } from '@main/stores/address'
import { useComposeStore } from '@main/stores/compose'
import { useReviewStore } from '@main/stores/review'
import { addressLabel } from '@main/utils/address-display'
import { composeProblems, defaultFromAddress, joinRecipients, splitRecipients } from '@main/utils/compose'
import api from '@main/api'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const emitter = useEmitter()
const addressStore = useAddressStore()
const composeStore = useComposeStore()
const reviewStore = useReviewStore()
const editorRef = ref(null)

const emptyForm = () => ({ addressID: null, to: '', cc: '', bcc: '', subject: '', html: '', text: '' })
const form = ref(emptyForm())
const showCopies = ref(false)
const sending = ref(false)
const attempted = ref(false)
const confirmDiscard = ref(false)
const fileInput = ref(null)
// Remounts the editor so a prefill becomes its initial content.
const editorKey = ref(0)

const { uploadingFiles, mediaFiles, handleFileUpload, handleFileDelete, uploadFiles, setMediaFiles } = useFileUpload({
  linkedModel: 'messages'
})

const sendableAddresses = computed(() => addressStore.addresses.filter((address) => address.enabled))
const isResubmit = computed(() => Boolean(composeStore.prefill?.review_uuid))
const hasBody = computed(
  () => form.value.text.trim().length > 0 || hasInlineImage(form.value.html) || mediaFiles.value.length > 0
)
const problems = computed(() =>
  composeProblems({ ...form.value, addressID: form.value.addressID, hasBody: hasBody.value })
)
// Only point out problems once the user has tried to send.
const shownProblems = computed(() => (attempted.value ? problems.value : []))
const isDirty = computed(
  () =>
    Boolean(form.value.to || form.value.cc || form.value.bcc || form.value.subject) ||
    hasBody.value ||
    uploadingFiles.value.length > 0
)

function reset() {
  const prefill = composeStore.prefill
  form.value = {
    ...emptyForm(),
    addressID: prefill?.address_id ?? defaultFromAddress(sendableAddresses.value, route.params.addressID),
    to: joinRecipients(prefill?.to),
    cc: joinRecipients(prefill?.cc),
    bcc: joinRecipients(prefill?.bcc),
    subject: prefill?.subject || '',
    html: prefill?.content || ''
  }
  showCopies.value = Boolean(form.value.cc || form.value.bcc)
  setMediaFiles((prefill?.attachments || []).filter((file) => !file.inline).map((file) => ({ ...file })))
  attempted.value = false
  editorKey.value++
}

watch(
  () => composeStore.isOpen,
  async (open) => {
    if (!open) return
    await addressStore.fetchAddresses()
    reset()
  }
)

function close() {
  confirmDiscard.value = false
  composeStore.close()
}

function requestClose() {
  if (sending.value) return
  if (isDirty.value) confirmDiscard.value = true
  else close()
}

async function send() {
  if (sending.value) return
  attempted.value = true
  if (problems.value.length || hasPendingInlineUpload(form.value.html) || uploadingFiles.value.length) return
  sending.value = true
  const payload = {
    address_id: form.value.addressID,
    subject: form.value.subject.trim(),
    content: form.value.html,
    to: splitRecipients(form.value.to),
    cc: splitRecipients(form.value.cc),
    bcc: splitRecipients(form.value.bcc),
    attachments: mediaFiles.value.map((file) => file.id)
  }
  try {
    if (isResubmit.value) {
      await api.resubmitReview(composeStore.prefill.review_uuid, payload)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('review.toast.submitted') })
      reviewStore.fetchReviews()
      reviewStore.fetchCounts()
      close()
      return
    }
    const result = (await api.composeEmail(payload)).data?.data || {}
    close()
    if (result.sent) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('compose.toast.sent') })
      router.push({
        name: 'address-inbox-conversation',
        params: { addressID: payload.address_id, uuid: result.conversation_uuid }
      })
    } else {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('review.toast.submitted') })
      reviewStore.fetchCounts()
      if (reviewStore.loaded) reviewStore.fetchReviews()
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { variant: 'destructive', description: handleHTTPError(error).message })
  } finally {
    sending.value = false
  }
}
</script>
