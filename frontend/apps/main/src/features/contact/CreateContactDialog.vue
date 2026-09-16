<template>
  <Dialog :open="open" @update:open="$emit('update:open', $event)">
    <DialogContent class="sm:max-w-2xl">
      <DialogHeader>
        <DialogTitle>{{ t('contact.new') }}</DialogTitle>
      </DialogHeader>
      <ContactForm
        :formLoading="loading"
        :onSubmit="onSubmit"
        :submitLabel="t('globals.messages.create')"
      />
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@shared-ui/components/ui/dialog'
import ContactForm from './ContactForm.vue'
import { createFormSchema } from './formSchema.js'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const props = defineProps({
  open: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:open', 'created'])

const { t } = useI18n()
const emitter = useEmitter()
const loading = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

watch(
  () => props.open,
  (val) => {
    if (!val) form.resetForm()
  }
)

const onSubmit = form.handleSubmit(async (values) => {
  loading.value = true
  try {
    const { data } = await api.createContact(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('globals.messages.savedSuccessfully') })
    emit('update:open', false)
    emit('created', data.data)
  } catch (err) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(err).message
    })
  } finally {
    loading.value = false
  }
})
</script>
